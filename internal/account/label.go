package account

import (
	"strings"
	"unicode/utf8"
)

const (
	MaxJarLabelLen     = 20
	MaxJarsPerCurrency = 8
	DefaultJarLabel    = "Jar"
)

func NormalizeLabel(s string) (string, error) {
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return DefaultJarLabel, nil
	}
	if utf8.RuneCountInString(s) > MaxJarLabelLen {
		return "", ErrInvalidRequest
	}
	return s, nil
}
