import IntlMessageFormat from 'intl-messageformat';
import { writeFileSync } from 'node:fs';
import { CASES, LOCALES, META } from './corpus.mjs';

const fixtures = [];
for (const c of CASES) {
  const valueSets = c.valuesAlt || [c.values];
  for (const locale of LOCALES) {
    for (let i = 0; i < valueSets.length; i++) {
      const values = { ...valueSets[i] };
      const rec = { id: c.id, variant: i, locale, message: c.message, values };
      if (c.tags) {
        // tag values become marker-wrapping functions; Go mirrors with same markers
        for (const k of Object.keys(values)) {
          if (typeof values[k] === 'string' && values[k].startsWith('<') && values[k].endsWith('>')) {
            const marker = values[k];
            values[k] = (parts) => `${marker}${parts.join('')}</${marker.slice(1)}`;
          }
        }
      }
      if (typeof values.d === 'number') values.d = new Date(values.d);
      if (typeof values.t === 'number') values.t = new Date(values.t);
      try {
        const overrideFormats = c.needsCurrency ? { number: { currency: { style: 'currency', currency: 'USD' } } } : undefined;
        const mf = new IntlMessageFormat(c.message, locale, overrideFormats);
        rec.expected = mf.format(values);
        if (typeof rec.expected !== 'string') rec.expected = rec.expected.join('');
      } catch (e) {
        rec.error = e.constructor.name + ': ' + e.message;
      }
      if (c.expectError && !rec.error) rec.jsFormattedAnyway = true;
      fixtures.push(rec);
    }
  }
}
const meta = {
  generatedAt: new Date().toISOString(),
  node: process.version,
  icu: process.config.variables.icu_ver_major || null,
  intlMessageFormat: JSON.parse((await import('node:fs')).readFileSync(new URL('./node_modules/intl-messageformat/package.json', import.meta.url))).version,
  cldr: null,
  locales: LOCALES,
  count: fixtures.length,
};
writeFileSync('../fixtures.json', JSON.stringify({ meta, fixtures }, null, 1));
console.log(`wrote ${fixtures.length} fixtures, node ${meta.node}, imf ${meta.intlMessageFormat}`);
const errs = fixtures.filter(f => f.error);
console.log(`js-side errors: ${errs.length}`);
for (const e of errs.slice(0, 8)) console.log(' ', e.id, e.locale, e.error.slice(0, 90));
