package messageformat_test

import (
	"fmt"
	"time"

	messageformat "github.com/ofershap/intl-messageformat-go"
)

func Example() {
	mf, err := messageformat.New("en", "{count, plural, =0 {No items} one {# item} other {# items}}")
	if err != nil {
		panic(err)
	}
	out, _ := mf.Format(map[string]any{"count": 3})
	fmt.Println(out)
	// Output: 3 items
}

func ExampleMessageFormat_Format_locale() {
	mf, _ := messageformat.New("he", "הילל {name}, יש לך {count, plural, one {הודעה אחת} other {# הודעות}}")
	out, _ := mf.Format(map[string]any{"name": "דנה", "count": 5})
	fmt.Println(out)
	// Output: הילל דנה, יש לך 5 הודעות
}

func ExampleMessageFormat_Format_date() {
	mf, _ := messageformat.New("fr", "Arrive {d, date, long} a {t, time, short}")
	out, _ := mf.Format(map[string]any{
		"d": time.Date(2026, 11, 14, 9, 30, 0, 0, time.UTC),
		"t": time.Date(2026, 11, 14, 9, 30, 0, 0, time.UTC),
	})
	fmt.Println(out)
	// Output: Arrive 14 novembre 2026 a 09:30
}

func ExampleTagFunc() {
	mf, _ := messageformat.New("en", "Click <b>{what}</b> now")
	out, _ := mf.Format(map[string]any{
		"what": "here",
		"b": messageformat.TagFunc(func(parts []string) string {
			return "<B>" + parts[0] + "</B>"
		}),
	})
	fmt.Println(out)
	// Output: Click <B>here</B> now
}
