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
	// Subtag inheritance: fa-IR resolves to fa's non-Gregorian default.
	mfIR, _ := New("fa-IR", "{d, date, long}")
	if _, err := mfIR.Format(map[string]any{"d": "2024-03-15T14:30:45Z"}); err == nil {
		t.Fatal("fa-IR: expected non-Gregorian typed error")
	}
	// ar-SA defaults to Gregorian in CLDR 48 / ICU 78 (Saudi switched) and works.
	mfSA, _ := New("ar-SA", "{d, date, short}")
	if _, err := mfSA.Format(map[string]any{"d": "2024-03-15T14:30:45Z"}); err != nil {
		t.Fatalf("ar-SA should work (Gregorian default): %v", err)
	}

	// Gregorian-default locales keep working.
	mf, _ := New("ar-EG", "{d, date, short}")
	if _, err := mf.Format(map[string]any{"d": "2024-03-15T14:30:45Z"}); err != nil {
		t.Fatalf("ar-EG should work: %v", err)
	}
	// Explicit calendar extensions decide first, both ways.
	mfGreg, _ := New("fa-u-ca-gregory", "{d, date, short}")
	if _, err := mfGreg.Format(map[string]any{"d": "2024-03-15T14:30:45Z"}); err != nil {
		t.Fatalf("fa-u-ca-gregory should work: %v", err)
	}
	for _, loc := range []string{"en-u-ca-islamic", "th-TH-u-ca-buddhist", "en-u-ca-roc"} {
		mfCal, _ := New(loc, "{d, date, short}")
		if _, err := mfCal.Format(map[string]any{"d": "2024-03-15T14:30:45Z"}); err == nil {
			t.Fatalf("%s: expected non-Gregorian typed error", loc)
		} else if !strings.Contains(err.Error(), "calendar") {
			t.Fatalf("%s: unexpected error %v", loc, err)
		}
	}
	// Unknown tags fall back to Gregorian root behavior, mirroring ICU.
	mfUnk, _ := New("xx-UNKNOWN", "{d, date, short}")
	if _, err := mfUnk.Format(map[string]any{"d": "2024-03-15T14:30:45Z"}); err != nil {
		t.Fatalf("unknown tag should fall back, not fail: %v", err)
	}

	// Number formatting is calendar-independent and unaffected.
	mfn, _ := New("fa", "{n, number}")
	if _, err := mfn.Format(map[string]any{"n": 1234.5}); err != nil {
		t.Fatalf("fa number should work: %v", err)
	}
}
