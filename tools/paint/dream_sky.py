import numpy as np, sys
from PIL import Image
from scipy.ndimage import gaussian_filter
sys.path.insert(0,'/tmp/paint'); from common import fbm,col,lerp
W,H=2048,1152; HY=int(H*0.72)
seed=int(sys.argv[2]) if len(sys.argv)>2 else 1
rng=np.random.default_rng(seed)
yy,xx=np.mgrid[0:H,0:W].astype(np.float32)
t=np.clip(yy/HY,0,1)[...,None]
img=lerp(col(8,4,34),col(46,16,104),t**1.3)
img=lerp(img,col(170,50,170),np.clip((t-0.72)/0.28,0,1)**2*0.85)
# nebula clouds
n1=fbm(rng,W,H,260,6); n2=fbm(rng,W,H,140,5); n3=fbm(rng,W,H,90,5)
neb=np.clip((n1-0.45)*2.2,0,1)
img+=neb[...,None]*lerp(col(40,30,160),col(170,40,180),n2[...,None])*0.55*(1-t*0.4)
img+=np.clip((n3-0.6)*3,0,1)[...,None]*col(60,120,255)*0.25
# galaxy: bold logarithmic-spiral arms (painted, not simulated)
gx,gy=(1180 if seed%2 else 1330),300
rot=0.3 if seed%2 else -0.2; tilt=0.52
X=xx-gx; Y=yy-gy
Xr=X*np.cos(-rot)-Y*np.sin(-rot); Yr=(X*np.sin(-rot)+Y*np.cos(-rot))/tilt
R=np.hypot(Xr,Yr)+1e-3; TH=np.arctan2(Yr,Xr)
b=0.21
ph=TH-np.log(R/20)/b
warp=(fbm(rng,W,H,60,4)-0.5)*1.4
armA=(0.5+0.5*np.cos(2*(ph+warp)))**3.0
armB=(0.5+0.5*np.cos(2*(ph+warp*1.3)+0.7))**8
fall=np.exp(-(R/560)**1.6)
core=np.exp(-(R/55)**2)*1.4+np.exp(-(R/150)**1.5)*0.7
cr=np.clip(R/560,0,1)[...,None]
cA=col(255,236,170); cB=col(70,170,255); cC=col(220,70,230); cD=col(120,60,255)
gc=np.where(cr<0.3,lerp(cA,cB,cr/0.3),np.where(cr<0.7,lerp(cB,cC,(cr-0.3)/0.4),lerp(cC,cD,(cr-0.7)/0.3)))
grain=fbm(rng,W,H,8,3)
G=(armA*0.9+armB*0.6)*fall*(0.55+0.7*grain)
dust=(0.5+0.5*np.cos(2*(ph+warp)+1.9))**6*fall
img=img*(1-np.clip(G,0,0.9)[...,None]*0.5)+gc*G[...,None]*1.25
img*= (1-dust*0.35)[...,None]
img+=core[...,None]*col(255,235,190)
img+=gaussian_filter(G,18)[...,None]*gc*0.6
# sparkly star clusters riding the arms
n=14000; px=rng.random(n)*W; py=rng.random(n)*H
keep=G[py.astype(int),px.astype(int)]>rng.random(n)*0.9
st=np.zeros((H,W),np.float32); np.add.at(st,(py[keep].astype(int),px[keep].astype(int)),1.5)
img+=gaussian_filter(st,0.8)[...,None]*col(255,240,255)*3
# stars
for k,(n,sz,b) in enumerate([(9000,0.6,0.7),(1600,1.0,1.0),(260,1.6,1.4)]):
    sx=rng.random(n)*W; sy=rng.random(n)**1.15*HY*0.98
    st=np.zeros((H,W),np.float32); np.add.at(st,(sy.astype(int),sx.astype(int)),rng.random(n)*b+0.3)
    st=gaussian_filter(st,sz)*(sz*sz*6)
    tint=lerp(col(200,220,255),col(255,220,250),rng.random()); img+=st[...,None]*tint
# flare stars (4-point spikes)
def flare(cx,cy,L,c,a):
    global img
    dx=np.abs(xx-cx); dy=np.abs(yy-cy)
    m=(dx<L)&(dy<L)
    sp=(np.exp(-dy/1.4)*np.clip(1-dx/L,0,1)**2+np.exp(-dx/1.4)*np.clip(1-dy/L,0,1)**2)
    sp+=np.exp(-(dx**2+dy**2)/(L*0.08)**2)*2
    sp+=0.4*(np.exp(-np.abs(dx-dy)/1.2)+np.exp(-np.abs(dx+dy-0)/1.2)*0)*np.clip(1-np.hypot(dx,dy)/(L*0.5),0,1)**2
    img+=np.where(m,sp,0)[...,None]*c*a
pal=[col(255,220,120),col(255,140,230),col(160,220,255),col(255,255,255),col(200,150,255)]
for i in range(46):
    cx,cy=rng.random()*W,rng.random()**1.3*HY*0.92
    L=rng.uniform(14,60)*(1.6 if i<8 else 1)
    flare(cx,cy,L,pal[i%5],rng.uniform(0.6,1.1))
# shooting stars
for i in range(3):
    x0,y0=rng.uniform(100,W-300),rng.uniform(30,HY*0.5); L=rng.uniform(160,320); a=rng.uniform(0.35,0.6)
    tt=np.linspace(0,1,600); px=x0+tt*L*np.cos(a); py=y0+tt*L*np.sin(a)
    sb=np.zeros((H,W),np.float32); m=(px<W)&(py<H)
    np.add.at(sb,(py[m].astype(int),px[m].astype(int)),tt[m]**2*2)
    img+=gaussian_filter(sb,1.3)[...,None]*col(255,240,255)*6
# far hills silhouette with rim light + glowing horizon
hill=HY-30-fbm(rng,W,1,180,4)[0]*90-40*np.sin(xx[0]/W*np.pi*2+seed)
hm=(yy>hill[None,:]).astype(np.float32)
hm=gaussian_filter(hm,1.0)
rim=np.clip(1-np.abs(yy-hill[None,:])/6,0,1)
hc=lerp(col(30,40,60),col(20,70,50),fbm(rng,W,H,60,4)[...,None])
img=img*(1-hm[...,None])+hc*hm[...,None]+rim[...,None]*col(255,120,230)*0.6
# pine silhouettes on the hills
for i in range(80):
    px=rng.random()*W; base=hill[int(px)]+rng.uniform(0,30); h=rng.uniform(30,85); w=h*0.33
    for tier in range(3):
        tb=base-h*tier*0.28; th_=h*0.55
        m=(yy<tb)&(yy>tb-th_)&(np.abs(xx-px)<(yy-(tb-th_))/th_*w*(1-tier*0.22))
        img=np.where(m[...,None],col(12,34,34)+(np.abs(xx-px)<2)[...,None]*0,img)
# HDR crunch: local contrast + unsharp + saturation
lum=img.mean(2,keepdims=True)
img=img+(img-gaussian_filter(img,(10,10,0)))*0.35+(img-gaussian_filter(img,(1.5,1.5,0)))*0.7
lum=img.mean(2,keepdims=True); img=lum+(img-lum)*1.35
img=img/(1+img*0.25)*1.1
Image.fromarray((np.clip(img,0,1)*255+0.5).astype(np.uint8)).save(sys.argv[1],quality=92)
