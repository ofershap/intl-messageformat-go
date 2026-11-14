// Fixture corpus: grammar coverage x locales x values.
// Each case: {id, message, values (per-locale shared or overridden)}.
// Fixture record: {id, locale, message, values, expected, error?}

export const LOCALES = ['en', 'he', 'ar', 'ru', 'pl', 'cy', 'ja', 'zh', 'ko', 'fr', 'de', 'es', 'pt-BR', 'hi', 'ar-EG'];

const D = 1731686400000; // fixed ms: 2024-11-15T13:20:00Z

export const CASES = [
  { id: 'simple-arg', message: 'Hello {name}', values: { name: 'Dana' } },
  { id: 'simple-two-args', message: '{first} and {second}', values: { first: 'A', second: 'B' } },
  { id: 'missing-arg', message: 'Hello {name}', values: {}, expectError: true },
  { id: 'select-basic', message: '{gender, select, male {He} female {She} other {They}} replied', values: { gender: 'female' } },
  { id: 'select-other-fallback', message: '{gender, select, male {He} female {She} other {They}} replied', values: { gender: 'x' } },
  { id: 'plural-basic', message: '{count, plural, one {# item} other {# items}}', values: { count: 1 }, valuesAlt: [{ count: 1 }, { count: 2 }, { count: 0 }, { count: 5 }, { count: 11 }, { count: 100 }, { count: 1.5 }] },
  { id: 'plural-exact-match', message: '{count, plural, =0 {No items} one {One item} other {# items}}', values: { count: 0 }, valuesAlt: [{ count: 0 }, { count: 1 }, { count: 7 }] },
  { id: 'plural-offset', message: '{count, plural, offset:1 =0 {Nobody} =1 {You} one {You and one other} other {You and # others}}', values: { count: 2 }, valuesAlt: [{ count: 0 }, { count: 1 }, { count: 2 }, { count: 4 }] },
  { id: 'selectordinal', message: '{pos, selectordinal, one {#st} two {#nd} few {#rd} other {#th}}', values: { pos: 1 }, valuesAlt: [{ pos: 1 }, { pos: 2 }, { pos: 3 }, { pos: 4 }, { pos: 11 }, { pos: 22 }] },
  { id: 'nested-plural-in-select', message: '{kind, select, a {{n, plural, one {one a} other {# as}}} other {{n, plural, one {one b} other {# bs}}}}', values: { kind: 'a', n: 3 } },
  { id: 'number-default', message: 'Total: {n, number}', values: { n: 1234567.891 }, valuesAlt: [{ n: 1234567.891 }, { n: 0 }, { n: -3.14159 }, { n: 1000000 }, { n: 0.5 }, { n: 1e21 }] },
  { id: 'number-integer', message: '{n, number, integer}', values: { n: 1234.5 }, valuesAlt: [{ n: 1234.5 }, { n: 1234.4 }, { n: 2.5 }, { n: 3.5 }, { n: -2.5 }] },
  { id: 'number-percent', message: '{n, number, percent}', values: { n: 0.42 } },
  { id: 'number-currency', message: '{n, number, currency}', needsCurrency: true, values: { n: 1234.56 }, valuesAlt: [{ n: 1234.56 }, { n: 0.5 }, { n: -9.99 }] },
  { id: 'number-rounding-boundary', message: '{n, number}', values: { n: 1.005 }, valuesAlt: [{ n: 1.005 }, { n: 2.675 }, { n: 0.125 }, { n: 0.375 }, { n: 1.0005 }, { n: 123.4565 }] },
  { id: 'date-default', message: 'Arrives {d, date}', values: { d: D } },
  { id: 'date-styles', message: '{d, date, short} | {d, date, medium} | {d, date, long} | {d, date, full}', values: { d: D } },
  { id: 'time-styles', message: '{t, time, short} | {t, time, medium} | {t, time, long} | {t, time, full}', values: { t: D } },
  { id: 'date-explicit-pattern', message: '{d, date, yyyy-MM-dd}', values: { d: D } },
  { id: 'date-pattern-names', message: '{d, date, EEEE d MMMM yyyy}', values: { d: D } },
  { id: 'rich-text', message: 'Click <b>{what}</b> now', values: { what: 'here', b: '<B>' }, tags: true },
  { id: 'apostrophe-quoting', message: "It''s '{count}' {count, plural, one {item} other {items}}", values: { count: 2 } },
  { id: 'apostrophe-literal-brace', message: "'{'{x}'}'", values: { x: 'v' } },
  { id: 'unsupported-number-skeleton', message: '{n, number, ::currency/USD}', values: { n: 5 }, expectError: true, note: 'v1: skeletons unsupported -> Go must return typed error; JS may format fine' },
  { id: 'unsupported-date-skeleton', message: '{d, date, ::yyyyMMdd}', values: { d: D }, expectError: true, note: 'v1: skeletons unsupported' },
];

export const META = { fixedDateUTC: D };
