package currency

import (
	"strings"

	"github.com/google/uuid"
)

type Code string

const (
	USD Code = "USD"
	EUR Code = "EUR"
	GBP Code = "GBP"
)

const (
	ScaleE8    int64 = 100_000_000
	SortCode         = "040004"
	BankCode         = "THEB"
	BIC              = "THEBGB2L"
	RoutingABA       = "121000248"
	CountryGB        = "GB"
)

var (
	VaultUSD = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	VaultEUR = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	VaultGBP = uuid.MustParse("33333333-3333-3333-3333-333333333333")
)

func Parse(raw string) (Code, bool) {
	switch Code(strings.ToUpper(strings.TrimSpace(raw))) {
	case USD:
		return USD, true
	case EUR:
		return EUR, true
	case GBP:
		return GBP, true
	default:
		return "", false
	}
}

func (c Code) Valid() bool {
	_, ok := Parse(string(c))
	return ok
}

func (c Code) Symbol() string {
	switch c {
	case EUR:
		return "€"
	case GBP:
		return "£"
	default:
		return "$"
	}
}

func Vault(c Code) uuid.UUID {
	switch c {
	case EUR:
		return VaultEUR
	case GBP:
		return VaultGBP
	default:
		return VaultUSD
	}
}

func All() []Code {
	return []Code{USD, EUR, GBP}
}

func (c Code) Rank() int {
	switch c {
	case USD:
		return 0
	case EUR:
		return 1
	case GBP:
		return 2
	default:
		return 9
	}
}
