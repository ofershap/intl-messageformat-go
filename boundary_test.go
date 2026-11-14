package messageformat

import (
	"strings"
	"testing"
)

func TestDeepNestingReturnsError(t *testing.T) {
	msg := strings.Repeat("{a, plural, other {", 5000) + "x"
	if _, err := New("en", msg); err == nil {
		t.Fatal("expected nesting-depth error")
	} else if !strings.Contains(err.Error(), "nested too deeply") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNonGregorianLocaleReturnsError(t *testing.T) {
	for _, loc := range []string{"fa", "th", "ps", "fa-IR", "th-TH"} {
		mf, err := New(loc, "{d, date, long}")
		if err != nil {
			t.Fatalf("%s: parse err %v", loc, err)
		}
		if _, err := mf.Format(map[string]any{"d": "2024-03-15T14:30:45Z"}); err == nil {
			t.Fatalf("%s: expected non-Gregorian typed error", loc)
		} else if !strings.Contains(err.Error(), "non-Gregorian") {
			t.Fatalf("%s: unexpected error %v", loc, err)
		}
	}
	// Gregorian-default locales keep working.
	mf, _ := New("ar-EG", "{d, date, short}")
	if _, err := mf.Format(map[string]any{"d": "2024-03-15T14:30:45Z"}); err != nil {
		t.Fatalf("ar-EG should work: %v", err)
	}
	// Number formatting is calendar-independent and unaffected.
	mfn, _ := New("fa", "{n, number}")
	if _, err := mfn.Format(map[string]any{"n": 1234.5}); err != nil {
		t.Fatalf("fa number should work: %v", err)
	}
}
