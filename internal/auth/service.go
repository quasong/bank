package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"bank/internal/audit"
	"bank/internal/customer"
)

const (
	minPasswordRunes = 8
	maxPasswordRunes = 72
	maxEmailLen      = 254
)

type Session struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	RefreshExp   time.Time
	Customer     customer.Customer
}

type Service struct {
	store      Store
	hasher     Hasher
	tokens     *TokenManager
	refreshTTL time.Duration
	now        func() time.Time
}

func NewService(store Store, hasher Hasher, tokens *TokenManager, refreshTTL time.Duration) *Service {
	return &Service{
		store:      store,
		hasher:     hasher,
		tokens:     tokens,
		refreshTTL: refreshTTL,
		now:        time.Now,
	}
}

type RegisterInput struct {
	Email     string
	Password  string
	IP        string
	UserAgent string
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (Session, error) {
	email, err := normalizeEmail(in.Email)
	if err != nil {
		return Session{}, err
	}
	if err := validatePassword(in.Password); err != nil {
		return Session{}, err
	}
	hash, err := s.hasher.Hash(in.Password)
	if err != nil {
		return Session{}, err
	}
	id := uuid.New()
	cust, err := s.store.CreateCustomer(ctx, id, email, hash, customer.StatusActive)
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			return Session{}, ErrEmailTaken
		}
		return Session{}, err
	}
	_ = s.store.InsertAudit(ctx, AuditRecord{
		ID:        uuid.New(),
		ActorID:   &cust.ID,
		Action:    audit.Register,
		IP:        in.IP,
		UserAgent: in.UserAgent,
		Metadata:  map[string]string{"email": email},
	})
	return s.issueSession(ctx, cust, in.IP, in.UserAgent)
}

type LoginInput struct {
	Email     string
	Password  string
	IP        string
	UserAgent string
}

func (s *Service) Login(ctx context.Context, in LoginInput) (Session, error) {
	email, err := normalizeEmail(in.Email)
	if err != nil {
		return Session{}, ErrInvalidCredentials
	}
	rec, err := s.store.GetCustomerByEmail(ctx, email)
	if err != nil {
		_ = s.auditFail(ctx, nil, in.IP, in.UserAgent, email)
		if errors.Is(err, ErrNotFound) {
			return Session{}, ErrInvalidCredentials
		}
		return Session{}, err
	}
	ok, err := s.hasher.Compare(rec.PasswordHash, in.Password)
	if err != nil {
		return Session{}, err
	}
	if !ok {
		_ = s.auditFail(ctx, &rec.Customer.ID, in.IP, in.UserAgent, email)
		return Session{}, ErrInvalidCredentials
	}
	if !rec.Customer.Status.Active() {
		_ = s.auditFail(ctx, &rec.Customer.ID, in.IP, in.UserAgent, email)
		return Session{}, ErrLocked
	}
	sess, err := s.issueSession(ctx, rec.Customer, in.IP, in.UserAgent)
	if err != nil {
		return Session{}, err
	}
	_ = s.store.InsertAudit(ctx, AuditRecord{
		ID:        uuid.New(),
		ActorID:   &rec.Customer.ID,
		Action:    audit.LoginSuccess,
		IP:        in.IP,
		UserAgent: in.UserAgent,
		Metadata:  map[string]string{"email": email},
	})
	return sess, nil
}

func (s *Service) Refresh(ctx context.Context, plaintext, ip, userAgent string) (Session, error) {
	if strings.TrimSpace(plaintext) == "" {
		return Session{}, ErrUnauthorized
	}
	now := s.now()
	nextPlain, err := NewRefreshPlaintext()
	if err != nil {
		return Session{}, err
	}
	nextID := uuid.New()
	old, err := s.store.RotateRefreshToken(ctx, HashRefresh(plaintext), NewRefresh{
		ID:        nextID,
		TokenHash: HashRefresh(nextPlain),
		ExpiresAt: now.Add(s.refreshTTL),
		IP:        ip,
		UserAgent: userAgent,
	}, now)
	if err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrRefreshInvalid) {
			return Session{}, ErrUnauthorized
		}
		return Session{}, err
	}
	rec, err := s.store.GetCustomerByID(ctx, old.CustomerID)
	if err != nil {
		return Session{}, err
	}
	if !rec.Customer.Status.Active() {
		return Session{}, ErrLocked
	}
	access, _, err := s.tokens.IssueAccess(rec.Customer.ID, rec.Customer.Email, now)
	if err != nil {
		return Session{}, err
	}
	return Session{
		AccessToken:  access,
		RefreshToken: nextPlain,
		ExpiresIn:    int(s.tokens.AccessTTL().Seconds()),
		RefreshExp:   now.Add(s.refreshTTL),
		Customer:     rec.Customer,
	}, nil
}

func (s *Service) Logout(ctx context.Context, plaintext, ip, userAgent string, actorID *uuid.UUID) error {
	if strings.TrimSpace(plaintext) == "" {
		return nil
	}
	if err := s.store.RevokeRefreshTokenByHash(ctx, HashRefresh(plaintext), s.now()); err != nil {
		return err
	}
	_ = s.store.InsertAudit(ctx, AuditRecord{
		ID:        uuid.New(),
		ActorID:   actorID,
		Action:    audit.Logout,
		IP:        ip,
		UserAgent: userAgent,
	})
	return nil
}

func (s *Service) Me(ctx context.Context, id uuid.UUID) (customer.Customer, error) {
	rec, err := s.store.GetCustomerByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return customer.Customer{}, ErrUnauthorized
		}
		return customer.Customer{}, err
	}
	if !rec.Customer.Status.Active() {
		return customer.Customer{}, ErrLocked
	}
	return rec.Customer, nil
}

func (s *Service) ListAudit(ctx context.Context, actorID uuid.UUID, limit, offset int32) ([]AuditRecord, error) {
	if actorID == uuid.Nil {
		return nil, ErrUnauthorized
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	out, err := s.store.ListAudit(ctx, actorID, limit, offset)
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = []AuditRecord{}
	}
	return out, nil
}

func (s *Service) ParseAccess(token string) (uuid.UUID, error) {
	id, _, err := s.tokens.ParseAccess(token)
	return id, err
}

func (s *Service) issueSession(ctx context.Context, cust customer.Customer, ip, userAgent string) (Session, error) {
	now := s.now()
	access, _, err := s.tokens.IssueAccess(cust.ID, cust.Email, now)
	if err != nil {
		return Session{}, err
	}
	plain, err := NewRefreshPlaintext()
	if err != nil {
		return Session{}, err
	}
	rec := NewRefresh{
		ID:         uuid.New(),
		CustomerID: cust.ID,
		TokenHash:  HashRefresh(plain),
		ExpiresAt:  now.Add(s.refreshTTL),
		IP:         ip,
		UserAgent:  userAgent,
	}
	if err := s.store.InsertRefreshToken(ctx, rec); err != nil {
		return Session{}, err
	}
	return Session{
		AccessToken:  access,
		RefreshToken: plain,
		ExpiresIn:    int(s.tokens.AccessTTL().Seconds()),
		RefreshExp:   rec.ExpiresAt,
		Customer:     cust,
	}, nil
}

func (s *Service) auditFail(ctx context.Context, actor *uuid.UUID, ip, ua, email string) error {
	return s.store.InsertAudit(ctx, AuditRecord{
		ID:        uuid.New(),
		ActorID:   actor,
		Action:    audit.LoginFailed,
		IP:        ip,
		UserAgent: ua,
		Metadata:  map[string]string{"email": email},
	})
}

func normalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" || len(email) > maxEmailLen {
		return "", ErrInvalidRequest
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "", ErrInvalidRequest
	}
	return email, nil
}

func validatePassword(password string) error {
	n := utf8.RuneCountInString(password)
	if n < minPasswordRunes || n > maxPasswordRunes {
		return ErrInvalidRequest
	}
	return nil
}

func MetadataJSON(m map[string]string) []byte {
	if m == nil {
		return []byte("{}")
	}
	b, err := json.Marshal(m)
	if err != nil {
		return []byte("{}")
	}
	return b
}
