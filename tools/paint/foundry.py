import numpy as np, sys
from PIL import Image
from scipy.ndimage import gaussian_filter, zoom
sys.path.insert(0,'/tmp/paint'); from common import fbm,col,lerp
W,H=2048,1152; HY=int(H*0.72)
def ridge(rng,w,base,oct=6,rough=0.55):
    out=np.zeros(w,np.float32); amp=1; tot=0
    for o in range(oct):
        n=max(3,int(w/base*(2**o))+3); g=rng.random(n).astype(np.float32)
        z=zoom(g,w/(n-1),order=3)[:w]
        if len(z)<w: z=np.pad(z,(0,w-len(z)),mode='edge')
        out+=z*amp; tot+=amp; amp*=rough
    return out/tot
def paint(seed,out):
    rng=np.random.default_rng(seed)
    yy,xx=np.mgrid[0:H,0:W].astype(np.float32)
    t=np.clip(yy/HY,0,1)
    # smoky dusk: near-black top to ember orange at the horizon
    sky=lerp(col(14,11,14),col(44,26,24),t[...,None]**1.4)
    sky=lerp(sky,col(170,74,26),(np.clip((t-0.6)/0.4,0,1)**2.2)[...,None])
    img=sky.copy()
    # smoke banks
    for k,(yc,sc,a) in enumerate([(0.22,260,0.55),(0.42,200,0.5),(0.6,150,0.45)]):
        sm=fbm(rng,W,H,sc,6)
        band=np.exp(-((yy-HY*yc)/(HY*0.14))**2)
        m=np.clip((sm-0.45)*3,0,1)*band*a
        lit=np.clip((yy/HY-0.2),0,1)
        sc_=lerp(col(40,32,34),col(150,72,34),lit[...,None])
        img=img*(1-m[...,None])+sc_*m[...,None]
    # embers drifting up
    e=np.zeros((H,W),np.float32)
    ex=rng.integers(0,W,260); ey=(HY*(1-0.75*rng.random(260)**1.8)).astype(int)
    ey=np.clip(ey,0,H-1); e[ey,ex]=rng.random(260)**2
    img+=(gaussian_filter(e,1.6)*20+gaussian_filter(e,0.6)*3)[...,None]*col(255,150,50)
    # far ridge of slag heaps
    r=ridge(rng,W,600,6); top=HY-(0.04+0.12*r)*HY
    m=np.clip((yy-top[None,:])/1.5,0,1)
    img=img*(1-m[...,None])+col(46,28,24)[None,None,:]*m[...,None]
    # citadel silhouette: towers, chimneys, a great gear, gantries
    sil=np.zeros((H,W),bool); dk=col(16,12,14)
    rim=np.zeros((H,W),np.float32)
    def box(x0,x1,y0,y1):
        sil[max(0,int(y0)):int(y1),max(0,int(x0)):min(W,int(x1))]=True
    cx=W*(0.35+0.3*rng.random())
    base=HY-HY*0.035
    # massive central keep with stepped buttresses
    for i,(w,h) in enumerate([(380,170),(300,240),(200,320),(110,390)]):
        box(cx-w/2,cx+w/2,base-h,base+40)
    # battlements on the keep top
    for k in range(-2,3):
        box(cx+k*24-8,cx+k*24+8,base-405,base-390)
    # flanking towers with conical caps
    for s in (-1,1):
        tx=cx+s*(260+rng.integers(0,60)); th=230+rng.integers(0,60)
        box(tx-34,tx+34,base-th,base+40)
        for yy_ in range(int(base-th-70),int(base-th)):
            hw=(yy_-(base-th-70))/70*40
            sil[yy_,int(tx-hw):int(tx+hw)]=True
    # chimneys across the skyline with smoke plumes
    chims=[]
    for k in range(9):
        x=rng.random()*W
        if abs(x-cx)<230: continue
        h=150+rng.random()*230; w=16+rng.random()*14
        box(x-w/2,x+w/2,base-h,base+40); box(x-w/2-5,x+w/2+5,base-h,base-h+10)
        chims.append((x,base-h,w))
    # low workshop blocks with saw-tooth roofs
    x=0
    while x<W:
        bw=60+rng.random()*140; bh=40+rng.random()*70
        box(x,x+bw,base-bh,base+40)
        for k in range(int(bw//22)):
            x0=x+k*22
            for yy_ in range(int(base-bh-14),int(base-bh)):
                f=(yy_-(base-bh-14))/14
                sil[yy_,int(x0):int(x0+22*f)]=True
        x+=bw+rng.random()*40
    # great gear half-sunk behind the keep
    gx,gy,gr=cx+(150 if rng.random()<0.5 else -150),base-300,120
    d=np.hypot(xx-gx,yy-gy); a=np.arctan2(yy-gy,xx-gx)
    teeth=gr+14*(np.mod(a*12/np.pi,2)<1)
    gear=(d<teeth)&(d>gr*0.35)&~((d>gr*0.55)&(d<gr*0.82)&(np.mod(a*3/np.pi,2)>0.35))
    sil|=gear&(yy<base)
    # gantry crane
    kx=W*0.12 if cx>W/2 else W*0.82
    box(kx-6,kx+6,base-300,base); box(kx-140,kx+90,base-300,base-288)
    for k in range(12):
        x0=kx-140+k*20
        for i in range(12): 
            yy_=int(base-300+i); xx_=int(x0+i*1.6)
            if 0<=xx_<W: sil[yy_,xx_]=True
    box(kx-112,kx-110,base-288,base-180); box(kx-124,kx-98,base-180,base-160)
    dgl=np.hypot((xx-cx)/(W*0.32),(yy-base)/(HY*0.42))
    img+=np.exp(-dgl**2*1.4)[...,None]*col(255,96,24)*0.55
    # apply silhouette with a hot rim from below
    sf=gaussian_filter(sil.astype(np.float32),0.6)
    up=np.roll(sil,-3,axis=0)&~sil
    rimg=gaussian_filter(up.astype(np.float32),1.2)
    img=img*(1-sf[...,None])+dk*sf[...,None]
    img+=rimg[...,None]*col(255,120,40)*0.6
    # furnace windows: arched slits, all warm (no office grids)
    glow=np.zeros((H,W),np.float32)
    for k in range(70):
        x=rng.integers(0,W); y=int(base-rng.random()*360)
        if not sil[min(y,H-1),x] or not sil[min(y+10,H-1),x]: continue
        w=rng.integers(3,7); h=rng.integers(8,18)
        glow[y:y+h,x-w//2:x+w//2+1]=0.7+0.3*rng.random()
    # big furnace mouth in the keep
    fm=((np.abs(xx-cx)<38)&(yy>base-95)&(yy<base-30))|((np.hypot(xx-cx,yy-(base-95))<38)&(yy<=base-95))
    glow[fm]=np.clip(1.6-(base-30-yy[fm])/120,0.6,1.6)
    img+=gaussian_filter(glow,10)[...,None]*col(255,110,30)*3.2+gaussian_filter(glow,1)[...,None]*col(255,200,110)*1.2
    # plumes rising from chimneys
    pl=np.zeros((H,W),np.float32)
    for (x,y,w) in chims:
        for k in range(40):
            py=y-k*14; px=x+k*k*0.35*(1 if rng.random()<0.7 else -0.5)+rng.normal(0,4)
            r=w*0.7+k*4.2
            m=((xx-px)**2+(yy-py)**2)<r*r
            pl[m]=np.maximum(pl[m],0.85-k*0.019)
    pl=gaussian_filter(pl,6)*np.clip(fbm(rng,W,H,60,4)*1.6-0.2,0,1)
    plc=lerp(col(26,22,24),col(110,56,32),np.clip(yy/HY,0,1)[...,None])
    plc=plc+np.exp(-((xx-cx)/(W*0.3))**2)[...,None]*col(90,40,14)
    img=img*(1-pl[...,None]*0.9)+plc*pl[...,None]*0.9
    # haze at the horizon
    hz=np.exp(-((yy-base)/(HY*0.06))**2)
    img+=hz[...,None]*col(150,60,20)*0.22
    # ground: scorched plain under the horizon
    g=np.clip((yy-HY)/(H-HY),0,1)
    gn=fbm(rng,W,H,90,4)
    gcol=lerp(col(46,32,28),col(26,22,24),(g**0.5)[...,None])*(0.8+0.4*gn[...,None])
    gm=np.clip((yy-HY)/2,0,1)[...,None]
    img=img*(1-gm)+gcol*gm
    vig=1-0.24*(((xx-W/2)/(W/2))**2+((yy-H*0.45)/(H*0.75))**2)
    img*=vig[...,None]
    img=1-np.exp(-img*1.3)
    img+=(rng.random((H,W,1)).astype(np.float32)-0.5)/255*1.5
    Image.fromarray((np.clip(img,0,1)*255+0.5).astype(np.uint8)).save(out,quality=92)
paint(int(sys.argv[1]),sys.argv[2])
