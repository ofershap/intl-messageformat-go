package messageformat

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func resolveDateData(locale string) *dateLocaleData {
	locale = strings.ReplaceAll(locale, "-", "_")
	for l := locale; l != ""; {
		if d, ok := dateData[l]; ok {
			return d
		}
		if idx := strings.LastIndex(l, "_"); idx > 0 {
			l = l[:idx]
		} else {
			break
		}
	}
	if d, ok := dateData["en"]; ok {
		return d
	}
	return &dateLocaleData{}
}

// formatDateValue and formatTimeValue implement the MF1 date/time types with
// the four CLDR style presets (full/long/medium/short) at UTC, mirroring
// Intl.DateTimeFormat({dateStyle|timeStyle, timeZone:'UTC'}).
func formatDateValue(t time.Time, style, locale string) (string, error) {
	d := resolveDateData(locale)
	pat, ok := d.DateFormats[style]
	if !ok {
		return "", fmt.Errorf("messageformat: unsupported date style %q (want full, long, medium or short)", style)
	}
	return formatDatePattern(t, pat, d, resolveNumberData(locale).NS), nil
}

func formatTimeValue(t time.Time, style, locale string) (string, error) {
	d := resolveDateData(locale)
	pat, ok := d.TimeFormats[style]
	if !ok {
		return "", fmt.Errorf("messageformat: unsupported time style %q (want full, long, medium or short)", style)
	}
	return formatDatePattern(t, pat, d, resolveNumberData(locale).NS), nil
}

// formatDatePattern renders a CLDR date-time pattern for a UTC instant.
// Supported fields: y M d E H h m s a z. Literals pass through; non-ASCII
// letters (e.g. Hebrew ב) are literal per ICU rules. 'z' formats the UTC zone
// as "UTC" (short) or the locale's long UTC name (zzzz).
func formatDatePattern(t time.Time, pat string, d *dateLocaleData, ns string) string {
	var b strings.Builder
	runes := []rune(pat)
	i := 0
	for i < len(runes) {
		r := runes[i]
		if r == '\'' {
			i++
			for i < len(runes) {
				if runes[i] == '\'' {
					if i+1 < len(runes) && runes[i+1] == '\'' {
						b.WriteByte('\'')
						i += 2
						continue
					}
					i++
					break
				}
				b.WriteRune(runes[i])
				i++
			}
			continue
		}
		if r < 'A' || (r > 'Z' && r < 'a') || r > 'z' {
			b.WriteRune(r)
			i++
			continue
		}
		j := i
		for j < len(runes) && runes[j] == r {
			j++
		}
		n := j - i
		b.WriteString(formatDateField(t, r, n, d))
		i = j
	}
	// V8/Node date-time output uses plain spaces where CLDR 48 patterns carry
	// narrow no-break spaces (web-compat decision in V8); match the reference.
	out := strings.ReplaceAll(b.String(), " ", " ")
	return transliterate(out, ns)
}

func padNum(v, width int) string {
	s := strconv.Itoa(v)
	for len(s) < width {
		s = "0" + s
	}
	return s
}

func formatDateField(t time.Time, field rune, n int, d *dateLocaleData) string {
	switch field {
	case 'y':
		y := t.Year()
		if n == 2 {
			return padNum(y%100, 2)
		}
		if n >= 4 {
			return padNum(y, 4)
		}
		return strconv.Itoa(y)
	case 'M', 'L':
		m := int(t.Month())
		switch {
		case n >= 4:
			return d.MonthsWide[m-1]
		case n == 3:
			return d.MonthsAbbr[m-1]
		case n == 2:
			return padNum(m, 2)
		default:
			return strconv.Itoa(m)
		}
	case 'd':
		return padNum(t.Day(), n)
	case 'E', 'e', 'c':
		wd := int(t.Weekday())
		if n >= 4 {
			return d.DaysWide[wd]
		}
		return d.DaysAbbr[wd]
	case 'H':
		return padNum(t.Hour(), n)
	case 'h':
		h := t.Hour() % 12
		if h == 0 {
			h = 12
		}
		return padNum(h, n)
	case 'm':
		return padNum(t.Minute(), n)
	case 's':
		return padNum(t.Second(), n)
	case 'a':
		if t.Hour() < 12 {
			return d.AM
		}
		return d.PM
	case 'z', 'v':
		if n >= 4 {
			return d.UTCLong
		}
		return "UTC"
	default:
		// Unknown field: emit literally (ICU treats unsupported letters as
		// literal text in most cases; we keep them visible rather than silent).
		return strings.Repeat(string(field), n)
	}
}
