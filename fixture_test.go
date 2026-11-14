package messageformat

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type msgFixture struct {
	ID       string         `json:"id"`
	Variant  int            `json:"variant"`
	Locale   string         `json:"locale"`
	Message  string         `json:"message"`
	Values   map[string]any `json:"values"`
	Expected string         `json:"expected"`
	Error    string         `json:"error"`
	JSAnyway bool           `json:"jsFormattedAnyway"`
}

// goMustError: cases where the Go port deliberately returns typed errors
// (FormatJS formats silently). Grounded DDD decisions, see ddd doc.
var goMustError = map[string]bool{
	"unsupported-number-skeleton": true,
	"unsupported-date-skeleton":   true,
	"date-explicit-pattern":       true,
	"date-pattern-names":          true,
}

func TestMessageFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/fixtures.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Fixtures []msgFixture `json:"fixtures"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	bad := 0
	for _, f := range doc.Fixtures {
		values := map[string]any{}
		for k, v := range f.Values {
			values[k] = v
		}
		if f.ID == "rich-text" {
			// gen.mjs converted the '<B>' marker into a function, which JSON
			// dropped; re-synthesize the same marker-wrapping TagFunc here.
			marker := "<B>"
			values["b"] = TagFunc(func(parts []string) string {
				return marker + strings.Join(parts, "") + "</" + marker[1:]
			})
		}
		var opts []Option
		if f.ID == "number-currency" {
			opts = append(opts, WithNumberFormats(map[string]NumberFormatSpec{
				"currency": {Style: "currency", Currency: "USD"},
			}))
		}
		mf, err := New(f.Locale, f.Message, opts...)
		if err != nil {
			t.Errorf("%s v%d %s: parse error %v", f.ID, f.Variant, f.Locale, err)
			bad++
			continue
		}
		got, err := mf.Format(values)
		if goMustError[f.ID] || f.Error != "" {
			if err == nil {
				bad++
				if bad <= 20 {
					t.Errorf("%s v%d %s: want error, got %q (js: %q)", f.ID, f.Variant, f.Locale, got, f.Expected)
				}
			}
			continue
		}
		if err != nil {
			bad++
			if bad <= 20 {
				t.Errorf("%s v%d %s: err %v (want %q)", f.ID, f.Variant, f.Locale, err, f.Expected)
			}
			continue
		}
		if got != f.Expected {
			bad++
			if bad <= 20 {
				t.Errorf("%s v%d %s: got %q want %q", f.ID, f.Variant, f.Locale, got, f.Expected)
			}
		}
	}
	if bad > 0 {
		t.Errorf("%d mismatches of %d", bad, len(doc.Fixtures))
	}
}
