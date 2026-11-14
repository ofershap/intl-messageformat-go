package messageformat

import (
	"encoding/json"
	"os"
	"testing"
)

type numberFixture struct {
	Locale      string  `json:"locale"`
	Value       float64 `json:"value"`
	Decimal     string  `json:"decimal"`
	Integer     string  `json:"integer"`
	Percent     string  `json:"percent"`
	CurrencyUSD string  `json:"currencyUSD"`
	CurrencyJPY string  `json:"currencyJPY"`
	CurrencyILS string  `json:"currencyILS"`
}

func TestNumberFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/number_fixtures.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Fixtures []numberFixture `json:"fixtures"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	bad := 0
	check := func(f numberFixture, style numberStyle, cur, want, label string) {
		got, err := formatNumber(f.Value, style, cur, resolveNumberData(f.Locale))
		if err != nil {
			t.Errorf("%s %v %s: err %v", f.Locale, f.Value, label, err)
			bad++
			return
		}
		if got != want {
			bad++
			if bad <= 20 {
				t.Errorf("%s %v %s: got %q want %q", f.Locale, f.Value, label, got, want)
			}
		}
	}
	for _, f := range doc.Fixtures {
		check(f, styleDecimal, "", f.Decimal, "decimal")
		check(f, styleInteger, "", f.Integer, "integer")
		check(f, stylePercent, "", f.Percent, "percent")
		check(f, styleCurrency, "USD", f.CurrencyUSD, "currencyUSD")
		check(f, styleCurrency, "JPY", f.CurrencyJPY, "currencyJPY")
		check(f, styleCurrency, "ILS", f.CurrencyILS, "currencyILS")
	}
	if bad > 0 {
		t.Fatalf("%d mismatches", bad)
	}
	t.Logf("%d x 6 number fixtures all match", len(doc.Fixtures))
}
