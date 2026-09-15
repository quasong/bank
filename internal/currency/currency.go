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
	AUD Code = "AUD"
	BGN Code = "BGN"
	BRL Code = "BRL"
	CAD Code = "CAD"
	CHF Code = "CHF"
	CNY Code = "CNY"
	CZK Code = "CZK"
	DKK Code = "DKK"
	HKD Code = "HKD"
	ILS Code = "ILS"
	INR Code = "INR"
	MXN Code = "MXN"
	MYR Code = "MYR"
	NOK Code = "NOK"
	NZD Code = "NZD"
	PHP Code = "PHP"
	PLN Code = "PLN"
	RON Code = "RON"
	SEK Code = "SEK"
	SGD Code = "SGD"
	THB Code = "THB"
	TRY Code = "TRY"
	ZAR Code = "ZAR"
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
	vaultNS  = uuid.MustParse("7b7a6e10-4c2f-4e1a-9d3b-0a1b2c3d4e5f")
)

type Meta struct {
	Name   string
	Short  string
	Symbol string
}

var order = []Code{
	USD, EUR, GBP,
	AUD, BGN, BRL, CAD, CHF, CNY, CZK, DKK, HKD, ILS, INR,
	MXN, MYR, NOK, NZD, PHP, PLN, RON, SEK, SGD, THB, TRY, ZAR,
}

var catalog = map[Code]Meta{
	USD: {Name: "US dollar", Short: "Dollar", Symbol: "$"},
	EUR: {Name: "Euro", Short: "Euro", Symbol: "€"},
	GBP: {Name: "British pound", Short: "Pound", Symbol: "£"},
	AUD: {Name: "Australian dollar", Short: "Aussie dollar", Symbol: "A$"},
	BGN: {Name: "Bulgarian lev", Short: "Lev", Symbol: "лв"},
	BRL: {Name: "Brazilian real", Short: "Real", Symbol: "R$"},
	CAD: {Name: "Canadian dollar", Short: "Canadian dollar", Symbol: "C$"},
	CHF: {Name: "Swiss franc", Short: "Franc", Symbol: "CHF"},
	CNY: {Name: "Chinese yuan", Short: "Yuan", Symbol: "¥"},
	CZK: {Name: "Czech koruna", Short: "Koruna", Symbol: "Kč"},
	DKK: {Name: "Danish krone", Short: "Krone", Symbol: "kr"},
	HKD: {Name: "Hong Kong dollar", Short: "HK dollar", Symbol: "HK$"},
	ILS: {Name: "Israeli shekel", Short: "Shekel", Symbol: "₪"},
	INR: {Name: "Indian rupee", Short: "Rupee", Symbol: "₹"},
	MXN: {Name: "Mexican peso", Short: "Peso", Symbol: "MX$"},
	MYR: {Name: "Malaysian ringgit", Short: "Ringgit", Symbol: "RM"},
	NOK: {Name: "Norwegian krone", Short: "Krone", Symbol: "kr"},
	NZD: {Name: "New Zealand dollar", Short: "Kiwi dollar", Symbol: "NZ$"},
	PHP: {Name: "Philippine peso", Short: "Peso", Symbol: "₱"},
	PLN: {Name: "Polish zloty", Short: "Zloty", Symbol: "zł"},
	RON: {Name: "Romanian leu", Short: "Leu", Symbol: "lei"},
	SEK: {Name: "Swedish krona", Short: "Krona", Symbol: "kr"},
	SGD: {Name: "Singapore dollar", Short: "Singapore dollar", Symbol: "S$"},
	THB: {Name: "Thai baht", Short: "Baht", Symbol: "฿"},
	TRY: {Name: "Turkish lira", Short: "Lira", Symbol: "₺"},
	ZAR: {Name: "South African rand", Short: "Rand", Symbol: "R"},
}

func Parse(raw string) (Code, bool) {
	c := Code(strings.ToUpper(strings.TrimSpace(raw)))
	_, ok := catalog[c]
	if !ok {
		return "", false
	}
	return c, true
}

func (c Code) Valid() bool {
	_, ok := catalog[c]
	return ok
}

func (c Code) Prefixed() bool {
	return c.Valid() && c != USD && c != EUR && c != GBP
}

func (c Code) meta() Meta {
	if m, ok := catalog[c]; ok {
		return m
	}
	return Meta{Name: string(c), Short: string(c), Symbol: string(c)}
}

func (c Code) Symbol() string {
	return c.meta().Symbol
}

func (c Code) Name() string {
	return c.meta().Name
}

func (c Code) ShortName() string {
	return c.meta().Short
}

func Vault(c Code) uuid.UUID {
	switch c {
	case EUR:
		return VaultEUR
	case GBP:
		return VaultGBP
	case USD:
		return VaultUSD
	default:
		return uuid.NewSHA1(vaultNS, []byte("vault:"+string(c)))
	}
}

func All() []Code {
	out := make([]Code, len(order))
	copy(out, order)
	return out
}

func (c Code) Rank() int {
	for i, x := range order {
		if x == c {
			return i
		}
	}
	return len(order)
}
