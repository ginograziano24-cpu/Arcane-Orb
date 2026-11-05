// usage: WEB=12 node planb.js scene,scene reps
const { chromium } = require('playwright');
(async () => {
  const [list, repsS] = process.argv.slice(2);
  const reps = +(repsS||60);
  const b = await chromium.launch({ args: ['--use-gl=swiftshader', '--enable-unsafe-swiftshader'] });
  for (const sc of list.split(',')) {
    const p = await b.newPage({ viewport: { width: 1920, height: 1080 } });
    p.on('console', m => { const t = m.text(); if (/panic|goroutine|runtime error|fatal/i.test(t)) console.log('C:', t.slice(0, 2000)) });
    await p.goto('http://localhost:8767/web'+(process.env.WEB||'12')+'/index.html');
    await p.waitForFunction(() => typeof window.__capTotal === 'number', null, { timeout: 60000 });
    await p.evaluate(s => { window.__golden = s; window.__capFPS = 60 }, sc);
    await p.waitForTimeout(2500);
    for (let i = 1; i <= 40; i++) {
      await p.evaluate(([n, d]) => { window.__capDraw = d; window.__capReq = n }, [i, i >= 38]);
      await p.waitForFunction(n => window.__capDone >= n, i, { timeout: 120000, polling: 16 });
    }
    await p.evaluate(() => { window.__jobBenchRes = ''; window.__jobBench(30) });
    for (let i = 41; i <= 75; i++) {
      await p.evaluate(([n]) => { window.__capDraw = true; window.__capReq = n }, [i]);
      await p.waitForFunction(n => window.__capDone >= n, i, { timeout: 120000, polling: 16 });
    }
    console.log(sc.padEnd(10), 'jobs', await p.evaluate(() => window.__jobBenchRes));
    if (process.env.PIECE) {
      await p.evaluate(() => { window.__pieceRes = ''; window.__pieceCheck() });
      await p.evaluate(([n]) => { window.__capDraw = true; window.__capReq = n }, [76]);
      await p.waitForFunction(() => window.__pieceRes, null, { timeout: 120000, polling: 50 });
      console.log('   piece', await p.evaluate(() => window.__pieceRes));
      await p.close(); continue;
    }
    if (process.env.MESH) {
      await p.evaluate(() => { window.__meshBenchRes = ''; window.__meshBench(20) });
      for (let i = 76; i <= 99; i++) {
        await p.evaluate(([n]) => { window.__capDraw = true; window.__capReq = n }, [i]);
        await p.waitForFunction(n => window.__capDone >= n, i, { timeout: 120000, polling: 16 });
      }
      console.log('   mesh', await p.evaluate(() => window.__meshBenchRes));
    }
    if (process.env.JOBS) { await p.close(); continue; }
    const res = [];
    for (let k = 0; k < 3; k++) {
      await p.evaluate(r => { window.__planBenchRes = ''; window.__planBench(r) }, reps);
      const i = 81 + k;
      await p.evaluate(([n]) => { window.__capDraw = true; window.__capReq = n }, [i]);
      await p.waitForFunction(() => window.__planBenchRes, null, { timeout: 120000, polling: 50 });
      res.push(await p.evaluate(() => window.__planBenchRes));
    }
    console.log(sc.padEnd(10), res.join('\n           ')); 
    await p.close();
  }
  await b.close();
})();
