import numpy as np, sys
from PIL import Image
from scipy.ndimage import gaussian_filter
sys.path.insert(0,'/tmp/paint'); from common import fbm,col,lerp
W,H=2048,2633
rng=np.random.default_rng(31)
yy,xx=np.mgrid[0:H,0:W].astype(np.float32)
# large slabs in a staggered layout: 7 across, rows of 2:1 slabs
SW=W/7.0; SH=SW*0.62
row=np.floor(yy/SH).astype(int); off=(row%2)*SW*0.5
cx=np.floor((xx+off)/SW).astype(int)
fx=((xx+off)/SW)%1; fy=(yy/SH)%1
sid=(row*53+cx*17)%4096
sr=np.random.default_rng(4).random(4096).astype(np.float32)[sid]
sr2=np.random.default_rng(8).random(4096).astype(np.float32)[sid]
ex=np.minimum(fx,1-fx)*SW; ey=np.minimum(fy,1-fy)*SH; edge=np.minimum(ex,ey)
# marble: turbulent veins, per-slab orientation offset
t1=fbm(rng,W,H,180,6,0.55); t2=fbm(rng,W,H,60,5,0.5)
ang=sr*3.0
u=xx*np.cos(ang)+yy*np.sin(ang)
v=np.abs(np.sin(u/90+t1*9+sr2*20))
vein=np.clip(1-v/0.06,0,1)**1.5*0.8+np.clip(1-np.abs(np.sin(u/37+t2*14))/0.03,0,1)*0.35
cloud=fbm(rng,W,H,120,5)
base=lerp(col(236,236,242),col(250,248,244),cloud[...,None])
base=lerp(base,col(226,230,244),(sr[...,None]>0.7)*0.6)   # a few cooler slabs
veinc=lerp(col(140,140,160),col(184,160,120),sr2[...,None])
img=lerp(base,veinc,np.clip(vein*0.85,0,1)[...,None])
haze=gaussian_filter(vein,6); img=lerp(img,veinc,np.clip(haze*0.35,0,1)[...,None])
img*=(0.95+0.08*t2[...,None])
# soft polish highlight per slab (top-left lit)
img*=(1+0.05*(0.5-fy)[...,None]+0.03*(0.5-fx)[...,None])
# gold inlay seams with a bright core and darker channel
seam=np.clip(1-edge/7.0,0,1)
core=np.clip(1-edge/2.5,0,1)
gold=lerp(col(170,120,40),col(252,214,120),fbm(rng,W,H,30,3)[...,None])
img=lerp(img,col(120,100,70),seam[...,None]*0.5)
img=lerp(img,gold,core[...,None]*0.95)
# lapis diamonds where seams cross
diam=np.zeros((H,W),np.float32)
for r_ in range(int(H/SH)+2):
    o=(r_%2)*SW*0.5
    for c_ in range(9):
        px=c_*SW-o; py=r_*SH
        d=(np.abs(xx-px)+np.abs(yy-py)*1.1)
        m=d<26
        if not m.any(): continue
        diam=np.maximum(diam,np.clip((26-d)/3,0,1))
dring=np.clip(1-np.abs(gaussian_filter(diam,1)-0.5)/0.5,0,1)
img=lerp(img,col(110,150,214)*(0.85+0.3*cloud[...,None]),np.clip(diam,0,1)[...,None])
img=lerp(img,gold,(diam>0.05)*(diam<0.6)*0.9*np.ones_like(diam))[...] if False else img
gr=gaussian_filter(diam,2); ring=np.clip((gr-0.05)*6,0,1)*np.clip((0.95-gr)*6,0,1)
img=lerp(img,gold,ring[...,None]*0.8)
# faint sky reflection gradient
img*=(0.97+0.06*(1-yy/H))[...,None]
img=np.clip(img,0,1)
Image.fromarray((img*255+0.5).astype(np.uint8)).save(sys.argv[1],quality=90)
