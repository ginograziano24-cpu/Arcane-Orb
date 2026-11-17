# 2D: replace the old painted lashes above her right eye (viewer left) with plain skin.
import numpy as np, json
from PIL import Image
from scipy import ndimage
src=np.array(Image.open('/tmp/eyes_front_v1.png').convert('RGBA')).astype(np.float32)/255
im=src[...,:3].copy(); H,W=im.shape[:2]; yy,xx=np.mgrid[0:H,0:W].astype(np.float32)
x0,x1,c=json.load(open('/tmp/eyefix/lidcurves.json'))[0]
cols=np.arange(W); lid=np.polyval(c,np.clip(cols,x0,x1))
inx=(cols>=x0-22)&(cols<=x1+16)
zone=(yy<lid[None,:]+1)&(yy>lid[None,:]-52)&inx[None,:]
mx=im.max(2); st=(mx-im.min(2))/np.maximum(mx,1e-3)
hair=ndimage.binary_opening((st<0.16)&(mx>0.4),iterations=2)&(yy<lid[None,:]-10)
zs=np.clip(ndimage.gaussian_filter(zone.astype(np.float32),3)*1.5,0,1)*(yy<lid[None,:]+1)
w=zs*(1-np.clip(ndimage.gaussian_filter(hair.astype(np.float32),1.5)*1.5,0,1))
good=(mx>0.55)&(im[...,0]>im[...,2]*1.05)&(st>0.12)&~zone
est=im.copy(); got=np.zeros((H,W),bool)
for sg in (8,16,30,50):
    num=ndimage.gaussian_filter(im*good[...,None],(sg,sg,0)); den=ndimage.gaussian_filter(good.astype(np.float32),sg)
    t=(~got)&(den>0.06); est[t]=(num/np.maximum(den,1e-4)[...,None])[t]; got|=t
im=im*(1-w[...,None])+est*w[...,None]
m=ndimage.gaussian_filter((w>0.03).astype(np.float32),1.5)
Image.fromarray((np.concatenate([im,src[...,3:]],-1)*255).astype(np.uint8)).save('skinfill_fixed.png')
Image.fromarray((np.clip(m,0,1)*255).astype(np.uint8)).save('skinfill_mask.png')
Image.open('skinfill_fixed.png').convert('RGB').crop((1380,560,1780,800)).resize((1000,600)).save('/tmp/sf.png')
