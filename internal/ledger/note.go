package ledger

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const MaxNoteLen = 40

var ErrInvalidNote = errors.New("invalid note")

// NormalizeNote trims and collapses whitespace. Empty is allowed.
func NormalizeNote(s string) (string, error) {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) > MaxNoteLen {
		return "", ErrInvalidNote
	}
	return s, nil
}
