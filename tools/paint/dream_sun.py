import numpy as np, sys
from PIL import Image, ImageDraw
from scipy.ndimage import gaussian_filter
S=768; c=S/2
yy,xx=np.mgrid[0:S,0:S].astype(np.float32)
X=xx-c; Y=yy-c; R=np.hypot(X,Y); TH=np.arctan2(Y,X)
R0=205
# flame rays: wavy, curling spikes
n=16
wig=np.sin(TH*n+np.sin(TH*3)*0.8)
spike=np.clip(0.5+0.5*np.cos(TH*n+0.6*np.sin(R/18)),0,1)**2.2
rayR=R0+40+spike*150
rayMask=np.clip((rayR-R)/6,0,1)*(R>R0-10)
diskMask=np.clip((R0-R)/2.5,0,1)
# height map
h=np.zeros((S,S),np.float32)
h+=np.sqrt(np.clip(1-(R/R0)**2,0,1))*1.0*diskMask
rayH=np.clip(spike,0,1)*np.clip(1-(R-R0)/170,0,1)*0.45
h+=rayH*rayMask*(1-diskMask)
# face features (raised)
def blob(cx,cy,rx,ry,amp,sh=2):
    d=((xx-cx)/rx)**2+((yy-cy)/ry)**2
    return np.clip(1-d,0,1)**(1/sh)*amp
eyeL=(c-72,c-48); eyeR=(c+72,c-48)
for ex,ey in (eyeL,eyeR):
    h+=blob(ex,ey,58,66,0.35)
# brows
for s in (-1,1):
    bx=c+s*78
    d=np.abs((yy-(c-122- (xx-bx)*s*0.32)))
    m=(np.abs(xx-bx)<55)
    h+=np.where(m,np.clip(1-d/14,0,1)*0.25,0)
# cheeks
for s in (-1,1): h+=blob(c+s*128,c+40,52,40,0.22)
# nose
h+=blob(c,c+12,26,30,0.3)
# mouth: wide grin crescent
mouth=((X/150)**2+((Y-58)/70)**2<1)&(Y>40+ (X/150)**2*-10)
mu=((X/150)**2+((Y-30)/50)**2<1)
mouth=mouth&~mu
lip=gaussian_filter(mouth.astype(np.float32),5)
h+=lip*0.25-mouth*0.35
h=gaussian_filter(h,1.2)
# normals + lighting
gyh,gxh=np.gradient(h*120)
nx,ny,nz=-gxh,-gyh,np.ones_like(h)
ln=np.sqrt(nx*nx+ny*ny+nz*nz); nx/=ln;ny/=ln;nz/=ln
L=np.array([-0.5,-0.6,0.62]);L/=np.linalg.norm(L)
dif=np.clip(nx*L[0]+ny*L[1]+nz*L[2],0,1)
hv=L+np.array([0,0,1]);hv/=np.linalg.norm(hv)
spec=np.clip(nx*hv[0]+ny*hv[1]+nz*hv[2],0,1)**40
# albedo
def C(*a): return np.array(a,np.float32)/255
alb=np.zeros((S,S,3),np.float32)
t=np.clip(R/R0,0,1)[...,None]
alb[:]=C(255,120,40)*(1-t)+C(235,40,30)*t
rt=np.clip((R-R0)/170,0,1)[...,None]
rayc=C(255,150,30)*(1-rt)+C(255,70,20)*rt
alb=np.where(diskMask[...,None]>0.5,alb,rayc)
# cheeks rosy
for s in (-1,1):
    alb=alb*(1-blob(c+s*128,c+40,60,46,0.5)[...,None])+C(255,80,110)*blob(c+s*128,c+40,60,46,0.5)[...,None]
# eyes: white, blue iris, pupil, glint
for ex,ey in (eyeL,eyeR):
    ew=((xx-ex)/50)**2+((yy-ey)/58)**2<1
    alb=np.where(ew[...,None],C(250,250,245),alb)
    ix,iy=ex+8,ey+8
    ir=np.hypot(xx-ix,yy-iy)
    alb=np.where((ir<28)[...,None]&ew[...,None],C(40,110,240)*(0.6+0.4*np.clip(1-ir/28,0,1))[...,None],alb)
    alb=np.where((ir<13)[...,None]&ew[...,None],C(10,10,20),alb)
    gl=np.hypot(xx-(ix-9),yy-(iy-9))<6
    alb=np.where(gl[...,None],C(255,255,255),alb)
    rim=np.abs(np.sqrt(((xx-ex)/50)**2+((yy-ey)/58)**2)-1)<0.06
    alb=np.where(rim[...,None],C(90,20,10),alb)
# brows dark
for s in (-1,1):
    bx=c+s*78; d=np.abs((yy-(c-122-(xx-bx)*s*0.32)))
    m=(np.abs(xx-bx)<55)&(d<11)
    alb=np.where(m[...,None],C(150,30,10),alb)
# mouth interior + teeth
alb=np.where(mouth[...,None],C(90,10,20),alb)
ub=30+50*np.sqrt(np.clip(1-(X/150)**2,0,1))
lb=58+70*np.sqrt(np.clip(1-(X/150)**2,0,1))
teeth=mouth&((Y<ub+22)|(Y>lb-16))
tooth_gap=(np.abs(((X+300)%24)-12)<1.6)|(np.abs(Y-(ub+22))<1.5)|(np.abs(Y-(lb-16))<1.5)
alb=np.where((teeth&~tooth_gap)[...,None],C(255,250,235),alb)
alb=np.where((teeth&tooth_gap)[...,None],C(160,140,120),alb)
lipline=gaussian_filter(mouth.astype(np.float32),2)
edge=(lipline>0.15)&(lipline<0.6)
alb=np.where(edge[...,None],C(140,20,20),alb)
# compose shading
col=alb*(0.35+0.85*dif[...,None])+spec[...,None]*0.75
# emboss micro texture
rng=np.random.default_rng(3)
nz_=gaussian_filter(rng.random((S,S)).astype(np.float32),1.0)
col*= (0.92+0.16*nz_)[...,None]
# HDR crunch
col=col+(col-gaussian_filter(col,(2,2,0)))*0.9
alpha=np.clip(np.maximum(diskMask,rayMask),0,1)
alpha=gaussian_filter(alpha,0.7)
# outer glow in alpha (soft halo)
halo=np.exp(-np.clip(R-R0-100,0,None)/60)*0.35*(1-alpha)
out=np.zeros((S,S,4),np.float32)
out[...,:3]=np.where(alpha[...,None]>0.01,col,C(255,120,40))
out[...,3]=np.clip(alpha+halo,0,1)
Image.fromarray((np.clip(out,0,1)*255+0.5).astype(np.uint8),'RGBA').save(sys.argv[1])
