import numpy as np, json
from PIL import Image
from scipy import ndimage
src=np.array(Image.open('/tmp/eyes_front_v1.png').convert('RGBA')).astype(np.float32)/255
o=src[...,:3]; im=o.copy(); H,W=im.shape[:2]; yy,xx=np.mgrid[0:H,0:W].astype(np.float32)
x0,x1,c=json.load(open('/tmp/eyefix/lidcurves.json'))[1]
cols=np.arange(W); T=np.polyval(c,np.clip(cols,x0,x1))
LO=[(1887,714),(2000,725),(2120,680)]; B=np.polyval(np.polyfit([p[0] for p in LO],[p[1] for p in LO],2),cols)
inx=(cols>=x0+2)&(cols<=x1-2)
opening=(yy>T[None,:]+2)&(yy<B[None,:]-1)&inx[None,:]
mx=o.max(2); notblue=o[...,2]<o[...,0]*1.25
lashin=opening&(mx<0.3)&(o[...,2]<o[...,0]*1.12)&(yy<T[None,:]+34)
lashin=ndimage.binary_dilation(ndimage.binary_opening(lashin,iterations=1),iterations=2)&opening
# 1) reveal the eye under the lashes: mirrored copy of her right eye
sx,sy,dx,dy=1530,698,2004,690
ys,xs=np.mgrid[0:H,0:W]
mir=o[np.clip(ys-(dy-sy),0,H-1),np.clip(sx-(xs-dx),0,W-1)]
a=np.clip(ndimage.gaussian_filter(opening.astype(np.float32),1.6)*1.25-0.12,0,1)
im=im*(1-a[...,None])+mir*a[...,None]
# 2) slide the lash band up onto the fold above the lid
D=int(S) if (S:=26) else 26
up=np.roll(o,-D,axis=0); upm=np.roll(lashin,-D,axis=0)
dark=np.clip((0.5-up.max(2))/0.25,0,1)
b=np.clip(ndimage.gaussian_filter(upm.astype(np.float32),1.2)*1.5,0,1)*dark*(yy<T[None,:]+3)
im=im*(1-b[...,None])+up*b[...,None]
m=ndimage.gaussian_filter(np.maximum(a,b)>0.02,1.5)
Image.fromarray((np.concatenate([np.clip(im,0,1),src[...,3:]],-1)*255).astype(np.uint8)).save('eyes_fixed.png')
Image.fromarray((np.clip(m*1.5,0,1)*255).astype(np.uint8)).save('eyes_mask.png')
A=Image.open('/tmp/eyes_front_v1.png').convert('RGB').crop((1380,540,2200,800)); Bm=Image.open('eyes_fixed.png').convert('RGB').crop((1380,540,2200,800))
cc=Image.new('RGB',(820,530)); cc.paste(A,(0,0)); cc.paste(Bm,(0,270)); cc.resize((1230,795)).save('/tmp/lm.png')
