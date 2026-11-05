// usage: node golden.js outDir [scene,scene...]
const { chromium } = require('playwright');
const fs = require('fs');
const ALL = ['menu','classsel','map','world1','world2','world3','world4','world5','world6','multiball','classic2','classic5','boss','shop','reward'];
(async () => {
  const [out, list] = process.argv.slice(2);
  const scenes = list ? list.split(',') : ALL;
  fs.mkdirSync(out, { recursive: true });
  const b = await chromium.launch({ args: ['--use-gl=swiftshader', '--enable-unsafe-swiftshader'] });
  for (const sc of scenes) {
    const p = await b.newPage({ viewport: { width: 1920, height: 1080 } });
    p.on('pageerror', e => console.log('ERR ' + e.message));
    p.on('console', m => { const t = m.text(); if (/panic|goroutine|runtime error|fatal/i.test(t)) console.log('C:', t.slice(0, 2000)) });
    await p.goto('http://localhost:8767/web'+(process.env.WEB||'6')+'/index.html');
    await p.waitForFunction(() => typeof window.__capTotal === 'number', null, { timeout: 60000 });
    await p.evaluate(s => { window.__golden = s; window.__capFPS = 60 }, sc);
    await p.waitForTimeout(2500);
    const N = 90;
    for (let i = 1; i <= N; i++) {
      await p.evaluate(([n, d]) => { window.__capDraw = d; window.__capReq = n }, [i, i >= N - (+(process.env.DRAWS||1)) + 1]);
      await p.waitForFunction(n => window.__capDone >= n, i, { timeout: 120000, polling: 16 });
    }
    await p.evaluate(() => new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r))));
    await p.screenshot({ path: `${out}/${sc}.png` });
    console.log('shot', sc);
    await p.close();
  }
  await b.close();
})();
