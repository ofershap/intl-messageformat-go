// Package messageformat renders ICU MessageFormat 1 messages in Go,
// differentially tested against intl-messageformat (FormatJS) 12.1.5 on
// Node v22.23.4 (ICU 78.3, Unicode 17.0, CLDR 48.0).
//
// Supported: simple arguments, select, plural and selectordinal with offset
// and =N exact selectors, the # pound sign, nested messages, number
// formatting (default, integer, percent and named custom formats such as
// currency), date and time formatting in the four CLDR style presets
// (full/long/medium/short) at UTC, rich-text tags via TagFunc, and FormatJS
// apostrophe quoting rules. Plural categories, number formats and date/time
// patterns are computed from generated CLDR 48 data for 700+ locales.
//
// Deliberate v1 exclusions (typed errors, never silent wrong output):
// number and date/time skeletons (::...), user-supplied date pattern
// strings (FormatJS 12 silently ignores them and falls back to medium),
// and locales whose default calendar is non-Gregorian.
//
// Locale-dependent output mirrors V8/Node behavior, including its plain-space
// (not narrow-no-break-space) date-time separators and its String(number)
// exponent boundaries.
package messageformat
