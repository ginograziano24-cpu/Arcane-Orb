import numpy as np, sys
from PIL import Image, ImageDraw
from scipy import ndimage
sys.path.insert(0,'/tmp/lash3')
src=np.array(Image.open('/tmp/eyes_front_v1.png').convert('RGB')).astype(np.float32)/255
A=np.array(Image.open('eyes_fixed.png').convert('RGBA')).astype(np.float32)/255; al=A[...,3:]; A=A[...,:3]
mask=np.array(Image.open('eyes_mask.png').convert('L')).astype(np.float32)/255
zone=np.load('mask2.npy')
H_,W_=src.shape[:2]; yy,xx=np.mgrid[0:H_,0:W_].astype(np.float32)
FX=[1180,1380,1400,1420,1445,1470,1492,1530,1580,1630,1680,1720,1730,1770,1810,1835,1850,1872,1890,1900,1906]
FY=[840,745,722,700,678,658,640,628,614,600,585,574,565,525,480,430,380,310,250,180,100]
poly=(yy<np.interp(xx,FX,FY))&(xx>1300)&(xx<1906)&(yy>236)
# forehead under the old fringe: her own left forehead, mirrored (8 px lower on her right)
ys,xs=np.mgrid[0:H_,0:W_]
mir=A[np.clip(ys-8,0,H_-1),np.clip(2*1767-xs,0,W_-1)]
mx=src.max(2); st=(mx-src.min(2))/np.maximum(mx,1e-3)
skin=lambda im:(im[...,0]>im[...,2]*1.12)&(im.max(2)>0.55)&((im.max(2)-im.min(2))/np.maximum(im.max(2),1e-3)>0.12)
R=poly|((xx>1680)&(xx<1880)&(yy>236)&(yy<530))
R&=~(zone>0.15)
mid=(xx>1690)&(xx<1840)&(yy>318)&(yy<530)
R&=(ndimage.binary_dilation(skin(mir)|skin(A),iterations=6)|mid)&(xx>1360)&~ndimage.binary_dilation((A.max(2)<0.6)&(xx>1800)&(yy>480),iterations=2)
good=skin(A)&~R
est=None
for sg in (60,34,18,10):
    num=ndimage.gaussian_filter(A*good[...,None],(sg,sg,0)); den=ndimage.gaussian_filter(good.astype(np.float32),sg)
    e=num/np.maximum(den,1e-4)[...,None]
    if est is None: est=e
    else:
        c=np.clip((den-0.15)/0.35,0,1)[...,None]; est=est*(1-c)+e*c
est=ndimage.gaussian_filter(est,(2,2,0))
use_m=(skin(mir)&(xx<1767)).astype(np.float32)
use_m=np.clip(ndimage.gaussian_filter(use_m,2)*1.3-0.15,0,1)
fill=mir*use_m[...,None]+est*(1-use_m[...,None])
fw=np.clip(ndimage.gaussian_filter(R.astype(np.float32),3)*1.4-0.15,0,1)*(1-np.clip(zone*1.6,0,1))*ndimage.binary_dilation(R,iterations=1)
B=A*(1-fw[...,None])+fill*fw[...,None]
m2=np.maximum(mask,np.clip(fw*2,0,1))
# matte brows, both sides (as in the hair pass)
browbox=((xx>1840)&(xx<2110)&(yy>492)&(yy<580))
brow=browbox&(st<0.3)&(mx<0.62)&(mx>0.2)
brow=ndimage.binary_dilation(ndimage.binary_opening(brow,iterations=2),iterations=2)
lab,n=ndimage.label(brow); sz=ndimage.sum(brow,lab,range(1,n+1)); brow=lab==(int(np.argmax(sz))+1)
bw=np.clip(ndimage.gaussian_filter(brow.astype(np.float32),1.6)*1.3-0.1,0,1)
bc=src*0.88
B=B*(1-bw[...,None])+bc*bw[...,None]; m2=np.maximum(m2,bw)
bwm=bw[np.clip(ys-8,0,H_-1),np.clip(2*1767-xs,0,W_-1)]; bcm=bc[np.clip(ys-8,0,H_-1),np.clip(2*1767-xs,0,W_-1)]
B=B*(1-bwm[...,None])+bcm*bwm[...,None]; m2=np.maximum(m2,bwm)
Image.fromarray((np.concatenate([np.clip(B,0,1),al],-1)*255).astype(np.uint8)).save('eyes_fixed_fr.png')
Image.fromarray((np.clip(m2,0,1)*255).astype(np.uint8)).save('eyes_mask_fr.png')
# hair texture for the strands: root (v=0) dark to silver tips, fine streaks along the strand
ref=(st<0.18)&(mx>0.12)&(mx<0.8)&(xx>2150)&(yy>200)&(yy<700)
lum=src.mean(2); rl=lum[ref]; rp=src[ref]
dark=rp[rl<np.percentile(rl,10)].mean(0); light=rp[rl>np.percentile(rl,95)].mean(0)
S=512; vv,uu=np.mgrid[0:S,0:S].astype(np.float32)/S
rng=np.random.RandomState(2)
streak=ndimage.gaussian_filter(rng.randn(S,S).astype(np.float32),(40,1.2)); streak/=streak.std()
streak2=ndimage.gaussian_filter(rng.randn(S,S).astype(np.float32),(60,4)); streak2/=streak2.std()
tt=np.clip(0.18+0.62*vv**0.8+0.10*streak+0.09*streak2,0,1)
tex=dark*(1-tt[...,None])+light*tt[...,None]
Image.fromarray((np.clip(tex,0,1)*255).astype(np.uint8)).save('fringe_tex.png')
print('dark',dark.round(2),'light',light.round(2))
