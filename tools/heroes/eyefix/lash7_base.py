# step 1: symmetric clean base, built only from her own pixels (no lashes drawn here)
import numpy as np, json
from PIL import Image
from scipy import ndimage
src = np.array(Image.open('/tmp/eyes_front_v1.png').convert('RGBA')).astype(np.float32)/255
al = src[...,3:4]; im = src[...,:3].copy()
H,W = im.shape[:2]; yy,xx = np.mgrid[0:H,0:W].astype(np.float32)
mask = np.zeros((H,W),np.float32)
sx,sy = 1530,698; dx,dy = 2004,690
ys,xs = np.mgrid[0:H,0:W]
mir = im[np.clip(ys-(dy-sy),0,H-1), np.clip(sx-(xs-dx),0,W-1)]
e = ((xx-(dx+6))/158)**2 + ((yy-(dy-20))/98)**2
w = np.clip((1-e)/0.22,0,1)*np.clip((yy-(dy-84))/10,0,1)
mmx = mir.max(2); msat = (mmx-mir.min(2))/np.maximum(mmx,1e-3)
hair = ndimage.binary_dilation(ndimage.binary_opening((msat<0.16)&(mmx>0.4)&(mmx<0.85)&(np.abs(xx-dx)>108),iterations=2),iterations=3)
w = w*(1-np.clip(ndimage.gaussian_filter(hair.astype(np.float32),2)*1.5,0,1))*np.clip((dx+122-xx)/16,0,1)
im = im*(1-w[...,None]) + mir*w[...,None]
mask = np.maximum(mask,(w>0.02).astype(np.float32))
def skinfill(im, zone, protect, sgs=(10,18,30,48), blur=5):
    mx_=im.max(2); st=(mx_-im.min(2))/np.maximum(mx_,1e-3)
    good=(mx_>0.6)&(st>0.1)&(im[...,0]>im[...,2])&~zone
    est=im.copy(); got=np.zeros((H,W),bool)
    for sg in sgs:
        num=ndimage.gaussian_filter(im*good[...,None],(sg,sg,0)); den=ndimage.gaussian_filter(good.astype(np.float32),sg)
        take=(~got)&(den>0.1); est[take]=(num/np.maximum(den,1e-4)[...,None])[take]; got|=take
    zw=np.clip(ndimage.gaussian_filter(zone.astype(np.float32),blur)*1.5-0.2,0,1)*(1-np.clip(ndimage.gaussian_filter(protect.astype(np.float32),2)*1.5,0,1))
    return im*(1-zw[...,None])+est*zw[...,None], zw
mx_=im.max(2); st=(mx_-im.min(2))/np.maximum(mx_,1e-3)
hairk=ndimage.binary_dilation(ndimage.binary_opening((st<0.16)&(mx_>0.4),iterations=2),iterations=2)
zone=((xx>dx+90)&(xx<dx+178)&(yy>dy-48)&(yy<dy+40))|((xx>dx+104)&(xx<dx+178)&(yy>dy-84)&(yy<dy+40))
im,zw=skinfill(im,zone,hairk); mask=np.maximum(mask,(zw>0.02).astype(np.float32))
np.save('base.npy',im); np.save('mask.npy',mask); np.save('alpha.npy',al)
Image.fromarray((im*255).astype(np.uint8)).crop((1380,560,2200,800)).resize((1640,480),Image.LANCZOS).save('base.png')
Image.fromarray((im*255).astype(np.uint8)).crop((1850,580,2180,780)).resize((1650,1000),Image.LANCZOS).save('base_l.png')
