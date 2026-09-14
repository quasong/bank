package moneyjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

var ErrNotIntegerCents = errors.New("amount_cents must be a positive integer")

func Decode(r io.Reader, dst *map[string]json.RawMessage) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var m map[string]json.RawMessage
	if err := dec.Decode(&m); err != nil {
		return err
	}
	*dst = m
	return nil
}

func PositiveCents(raw json.RawMessage) (int64, error) {
	if len(raw) == 0 || bytes.Contains(raw, []byte{'.'}) || bytes.Contains(raw, []byte{'e'}) || bytes.Contains(raw, []byte{'E'}) || bytes.Contains(raw, []byte{'"'}) {
		return 0, ErrNotIntegerCents
	}
	var n int64
	if err := json.Unmarshal(raw, &n); err != nil {
		return 0, ErrNotIntegerCents
	}
	if n <= 0 {
		return 0, ErrNotIntegerCents
	}
	return n, nil
}

func String(raw json.RawMessage) (string, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", err
	}
	return s, nil
}
