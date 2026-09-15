package fx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"bank/internal/currency"
)

const (
	DefaultBase = "https://api.frankfurter.dev/v1"
	cacheTTL    = 60 * time.Second
)

type Table struct {
	Date    string
	PerUSD  map[currency.Code]int64
	Fetched time.Time
}

type Client struct {
	base   string
	http   *http.Client
	mu     sync.Mutex
	cached *Table
}

func NewClient(base string, h *http.Client) *Client {
	if strings.TrimSpace(base) == "" {
		base = DefaultBase
	}
	if h == nil {
		h = &http.Client{Timeout: 8 * time.Second}
	}
	return &Client{base: strings.TrimRight(base, "/"), http: h}
}

func (c *Client) Snapshot(ctx context.Context) (*Table, error) {
	c.mu.Lock()
	if c.cached != nil && time.Since(c.cached.Fetched) < cacheTTL {
		t := *c.cached
		c.mu.Unlock()
		return &t, nil
	}
	c.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/latest?from=USD", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "TheBankDemo/1.0")
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<16))
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rates status %d", res.StatusCode)
	}
	dec := json.NewDecoder(strings.NewReader(string(body)))
	dec.UseNumber()
	var payload struct {
		Date  string                 `json:"date"`
		Rates map[string]json.Number `json:"rates"`
	}
	if err := dec.Decode(&payload); err != nil {
		return nil, err
	}
	per := map[currency.Code]int64{currency.USD: currency.ScaleE8}
	for rawCode, raw := range payload.Rates {
		code, ok := currency.Parse(rawCode)
		if !ok || code == currency.USD {
			continue
		}
		n, err := currency.RateE8(raw.String())
		if err != nil {
			continue
		}
		per[code] = n
	}
	t := &Table{Date: payload.Date, PerUSD: per, Fetched: time.Now().UTC()}
	c.mu.Lock()
	c.cached = t
	c.mu.Unlock()
	copyT := *t
	return &copyT, nil
}
