// Differential fixtures: Intl.NumberFormat default + integer/percent/currency styles.
import { writeFileSync } from 'node:fs';
import { LOCALES } from './corpus.mjs';

const VALUES = [0, 1, -1, 0.5, -0.5, 1.005, 2.675, 0.125, 0.375, 1.0005, 123.4565,
  3.14159, -3.14159, 1000, 1234567.891, 1000000, 1e9, 1e21, 0.0001, 999999.9999,
  0.9999, 2.5, 3.5, -2.5, 42, 100, 0.42, 1234.56, 0.001, 123456789.123456];

const out = [];
for (const loc of LOCALES) {
  for (const v of VALUES) {
    const rec = { locale: loc, value: v, str: String(v) };
    try { rec.decimal = new Intl.NumberFormat(loc).format(v); } catch (e) { rec.decimalErr = e.message; }
    try { rec.integer = new Intl.NumberFormat(loc, { maximumFractionDigits: 0 }).format(v); } catch (e) { rec.integerErr = e.message; }
    try { rec.percent = new Intl.NumberFormat(loc, { style: 'percent' }).format(v); } catch (e) { rec.percentErr = e.message; }
    try { rec.currencyUSD = new Intl.NumberFormat(loc, { style: 'currency', currency: 'USD' }).format(v); } catch (e) { rec.currencyErr = e.message; }
    try { rec.currencyJPY = new Intl.NumberFormat(loc, { style: 'currency', currency: 'JPY' }).format(v); } catch (e) { rec.currencyJPYErr = e.message; }
    try { rec.currencyILS = new Intl.NumberFormat(loc, { style: 'currency', currency: 'ILS' }).format(v); } catch (e) {}
    out.push(rec);
  }
}
writeFileSync('../number_fixtures.json', JSON.stringify({ meta: { node: process.version, icu: process.versions.icu, cldr: process.versions.cldr }, fixtures: out }));
console.log('number fixtures:', out.length);
