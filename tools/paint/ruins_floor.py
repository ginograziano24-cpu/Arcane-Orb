import numpy as np, sys
from PIL import Image
from scipy.ndimage import gaussian_filter
sys.path.insert(0,'/tmp/paint'); from common import fbm,col,lerp
W,H=2048,2633; U=W/11.0   # flagstone course unit
rng=np.random.default_rng(23)
yy,xx=np.mgrid[0:H,0:W].astype(np.float32)
warp=(fbm(rng,W,H,300,4)-0.5)*U*0.12
X=xx+warp; Y=yy-warp*0.7
# running-bond courses of varying height, stones of varying length
rows=[];y=0
while y<H+U: h=U*(0.75+rng.random()*0.5); rows.append((y,h)); y+=h
ri=np.zeros((H,W),np.int32); fy=np.zeros((H,W),np.float32); rh=np.zeros((H,W),np.float32)
starts=np.array([r[0] for r in rows]); ri=np.clip(np.searchsorted(starts,Y,side='right')-1,0,len(rows)-1)
hs=np.array([r[1] for r in rows],np.float32)
fy=(Y-starts[ri])/hs[ri]
# per-row stone breaks
maxs=60; brk=np.zeros((len(rows),maxs),np.float32)
for i in range(len(rows)):
    x=-rng.random()*U*1.5; k=0
    while k<maxs: brk[i,k]=x; x+=U*(0.9+rng.random()*1.3); k+=1
bi=np.zeros((H,W),np.int32)
for i in range(len(rows)):
    m=ri==i
    bi[m]=np.searchsorted(brk[i],X[m],side='right')-1
x0=brk[ri,bi]; x1=brk[ri,np.minimum(bi+1,maxs-1)]
fx=(X-x0)/np.maximum(x1-x0,1)
sid=(ri*97+bi*31)%4096
srand=np.random.default_rng(3).random(4096).astype(np.float32)
tone=srand[sid]; tone2=np.random.default_rng(5).random(4096).astype(np.float32)[sid]
# distance to stone edge in px (rounded corners)
ex=np.minimum(fx,1-fx)*(x1-x0); ey=np.minimum(fy,1-fy)*hs[ri]
edge=np.minimum(ex,ey)+ (fbm(rng,W,H,40,3)-0.5)*U*0.06
grain=fbm(rng,W,H,18,4); blot=fbm(rng,W,H,160,4); strata=fbm(rng,W,H,70,4,0.45)
base=lerp(col(178,138,92),col(226,192,138),tone[...,None]**0.8)
base=lerp(base,col(204,170,128),(tone2[...,None]>0.8)*0.6)  # a few cooler grey stones
stone=base*(0.8+0.36*grain[...,None])*(0.86+0.26*blot[...,None])*(0.94+0.12*strata[...,None])
# bevel: light from upper-left
bev=np.clip(1-edge/(U*0.14),0,1)
lx=np.where(fx<0.5,1,-1)*(ex<ey); ly=np.where(fy<0.5,1,-1)*(ey<=ex)
stone*=(1+(lx*0.5+ly*0.7)[...,None]*bev[...,None]**1.3*0.45)
stone*=(1-0.18*np.clip(1-edge/(U*0.35),0,1)[...,None]*(tone2[...,None]))
joint=np.clip(1-edge/(U*0.065),0,1)**1.1
soil=col(120,88,56)*(0.8+0.4*grain[...,None])
img=stone*(1-joint[...,None])+soil*joint[...,None]
# sand drifts banked against joints, wind-streaked
mz=np.clip((fbm(rng,W,H,260,4)-0.42)*3,0,1)
streak=fbm(rng,W*1,H,60,3)
sand=np.clip((np.clip(1-edge/(U*0.16),0,1)**1.4*mz*1.5-(1-streak)*0.7)*5,0,1)*(mz>0.35)
scol=lerp(col(214,178,122),col(238,210,156),fbm(rng,W,H,8,3)[...,None])
sand=gaussian_filter(sand,2.5)
img=lerp(img,scol,np.clip(sand*0.8,0,0.8)[...,None])
# hairline cracks
cr=fbm(rng,W,H,120,5)
crack=np.clip(1-np.abs(cr-0.5)/0.003,0,1)*(fbm(rng,W,H,300,3)>0.7)*(1-joint)
img*=1-gaussian_filter(crack,0.7)[...,None]*0.45
# carved glyph roundels on a few stones, with turquoise inlay
from PIL import Image as _I, ImageDraw as _D
gl=_I.new('L',(W,H),0); dr=_D.Draw(gl)
inl=_I.new('L',(W,H),0); di=_D.Draw(inl)
cy_=0
for (y0r,hr) in rows:
    for kk in range(0,maxs-1):
        xa,xb=brk[rows.index((y0r,hr)),kk],brk[rows.index((y0r,hr)),kk+1]
        if xb<0 or xa>W or rng.random()>0.06: continue
        cx,cy=(xa+xb)/2,y0r+hr/2; rr=min(xb-xa,hr)*0.38
        kind=rng.integers(3)
        if kind==0:
            for q in range(3): dr.ellipse([cx-rr*(1-q*0.3),cy-rr*(1-q*0.3),cx+rr*(1-q*0.3),cy+rr*(1-q*0.3)],outline=255,width=7)
            di.ellipse([cx-rr*0.2,cy-rr*0.2,cx+rr*0.2,cy+rr*0.2],fill=255)
        elif kind==1:
            dr.ellipse([cx-rr*0.45,cy-rr*0.45,cx+rr*0.45,cy+rr*0.45],outline=255,width=7)
            for a in np.linspace(0,2*np.pi,9)[:-1]:
                dr.line([cx+np.cos(a)*rr*0.6,cy+np.sin(a)*rr*0.6,cx+np.cos(a)*rr,cy+np.sin(a)*rr],fill=255,width=7)
            di.ellipse([cx-rr*0.3,cy-rr*0.3,cx+rr*0.3,cy+rr*0.3],fill=255)
        else:
            dr.polygon([cx-rr,cy,cx,cy-rr*0.55,cx+rr,cy,cx,cy+rr*0.55],outline=255,width=7)
            dr.ellipse([cx-rr*0.3,cy-rr*0.3,cx+rr*0.3,cy+rr*0.3],outline=255,width=7)
            di.ellipse([cx-rr*0.14,cy-rr*0.14,cx+rr*0.14,cy+rr*0.14],fill=255)
g=gaussian_filter(np.asarray(gl,np.float32)/255,1.2); ii=gaussian_filter(np.asarray(inl,np.float32)/255,1.0)
img*=(1-g[...,None]*0.45)
img+=(np.roll(g,2,0)-g)[...,None]*0.08
img=lerp(img,col(60,200,180)*(0.8+0.3*grain[...,None]),np.clip(ii*0.85,0,1)[...,None])
img+=gaussian_filter(ii,6)[...,None]*col(60,200,180)*0.25
# warm sunset wash: brighter toward the far (top) end, gentle vignette at sides
img*=(0.92+0.14*(1-yy/H))[...,None]
img=img*lerp(col(255,255,255),col(255,232,200),0.25)
exx=np.minimum(xx,W-xx)/(U*1.6); img*=(0.74+0.26*np.clip(exx,0,1))[...,None]
speck=rng.random((H,W)).astype(np.float32)
img*=(0.93+0.1*speck)[...,None]*(0.9+0.2*fbm(rng,W,H,6,2))[...,None]
img=np.clip(img*1.16,0,1)
Image.fromarray((img*255+0.5).astype(np.uint8)).save(sys.argv[1],quality=90)
