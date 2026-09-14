package auth

import (
	"sync"
	"time"
)

type Limiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
}

func NewLimiter() *Limiter {
	return &Limiter{attempts: make(map[string][]time.Time)}
}

func (l *Limiter) Allow(key string, n int, window time.Duration) bool {
	now := time.Now()
	cutoff := now.Add(-window)
	l.mu.Lock()
	defer l.mu.Unlock()
	hits := l.attempts[key]
	kept := hits[:0]
	for _, t := range hits {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= n {
		l.attempts[key] = kept
		return false
	}
	l.attempts[key] = append(kept, now)
	return true
}
