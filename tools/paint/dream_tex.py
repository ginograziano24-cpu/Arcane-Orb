import numpy as np
from PIL import Image
from scipy.ndimage import gaussian_filter
def C(*a): return np.array(a,np.float32)/255
def lerp(a,b,t): return a+(b-a)*t
def pnoise(N,M,cyc,rng,beta=2.0):
    fy=np.fft.fftfreq(N)[:,None]*N; fx=np.fft.fftfreq(M)[None,:]*M
    fr=np.sqrt(fx**2+fy**2); fr[0,0]=1
    amp=fr**(-beta/2)*(fr>=cyc*0.5)*(fr<=cyc*4)
    n=np.real(np.fft.ifft2(np.fft.fft2(rng.standard_normal((N,M)))*amp))
    return ((n-n.min())/(n.max()-n.min())).astype(np.float32)
def shade(h,albedo,amp=90,L=(-0.5,-0.6,0.65),spec=30,specA=0.5,wrap=True):
    gy,gx=np.gradient(np.pad(h,1,mode='wrap' if wrap else 'edge')*amp); gy=gy[1:-1,1:-1]; gx=gx[1:-1,1:-1]
    nx,ny,nz=-gx,-gy,np.ones_like(h); ln=np.sqrt(nx*nx+ny*ny+1); nx/=ln;ny/=ln;nz/=ln
    L=np.array(L);L=L/np.linalg.norm(L)
    d=np.clip(nx*L[0]+ny*L[1]+nz*L[2],0,1)
    hv=L+np.array([0,0,1.]);hv/=np.linalg.norm(hv)
    sp=np.clip(nx*hv[0]+ny*hv[1]+nz*hv[2],0,1)**spec
    return albedo*(0.3+0.9*d[...,None])+sp[...,None]*specA
def crunch(img,a=0.8,s=1.25):
    img=img+(img-gaussian_filter(img,(1.6,1.6,0),mode='wrap'))*a
    l=img.mean(2,keepdims=True); return np.clip(l+(img-l)*s,0,1)
def save(img,p,q=92):
    Image.fromarray((np.clip(img,0,1)*255+0.5).astype(np.uint8)).save(p,quality=q)
def wrapstamp(H,W,cx,cy,r,fn,acc):
    # call fn on every wrapped copy whose box overlaps
    for ox in (-W,0,W):
        for oy in (-H,0,H):
            x0=int(max(cx+ox-r,0)); x1=int(min(cx+ox+r+1,W)); y0=int(max(cy+oy-r,0)); y1=int(min(cy+oy+r+1,H))
            if x0<x1 and y0<y1:
                fn(acc,x0,x1,y0,y1,cx+ox,cy+oy)

# ---------- flower bush (tileable 512)
def flowers(p,seed=1,N=512):
    rng=np.random.default_rng(seed)
    yy,xx=np.mgrid[0:N,0:N].astype(np.float32)
    h=pnoise(N,N,10,rng)*0.6+pnoise(N,N,30,rng)*0.4
    alb=lerp(C(16,60,24),C(60,140,40),h[...,None])
    H=h.copy()*0.6
    pal=[C(255,90,190),C(255,130,210),C(240,60,160),C(255,190,230),C(250,250,255)]
    for i in range(1100):
        cx,cy=rng.random()*N,rng.random()*N; r=rng.uniform(9,16); cc=pal[rng.integers(5)] if rng.random()<0.92 else C(255,220,80)
        ph=rng.random()*6.28
        def fn(acc,x0,x1,y0,y1,X,Y):
            dx=xx[y0:y1,x0:x1]-X; dy=yy[y0:y1,x0:x1]-Y; R=np.hypot(dx,dy); T=np.arctan2(dy,dx)
            petal=r*(0.62+0.38*np.abs(np.cos(2.5*(T+ph))))
            m=R<petal
            hh=np.clip(1-R/petal,0,1)**0.6*0.9+0.4
            sub=H[y0:y1,x0:x1]; H[y0:y1,x0:x1]=np.where(m,np.maximum(sub,hh),sub)
            a=alb[y0:y1,x0:x1]; a[m]=cc*(0.75+0.25*np.clip(1-R/petal,0,1))[m][:,None]
            cm=R<r*0.22; a[cm]=C(255,220,60)
        wrapstamp(N,N,cx,cy,r+1,fn,None)
    img=shade(H,alb,amp=60,spec=25,specA=0.35)
    save(crunch(img),p)

# ---------- candy stripes (tileable 256): diagonal red/white bands
def candy(p,N=256):
    yy,xx=np.mgrid[0:N,0:N].astype(np.float32)
    u=((xx+yy)/N*4)%1
    band=(u<0.5).astype(np.float32)
    e=np.minimum(np.abs(u-0),np.minimum(np.abs(u-0.5),np.abs(u-1)))
    h=np.clip(e*8,0,1)**0.5
    alb=np.where(band[...,None]>0.5,C(235,30,40),C(250,245,240))
    img=shade(h,alb,amp=25,spec=60,specA=0.8)
    save(crunch(img,0.6,1.2),p)

# ---------- castle brick (tileable 512): royal blue stone blocks
def brick(p,seed=4,N=512):
    rng=np.random.default_rng(seed)
    yy,xx=np.mgrid[0:N,0:N].astype(np.float32)
    rows=8; bh=N/rows; bw=N/4
    r=(yy//bh).astype(int); off=(r%2)*bw/2
    fx=((xx+off)%bw)/bw; fy=(yy%bh)/bh
    bid=((xx+off)//bw).astype(int)%4+r*4
    ex=np.minimum(fx,1-fx)*bw; ey=np.minimum(fy,1-fy)*bh; e=np.minimum(ex,ey)
    h=np.clip(e/7,0,1)**0.7*(0.85+0.3*pnoise(N,N,24,rng))
    tone=np.random.default_rng(9).random(64)[bid%64]
    alb=lerp(C(40,70,200),C(80,120,240),tone[...,None])*(0.85+0.3*pnoise(N,N,40,rng)[...,None])
    alb=np.where((e<3)[...,None],C(20,25,60),alb)
    img=shade(h,alb,amp=70,spec=30,specA=0.45)
    save(crunch(img),p)

# ---------- roof shingles (tileable 256): scalloped red
def roof(p,seed=5,N=256):
    rng=np.random.default_rng(seed)
    yy,xx=np.mgrid[0:N,0:N].astype(np.float32)
    rows=8; rh=N/rows; sw=N/8
    r=(yy//rh).astype(int); off=(r%2)*sw/2
    fx=((xx+off)%sw)/sw-0.5; fy=(yy%rh)/rh
    d=np.sqrt(fx**2*1.0+(1-fy)**2*0.7)
    h=np.clip(1-d*1.4,0,1)**0.6
    alb=lerp(C(160,20,30),C(240,50,50),h[...,None])*(0.9+0.2*pnoise(N,N,16,rng)[...,None])
    img=shade(h,alb,amp=40,spec=30,specA=0.5)
    save(crunch(img),p)

# ---------- grass ground (tileable 1024)
def grass(p,seed=6,N=1024):
    rng=np.random.default_rng(seed)
    yy,xx=np.mgrid[0:N,0:N].astype(np.float32)
    base=pnoise(N,N,6,rng)*0.5+pnoise(N,N,20,rng)*0.5
    alb=lerp(C(20,90,30),C(70,170,50),base[...,None])
    H=base*0.3
    # blade strokes
    for i in range(16000):
        cx,cy=rng.random()*N,rng.random()*N; L=rng.uniform(8,20); a=rng.uniform(-2.2,-0.9)
        cc=lerp(C(40,140,40),C(140,230,70),rng.random())
        ts=np.linspace(0,1,int(L))
        px=((cx+np.cos(a)*L*ts)%N).astype(int); py=((cy+np.sin(a)*L*ts)%N).astype(int)
        H[py,px]=np.maximum(H[py,px],0.45+0.5*ts)
        alb[py,px]=cc*(0.7+0.4*ts)[:,None]
    for i in range(500):
        cx,cy=int(rng.random()*N),int(rng.random()*N)
        cc=[C(255,80,180),C(255,230,90),C(240,240,255),C(180,120,255)][rng.integers(4)]
        for dx,dy in ((0,0),(1,0),(-1,0),(0,1),(0,-1)):
            alb[(cy+dy)%N,(cx+dx)%N]=cc; H[(cy+dy)%N,(cx+dx)%N]=1
    H=gaussian_filter(H,0.7,mode='wrap')
    img=shade(H,alb,amp=50,spec=20,specA=0.25)
    save(crunch(img,0.7,1.3),p)

# ---------- violet glossy cobblestone floor (2048x2633, not tiled)
def cobble(p,seed=7,W=2048,H=2633):
    rng=np.random.default_rng(seed)
    from scipy.spatial import cKDTree
    cell=105
    pts=[]
    for y in np.arange(-cell,H+cell,cell*0.8):
        for x in np.arange(-cell,W+cell,cell):
            pts.append((x+rng.uniform(-0.32,0.32)*cell+(cell/2 if int(y/(cell*0.8))%2 else 0),y+rng.uniform(-0.3,0.3)*cell))
    pts=np.array(pts); tree=cKDTree(pts)
    yy,xx=np.mgrid[0:H,0:W].astype(np.float32)
    q=np.stack([xx.ravel(),yy.ravel()],1)
    d,ix=tree.query(q,k=2)
    d1=d[:,0].reshape(H,W); d2=d[:,1].reshape(H,W); idx=ix[:,0].reshape(H,W)
    edge=(d2-d1)/2
    hh=np.clip(edge/28,0,1)**0.55
    tone=np.random.default_rng(3).random(len(pts)).astype(np.float32)[idx]
    noise=gaussian_filter(rng.random((H,W)).astype(np.float32),1.2)
    big=gaussian_filter(rng.random((H//8,W//8)).astype(np.float32),3)
    big=np.array(Image.fromarray((big*255).astype(np.uint8)).resize((W,H),Image.BILINEAR),np.float32)/255
    h=hh*(0.85+0.25*noise)
    alb=lerp(C(110,60,200),C(190,140,255),tone[...,None])
    alb=lerp(alb,C(230,120,255),(tone[...,None]>0.85)*0.4)
    alb*= (0.85+0.3*big[...,None])
    alb=np.where((edge<4)[...,None],C(30,10,50),alb)
    img=shade(h,alb,amp=80,spec=22,specA=0.55,wrap=False)
    # wet sheen streaks
    img+=np.clip(gaussian_filter(rng.random((H,W)).astype(np.float32),18)-0.52,0,1)[...,None]*C(255,200,255)*1.5*hh[...,None]
    img=img+(img-gaussian_filter(img,(1.6,1.6,0)))*0.9
    l=img.mean(2,keepdims=True); img=np.clip(l+(img-l)*1.2,0,1)
    # gentle side vignette
    ex=np.minimum(xx,W-xx)/(W*0.12); img*=(0.7+0.3*np.clip(ex,0,1))[...,None]
    save(img*1.05,p,90)

import sys
o=sys.argv[1]
flowers(o+'/dream_flowers.jpg'); candy(o+'/dream_candy.jpg'); brick(o+'/dream_brick.jpg'); roof(o+'/dream_roof.jpg')
grass(o+'/ground_dream_meadow.jpg'); cobble(o+'/dream_meadow.jpg')
