import numpy as np
from PIL import Image
from scipy import ndimage
im=np.load('im2.npy'); mask=np.load('mask2.npy'); al=np.load('alpha.npy')
H,W=im.shape[:2]; yy,xx=np.mgrid[0:H,0:W].astype(np.float32)
# subtle lip gloss only (no reshaping)
reg=np.zeros((H,W),bool); reg[960:1215,1540:2000]=True
r,g=im[...,0],im[...,1]
lips=ndimage.binary_opening(reg&(r-g>0.2)&(r>0.5)&(g<0.62),iterations=2)
lab,n=ndimage.label(lips); sz=ndimage.sum(lips,lab,range(1,n+1)); lips=lab==(np.argmax(sz)+1)
lips=ndimage.binary_fill_holes(ndimage.binary_closing(lips,iterations=4))
ly,lx=np.where(lips); lw=np.clip(ndimage.gaussian_filter(lips.astype(np.float32),2.5)*1.2,0,1)
tint=np.clip(im*np.array([1.05,0.88,0.94])*1.03,0,1); im=im*(1-0.5*lw[...,None])+tint*0.5*lw[...,None]
cx=(lx.min()+lx.max())/2; top=ly.min(); hl=ly.max()-top; wl=lx.max()-lx.min()
sp=lambda a,b,rx,ry,k: np.exp(-(((xx-a)/rx)**2+((yy-b)/ry)**2))*k
gl=sp(cx-0.08*wl,top+0.66*hl,0.085*wl,0.07*hl,0.7)+sp(cx+0.1*wl,top+0.7*hl,0.035*wl,0.045*hl,0.45)+sp(cx-0.09*wl,top+0.2*hl,0.05*wl,0.035*hl,0.3)+sp(cx,top+0.62*hl,0.3*wl,0.2*hl,0.1)
gl=np.clip(gl,0,0.8)*lw; im=im*(1-gl[...,None])+np.array([1.0,0.97,0.98])*gl[...,None]
mask=np.maximum(mask,ndimage.gaussian_filter(ndimage.binary_dilation(lips,iterations=6).astype(np.float32),1.0))
Image.fromarray((np.concatenate([np.clip(im,0,1),al],-1)*255).astype(np.uint8)).save('eyes_fixed.png')
Image.fromarray((np.clip(mask,0,1)*255).astype(np.uint8)).save('eyes_mask.png')
f=Image.open('eyes_fixed.png').convert('RGB'); o=Image.open('/tmp/eyes_front_v1.png').convert('RGB')
f.crop((1330,480,2230,1220)).save('face.png')
A=o.crop((1380,540,2200,800)); Bm=f.crop((1380,540,2200,800))

