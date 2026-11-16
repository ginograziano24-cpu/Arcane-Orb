import numpy as np, sys
from PIL import Image
from scipy.ndimage import gaussian_filter
from scipy.spatial import cKDTree
sys.path.insert(0,'/tmp/paint'); from common import fbm,col,lerp
W,H=2048,2633; U=W/28.0
rng=np.random.default_rng(7)
yy,xx=np.mgrid[0:H,0:W].astype(np.float32)
# irregular slate slabs: jittered points, euclidean voronoi, warped
n=int(W*H/(U*2.1)**2)
pts=np.stack([rng.random(n)*W,rng.random(n)*H],1)
warp=(fbm(rng,W,H,260,4)-0.5)*U*0.9
q=np.stack([(xx+warp).ravel(),(yy-warp).ravel()],1)
d,idx=cKDTree(pts).query(q,k=2)
d1=d[:,0].reshape(H,W); d2=d[:,1].reshape(H,W); cid=idx[:,0].reshape(H,W)
edge=d2-d1
tone=rng.random(n).astype(np.float32)[cid]
grain=fbm(rng,W,H,22,4); strata=fbm(rng,W,H,120,4,0.45)
blot=fbm(rng,W,H,200,4)
base=lerp(col(34,40,62),col(58,66,96),tone[...,None])
stone=base*(0.78+0.44*grain[...,None])*(0.85+0.3*blot[...,None])*(0.92+0.16*strata[...,None])
# bevel lighting on slab edges
gy,gx=np.gradient(gaussian_filter(np.minimum(edge,U*0.4),2))
bev=np.clip(1-edge/(U*0.25),0,1)
stone*=(1+np.clip(-gx-gy,-1,1)[...,None]*bev[...,None]*0.6)
joint=np.clip(1-edge/(U*0.06),0,1)**1.4
img=stone*(1-joint[...,None])+col(8,10,20)*joint[...,None]
# crystal veins: along some slab joints plus thin branching cracks
vsel=rng.random(n)<0.22
vein_j=joint*np.maximum(vsel[cid],vsel[idx[:,1].reshape(H,W)])
cr=fbm(rng,W,H,140,5)
crack=np.clip(1-np.abs(cr-0.5)/0.0035,0,1)*(fbm(rng,W,H,400,3)>0.6)
vein=np.clip(vein_j+crack,0,1)
hue=fbm(rng,W,H,500,3)
vcol=lerp(col(70,230,255),col(170,110,255),np.clip((hue-0.35)*2.5,0,1)[...,None])
core=gaussian_filter(vein,0.8); glow=gaussian_filter(vein,7)
img=img*(1-core[...,None]*0.9)+vcol*core[...,None]*1.1
img+=glow[...,None]*vcol*0.55
# soft glow pooled where veins meet
for k in range(26):
    cx,cy=rng.random()*W,rng.random()*H
    c=vcol[int(min(cy,H-1)),int(min(cx,W-1))]
    gl=np.exp(-(((xx-cx)**2+(yy-cy)**2)/(U*1.1)**2))
    img+=gl[...,None]*c*0.22
# wet sheen: puddle-ish areas darker with a cool specular streak
wet=np.clip((fbm(rng,W,H,350,4)-0.55)*4,0,1)
img=img*(1-wet[...,None]*0.25)+wet[...,None]*col(40,70,110)*0.25*(0.6+0.8*grain[...,None])
# darken toward edges
ex=np.minimum(xx,W-xx)/(U*2.2); ey=np.minimum(yy,H-yy)/(U*2.2)
img*=(0.7+0.3*np.clip(np.minimum(ex,ey),0,1))[...,None]
img=np.clip(img*1.25,0,1)
Image.fromarray((img*255+0.5).astype(np.uint8)).save(sys.argv[1],quality=90)
