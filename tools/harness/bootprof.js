const { chromium } = require('playwright');
(async () => {
  const b = await chromium.launch({ args: ['--use-gl=swiftshader', '--enable-unsafe-swiftshader'] });
  const p = await b.newPage({ viewport: { width: 1920, height: 1080 } });
  const cdp = await p.context().newCDPSession(p);
  await cdp.send('Profiler.enable');
  await cdp.send('Profiler.setSamplingInterval', { interval: 500 });
  await cdp.send('Profiler.start');
  const t0 = Date.now();
  await p.goto('http://localhost:8767/web'+process.env.WEB+'/index.html');
  await p.waitForFunction(() => typeof window.__capTotal === 'number', null, { timeout: 120000, polling: 10 });
  const t1 = Date.now();
  await p.waitForTimeout(+(process.env.WAIT||3000));
  const { profile } = await cdp.send('Profiler.stop');
  console.log('until game globals:', t1 - t0, 'ms');
  // self time per function
  const self = new Map(); const byId = new Map();
  for (const n of profile.nodes) byId.set(n.id, n);
  const dt = profile.timeDeltas; const counts = new Map();
  profile.samples.forEach((id, i) => counts.set(id, (counts.get(id)||0) + (dt[i]||0)));
  for (const [id, us] of counts) { const n = byId.get(id); const k = n.callFrame.functionName || '(anon)'; self.set(k, (self.get(k)||0) + us); }
  const top = [...self.entries()].sort((a,b)=>b[1]-a[1]).slice(0, 35);
  for (const [k, us] of top) console.log((us/1000).toFixed(1).padStart(8), 'ms', k);
  await b.close();
})();
