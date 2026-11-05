const { chromium } = require('playwright');
(async () => {
  const [sc, frame] = process.argv.slice(2);
  const b = await chromium.launch({ args: ['--use-gl=swiftshader', '--enable-unsafe-swiftshader'] });
  const p = await b.newPage({ viewport: { width: 1920, height: 1080 } });
  await p.goto('http://localhost:8767/web'+process.env.WEB+'/index.html');
  await p.waitForFunction(() => typeof window.__capTotal === 'number', null, { timeout: 60000 });
  await p.evaluate(s => { window.__golden = s; window.__capFPS = 60 }, sc);
  await p.waitForTimeout(2500);
  const N = +(frame||90);
  for (let i = 1; i <= N; i++) {
    if (i === N) await p.evaluate(() => { window.__hashRes=''; window.__hashFrame() });
    await p.evaluate(([n, d]) => { window.__capDraw = d; window.__capReq = n }, [i, i >= N - (+(process.env.DRAWS||1)) + 1]);
    await p.waitForFunction(n => window.__capDone >= n, i, { timeout: 120000, polling: 16 });
  }
  console.log(await p.evaluate(() => window.__hashRes));
  await b.close();
})();
