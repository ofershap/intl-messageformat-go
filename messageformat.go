package messageformat

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// TagFunc renders a rich-text tag around its inner parts, mirroring
// FormatJS's rich-text formatting (values[name] = (parts) => ...).
type TagFunc func(parts []string) string

// NumberFormatSpec is a named custom number format, mirroring FormatJS's
// `formats.number` overrides (e.g. {n, number, currency} with
// formats={number:{currency:{style:'currency',currency:'USD'}}}).
type NumberFormatSpec struct {
	Style    string // "decimal", "integer", "percent" or "currency"
	Currency string // required when Style is "currency"
}

// MessageFormat is a parsed ICU MessageFormat 1 message bound to a locale.
type MessageFormat struct {
	locale     string
	nodes      []node
	numFormats map[string]NumberFormatSpec
}

// Option configures a MessageFormat at construction.
type Option func(*MessageFormat)

// WithNumberFormats registers named custom number formats (FormatJS
// `formats.number`).
func WithNumberFormats(f map[string]NumberFormatSpec) Option {
	return func(m *MessageFormat) { m.numFormats = f }
}

// New parses message for locale. Parse errors are returned eagerly, mirroring
// `new IntlMessageFormat(message, locale)`.
func New(locale, message string, opts ...Option) (*MessageFormat, error) {
	nodes, err := parseMessage(message)
	if err != nil {
		return nil, err
	}
	m := &MessageFormat{locale: locale, nodes: nodes}
	for _, o := range opts {
		o(m)
	}
	return m, nil
}

// Format renders the message with the given values. Missing arguments,
// malformed value types and unsupported styles return typed errors.
func (m *MessageFormat) Format(values map[string]any) (string, error) {
	var b strings.Builder
	if err := m.formatNodes(&b, m.nodes, values, nil); err != nil {
		return "", err
	}
	return b.String(), nil
}

// pluralCtx carries the pound-sign context of an enclosing plural.
type pluralCtx struct {
	value  float64
	offset float64
}

func (m *MessageFormat) formatNodes(b *strings.Builder, nodes []node, values map[string]any, pound *pluralCtx) error {
	for _, n := range nodes {
		switch v := n.(type) {
		case literalNode:
			b.WriteString(string(v))
		case poundNode:
			s, err := formatNumber(pound.value-pound.offset, styleDecimal, "", resolveNumberData(m.locale))
			if err != nil {
				return err
			}
			b.WriteString(s)
		case argNode:
			val, ok := values[v.name]
			if !ok {
				return fmt.Errorf("messageformat: missing value %q", v.name)
			}
			b.WriteString(simpleString(val))
		case numberNode:
			val, ok := values[v.name]
			if !ok {
				return fmt.Errorf("messageformat: missing value %q", v.name)
			}
			f, err := toNumber(val)
			if err != nil {
				return fmt.Errorf("messageformat: argument %q: %w", v.name, err)
			}
			s, err := m.formatNumberArg(f, v.style)
			if err != nil {
				return err
			}
			b.WriteString(s)
		case dateTimeNode:
			val, ok := values[v.name]
			if !ok {
				return fmt.Errorf("messageformat: missing value %q", v.name)
			}
			t, err := toTime(val)
			if err != nil {
				return fmt.Errorf("messageformat: argument %q: %w", v.name, err)
			}
			s, err := m.formatDateTimeArg(t, v.isTime, v.style)
			if err != nil {
				return err
			}
			b.WriteString(s)
		case selectNode:
			val, ok := values[v.name]
			if !ok {
				return fmt.Errorf("messageformat: missing value %q", v.name)
			}
			key := simpleString(val)
			body := pickOption(v.options, key)
			if err := m.formatNodes(b, body, values, pound); err != nil {
				return err
			}
		case pluralNode:
			val, ok := values[v.name]
			if !ok {
				return fmt.Errorf("messageformat: missing value %q", v.name)
			}
			f, err := toNumber(val)
			if err != nil {
				return fmt.Errorf("messageformat: argument %q: %w", v.name, err)
			}
			// Exact =N selectors match the raw value; keyword selection uses
			// value - offset (ICU semantics).
			var body []node
			matched := false
			for _, e := range v.exact {
				if e.value == f {
					body = e.nodes
					matched = true
					break
				}
			}
			if !matched {
				op, err := operandsFromString(jsNumberString(f - v.offset))
				if err != nil {
					return fmt.Errorf("messageformat: argument %q: %w", v.name, err)
				}
				rules := cldrCardinalRules
				if v.ordinal {
					rules = cldrOrdinalRules
				}
				body = pickOption(v.options, selectPlural(rules, m.locale, op))
			}
			ctx := &pluralCtx{value: f, offset: v.offset}
			if err := m.formatNodes(b, body, values, ctx); err != nil {
				return err
			}
		case tagNode:
			val, ok := values[v.name]
			if !ok {
				return fmt.Errorf("messageformat: missing value for tag %q", v.name)
			}
			fn, ok := val.(TagFunc)
			if !ok {
				if raw, ok2 := val.(func([]string) string); ok2 {
					fn = TagFunc(raw)
				} else {
					return fmt.Errorf("messageformat: tag %q requires a TagFunc value", v.name)
				}
			}
			var inner strings.Builder
			if err := m.formatNodes(&inner, v.children, values, pound); err != nil {
				return err
			}
			b.WriteString(fn([]string{inner.String()}))
		}
	}
	return nil
}

func pickOption(opts []selectOption, key string) []node {
	for _, o := range opts {
		if o.key == key {
			return o.nodes
		}
	}
	for _, o := range opts {
		if o.key == "other" {
			return o.nodes
		}
	}
	return nil
}

func (m *MessageFormat) formatNumberArg(f float64, style string) (string, error) {
	data := resolveNumberData(m.locale)
	// Custom formats shadow built-in style names, as in FormatJS.
	if style != "" {
		if spec, ok := m.numFormats[style]; ok {
			switch spec.Style {
			case "decimal":
				return formatNumber(f, styleDecimal, "", data)
			case "integer":
				return formatNumber(f, styleInteger, "", data)
			case "percent":
				return formatNumber(f, stylePercent, "", data)
			case "currency":
				if spec.Currency == "" {
					return "", errCurrencyCodeRequired
				}
				return formatNumber(f, styleCurrency, spec.Currency, data)
			default:
				return "", fmt.Errorf("messageformat: unsupported custom number format style %q", spec.Style)
			}
		}
	}
	switch style {
	case "":
		return formatNumber(f, styleDecimal, "", data)
	case "integer":
		return formatNumber(f, styleInteger, "", data)
	case "percent":
		return formatNumber(f, stylePercent, "", data)
	case "currency":
		return "", errCurrencyCodeRequired
	default:
		if strings.HasPrefix(style, "::") {
			return "", fmt.Errorf("messageformat: number skeletons are not supported (style %q); use integer, percent or a registered currency format", style)
		}
		return "", fmt.Errorf("messageformat: unknown number format %q", style)
	}
}

func (m *MessageFormat) formatDateTimeArg(t time.Time, isTime bool, style string) (string, error) {
	// Locales whose default calendar is not Gregorian would silently render
	// wrong dates from the gregorian-only data; fail clearly instead.
	loc := strings.ReplaceAll(m.locale, "-", "_")
	for l := loc; l != ""; {
		if nonGregorianDefaultLocales[l] {
			return "", fmt.Errorf("messageformat: locale %q defaults to a non-Gregorian calendar, which is not supported; use a Gregorian-calendar locale", m.locale)
		}
		if idx := strings.LastIndex(l, "_"); idx > 0 {
			l = l[:idx]
		} else {
			break
		}
	}
	t = t.UTC()
	if style == "" {
		if isTime {
			style = "medium"
		} else {
			style = "default"
		}
	}
	if strings.HasPrefix(style, "::") {
		return "", fmt.Errorf("messageformat: date/time skeletons are not supported (style %q); use full, long, medium or short", style)
	}
	switch style {
	case "full", "long", "medium", "short", "default":
		key := "date." + style
		if isTime {
			key = "time." + style
		}
		pat, d := resolveFormatjsStyle(m.locale, key)
		if pat == "" {
			return "", fmt.Errorf("messageformat: no %s style data for locale %q", key, m.locale)
		}
		return formatDatePattern(t, pat, d, resolveNumberData(m.locale).NS), nil
	default:
		// FormatJS 12 silently ignores user date patterns (falls back to
		// medium); this port returns a typed error instead so callers notice.
		return "", fmt.Errorf("messageformat: user date patterns are not supported (style %q); use full, long, medium or short", style)
	}
}

// toNumber accepts the numeric types JSON callers actually hold.
func toNumber(v any) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case int32:
		return float64(n), nil
	case json.Number:
		return n.Float64()
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
		if err != nil {
			return 0, fmt.Errorf("value %q is not a number", n)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("value of type %T is not a number", v)
	}
}

func toTime(v any) (time.Time, error) {
	switch t := v.(type) {
	case time.Time:
		return t, nil
	case string:
		parsed, err := time.Parse(time.RFC3339, t)
		if err != nil {
			return time.Time{}, fmt.Errorf("value %q is not an RFC3339 timestamp", t)
		}
		return parsed, nil
	default:
		return time.Time{}, fmt.Errorf("value of type %T is not a time.Time", v)
	}
}

// simpleString mirrors JS String(value) for simple {arg} substitution.
func simpleString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		return jsNumberString(x)
	case float32:
		return jsNumberString(float64(x))
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case nil:
		return "undefined"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// jsNumberString mirrors V8 String(number): decimal for |x| in [1e-6, 1e21),
// exponent form outside, shortest round-trip representation.
func jsNumberString(v float64) string {
	if v == 0 {
		return "0"
	}
	if math.IsNaN(v) {
		return "NaN"
	}
	if math.IsInf(v, 1) {
		return "Infinity"
	}
	if math.IsInf(v, -1) {
		return "-Infinity"
	}
	av := math.Abs(v)
	if av >= 1e-6 && av < 1e21 {
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	s := strconv.FormatFloat(v, 'e', -1, 64)
	// Go: "1e+21" already matches JS; Go pads small exponents as "e-06": fix.
	if i := strings.IndexByte(s, 'e'); i >= 0 {
		mant, exp := s[:i], s[i+1:]
		sign := ""
		if strings.HasPrefix(exp, "+") || strings.HasPrefix(exp, "-") {
			sign = string(exp[0])
			exp = exp[1:]
		}
		exp = strings.TrimLeft(exp, "0")
		if exp == "" {
			exp = "0"
		}
		s = mant + "e" + sign + exp
	}
	return s
}
