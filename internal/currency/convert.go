package currency

import (
	"errors"
	"math/big"
	"strings"
)

var ErrInvalidRate = errors.New("invalid rate")

func RateE8(raw string) (int64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, ErrInvalidRate
	}
	neg := strings.HasPrefix(s, "-")
	if neg {
		return 0, ErrInvalidRate
	}
	parts := strings.SplitN(s, ".", 2)
	whole := parts[0]
	if whole == "" {
		whole = "0"
	}
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
	}
	if len(frac) > 8 {
		frac = frac[:8]
	}
	for _, c := range whole + frac {
		if c < '0' || c > '9' {
			return 0, ErrInvalidRate
		}
	}
	frac = frac + strings.Repeat("0", 8-len(frac))
	n := new(big.Int)
	if _, ok := n.SetString(whole+frac, 10); !ok {
		return 0, ErrInvalidRate
	}
	if !n.IsInt64() || n.Sign() <= 0 {
		return 0, ErrInvalidRate
	}
	return n.Int64(), nil
}

func CrossRateE8(fromPerUSD, toPerUSD int64) (int64, error) {
	if fromPerUSD <= 0 || toPerUSD <= 0 {
		return 0, ErrInvalidRate
	}
	n := big.NewInt(toPerUSD)
	n.Mul(n, big.NewInt(ScaleE8))
	n.Add(n, big.NewInt(fromPerUSD/2))
	n.Div(n, big.NewInt(fromPerUSD))
	if !n.IsInt64() || n.Sign() <= 0 {
		return 0, ErrInvalidRate
	}
	return n.Int64(), nil
}

func ConvertCents(fromCents, rateE8 int64) (int64, error) {
	if fromCents <= 0 || rateE8 <= 0 {
		return 0, ErrInvalidRate
	}
	n := big.NewInt(fromCents)
	n.Mul(n, big.NewInt(rateE8))
	n.Add(n, big.NewInt(ScaleE8/2))
	n.Div(n, big.NewInt(ScaleE8))
	if !n.IsInt64() || n.Sign() <= 0 {
		return 0, ErrInvalidRate
	}
	return n.Int64(), nil
}

func FormatRate(rateE8 int64) string {
	if rateE8 <= 0 {
		return ""
	}
	whole := rateE8 / ScaleE8
	frac := rateE8 % ScaleE8
	s := strings.TrimRight(strings.TrimRight(pad8(frac), "0"), ".")
	if s == "" {
		return itoa(whole)
	}
	return itoa(whole) + "." + s
}

func pad8(n int64) string {
	s := itoa(n)
	if len(s) >= 8 {
		return s
	}
	return strings.Repeat("0", 8-len(s)) + s
}

func itoa(n int64) string {
	return strings.TrimPrefix(big.NewInt(n).String(), "+")
}
