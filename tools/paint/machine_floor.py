import numpy as np, sys
from PIL import Image
from scipy.ndimage import gaussian_filter
sys.path.insert(0,'/tmp/paint'); from common import fbm,col,lerp
W,H=2048,2633; U=W/28.0
rng=np.random.default_rng(13)
yy,xx=np.mgrid[0:H,0:W].astype(np.float32)
wx=xx/U-14; wz=33-yy/U      # world coords: x -14..14, z -3..33
grain=fbm(rng,W,H,18,4); blot=fbm(rng,W,H,160,4); rust=fbm(rng,W,H,90,5); big=fbm(rng,W,H,600,3)
# plates 4 x 2 units in a running bond
pw,ph=4.0,2.0
row=np.floor(wz/ph); off=(row%2)*pw/2
px=np.mod(wx+off,pw); pz=np.mod(wz,ph)
ex=np.minimum(px,pw-px); ez=np.minimum(pz,ph-pz); e=np.minimum(ex,ez)
pid=(np.floor((wx+off)/pw)*31+row*17).astype(np.int64)
ptone=(np.sin(pid*12.9898)*43758.5453)%1
iron=lerp(col(62,58,56),col(92,84,76),ptone[...,None].astype(np.float32))
# tread pattern on every third plate
tread=(np.mod(pid,3)==0)
tu=np.mod(wx*3+wz*3,1.0); tv=np.mod(wx*3-wz*3,1.0)
td=np.minimum(np.abs(tu-0.5),np.abs(tv-0.5))
treadm=(td<0.07)&tread
metal=iron*(0.8+0.4*grain[...,None])*(0.86+0.28*blot[...,None])
metal=np.where(treadm[...,None],metal*1.25+col(10,8,4),metal)
# rust and oil
r=np.clip((rust-0.58)*4,0,1)*(0.4+0.6*np.clip(1-e/0.5,0,1))
metal=metal*(1-r[...,None]*0.6)+col(120,58,30)*(0.7+0.5*grain[...,None])*r[...,None]*0.6
oil=np.clip((fbm(rng,W,H,260,4)-0.62)*5,0,1)
metal*=(1-oil[...,None]*0.35)
# bevel + seam
bev=np.clip(1-e/0.12,0,1)
lit=np.where(((px<pw/2)&(ex<ez))|((pz>ph/2)&(ez<=ex)),1.0,-0.8)
metal*=(1+lit[...,None]*bev[...,None]*0.18)
seam=np.clip(1-e/0.035,0,1)
img=metal*(1-seam[...,None])+col(14,11,10)*seam[...,None]
# brass strips along every plate's long edges with rivets
strip=(ez<0.11)&(ez>0.035)
brass=col(176,128,56)*(0.75+0.5*grain[...,None])*(1-0.35*np.clip((rust-0.5)*3,0,1)[...,None])
img=np.where(strip[...,None],brass,img)
rx=np.mod(wx+off,0.5)-0.25; rz=ez-0.07
rd=np.sqrt(rx*rx+rz*rz)
rv=(rd<0.045)
shade=np.clip(0.6+(-rx-rz)*8,0.3,1.4)
img=np.where(rv[...,None],col(214,170,96)*shade[...,None],img)
img*= (1-np.clip(1-np.abs(rd-0.05)/0.012,0,1)*0.35*(ez<0.13))[...,None]
# corner bolts on iron
cxd=np.minimum(px,pw-px)-0.22; czd=np.minimum(pz,ph-pz)-0.32
bd=np.sqrt(cxd**2+czd**2)
bolt=bd<0.07
img=np.where(bolt[...,None],col(120,112,104)*np.clip(0.7+(-cxd-czd)*5,0.4,1.4)[...,None],img)
# drain grates on a few plates
gsel=(np.mod(pid*7+3,11)==0)&(ex>0.8)&(ez>0.45)
gl=np.mod(wx*5,1.0)<0.45
img=np.where((gsel&gl)[...,None],col(10,8,8),img)
img+=(gsel&gl)[...,None]*col(120,40,10)*0.15
# warm furnace light from the far end, cooler near the paddle
warm=np.clip((wz-4)/30,0,1)**1.2
img*= (0.8+0.35*warm)[...,None]
img+= warm[...,None]*col(60,24,6)*0.25*(0.6+0.8*big[...,None])
# hazard strip by the paddle (z < 1): worn yellow and black
hz=wz<1.0
stripe=np.mod((wx+wz)*0.8,1.0)<0.5
wear=np.clip((fbm(rng,W,H,30,4)-0.22)*5,0,1)
hcol=np.where(stripe[...,None],col(214,160,40),col(24,20,18))
hcol=lerp(metal,hcol,wear[...,None]*0.9)
img=np.where(hz[...,None]&(np.abs(wz-1.0)>0.05)[...,None],hcol,img)
img=np.where((np.abs(wz-1.0)<0.05)[...,None],brass,img)
# edges darker
exx=np.minimum(xx,W-xx)/(U*2); eyy=np.minimum(yy,H-yy)/(U*2)
img*=(0.72+0.28*np.clip(np.minimum(exx,eyy),0,1))[...,None]
img=np.clip(img*1.18,0,1)
Image.fromarray((img*255+0.5).astype(np.uint8)).save(sys.argv[1],quality=90)
