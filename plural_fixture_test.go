package messageformat

import (
	"encoding/json"
	"os"
	"testing"
)

type pluralFixture struct {
	Locale   string  `json:"locale"`
	Type     string  `json:"type"`
	Value    float64 `json:"value"`
	Str      string  `json:"str"`
	Expected string  `json:"expected"`
}

func TestPluralFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/plural_fixtures.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Fixtures []pluralFixture `json:"fixtures"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	bad := 0
	for _, f := range doc.Fixtures {
		rules := cldrCardinalRules
		if f.Type == "ordinal" {
			rules = cldrOrdinalRules
		}
		op, err := operandsFromString(f.Str)
		if err != nil {
			t.Errorf("%s %v: %v", f.Locale, f.Str, err)
			continue
		}
		got := selectPlural(rules, f.Locale, op)
		if got != f.Expected {
			bad++
			if bad <= 15 {
				t.Errorf("%s/%s %s: got %q want %q", f.Locale, f.Type, f.Str, got, f.Expected)
			}
		}
	}
	if bad > 0 {
		t.Fatalf("%d/%d mismatches", bad, len(doc.Fixtures))
	}
	t.Logf("%d plural fixtures all match", len(doc.Fixtures))
}
