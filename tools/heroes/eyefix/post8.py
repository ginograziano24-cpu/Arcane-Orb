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
w = w*(1-np.clip(ndimage.gaussian_filter(hair.astype(np.float32),2)*1.5,0,1))
im = im*(1-w[...,None]) + mir*w[...,None]
mask = np.maximum(mask,(w>0.02).astype(np.float32))
# --- lip gloss
reg = np.zeros((H,W),bool); reg[980:1200,1560:1980] = True
r,g = im[...,0],im[...,1]
lips = ndimage.binary_opening(reg&(r-g>0.2)&(r>0.5)&(g<0.62),iterations=2)
lab,n = ndimage.label(lips); sz = ndimage.sum(lips,lab,range(1,n+1)); lips = lab==(np.argmax(sz)+1)
lips = ndimage.binary_fill_holes(ndimage.binary_closing(lips,iterations=4))
ly,lx = np.where(lips)
lw = np.clip(ndimage.gaussian_filter(lips.astype(np.float32),2.5)*1.2,0,1)
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
Image.open('eyes_fixed2.png').convert('RGB').crop((1330,480,2230,1220)).save('/tmp/post8.png')
