package auth

import (
	"testing"
	"time"
)

func TestLimiterAllowsThenBlocks(t *testing.T) {
	l := NewLimiter()
	key := "127.0.0.1\x00a@b.com"
	for i := 0; i < 10; i++ {
		if !l.Allow(key, 10, time.Minute) {
			t.Fatalf("attempt %d should be allowed", i)
		}
	}
	if l.Allow(key, 10, time.Minute) {
		t.Fatal("expected rate limit")
	}
	if !l.Allow("other", 10, time.Minute) {
		t.Fatal("different key should be independent")
	}
}
