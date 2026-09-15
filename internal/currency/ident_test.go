package currency

import "testing"

func TestIssueRoundTrip(t *testing.T) {
	if !ValidABA(RoutingABA) {
		t.Fatalf("routing %s failed ABA checksum", RoutingABA)
	}
	core := "12345678"
	usd, err := Issue(USD, core)
	if err != nil || usd != RoutingABA+core {
		t.Fatalf("usd %q %v", usd, err)
	}
	gbp, err := Issue(GBP, core)
	if err != nil || gbp != "04000412345678" {
		t.Fatalf("gbp %q %v", gbp, err)
	}
	eur, err := Issue(EUR, core)
	if err != nil {
		t.Fatal(err)
	}
	spaced := eur[:4] + " " + eur[4:8] + " " + eur[8:12] + " " + eur[12:16] + " " + eur[16:]
	got, code, ok := Normalize(spaced)
	if !ok || code != EUR || got != eur || Core(eur) != core {
		t.Fatalf("eur %q spaced=%q got=%q code=%s ok=%v core=%s", eur, spaced, got, code, ok, Core(eur))
	}
	if _, _, ok := Normalize("GB00THEB04000412345678"); ok {
		t.Fatal("bad iban checksum accepted")
	}
}

func TestNormalizeGBPAndUSD(t *testing.T) {
	n, code, ok := Normalize("04-00-04 1234 5678")
	if !ok || code != GBP || n != "04000412345678" {
		t.Fatalf("%q %s %v", n, code, ok)
	}
	n, code, ok = Normalize(RoutingABA + " · 1234 5678")
	if !ok || code != USD || n != RoutingABA+"12345678" || Core(n) != "12345678" {
		t.Fatalf("%q %s %v", n, code, ok)
	}
	if _, _, ok := Normalize("1234 · 5678"); ok {
		t.Fatal("bare 8-digit USD should not be an account number")
	}
	if Format(RoutingABA+"12345678") != RoutingABA+" · 1234 5678" {
		t.Fatalf("format %q", Format(RoutingABA+"12345678"))
	}
}

func TestCrossRateE8(t *testing.T) {
	usd := ScaleE8
	eur := int64(85_000_000)
	got, err := CrossRateE8(usd, eur)
	if err != nil || got != eur {
		t.Fatalf("usd->eur %d %v", got, err)
	}
	back, err := CrossRateE8(eur, usd)
	if err != nil {
		t.Fatal(err)
	}
	out, err := ConvertCents(8500, back)
	if err != nil || out != 10000 {
		t.Fatalf("round trip %d %v rate=%d", out, err, back)
	}
}

func TestConvertCents(t *testing.T) {
	rate, err := RateE8("0.85")
	if err != nil || rate != 85_000_000 {
		t.Fatalf("rate %d %v", rate, err)
	}
	out, err := ConvertCents(10000, rate)
	if err != nil || out != 8500 {
		t.Fatalf("out %d %v", out, err)
	}
	if FormatRate(rate) != "0.85" {
		t.Fatalf("format %q", FormatRate(rate))
	}
}
