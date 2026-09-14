package auth

import (
	"testing"
)

func TestArgon2Hasher(t *testing.T) {
	h := NewArgon2Hasher()
	hash, err := h.Hash("correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	ok, err := h.Compare(hash, "correct-horse")
	if err != nil || !ok {
		t.Fatalf("expected match, ok=%v err=%v", ok, err)
	}
	ok, err = h.Compare(hash, "wrong-password")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected mismatch")
	}
}
