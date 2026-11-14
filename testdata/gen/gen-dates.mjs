// Differential fixtures: Intl.DateTimeFormat 4 date styles + 4 time styles, fixed UTC instants.
import { writeFileSync } from 'node:fs';
import { LOCALES } from './corpus.mjs';

const INSTANTS = [
  '2024-03-15T14:30:45Z', '2024-07-04T00:05:09Z', '1969-12-31T23:59:59Z',
  '2025-01-01T12:00:00Z', '2000-02-29T06:07:08Z', '2024-11-30T18:45:30Z',
];
const out = [];
for (const loc of LOCALES) {
  for (const iso of INSTANTS) {
    const d = new Date(iso);
    const rec = { locale: loc, iso };
    for (const style of ['full', 'long', 'medium', 'short']) {
      try { rec['date' + style] = new Intl.DateTimeFormat(loc, { dateStyle: style, timeZone: 'UTC' }).format(d); } catch (e) { rec['date' + style + 'Err'] = e.message; }
      try { rec['time' + style] = new Intl.DateTimeFormat(loc, { timeStyle: style, timeZone: 'UTC' }).format(d); } catch (e) { rec['time' + style + 'Err'] = e.message; }
    }
    out.push(rec);
  }
}
writeFileSync('../date_fixtures.json', JSON.stringify({ meta: { node: process.version, icu: process.versions.icu, cldr: process.versions.cldr, tz: 'UTC' }, fixtures: out }));
console.log('date fixtures:', out.length);
