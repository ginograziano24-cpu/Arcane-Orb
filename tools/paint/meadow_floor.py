import numpy as np, sys
from PIL import Image
from scipy.ndimage import gaussian_filter
sys.path.insert(0,'/tmp/paint'); from common import fbm,col,lerp
W,H=2048,2633; U=W/14.0   # flagstone course unit
rng=np.random.default_rng(11)
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
base=lerp(col(138,104,74),col(206,170,120),tone[...,None]**0.8)
base=lerp(base,col(170,150,128),(tone2[...,None]>0.75)*0.5)  # a few cooler grey stones
stone=base*(0.8+0.36*grain[...,None])*(0.86+0.26*blot[...,None])*(0.94+0.12*strata[...,None])
# bevel: light from upper-left
bev=np.clip(1-edge/(U*0.14),0,1)
lx=np.where(fx<0.5,1,-1)*(ex<ey); ly=np.where(fy<0.5,1,-1)*(ey<=ex)
stone*=(1+(lx*0.5+ly*0.7)[...,None]*bev[...,None]**1.3*0.45)
stone*=(1-0.18*np.clip(1-edge/(U*0.35),0,1)[...,None]*(tone2[...,None]))
joint=np.clip(1-edge/(U*0.065),0,1)**1.1
soil=col(70,52,34)*(0.8+0.4*grain[...,None])
img=stone*(1-joint[...,None])+soil*joint[...,None]
# moss creeping from joints, patchy
mz=np.clip((fbm(rng,W,H,220,4)-0.45)*3,0,1)
mn=fbm(rng,W,H,9,3)
moss=np.clip((np.clip(1-edge/(U*0.2),0,1)**1.2*mz*1.6-(1-mn)*0.9)*4,0,1)
mcol=lerp(col(60,80,32),col(112,132,54),mn[...,None])
img=lerp(img,mcol,np.clip(moss*1.3,0,0.95)[...,None])
# hairline cracks
cr=fbm(rng,W,H,120,5)
crack=np.clip(1-np.abs(cr-0.5)/0.003,0,1)*(fbm(rng,W,H,300,3)>0.7)*(1-joint)
img*=1-gaussian_filter(crack,0.7)[...,None]*0.45
# scattered petals and tiny grass sprigs
pet=np.zeros((H,W),np.float32); pc=np.zeros((H,W,3),np.float32)
pal=[col(250,236,210),col(246,180,90),col(232,120,120),col(214,170,236)]
for k in range(420):
    cx,cy=rng.random()*W,rng.random()*H; r=3+rng.random()*4
    x0_,y0_=int(max(cx-r*2,0)),int(max(cy-r*2,0)); x1_,y1_=int(min(cx+r*2,W)),int(min(cy+r*2,H))
    g=np.exp(-(((xx[y0_:y1_,x0_:x1_]-cx)**2+((yy[y0_:y1_,x0_:x1_]-cy)*1.6)**2)/r**2)*1.5)
    c=pal[rng.integers(4)]
    pet[y0_:y1_,x0_:x1_]=np.maximum(pet[y0_:y1_,x0_:x1_],g); pc[y0_:y1_,x0_:x1_]=np.where(g[...,None]>0.3,c,pc[y0_:y1_,x0_:x1_])
pet*=np.clip(mz*1.5+0.25,0,1)
img=lerp(img,pc,np.clip(pet*0.9,0,0.9)[...,None])
# warm sunset wash: brighter toward the far (top) end, gentle vignette at sides
img*=(0.92+0.14*(1-yy/H))[...,None]
img=img*lerp(col(255,255,255),col(255,226,190),0.35)
exx=np.minimum(xx,W-xx)/(U*1.6); img*=(0.74+0.26*np.clip(exx,0,1))[...,None]
img=np.clip(img*1.17,0,1)
Image.fromarray((img*255+0.5).astype(np.uint8)).save(sys.argv[1],quality=90)
