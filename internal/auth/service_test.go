package auth

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"bank/internal/audit"
)

type stubHasher struct{}

func (stubHasher) Hash(password string) (string, error) { return "h:" + password, nil }
func (stubHasher) Compare(hash, password string) (bool, error) {
	return hash == "h:"+password, nil
}

func testService(store Store) *Service {
	tokens := NewTokenManager("dev-change-me-use-at-least-32-bytes", 15*time.Minute)
	return NewService(store, stubHasher{}, tokens, 7*24*time.Hour)
}

func TestRegisterAndLogin(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	svc := testService(store)

	_, err := svc.Register(ctx, RegisterInput{Email: "Ada@Example.com", Password: "password1"})
	if err != nil {
		t.Fatal(err)
	}
	sess, err := svc.Login(ctx, LoginInput{Email: "ada@example.com", Password: "password1"})
	if err != nil {
		t.Fatal(err)
	}
	if sess.AccessToken == "" || sess.RefreshToken == "" || sess.Customer.Email != "ada@example.com" {
		t.Fatalf("unexpected session: %+v", sess)
	}
	if !slices.Contains(store.actions(), audit.Register) || !slices.Contains(store.actions(), audit.LoginSuccess) {
		t.Fatalf("audit log missing: %v", store.actions())
	}
	events, err := svc.ListAudit(ctx, sess.Customer.ID, 50, 0)
	if err != nil || len(events) < 2 {
		t.Fatalf("list audit %+v %v", events, err)
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	ctx := context.Background()
	svc := testService(newMemStore())
	in := RegisterInput{Email: "a@b.com", Password: "password1"}
	if _, err := svc.Register(ctx, in); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Register(ctx, in); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("got %v", err)
	}
}

func TestRegisterRejectsShortPassword(t *testing.T) {
	svc := testService(newMemStore())
	_, err := svc.Register(context.Background(), RegisterInput{Email: "a@b.com", Password: "short"})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("got %v", err)
	}
}

func TestLoginUnknownUserAndWrongPassword(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	svc := testService(store)
	if _, err := svc.Register(ctx, RegisterInput{Email: "a@b.com", Password: "password1"}); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Login(ctx, LoginInput{Email: "nobody@b.com", Password: "password1"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("unknown user: %v", err)
	}
	_, err = svc.Login(ctx, LoginInput{Email: "a@b.com", Password: "nope-nope"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password: %v", err)
	}
	if !slices.Contains(store.actions(), audit.LoginFailed) {
		t.Fatalf("expected failed login audit, got %v", store.actions())
	}
}

func TestLoginLockedAccount(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	svc := testService(store)
	if _, err := svc.Register(ctx, RegisterInput{Email: "a@b.com", Password: "password1"}); err != nil {
		t.Fatal(err)
	}
	store.lock("a@b.com")
	_, err := svc.Login(ctx, LoginInput{Email: "a@b.com", Password: "password1"})
	if !errors.Is(err, ErrLocked) {
		t.Fatalf("got %v", err)
	}
}

func TestRefreshRotation(t *testing.T) {
	ctx := context.Background()
	svc := testService(newMemStore())
	first, err := svc.Register(ctx, RegisterInput{Email: "a@b.com", Password: "password1"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Refresh(ctx, first.RefreshToken, "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	if second.RefreshToken == first.RefreshToken || second.AccessToken == "" {
		t.Fatal("expected new refresh and access tokens")
	}
	_, err = svc.Refresh(ctx, first.RefreshToken, "127.0.0.1", "test")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("reused refresh: %v", err)
	}
	third, err := svc.Refresh(ctx, second.RefreshToken, "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	if third.AccessToken == "" {
		t.Fatal("expected rotated session")
	}
}

func TestRefreshExpired(t *testing.T) {
	ctx := context.Background()
	svc := testService(newMemStore())
	sess, err := svc.Register(ctx, RegisterInput{Email: "a@b.com", Password: "password1"})
	if err != nil {
		t.Fatal(err)
	}
	svc.now = func() time.Time { return time.Now().Add(8 * 24 * time.Hour) }
	_, err = svc.Refresh(ctx, sess.RefreshToken, "", "")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
}

func TestLogoutRevokesRefresh(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	svc := testService(store)
	sess, err := svc.Register(ctx, RegisterInput{Email: "a@b.com", Password: "password1"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Logout(ctx, sess.RefreshToken, "", "", &sess.Customer.ID); err != nil {
		t.Fatal(err)
	}
	_, err = svc.Refresh(ctx, sess.RefreshToken, "", "")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
	if !slices.Contains(store.actions(), audit.Logout) {
		t.Fatalf("expected logout audit, got %v", store.actions())
	}
}

func TestMe(t *testing.T) {
	ctx := context.Background()
	svc := testService(newMemStore())
	sess, err := svc.Register(ctx, RegisterInput{Email: "a@b.com", Password: "password1"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := svc.Me(ctx, sess.Customer.ID)
	if err != nil || c.Email != "a@b.com" {
		t.Fatalf("got %+v %v", c, err)
	}
}
