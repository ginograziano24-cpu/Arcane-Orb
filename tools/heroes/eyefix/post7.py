import numpy as np, json
from PIL import Image
from scipy import ndimage
orig = np.array(Image.open('/tmp/eyes_front_v1.png').convert('RGB')).astype(np.float32)/255
im = np.array(Image.open('eyes_fixed.png').convert('RGBA')).astype(np.float32)/255
al = im[...,3:4]; im = im[...,:3]
mask = np.array(Image.open('eyes_mask.png').convert('L')).astype(np.float32)/255
H,W = im.shape[:2]; yy,xx = np.mgrid[0:H,0:W].astype(np.float32)
curves = json.load(open('lidcurves.json'))
# --- open her left eye (viewer right): give it the other eye's clean iris, clear the dark blotch
x0,x1,c = curves[1]
cols = np.arange(W); T = np.polyval(c, np.clip(cols,x0,x1))
LO = [(1887,714),(2000,725),(2120,680)]
B = np.polyval(np.polyfit([p[0] for p in LO],[p[1] for p in LO],2), cols)
inx = (cols>=x0+4)&(cols<=x1-4)
opening = (yy > T[None,:]+3) & (yy < B[None,:]-4) & inx[None,:]
osoft = np.clip(ndimage.gaussian_filter(opening.astype(np.float32),2.2)*1.15-0.1,0,1)
sx,sy,R = 1530,698,56; dx_,dy_ = 2000,690
d = np.hypot(xx-dx_, yy-dy_)
ys,xs = np.mgrid[0:H,0:W]
src_y = np.clip(ys-(dy_-sy),0,H-1); src_x = np.clip(xs-(dx_-sx),0,W-1)
moved = orig[src_y, src_x]
mo = moved.max(2); isblue = (moved[...,2] > moved[...,0]*1.2)
iris = (d < R) & opening & isblue
iw = np.clip(ndimage.gaussian_filter(iris.astype(np.float32),1.2)*1.3,0,1)*osoft
# sclera for the rest of the opening where it is dark
cur = im.copy(); mx = cur.max(2)
dark = opening & (d >= R-2) & ((mx < 0.62) | (cur[...,2] > cur[...,0]*1.12))
corner = np.clip((np.abs(xx-(x0+x1)/2)/((x1-x0)/2)-0.45)/0.55,0,1)
scl = np.array([0.9,0.88,0.9]) * (1 - 0.4*np.clip(1-(yy-T[None,:])/24,0,1))[...,None] * (1-0.22*corner)[...,None]
dw = np.clip(ndimage.gaussian_filter(dark.astype(np.float32),2)*1.6,0,1)*osoft*(1-iw)
im = im*(1-dw[...,None]) + scl*dw[...,None]
lidsh = (1 - 0.35*np.clip(1-(yy-T[None,:])/16,0,1))[...,None]
im = im*(1-iw[...,None]) + moved*lidsh*iw[...,None]
mask = np.maximum(mask, ndimage.binary_dilation(opening,iterations=3).astype(np.float32))
# --- lip gloss
reg = np.zeros((H,W),bool); reg[980:1200,1560:1980] = True
r,g,b = im[...,0],im[...,1],im[...,2]
lips = reg & (r-g > 0.2) & (r > 0.5) & (g < 0.62)
lips = ndimage.binary_opening(lips,iterations=2)
lab,n = ndimage.label(lips)
if n:
    sz = ndimage.sum(lips,lab,range(1,n+1)); lips = lab == (np.argmax(sz)+1)
lips = ndimage.binary_fill_holes(ndimage.binary_closing(lips,iterations=4))
ly,lx = np.where(lips); print('lips',lips.sum(), lx.min(),lx.max(),ly.min(),ly.max())
lw = np.clip(ndimage.gaussian_filter(lips.astype(np.float32),2.5)*1.2,0,1)
tint = im*np.array([1.06,0.86,0.93]); tint = np.clip(tint*1.04,0,1)
im = im*(1-0.6*lw[...,None]) + tint*0.6*lw[...,None]
cxl = (lx.min()+lx.max())/2; top,bot = ly.min(),ly.max(); hl = bot-top; wl = lx.max()-lx.min()
def spot(cx,cy,rx,ry,a):
    g_ = np.exp(-(((xx-cx)/rx)**2 + ((yy-cy)/ry)**2))
    return g_*a
gl = spot(cxl-0.08*wl, top+0.66*hl, 0.085*wl, 0.075*hl, 0.8) + spot(cxl+0.1*wl, top+0.7*hl, 0.035*wl, 0.045*hl, 0.55) + spot(cxl-0.09*wl, top+0.2*hl, 0.05*wl, 0.035*hl, 0.4) + spot(cxl+0.1*wl, top+0.2*hl, 0.035*wl, 0.03*hl, 0.3) + spot(cxl, top+0.62*hl, 0.3*wl, 0.2*hl, 0.14)
gl = np.clip(gl,0,0.85)*lw
im = im*(1-gl[...,None]) + np.array([1.0,0.97,0.98])*gl[...,None]
mask = np.maximum(mask, ndimage.binary_dilation(lips,iterations=6).astype(np.float32))
mask = ndimage.gaussian_filter(mask,1.0)
Image.fromarray((np.concatenate([np.clip(im,0,1),al],-1)*255).astype(np.uint8)).save('eyes_fixed2.png')
Image.fromarray((np.clip(mask,0,1)*255).astype(np.uint8)).save('eyes_mask2.png')
a = Image.open('eyes_fixed.png').convert('RGB'); b_ = Image.open('eyes_fixed2.png').convert('RGB')
cc = Image.new('RGB',(1300,1040))
cc.paste(a.crop((1380,560,2180,800)).resize((1300,390)),(0,0)); cc.paste(b_.crop((1380,560,2180,800)).resize((1300,390)),(0,390))
cc.paste(a.crop((1560,990,1980,1160)).resize((640,259)),(0,781)); cc.paste(b_.crop((1560,990,1980,1160)).resize((640,259)),(660,781))
cc.save('/tmp/post7.png')
