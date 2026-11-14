package messageformat

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

type dateFixture struct {
	Locale     string `json:"locale"`
	ISO        string `json:"iso"`
	DateFull   string `json:"datefull"`
	DateLong   string `json:"datelong"`
	DateMedium string `json:"datemedium"`
	DateShort  string `json:"dateshort"`
	TimeFull   string `json:"timefull"`
	TimeLong   string `json:"timelong"`
	TimeMedium string `json:"timemedium"`
	TimeShort  string `json:"timeshort"`
}

func TestDateFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/date_fixtures.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Fixtures []dateFixture `json:"fixtures"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	bad := 0
	check := func(f dateFixture, tm time.Time, isDate bool, style, want string) {
		var got string
		var err error
		if isDate {
			got, err = formatDateValue(tm, style, f.Locale)
		} else {
			got, err = formatTimeValue(tm, style, f.Locale)
		}
		if err != nil {
			t.Errorf("%s %s %s: err %v", f.Locale, f.ISO, style, err)
			bad++
			return
		}
		if got != want {
			bad++
			if bad <= 20 {
				t.Errorf("%s %s %s: got %q want %q", f.Locale, f.ISO, style, got, want)
			}
		}
	}
	for _, f := range doc.Fixtures {
		tm, err := time.Parse(time.RFC3339, f.ISO)
		if err != nil {
			t.Fatal(err)
		}
		tm = tm.UTC()
		check(f, tm, true, "full", f.DateFull)
		check(f, tm, true, "long", f.DateLong)
		check(f, tm, true, "medium", f.DateMedium)
		check(f, tm, true, "short", f.DateShort)
		check(f, tm, false, "full", f.TimeFull)
		check(f, tm, false, "long", f.TimeLong)
		check(f, tm, false, "medium", f.TimeMedium)
		check(f, tm, false, "short", f.TimeShort)
	}
	if bad > 0 {
		t.Errorf("%d mismatches", bad)
	}
}
