package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTokenManagerIssueAndParse(t *testing.T) {
	m := NewTokenManager("dev-change-me-use-at-least-32-bytes", 15*time.Minute)
	id := uuid.New()
	tok, exp, err := m.IssueAccess(id, "a@b.com", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if exp.Before(time.Now()) {
		t.Fatal("expiry should be in the future")
	}
	gotID, email, err := m.ParseAccess(tok)
	if err != nil {
		t.Fatal(err)
	}
	if gotID != id || email != "a@b.com" {
		t.Fatalf("got %s %s", gotID, email)
	}
}

func TestTokenManagerExpired(t *testing.T) {
	m := NewTokenManager("dev-change-me-use-at-least-32-bytes", 15*time.Minute)
	tok, _, err := m.IssueAccess(uuid.New(), "a@b.com", time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := m.ParseAccess(tok); err != ErrUnauthorized {
		t.Fatalf("got %v", err)
	}
}

func TestTokenManagerRejectsGarbage(t *testing.T) {
	m := NewTokenManager("dev-change-me-use-at-least-32-bytes", 15*time.Minute)
	if _, _, err := m.ParseAccess("not-a-jwt"); err != ErrUnauthorized {
		t.Fatalf("got %v", err)
	}
}

func TestHashRefreshStable(t *testing.T) {
	if HashRefresh("abc") != HashRefresh("abc") {
		t.Fatal("hash should be deterministic")
	}
	if HashRefresh("abc") == HashRefresh("abd") {
		t.Fatal("different inputs should not collide in this test")
	}
}
