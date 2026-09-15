package ledger

import (
	"strings"
	"testing"
)

func TestNormalizeNote(t *testing.T) {
	got, err := NormalizeNote("  Rent   May  ")
	if err != nil || got != "Rent May" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := NormalizeNote(strings.Repeat("n", MaxNoteLen+1)); err != ErrInvalidNote {
		t.Fatalf("got %v", err)
	}
	ok, err := NormalizeNote(strings.Repeat("n", MaxNoteLen))
	if err != nil || len(ok) != MaxNoteLen {
		t.Fatalf("%q %v", ok, err)
	}
	empty, err := NormalizeNote("   \t")
	if err != nil || empty != "" {
		t.Fatalf("%q %v", empty, err)
	}
}
