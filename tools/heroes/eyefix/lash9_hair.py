# Hair pass on the face image: her fringe is textured see-through (skin tint, a brow drawn on the hair).
# Recolour it as real hair - charcoal to silver strands - keeping the gold ornament.
import numpy as np
from PIL import Image
from scipy import ndimage
src=np.array(Image.open('/tmp/eyes_front_v1.png').convert('RGB')).astype(np.float32)/255
A=np.array(Image.open('eyes_fixed.png').convert('RGBA')).astype(np.float32)/255; al=A[...,3:]; A=A[...,:3]
mask=np.array(Image.open('eyes_mask.png').convert('L')).astype(np.float32)/255
zone=np.load('mask2.npy')                                  # eye areas only (before lips/blush)
H_,W_=src.shape[:2]; yy,xx=np.mgrid[0:H_,0:W_].astype(np.float32)
FX=[1180,1380,1400,1420,1445,1470,1492,1530,1580,1630,1680,1720,1730,1770,1810,1835,1850,1872,1890,1900,1906]
FY=[840,745,722,700,678,658,640,628,614,600,585,574,565,525,480,430,380,310,250,180,100]
edge=np.interp(xx,FX,FY)
poly=(yy<edge)&(xx>1180)&(xx<1906)
r,g,b=src[...,0],src[...,1],src[...,2]; mx=src.max(2); st=(mx-src.min(2))/np.maximum(mx,1e-3)
mn=src.min(2); hue=np.degrees(np.arctan2(np.sqrt(3)*(g-b),2*r-g-b))
gold=(hue>30)&(hue<72)&(st>0.2)&(r>0.5)
purple=(b>r*1.02)&(b>g*1.12)&(st>0.28)
other=ndimage.binary_dilation(ndimage.binary_opening(gold|purple,iterations=1),iterations=3)
hair=poly&~other
lum=0.3*r+0.59*g+0.11*b
# blur along the strands (they run from upper right to lower left)
L=95; k=np.zeros((L,L),np.float32); c=L//2
for t in np.linspace(-c,c,4*L):
    x=int(round(c+t*-0.575)); y=int(round(c+t*0.818))
    if 0<=x<L and 0<=y<L: k[y,x]=1
k/=k.sum()
hm=hair.astype(np.float32)
num=ndimage.convolve(lum*hm,k,mode='nearest'); den=ndimage.convolve(hm,k,mode='nearest')
dl=num/np.maximum(den,1e-3)
lum2=0.30*lum+0.70*dl
rng=np.random.RandomState(3)
nz=ndimage.convolve(rng.randn(H_,W_).astype(np.float32),k,mode='nearest'); nz=ndimage.gaussian_filter(nz,0.7); nz/=nz.std()
nz2=ndimage.convolve(ndimage.gaussian_filter(rng.randn(H_,W_).astype(np.float32),3),k,mode='nearest'); nz2/=nz2.std()
t=np.clip((lum2-0.24)/0.46,0,1)**1.05+0.03*nz+0.045*nz2
t=np.clip(t,0,1)[...,None]
dark=np.array([0.15,0.15,0.20]); light=np.array([0.83,0.85,0.93])
rec=dark*(1-t)+light*t
hw=np.clip(ndimage.gaussian_filter(hm,1.2)*1.4-0.2,0,1)
Hh=src*(1-hw[...,None])+rec*hw[...,None]
# image A: recoloured hair outside the eye areas
wa=hw*(1-np.clip(zone*1.5,0,1))
A2=A*(1-wa[...,None])+rec*wa[...,None]
mask2=np.maximum(mask,np.clip(wa*2,0,1))
Image.fromarray((np.concatenate([np.clip(A2,0,1),al],-1)*255).astype(np.uint8)).save('eyes_fixed_h.png')
Image.fromarray((np.clip(mask2,0,1)*255).astype(np.uint8)).save('eyes_mask_h.png')
Image.fromarray((np.clip(Hh,0,1)*255).astype(np.uint8)).save('eyes_hair.png')
Image.fromarray((np.clip(hw*1.5,0,1)*255).astype(np.uint8)).save('eyes_hairmask.png')
Image.fromarray((np.clip(A2,0,1)*255).astype(np.uint8)).crop((1250,150,2300,1220)).save('face_h.png')
Image.fromarray((np.clip(Hh,0,1)*255).astype(np.uint8)).crop((1250,150,2300,800)).save('hair_only.png')
