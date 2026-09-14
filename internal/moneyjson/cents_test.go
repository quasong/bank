package moneyjson

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPositiveCents(t *testing.T) {
	n, err := PositiveCents([]byte("2500"))
	if err != nil || n != 2500 {
		t.Fatalf("got %d %v", n, err)
	}
	if _, err := PositiveCents([]byte("10.5")); err != ErrNotIntegerCents {
		t.Fatalf("float: %v", err)
	}
	if _, err := PositiveCents([]byte("0")); err != ErrNotIntegerCents {
		t.Fatalf("zero: %v", err)
	}
	if _, err := PositiveCents([]byte("-3")); err != ErrNotIntegerCents {
		t.Fatalf("neg: %v", err)
	}
}

func TestDecodeBody(t *testing.T) {
	var out map[string]json.RawMessage
	if err := Decode(strings.NewReader(`{"amount_cents":100,"idempotency_key":"k"}`), &out); err != nil {
		t.Fatal(err)
	}
	n, err := PositiveCents(out["amount_cents"])
	if err != nil || n != 100 {
		t.Fatalf("got %d %v", n, err)
	}
}
