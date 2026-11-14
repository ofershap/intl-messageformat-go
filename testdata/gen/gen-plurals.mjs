import { writeFileSync } from 'node:fs';
import { LOCALES } from './corpus.mjs';

const VALUES = [0,1,2,3,4,5,6,10,11,12,14,20,21,22,24,25,100,101,102,111,112,122,1000,1000000,
  0.0,1.0,2.0,10.0, 0.5,1.5,2.5,3.1, 0.1,0.01,2.25, -1,-2,-0.5, 1e3,1e6,1e21];
const out = [];
for (const loc of LOCALES) {
  for (const type of ['cardinal','ordinal']) {
    let pr;
    try { pr = new Intl.PluralRules(loc, {type}); } catch(e) { continue; }
    for (const v of VALUES) {
      out.push({locale: loc, type, value: v, str: String(v), expected: pr.select(v)});
    }
  }
}
writeFileSync('../plural_fixtures.json', JSON.stringify({meta:{node:process.version,icu:process.versions.icu,cldr:process.versions.cldr}, fixtures: out}));
console.log('plural fixtures:', out.length);
