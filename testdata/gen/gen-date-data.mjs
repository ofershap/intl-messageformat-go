// Generate cldr_date_data.go from cldr-dates-full: gregorian 4-style date/time
// patterns, month/day names (wide+abbreviated), am/pm day periods, UTC long
// zone name, per locale.
import { readFileSync, writeFileSync, readdirSync } from 'node:fs';

const dir = 'node_modules/cldr-dates-full/main';
const entries = [];
for (const loc of readdirSync(dir)) {
  let g, tzn;
  try { g = JSON.parse(readFileSync(`${dir}/${loc}/ca-gregorian.json`, 'utf8')).main[loc].dates.calendars.gregorian; } catch { continue; }
  try { tzn = JSON.parse(readFileSync(`${dir}/${loc}/timeZoneNames.json`, 'utf8')).main[loc].dates.timeZoneNames; } catch { tzn = null; }
  if (!g.dateFormats || !g.months) continue;
  const pick = (o) => o ? o : null;
  const utcLong = tzn?.zone?.Etc?.UTC?.long?.standard ?? 'Coordinated Universal Time';
  const pv = (v) => (v && typeof v === 'object') ? v._value : v;
  entries.push({
    loc: loc.replace(/-/g, '_'),
    date: { full: pv(g.dateFormats.full), long: pv(g.dateFormats.long), medium: pv(g.dateFormats.medium), short: pv(g.dateFormats.short) },
    time: { full: pv(g.timeFormats.full), long: pv(g.timeFormats.long), medium: pv(g.timeFormats.medium), short: pv(g.timeFormats.short) },
    monthsWide: g.months.format.wide, monthsAbbr: g.months.format.abbreviated,
    daysWide: g.days.format.wide, daysAbbr: g.days.format.abbreviated,
    // ICU 'a' uses abbreviated dayPeriods, but locales whose abbreviated
    // am/pm is the ASCII root value (ko) render the wide form instead.
    am: (g.dayPeriods?.format?.abbreviated?.am === 'AM' && g.dayPeriods?.format?.wide?.am && g.dayPeriods.format.wide.am !== 'AM') ? g.dayPeriods.format.wide.am : (g.dayPeriods?.format?.abbreviated?.am ?? 'AM'),
    pm: (g.dayPeriods?.format?.abbreviated?.pm === 'PM' && g.dayPeriods?.format?.wide?.pm && g.dayPeriods.format.wide.pm !== 'PM') ? g.dayPeriods.format.wide.pm : (g.dayPeriods?.format?.abbreviated?.pm ?? 'PM'),
    utcLong,
  });
}
let src = `// Code generated from cldr-dates-full@48.0.0 by testdata/gen/gen-date-data.mjs. DO NOT EDIT.\npackage messageformat\n\ntype dateLocaleData struct {\n\tDateFormats map[string]string\n\tTimeFormats map[string]string\n\tMonthsWide  []string // index 0 = month 1\n\tMonthsAbbr  []string\n\tDaysWide    []string // sun..sat\n\tDaysAbbr    []string\n\tAM, PM      string\n\tUTCLong     string\n}\n\nvar dateData = map[string]*dateLocaleData{\n`;
const dayKeys = ['sun', 'mon', 'tue', 'wed', 'thu', 'fri', 'sat'];
for (const e of entries) {
  const mw = [], ma = [], dw = [], da = [];
  for (let i = 1; i <= 12; i++) { mw.push(e.monthsWide[String(i)] ?? String(i)); ma.push(e.monthsAbbr[String(i)] ?? String(i)); }
  for (const k of dayKeys) { dw.push(e.daysWide[k] ?? k); da.push(e.daysAbbr[k] ?? k); }
  src += `\t${JSON.stringify(e.loc)}: {DateFormats: map[string]string{"full": ${JSON.stringify(e.date.full)}, "long": ${JSON.stringify(e.date.long)}, "medium": ${JSON.stringify(e.date.medium)}, "short": ${JSON.stringify(e.date.short)}}, TimeFormats: map[string]string{"full": ${JSON.stringify(e.time.full)}, "long": ${JSON.stringify(e.time.long)}, "medium": ${JSON.stringify(e.time.medium)}, "short": ${JSON.stringify(e.time.short)}}, MonthsWide: []string{${mw.map((m) => JSON.stringify(m)).join(', ')}}, MonthsAbbr: []string{${ma.map((m) => JSON.stringify(m)).join(', ')}}, DaysWide: []string{${dw.map((m) => JSON.stringify(m)).join(', ')}}, DaysAbbr: []string{${da.map((m) => JSON.stringify(m)).join(', ')}}, AM: ${JSON.stringify(e.am)}, PM: ${JSON.stringify(e.pm)}, UTCLong: ${JSON.stringify(e.utcLong)}},\n`;
}
src += '}\n';
writeFileSync('../../cldr_date_data.go', src);
console.log('date data:', entries.length, 'locales,', src.length, 'bytes');
