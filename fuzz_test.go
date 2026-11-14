package messageformat

import "testing"

// FuzzParse feeds arbitrary bytes to the parser: it must never panic, and
// every parsed message must round-trip through Format with an empty value
// map producing either output or a typed error.
func FuzzParse(f *testing.F) {
	seeds := []string{
		"Hello {name}",
		"{count, plural, one {# item} other {# items}}",
		"{gender, select, male {He} other {They}}",
		"Click <b>{what}</b> now",
		"It''s '{count}'",
		"{d, date, ::yyyyMMdd}",
		"{n, number, ::currency/USD}",
		"{'}",
		"{{{{",
		"<b>",
		"{x, plural, offset:2 =0 {a} other {#}}",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		mf, err := New("en", s)
		if err != nil {
			return
		}
		_, _ = mf.Format(map[string]any{})
		_, _ = mf.Format(map[string]any{
			"name": "x", "count": 1.5, "n": -2.5, "gender": "female",
			"what": "y", "x": "z",
			"b": TagFunc(func(parts []string) string { return parts[0] }),
		})
	})
}
