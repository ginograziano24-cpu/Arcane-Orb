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
skin=lambda im:(im[...,0]>im[...,2]*1.12)&(im[...,2]>im[...,0]*0.58)&(im[...,1]>im[...,2]*0.98)&(im.max(2)>0.55)&((im.max(2)-im.min(2))/np.maximum(im.max(2),1e-3)>0.12)
R=poly|((xx>1680)&(xx<1880)&(yy>236)&(yy<530))
R|=ndimage.binary_dilation(poly,iterations=9)&(yy>236)
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
# her clean left forehead sets the tone, with its own top-to-bottom gradient
fb=(xx>1900)&(xx<2080)&(yy>290)&(yy<470)&skin(A)
Yf=yy[fb]; Cf=A[fb]
co=[np.polyfit(Yf,Cf[:,c],1) for c in range(3)]
est=np.stack([np.polyval(co[c],np.clip(yy,260,520)) for c in range(3)],-1)
pe=ndimage.binary_dilation(poly,iterations=14)&~ndimage.binary_erosion(poly,iterations=4)
pe_m=pe[np.clip(ys-8,0,H_-1),np.clip(2*1767-xs,0,W_-1)]          # the old fringe outline, seen in the mirror
use_m=(skin(mir)&(xx<1767)&~pe_m).astype(np.float32)
use_m=np.clip(ndimage.gaussian_filter(use_m,2)*1.3-0.15,0,1)*(skin(mir)&~pe_m)
fill=mir*use_m[...,None]+est*(1-use_m[...,None])
fill=np.where((R&~(use_m>0.98))[...,None],ndimage.gaussian_filter(fill,(3,3,0)),fill)
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
# clear the dark smudges under both eyes (keep the lower lid line and lower lashes)
qB=np.polyfit([93,-3,-107],[7,26,23],2)
for (ex,ey,f) in ((1530,698,-1),(2004,690,1)):
    u=f*(xx-ex); lid=ey+np.polyval(qB,np.clip(u,-112,124))
    inx=(u>-118)&(u<128)
    d=yy-lid
    band=inx&(d>2.0)&(d<36)
    refm=inx&(d>34)&(d<54)
    lum=B.mean(2)
    # skin colour per column, from just below the band, smoothed
    num=ndimage.gaussian_filter(B*refm[...,None],(0,10,0)); den=ndimage.gaussian_filter(refm.astype(np.float32),(0,10))
    colref=np.zeros_like(B)
    for c in range(3):
        sm=num[...,c].sum(0)/np.maximum(den.sum(0),1e-6)
        colref[...,c]=sm[None,:]
    rlum=colref.mean(2)
    dark=np.clip((rlum-lum)/0.08,0,1)*(lum>0.24)
    fade=np.clip((d-2.0)/4.0,0,1)*np.clip((36-d)/16.0,0,1)
    w=np.clip(ndimage.gaussian_filter(dark*band*fade,1.2)*1.4,0,1)*band
    B=B*(1-w[...,None])+colref*w[...,None]
    m2=np.maximum(m2,np.clip(ndimage.gaussian_filter(band.astype(np.float32),2)*1.5,0,1))
# beauty mark just under her right eye (where the stray speck was)
bmx,bmy=1446,792
rr=np.sqrt((xx-bmx)**2+((yy-bmy)*1.08)**2)
bm=np.clip((7.2-rr)/2.2,0,1)
core=np.clip((4.2-rr)/2.0,0,1)
mark=np.array([0.40,0.25,0.22])*(1-core[...,None])+np.array([0.30,0.17,0.15])*core[...,None]
B=B*(1-0.9*bm[...,None])+mark*0.9*bm[...,None]
m2=np.maximum(m2,np.clip((14-rr)/4,0,1))
Image.fromarray((np.concatenate([np.clip(B,0,1),al],-1)*255).astype(np.uint8)).save('eyes_fixed_fr.png')
# the whole rebuilt forehead takes its colour from this image, temple to temple
mxx=np.where(xx>1767,2*1767-xx,xx); myy=np.where(xx>1767,yy+8,yy)
fh=(mxx>1376)&(myy>232)&(myy<630)&~((mxx<1494)&(myy<318))
m2=np.maximum(m2,np.clip(ndimage.gaussian_filter(fh.astype(np.float32),2)*1.5-0.2,0,1))
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
if '--dbg' in sys.argv:
    l=B.mean(2); d=ndimage.gaussian_filter(l,5)-l
    yy2,xx2=np.where((d>0.05)[350:500,1660:1780]); print('line px',len(xx2))
    for k in range(0,len(xx2),max(1,len(xx2)//6)):
        y,x=yy2[k]+350,xx2[k]+1660; print((x,y),'fw',round(float(fw[y,x]),2),'um',round(float(use_m[y,x]),2),'B',B[y,x].round(2),'A',A[y,x].round(2),'mir',mir[y,x].round(2))
    for p in ((1705,400),(1690,360),(1740,470),(1660,280)):
        x,y=p; print(p,'R',bool(R[y,x]),'fw',round(float(fw[y,x]),2),'A',A[y,x].round(2),'B',B[y,x].round(2),'mid',bool(mid[y,x]),'zone',round(float(zone[y,x]),2))
