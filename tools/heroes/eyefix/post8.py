import numpy as np
from PIL import Image
from scipy import ndimage
im = np.array(Image.open('eyes_fixed.png').convert('RGBA')).astype(np.float32)/255
al = im[...,3:4]; im = im[...,:3]
mask = np.array(Image.open('eyes_mask.png').convert('L')).astype(np.float32)/255
H,W = im.shape[:2]; yy,xx = np.mgrid[0:H,0:W].astype(np.float32)
# --- mirror her right eye (viewer left) onto her left eye
sx,sy = 1530,698; dx,dy = 2004,690
ys,xs = np.mgrid[0:H,0:W]
srcx = np.clip(sx-(xs-dx),0,W-1); srcy = np.clip(ys-(dy-sy),0,H-1)
mir = im[srcy,srcx]
e = ((xx-(dx+6))/158)**2 + ((yy-(dy-20))/98)**2
w = np.clip((1-e)/0.22,0,1)*np.clip((yy-(dy-84))/10,0,1)
mmx = mir.max(2); msat = (mmx-mir.min(2))/np.maximum(mmx,1e-3)
hair = ndimage.binary_dilation(ndimage.binary_opening((msat<0.16)&(mmx>0.4)&(mmx<0.85)&(np.abs(xx-dx)>108),iterations=2),iterations=3)
w = w*(1-np.clip(ndimage.gaussian_filter(hair.astype(np.float32),2)*1.5,0,1))*np.clip((dx+122-xx)/16,0,1)
im = im*(1-w[...,None]) + mir*w[...,None]
mask = np.maximum(mask,(w>0.02).astype(np.float32))
# --- clean outer corner of her left eye: skin, then a neat liner wing
import json
from PIL import ImageDraw
c0=json.load(open('lidcurves.json'))[0][2]
lidm=lambda x: np.polyval(c0, sx-(x-dx))+(dy-sy)
zone=((xx>dx+90)&(xx<dx+178)&(yy>dy-48)&(yy<dy+40))|((xx>dx+104)&(xx<dx+178)&(yy>dy-84)&(yy<dy+40))
mx_=im.max(2); st=(mx_-im.min(2))/np.maximum(mx_,1e-3)
hairk=ndimage.binary_dilation(ndimage.binary_opening((st<0.16)&(mx_>0.4),iterations=2),iterations=2)
good=(mx_>0.6)&(st>0.1)&(im[...,0]>im[...,2])&~zone
est=im.copy(); got=np.zeros((H,W),bool)
for sg in (10,18,30,48):
    num=ndimage.gaussian_filter(im*good[...,None],(sg,sg,0)); den=ndimage.gaussian_filter(good.astype(np.float32),sg)
    take=(~got)&(den>0.1); est[take]=(num/np.maximum(den,1e-4)[...,None])[take]; got|=take
zw=np.clip(ndimage.gaussian_filter(zone.astype(np.float32),5)*1.5-0.2,0,1)*(1-np.clip(ndimage.gaussian_filter(hairk.astype(np.float32),2)*1.5,0,1))
im=im*(1-zw[...,None])+est*zw[...,None]
SS=4; cv=Image.new("L",(W,H),0)
big=Image.new("L",(260*SS,160*SS),0); dr=ImageDraw.Draw(big); ox,oy=dx+10,dy-90
P=lambda x,y:((x-ox)*SS,(y-oy)*SS)
up=[];dn=[]
for x in np.linspace(dx+40,dx+95,24):
    up.append(P(x,lidm(x)-8.5)); dn.append(P(x,lidm(x)+2))
tipx=dx+124; tipy=lidm(dx+95)-13
dr.polygon(up+[P(tipx,tipy)]+dn[::-1],fill=255)
A=ndimage.gaussian_filter(np.array(big.resize((260,160),Image.LANCZOS)).astype(np.float32)/255,0.8)*0.95
im[oy:oy+160,ox:ox+260]=im[oy:oy+160,ox:ox+260]*(1-A[...,None])+np.array([0.08,0.05,0.06])*A[...,None]
mask=np.maximum(mask,(zw>0.02).astype(np.float32)); mask[oy:oy+160,ox:ox+260]=np.maximum(mask[oy:oy+160,ox:ox+260],(A>0.02).astype(np.float32))
# --- clear the tan blotches between the lashes and the brows
for (ex,ey) in ((dx,dy),):
    box = (np.abs(xx-(ex+8))<165)&(yy>ey-125)&(yy<ey-52)
    mx_=im.max(2); st=(mx_-im.min(2))/np.maximum(mx_,1e-3)
    bad=box&(mx_<0.84)&(mx_>0.3)&(st>0.13)&(yy>ey-112)
    lash=ndimage.binary_dilation(mx_<0.3,iterations=3)
    bad=ndimage.binary_dilation(ndimage.binary_opening(bad,iterations=1),iterations=5)&box&~lash
    good=(mx_>0.6)&(st>0.1)&(im[...,0]>im[...,2])&~bad
    est=im.copy(); got=~bad
    for sg in (5,9,16,28):
        num=ndimage.gaussian_filter(im*good[...,None],(sg,sg,0)); den=ndimage.gaussian_filter(good.astype(np.float32),sg)
        take=(~got)&(den>0.2); est[take]=(num/np.maximum(den,1e-4)[...,None])[take]; got|=take
    est=np.where(bad[...,None],ndimage.gaussian_filter(est,(4,4,0)),est)
    bw=np.clip(ndimage.gaussian_filter(bad.astype(np.float32),3)*1.3,0,1)*(1-np.clip(ndimage.gaussian_filter(lash.astype(np.float32),1.5)*1.6,0,1))
    im=im*(1-bw[...,None])+est*bw[...,None]
    mask=np.maximum(mask,(bw>0.02).astype(np.float32))
# --- fuller lips: gentle enlarge around the mouth, then sharpen
from scipy.ndimage import map_coordinates
lcx,lcy=1765.0,1108.0
e2=((xx-lcx)/270)**2+((yy-lcy)/135)**2
k=np.clip(1-e2,0,1)**1.5
sxs=1+0.015*k; sys_=1+0.045*k
srcx_=lcx+(xx-lcx)/sxs; srcy_=lcy+(yy-lcy)/sys_
warp=np.stack([map_coordinates(im[...,c],[srcy_,srcx_],order=3,mode='nearest') for c in range(3)],-1)
ww=(e2<1)[...,None]
im=np.where(ww,warp,im)
mask=np.maximum(mask,(e2<1.05).astype(np.float32))
# --- lip gloss
reg = np.zeros((H,W),bool); reg[960:1215,1540:2000] = True
r,g = im[...,0],im[...,1]
lips = ndimage.binary_opening(reg&(r-g>0.2)&(r>0.5)&(g<0.62),iterations=2)
lab,n = ndimage.label(lips); sz = ndimage.sum(lips,lab,range(1,n+1)); lips = lab==(np.argmax(sz)+1)
lips = ndimage.binary_fill_holes(ndimage.binary_closing(lips,iterations=4))
ly,lx = np.where(lips)
lw = np.clip(ndimage.gaussian_filter(lips.astype(np.float32),2.5)*1.2,0,1)
bl=ndimage.gaussian_filter(im,(2.5,2.5,0)); sharp=np.clip(im+0.9*(im-bl),0,1)
im=im*(1-lw[...,None])+sharp*lw[...,None]
# soft lip line and a little depth at the edges
edge=np.clip(ndimage.gaussian_filter(lips.astype(np.float32),5)*(1-ndimage.gaussian_filter(lips.astype(np.float32),1.5)),0,1)
im=im*(1-0.25*lw[...,None]*(1-np.clip(ndimage.gaussian_filter(lips.astype(np.float32),9)*1.6-0.5,0,1))[...,None])+im*np.array([0.82,0.55,0.62])*0.25*lw[...,None]*(1-np.clip(ndimage.gaussian_filter(lips.astype(np.float32),9)*1.6-0.5,0,1))[...,None]
tint = np.clip(im*np.array([1.06,0.86,0.93])*1.04,0,1)
im = im*(1-0.6*lw[...,None]) + tint*0.6*lw[...,None]
cx = (lx.min()+lx.max())/2; top = ly.min(); hl = ly.max()-top; wl = lx.max()-lx.min()
sp = lambda a,b,rx,ry,k: np.exp(-(((xx-a)/rx)**2+((yy-b)/ry)**2))*k
gl = sp(cx-0.08*wl,top+0.66*hl,0.085*wl,0.075*hl,0.8)+sp(cx+0.1*wl,top+0.7*hl,0.035*wl,0.045*hl,0.55)+sp(cx-0.09*wl,top+0.2*hl,0.05*wl,0.035*hl,0.4)+sp(cx+0.1*wl,top+0.2*hl,0.035*wl,0.03*hl,0.3)+sp(cx,top+0.62*hl,0.3*wl,0.2*hl,0.14)
gl = np.clip(gl,0,0.85)*lw
im = im*(1-gl[...,None]) + np.array([1.0,0.97,0.98])*gl[...,None]
mask = ndimage.gaussian_filter(np.maximum(mask,ndimage.binary_dilation(lips,iterations=6).astype(np.float32)),1.0)
Image.fromarray((np.concatenate([np.clip(im,0,1),al],-1)*255).astype(np.uint8)).save('eyes_fixed2.png')
Image.fromarray((np.clip(mask,0,1)*255).astype(np.uint8)).save('eyes_mask2.png')
f=Image.open('eyes_fixed2.png').convert('RGB'); f.crop((1330,480,2230,1220)).save('/tmp/post8.png'); f.crop((1840,540,2200,760)).resize((1080,660)).save('/tmp/post8_eye.png')
