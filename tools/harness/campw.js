const { chromium } = require('playwright');
const fs = require('fs');
(async () => {
  const [out, params] = process.argv.slice(2);
  const b = await chromium.launch({ args: ['--use-gl=swiftshader', '--enable-unsafe-swiftshader'] });
  const p = await b.newPage({ viewport: { width: 320, height: 180 } });
  p.on('console', m => { const t = m.text(); if (/panic|runtime error|fatal/i.test(t)) console.log('C:', t.slice(0, 4000)) });
  await p.goto('http://localhost:8767/web'+(process.env.WEB||'8')+'/index.html');
  await p.waitForFunction(() => typeof window.__campaign === 'function', null, { timeout: 60000 });
  await p.evaluate(() => { window.__capDraw = false; window.__capReq = 1 });
  await p.waitForFunction(() => window.__capDone >= 1, null, { timeout: 60000 });
  const t0 = Date.now();
  const r = await p.evaluate(s => window[s.startsWith("C")?"__campaign":"__balance"](s.startsWith("C")?s.slice(1):s), params || '{}');
  console.log('sim took', (Date.now() - t0) / 1000, 's');
  fs.writeFileSync(out, r);
  await b.close();
})();
