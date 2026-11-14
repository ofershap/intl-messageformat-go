package messageformat

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// pluralOperands are the CLDR plural operands, computed from the decimal
// representation of the value the way JS Intl.PluralRules derives them.
type pluralOperands struct {
	n float64 // absolute value
	i int64   // integer digits
	v int     // visible fraction digits, with trailing zeros
	w int     // visible fraction digits, without trailing zeros
	f int64   // visible fractional digits, with trailing zeros
	t int64   // visible fractional digits, without trailing zeros
	e int     // exponent (CLDR 44+ compact-exponent operand)
}

// operandsFromString computes operands from a shortest-round-trip decimal
// string like JS String(n) produces ("1.50" never occurs: shortest repr).
// Exponent forms ("1e+21") are expanded first.
func operandsFromString(s string) (pluralOperands, error) {
	var op pluralOperands
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	if idx := strings.IndexAny(s, "eE"); idx >= 0 {
		// Exponent form ("1e+21"): engines zero the digit operands and keep
		// only n (absolute value) and e (exponent). Verified against V8
		// (Node 22): fr.select(2e21)="one" (i=0 matches "i = 0,1"),
		// ru.select(1e21)="many" (v=0 and i%10=0), cy/ar keep n-based rules.
		// V8 zeroes ALL digit operands INCLUDING e for exponent-form input
		// (verified: es.select(1e21)="other" - es "many" needs e != 0..5,
		// which would fire if e were 21; it does not).
		exp, _ := strconv.Atoi(s[idx+1:])
		coeff, err := strconv.ParseFloat(s[:idx], 64)
		if err != nil {
			return op, fmt.Errorf("bad number %q: %w", s, err)
		}
		op.n = math.Pow10(exp) * coeff
		if op.n < 0 {
			op.n = -op.n
		}
		if neg {
			// n is absolute; sign already discarded
			_ = neg
		}
		return op, nil
	}
	intPart, fracPart := s, ""
	if idx := strings.IndexByte(s, '.'); idx >= 0 {
		intPart, fracPart = s[:idx], s[idx+1:]
	}
	if intPart == "" {
		intPart = "0"
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return op, fmt.Errorf("bad number %q: %w", s, err)
	}
	if neg {
		f = -f
	}
	op.n = f
	if f < 0 {
		op.n = -f
	}
	op.i, _ = strconv.ParseInt(intPart, 10, 64)
	op.v = len(fracPart)
	op.f, _ = strconv.ParseInt("0"+fracPart, 10, 64)
	trimmed := strings.TrimRight(fracPart, "0")
	op.w = len(trimmed)
	op.t, _ = strconv.ParseInt("0"+trimmed, 10, 64)
	return op, nil
}

// evalPluralRule evaluates one CLDR rule expression ("i = 1 and v = 0 or ...").
// Empty rule (category "other") always matches; callers handle that.
func evalPluralRule(rule string, op pluralOperands) (bool, error) {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return false, fmt.Errorf("empty rule")
	}
	for _, orPart := range splitOnKeyword(rule, "or") {
		ok := true
		for _, andPart := range splitOnKeyword(orPart, "and") {
			match, err := evalRelation(strings.TrimSpace(andPart), op)
			if err != nil {
				return false, err
			}
			ok = ok && match
			if !ok {
				break
			}
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

func splitOnKeyword(s, kw string) []string {
	return strings.Split(s, " "+kw+" ")
}

func evalRelation(rel string, op pluralOperands) (bool, error) {
	// forms: "n = 1..2, 5" | "n % 100 = 3..10" | "v != 0"
	neq := false
	var expr, list string
	if i := strings.Index(rel, "!="); i >= 0 {
		expr, list, neq = strings.TrimSpace(rel[:i]), strings.TrimSpace(rel[i+2:]), true
	} else if i := strings.Index(rel, "="); i >= 0 {
		expr, list = strings.TrimSpace(rel[:i]), strings.TrimSpace(rel[i+1:])
	} else {
		return false, fmt.Errorf("bad relation %q", rel)
	}
	parts := strings.Fields(expr)
	var val float64
	switch parts[0] {
	case "n":
		val = op.n
	case "i":
		val = float64(op.i)
	case "v":
		val = float64(op.v)
	case "w":
		val = float64(op.w)
	case "f":
		val = float64(op.f)
	case "t":
		val = float64(op.t)
	case "c", "e": // exponent operand (CLDR 44+; used by fr/pt "many" rules)
		val = float64(op.e)
	default:
		return false, fmt.Errorf("unknown operand %q", parts[0])
	}
	if len(parts) == 3 {
		if parts[1] != "%" {
			return false, fmt.Errorf("bad relation %q", rel)
		}
		mod, err := strconv.ParseFloat(parts[2], 64)
		if err != nil || mod == 0 {
			return false, fmt.Errorf("bad modulus in %q", rel)
		}
		val = modF(val, mod)
	} else if len(parts) != 1 {
		return false, fmt.Errorf("bad relation %q", rel)
	}
	match := false
	for _, item := range strings.Split(list, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if dd := strings.Index(item, ".."); dd >= 0 {
			lo, err1 := strconv.ParseFloat(strings.TrimSpace(item[:dd]), 64)
			hi, err2 := strconv.ParseFloat(strings.TrimSpace(item[dd+2:]), 64)
			if err1 != nil || err2 != nil {
				return false, fmt.Errorf("bad range %q", item)
			}
			// CLDR: ranges match integer values only (verified vs V8:
			// ar "n % 100 = 3..10" does not match 3.1).
			if val == math.Trunc(val) && val >= lo && val <= hi {
				match = true
				break
			}
		} else {
			v, err := strconv.ParseFloat(item, 64)
			if err != nil {
				return false, fmt.Errorf("bad value %q", item)
			}
			if val == v {
				match = true
				break
			}
		}
	}
	return match != neq, nil
}

// modF computes val % mod on decimal operands; inputs come from decimal
// strings so integer modulus is exact for realistic magnitudes.
func modF(val, mod float64) float64 {
	r := val
	if r >= mod {
		r = float64(int64(r) % int64(mod))
	}
	return r
}

// selectPlural picks the CLDR plural category for a locale.
// rules: cldrCardinalRules or cldrOrdinalRules. Falls back to root rules ("other").
func selectPlural(rules map[string]map[string]string, locale string, op pluralOperands) string {
	lang := resolveLocaleRules(rules, locale)
	if lang == nil {
		return "other"
	}
	for _, cat := range []string{"zero", "one", "two", "few", "many"} {
		rule, ok := lang[cat]
		if !ok {
			continue
		}
		match, err := evalPluralRule(rule, op)
		if err == nil && match {
			return cat
		}
	}
	return "other"
}

// resolveLocaleRules resolves a BCP-47 locale to its rule set: exact,
// then language subtag (CLDR rules key on language or language_Script/Region).
func resolveLocaleRules(rules map[string]map[string]string, locale string) map[string]string {
	locale = strings.ReplaceAll(locale, "-", "_")
	if r, ok := rules[locale]; ok {
		return r
	}
	lang := strings.SplitN(locale, "_", 2)[0]
	if r, ok := rules[lang]; ok {
		return r
	}
	// script/regional variants: e.g. "pt_PT" exists separately from "pt";
	// walk progressively shorter prefixes
	rest := locale
	for i := len(rest); i > 0; i-- {
		if r, ok := rules[rest[:i]]; ok {
			return r
		}
	}
	return nil
}
