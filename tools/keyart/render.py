import asyncio, base64, sys, subprocess, time, os
from playwright.async_api import async_playwright
D=sys.argv[1]
ASSETS=[('main',1232,706,'main_capsule_1232x706.png'),('header',920,430,'header_capsule_920x430.png'),('small',462,174,'small_capsule_462x174.png'),
        ('vertical',748,896,'vertical_capsule_748x896.png'),('libcap',600,900,'library_capsule_600x900.png'),('hero',3840,1240,'library_hero_3840x1240.png'),('logo',1280,720,'library_logo_1280x720.png')]
only=sys.argv[2].split(',') if len(sys.argv)>2 else None
async def main():
    srv=subprocess.Popen(['python3','-m','http.server','8791','--bind','127.0.0.1'],cwd=D,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL); time.sleep(1)
    try:
        async with async_playwright() as p:
            b=await p.chromium.launch(); pg=await b.new_page()
            errs=[]; pg.on('pageerror',lambda e: errs.append(str(e)))
            await pg.goto('http://127.0.0.1:8791/keyart.html'); await pg.evaluate('window.ready')
            for mode,w,h,name in ASSETS:
                if only and mode not in only: continue
                d=await pg.evaluate('([m,w,h])=>render(m,w,h)',[mode,w,h])
                open(os.path.join(D,'out',name),'wb').write(base64.b64decode(d.split(',')[1])); print('wrote',name)
            print('errors',errs); await b.close()
    finally: srv.terminate()
asyncio.run(main())
