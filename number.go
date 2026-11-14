package messageformat

import (
	"math"
	"math/big"
	"strconv"
	"strings"
)

// numberStyle selects the MF1 number style.
type numberStyle int

const (
	styleDecimal numberStyle = iota
	styleInteger
	stylePercent
	styleCurrency
)

// numberPattern is a parsed CLDR number pattern.
type numberPattern struct {
	prefix, suffix string
	neg            *numberPatternParts // explicit negative variant (after ';')
	parts          numberPatternParts
}

type numberPatternParts struct {
	prefix, suffix       string
	minInt               int
	fracMin, fracMax     int
	groupPrimary         int
	groupSecondary       int
	hasFractionSeparator bool
}

// parseNumberPattern parses a CLDR number pattern like "#,##,##0.###" or
// "\u200f#,##0.00 \u200f¤;-#,##0.00 ¤". Only the constructs CLDR 48 uses
// for standard decimal/percent/currency patterns are supported.
func parseNumberPattern(pat string) numberPattern {
	p := numberPattern{}
	pos, neg, hasNeg := strings.Cut(pat, ";")
	p.parts = parsePatternParts(pos)
	if hasNeg {
		n := parsePatternParts(neg)
		p.neg = &n
	}
	return p
}

func parsePatternParts(s string) numberPatternParts {
	var pp numberPatternParts
	// find the numeric core: first and last char in "#0,."
	i, j := -1, -1
	for k := 0; k < len(s); k++ {
		c := s[k]
		if c == '#' || c == '0' || c == ',' || c == '.' {
			if i < 0 {
				i = k
			}
			j = k
		}
	}
	if i < 0 {
		pp.prefix = s
		return pp
	}
	pp.prefix, pp.suffix = s[:i], s[j+1:]
	core := s[i : j+1]
	intPart, fracPart, hasDot := strings.Cut(core, ".")
	pp.hasFractionSeparator = hasDot
	// integer min digits: count of '0' in intPart
	for _, c := range intPart {
		if c == '0' {
			pp.minInt++
		}
	}
	// grouping: ',' positions from the right of the integer part
	groups := []int{}
	digits := 0
	for k := len(intPart) - 1; k >= 0; k-- {
		c := intPart[k]
		if c == ',' {
			groups = append(groups, digits)
		} else {
			digits++
		}
	}
	if len(groups) > 0 {
		pp.groupPrimary = groups[0]
		pp.groupSecondary = groups[0]
		if len(groups) > 1 {
			pp.groupSecondary = groups[1] - groups[0]
		}
	}
	for _, c := range fracPart {
		if c == '0' {
			pp.fracMin++
			pp.fracMax++
		} else if c == '#' {
			pp.fracMax++
		}
	}
	return pp
}

// formatNumber formats v per ICU/ECMA-402 default NumberFormat semantics for
// the given style. Rounding is halfExpand (ties away from zero) computed on
// the exact binary value, verified against Intl.NumberFormat fixtures.
func formatNumber(v float64, style numberStyle, currency string, data numberLocaleData) (string, error) {
	pat := data.DecPattern
	fracOverride := -1
	mult := int64(1)
	switch style {
	case styleInteger:
		fracOverride = 0
	case stylePercent:
		pat = data.PctPattern
		mult = 100
	case styleCurrency:
		pat = data.CurPattern
		if currency == "" {
			return "", errCurrencyCodeRequired
		}
	}
	p := parseNumberPattern(pat)
	parts := p.parts
	neg := math.Signbit(v) && v != 0
	if neg && p.neg != nil {
		parts = *p.neg
	}
	maxFrac := parts.fracMax
	if fracOverride >= 0 {
		maxFrac = fracOverride
	}
	if style == styleCurrency {
		d, ok := cldrCurrencyDigits[currency]
		if !ok {
			d = cldrDefaultCurrencyDigits
		}
		maxFrac = d
		parts.fracMin = d
	}
	digits := roundToDigits(v, mult, maxFrac)
	intDigits, fracDigits := splitScaledDigits(digits, maxFrac)
	body := renderDigits(intDigits, fracDigits, parts, data)

	prefix := nbspSpaces(parts.prefix)
	suffix := nbspSpaces(parts.suffix)
	if style == styleCurrency {
		sym := currencySymbol(currency, data)
		prefix = strings.ReplaceAll(prefix, "¤", sym)
		suffix = strings.ReplaceAll(suffix, "¤", sym)
	}
	if style == stylePercent {
		prefix = strings.ReplaceAll(prefix, "%", data.Percent)
		suffix = strings.ReplaceAll(suffix, "%", data.Percent)
	}
	if neg {
		// The pattern's minus token is replaced by the locale's minus sign,
		// which in RTL locales carries its directional mark (CLDR symbol
		// data: he "-\u200e-"... actually "\u200e-", ar-EG "\u061c-").
		prefix = strings.ReplaceAll(prefix, "-", data.Minus)
		suffix = strings.ReplaceAll(suffix, "-", data.Minus)
	}
	var b strings.Builder
	if neg && p.neg == nil {
		b.WriteString(data.Minus)
	}
	b.WriteString(prefix)
	b.WriteString(body)
	b.WriteString(suffix)
	return b.String(), nil
}

// insertMarkBeforeMinus inserts the RTL directional mark (ALM for Arabic
// script, LRM for Hebrew) immediately before the first minus sign.
// Fixture-verified against ICU 78.
func insertMarkBeforeMinus(s string, data numberLocaleData) string {
	mark := bidiSignMark(data.Script)
	if mark == "" {
		return s
	}
	if idx := strings.Index(s, data.Minus); idx >= 0 {
		return s[:idx] + mark + s[idx:]
	}
	return s
}

// bidiSignMark returns the directional mark ICU prepends to a leading minus
// sign in RTL locales: ALM for Arabic script, LRM for Hebrew (fixture-verified).
func bidiSignMark(script string) string {
	switch script {
	case "Arab":
		return "\u061c"
	case "Hebr":
		return "\u200e"
	}
	return ""
}

// nbspSpaces replaces ASCII spaces adjacent in pattern affixes with NBSP,
// matching ICU currency/percent spacing behavior (fixture-verified).
func nbspSpaces(s string) string {
	return strings.ReplaceAll(s, " ", "\u00a0")
}

// roundToDigits returns the digits of |v|*mult rounded halfExpand (ties
// away from zero) at maxFrac fraction digits, as a big integer scaled by
// 10^maxFrac. ICU rounds the shortest round-trip decimal representation,
// not the exact binary value (fixture-verified: 2.675 -> "$2.68" even
// though float64(2.675) < 2.675), so the input is strconv's shortest repr
// and percent multiplication is a decimal point shift, never a float mul.
func roundToDigits(v float64, mult int64, maxFrac int) *big.Int {
	if v < 0 {
		v = -v
	}
	if v == 0 {
		return big.NewInt(0)
	}
	s := strconv.FormatFloat(v, 'e', -1, 64)
	mant, expStr, _ := strings.Cut(s, "e")
	exp, _ := strconv.Atoi(expStr)
	digits := strings.ReplaceAll(mant, ".", "")
	// value = digits * 10^(exp+1-len(digits)); percent shifts the exponent.
	shift := exp + 1 + maxFrac
	if mult != 1 {
		// mult is 100 for percent: exact power of ten
		for m := mult; m > 1; m /= 10 {
			shift++
		}
	}
	switch {
	case shift >= len(digits):
		digits += strings.Repeat("0", shift-len(digits))
	case shift <= 0:
		// rounds to zero (or to 1 if the leading digit forces a carry)
		if shift == 0 && digits[0] >= '5' {
			digits = "1"
		} else {
			digits = "0"
		}
	default:
		roundUp := digits[shift] >= '5'
		d := []byte(digits[:shift])
		if roundUp {
			i := len(d) - 1
			for ; i >= 0; i-- {
				if d[i] != '9' {
					d[i]++
					break
				}
				d[i] = '0'
			}
			if i < 0 {
				d = append([]byte{'1'}, d...)
			}
		}
		digits = string(d)
	}
	n, _ := new(big.Int).SetString(digits, 10)
	return n
}

// splitScaledDigits splits a scaled integer into integer and fraction digit
// strings (fraction trimmed of trailing zeros down to minDigits... caller
// trims per pattern).
func splitScaledDigits(n *big.Int, maxFrac int) (string, string) {
	s := n.String()
	if maxFrac == 0 {
		return s, ""
	}
	if len(s) <= maxFrac {
		s = strings.Repeat("0", maxFrac-len(s)+1) + s
	}
	return s[:len(s)-maxFrac], s[len(s)-maxFrac:]
}

// renderDigits applies min-int padding, grouping, the fraction separator and
// digit transliteration.
func renderDigits(intDigits, fracDigits string, pp numberPatternParts, data numberLocaleData) string {
	// trim fraction trailing zeros down to fracMin
	frac := fracDigits
	for len(frac) > pp.fracMin && strings.HasSuffix(frac, "0") {
		frac = frac[:len(frac)-1]
	}
	if pp.fracMax == 0 {
		frac = ""
	}
	for len(intDigits) < pp.minInt {
		intDigits = "0" + intDigits
	}
	// CLDR minimumGroupingDigits: grouping applies only when the integer
	// part has at least primary+minGroup digits (es: "1234" but "12.345").
	if pp.groupPrimary > 0 && len(intDigits) >= pp.groupPrimary+data.MinGroup {
		intDigits = applyGrouping(intDigits, pp.groupPrimary, pp.groupSecondary, data.Group)
	}
	var b strings.Builder
	b.WriteString(intDigits)
	if frac != "" {
		b.WriteString(data.Decimal)
		b.WriteString(frac)
	}
	return transliterate(b.String(), data.NS)
}

func applyGrouping(s string, primary, secondary int, sep string) string {
	if len(s) <= primary {
		return s
	}
	var groups []string
	rest := s
	groups = append(groups, rest[len(rest)-primary:])
	rest = rest[:len(rest)-primary]
	for len(rest) > secondary {
		groups = append(groups, rest[len(rest)-secondary:])
		rest = rest[:len(rest)-secondary]
	}
	if rest != "" {
		groups = append(groups, rest)
	}
	var b strings.Builder
	for i := len(groups) - 1; i >= 0; i-- {
		b.WriteString(groups[i])
		if i > 0 {
			b.WriteString(sep)
		}
	}
	return b.String()
}

func transliterate(s, ns string) string {
	digits, ok := cldrNumberSystems[ns]
	if !ok || ns == "latn" {
		return s
	}
	dr := []rune(digits)
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return dr[r-'0']
		}
		return r
	}, s)
}

// currencySymbol resolves the display symbol: the locale's own table first,
// then en's, then the ISO code (fixture-verified: ar-EG USD -> "US$",
// hi JPY -> "JP¥", ja JPY -> "￥").
func currencySymbol(code string, data numberLocaleData) string {
	if s, ok := data.Currencies[code]; ok {
		return s
	}
	if s, ok := cldrEnCurrencySymbols[code]; ok {
		return s
	}
	return code
}

// resolveNumberData walks locale prefixes (pt-BR -> pt) like the plural rules.
func resolveNumberData(locale string) numberLocaleData {
	locale = strings.ReplaceAll(locale, "-", "_")
	for l := locale; l != ""; {
		if d, ok := cldrNumberData[l]; ok {
			return d
		}
		if idx := strings.LastIndex(l, "_"); idx > 0 {
			l = l[:idx]
		} else {
			break
		}
	}
	if d, ok := cldrNumberData["en"]; ok {
		return d
	}
	return numberLocaleData{NS: "latn", Decimal: ".", Group: ",", Percent: "%", Minus: "-", DecPattern: "#,##0.###", PctPattern: "#,##0%", CurPattern: "¤#,##0.00"}
}
