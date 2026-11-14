# Contributing

Issues and pull requests are welcome.

## Ground rules

- Output changes need differential evidence: extend the fixture corpus in
  `testdata/gen/corpus.mjs`, regenerate fixtures with `npm ci && node gen.mjs`
  (plus gen-numbers.mjs / gen-plurals.mjs / gen-dates.mjs as applicable) and
  make `go test ./...` pass. A change that alters rendered output without a
  fixture will not be merged.
- Never weaken a typed error into silent wrong output. Supporting a new
  construct means implementing it fully against the reference behavior.
- Keep the supported-subset documentation in README.md and doc.go accurate.

## Regenerating CLDR data

`testdata/gen/` contains the generators (gen-plurals, gen-number-data,
gen-date-data, gen-formatjs-styles). They read the pinned cldr-*-full
packages from the committed lockfile and emit the cldr_*_data.go files.
