package currency

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

func Issue(code Code, core8 string) (string, error) {
	if !validCore(core8) {
		return "", fmt.Errorf("invalid core")
	}
	switch code {
	case USD:
		return RoutingABA + core8, nil
	case GBP:
		return SortCode + core8, nil
	case EUR:
		bban := BankCode + SortCode + core8
		return CountryGB + ibanCheck(bban, CountryGB) + bban, nil
	default:
		if code.Prefixed() {
			return string(code) + core8, nil
		}
		return "", fmt.Errorf("unsupported currency")
	}
}

func Detect(canonical string) (Code, bool) {
	n := compact(canonical)
	switch {
	case isEURIBAN(n):
		return EUR, true
	case isPrefixedNumber(n):
		c, _ := Parse(n[:3])
		return c, true
	case len(n) == 14 && strings.HasPrefix(n, SortCode) && validCore(n[6:]):
		return GBP, true
	case isUSDACH(n):
		return USD, true
	default:
		return "", false
	}
}

func Normalize(raw string) (string, Code, bool) {
	n := compact(raw)
	if n == "" {
		return "", "", false
	}
	if looksLikeUSDAccount(n) {
		return RoutingABA + n, USD, true
	}
	if code, ok := Detect(n); ok {
		if code == EUR && !validIBAN(n) {
			return "", "", false
		}
		return n, code, true
	}
	return "", "", false
}

func Core(canonical string) string {
	n := compact(canonical)
	code, ok := Detect(n)
	if !ok {
		return ""
	}
	switch code {
	case USD:
		return n[len(RoutingABA):]
	case GBP:
		return n[len(n)-8:]
	case EUR:
		return n[len(n)-8:]
	default:
		if code.Prefixed() && len(n) == 11 {
			return n[3:]
		}
		return ""
	}
}

func Format(canonical string) string {
	n := compact(canonical)
	code, ok := Detect(n)
	if !ok {
		return canonical
	}
	switch code {
	case USD:
		return n[:9] + " · " + n[9:13] + " " + n[13:]
	case GBP:
		return n[0:2] + "-" + n[2:4] + "-" + n[4:6] + " · " + n[6:10] + " " + n[10:]
	case EUR:
		var b strings.Builder
		for i, r := range n {
			if i > 0 && i%4 == 0 {
				b.WriteByte(' ')
			}
			b.WriteRune(r)
		}
		return b.String()
	default:
		if code.Prefixed() && len(n) == 11 {
			return n[:3] + " · " + n[3:7] + " " + n[7:]
		}
		return n
	}
}

func LocalAccount(canonical string) string {
	return Core(canonical)
}

func SortDisplay() string {
	return SortCode[0:2] + "-" + SortCode[2:4] + "-" + SortCode[4:6]
}

func compact(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsSpace(r) || r == '-' || r == '·' {
			continue
		}
		b.WriteRune(unicode.ToUpper(r))
	}
	return b.String()
}

func isPrefixedNumber(n string) bool {
	if len(n) != 11 {
		return false
	}
	c, ok := Parse(n[:3])
	return ok && c.Prefixed() && validCore(n[3:])
}

func isUSDACH(n string) bool {
	return len(n) == 17 && strings.HasPrefix(n, RoutingABA) && validCore(n[9:])
}

func looksLikeUSDAccount(n string) bool {
	return validCore(n) && !strings.HasPrefix(RoutingABA, n)
}

// ValidABA reports whether n is a 9-digit ABA routing number with a valid checksum.
func ValidABA(n string) bool {
	if len(n) != 9 {
		return false
	}
	var d [9]int
	for i := 0; i < 9; i++ {
		if n[i] < '0' || n[i] > '9' {
			return false
		}
		d[i] = int(n[i] - '0')
	}
	sum := 3*(d[0]+d[3]+d[6]) + 7*(d[1]+d[4]+d[7]) + (d[2] + d[5] + d[8])
	return sum%10 == 0
}

func validCore(s string) bool {
	if len(s) != 8 {
		return false
	}
	for i := 0; i < 8; i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func isEURIBAN(n string) bool {
	return len(n) == 22 && strings.HasPrefix(n, CountryGB) && strings.Contains(n, BankCode+SortCode)
}

func ibanCheck(bban, country string) string {
	expanded := ibanDigits(bban + country + "00")
	rem := 0
	for _, c := range expanded {
		rem = rem*10 + int(c-'0')
		rem %= 97
	}
	return fmt.Sprintf("%02d", 98-rem)
}

func validIBAN(n string) bool {
	if len(n) < 5 {
		return false
	}
	return ibanMod97(ibanDigits(n[4:]+n[:4])) == 1
}

func ibanMod97(digits string) int {
	rem := 0
	for i := 0; i < len(digits); i++ {
		rem = rem*10 + int(digits[i]-'0')
		rem %= 97
	}
	return rem
}

func ibanDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteString(strconv.Itoa(int(r - 'A' + 10)))
		}
	}
	return b.String()
}
