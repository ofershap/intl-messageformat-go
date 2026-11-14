# intl-messageformat-go

ICU MessageFormat 1 for Go, verified byte-for-byte against [intl-messageformat](https://formatjs.github.io/docs/intl-messageformat) (FormatJS) 12.1.5 on 5,505 differential checks.

Your product already has its messages in MessageFormat: react-intl, FormatJS, Angular's i18n, ICUi18n exports, Crowdin and Lokalise all speak it. If any part of your stack runs Go (APIs, email renderers, notification workers, CLIs), those messages previously could not render there. Porting them to Go templates means rewriting every plural rule by hand and quietly losing the translator-facing syntax your localization pipeline depends on.

This package renders the same messages in Go with the same output.

```go
mf, _ := messageformat.New("he", "{count, plural, one {הודעה אחת} other {# הודעות}}")
out, _ := mf.Format(map[string]any{"count": 5})
// "5 הודעות"
```

## Install

```sh
go get github.com/ofershap/intl-messageformat-go
```

No cgo, no dependencies. The CLDR 48 locale data (766 locales) is generated Go source compiled into the package.

## Supported subset

- Simple arguments: `Hello {name}`
- `select`, `plural`, `selectordinal` with `offset:` and `=N` exact selectors
- `#` inside plural bodies (formats value minus offset)
- Nested messages (plural inside select, and so on)
- Numbers: `{n, number}`, `integer`, `percent`, and named custom formats via `WithNumberFormats` (currency included)
- Dates and times: `{d, date, full|long|medium|short}` and the same for `time`, at UTC, with the exact style mapping FormatJS 12 uses (which is not `Intl.DateTimeFormat`'s `dateStyle`/`timeStyle`)
- Rich-text tags through `TagFunc`: `Click <b>{what}</b>`
- FormatJS apostrophe quoting (`'{'`, `''`, `'#'`)

## Deliberate exclusions (typed errors, never silent wrong text)

Valid MessageFormat that uses these returns a descriptive error instead of wrong output:

- Number and date/time skeletons (`::currency/USD`, `::yyyyMMdd`)
- User-supplied date pattern strings (`{d, date, yyyy-MM-dd}`). FormatJS 12 silently ignores these and falls back to medium; this package tells you instead.
- Non-Gregorian default calendars
- Time zones other than UTC: `time.Time` values are rendered at UTC. Convert before formatting for other zones.

## Compatibility evidence

Every behavior above is pinned by golden fixtures generated from the reference implementation and compared exactly:

| Corpus | Checks | Reference |
| --- | --- | --- |
| Messages (parser + formatter) | 855 | intl-messageformat 12.1.5 |
| Numbers | 2,700 (450 values x 6 styles) | Intl.NumberFormat |
| Dates and times | 720 (90 instants x 8 styles) | Intl.DateTimeFormat |
| Plural categories | 1,230 | Intl.PluralRules |

Reference environment: Node v22.23.4, ICU 78.3, Unicode 17.0, CLDR 48.0. The fixture generator, its pinned lockfile and all fixtures live in `testdata/`, so the corpus is reproducible. "Verified" means byte-identical output on this corpus across the 15 exercised locales (en, he, ar, ar-EG, ru, pl, cy, ja, zh, ko, fr, de, es, pt-BR, hi), not a claim of universal equivalence on every possible message.

The differential approach also documents where JavaScript itself is surprising: V8 plural-category behavior on exponent-form numbers, FormatJS 12 silently dropping user date patterns, and its use of `timeZoneName: 'short'` for both long and full time styles are covered by tests.

## Related projects

Go searches today find a small MessageFormat 2 draft port, a few early-stage MF1 parsers without CLDR formatting, and file-format parsers; nothing rendering MF1 with generated CLDR 48 data against a FormatJS corpus (checks run 2026-11-14). For the JavaScript side of your stack, use [FormatJS](https://formatjs.github.io/) itself.

## License

MIT for the Go code. The generated locale data derives from [CLDR](https://cldr.unicode.org) 48.0 under the Unicode License v3; see NOTICE.
