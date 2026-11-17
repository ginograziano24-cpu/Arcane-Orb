# step 2: clean lids + drawn lash set, identical (mirrored) on both eyes
import numpy as np, json, sys
from PIL import Image, ImageDraw
from scipy import ndimage
from scipy.interpolate import CubicSpline
src = np.array(Image.open('/tmp/eyes_front_v1.png').convert('RGBA')).astype(np.float32)/255
src = src[...,:3]
im = np.load('base.npy'); mask = np.load('mask.npy'); al = np.load('alpha.npy')
H,W = im.shape[:2]; yy,xx = np.mgrid[0:H,0:W].astype(np.float32)
SX,SY = 1530,698
INK = np.array([0.060,0.036,0.052])
# ---- lid geometry in eye-local coords (u: + toward outer corner, v: + down)
UU = CubicSpline([-112,-110,-95,-80,-65,-50,-35,-20,-5,10,25,40,55,70,85,100,112],
                 [18,14.6,-3.4,-18.8,-31.7,-42.1,-50,-55.6,-58.8,-59.8,-58.5,-55,-49.3,-41.6,-31.8,-19,-6])
qB = np.polyfit([93,-3,-107],[7,26,23],2)
U = lambda u: UU(np.clip(u,-112,112))
B = lambda u: np.polyval(qB,np.clip(u,-112,112))
TH = lambda u: np.interp(u,[-108,-80,-40,0,40,70],[1.5,3.2,5.0,6.5,8.0,9.5])
TOP = CubicSpline([-108,-80,-40,0,40,70,90,104,116],
                  [U(-108)-1.5,U(-80)-3.2,U(-40)-5.0,U(0)-6.5,U(40)-8.0,U(70)-9.5,U(90)-10.5,U(104)-9.0,-7.0])
# hair lock over her right eye (source frame): lower edge
EX=[1380,1400,1420,1445,1470,1492,1530,1580,1630,1680,1720]; EY=[745,722,700,678,658,640,628,614,600,585,574]
E = lambda x: np.interp(x,EX,EY)
# ---- ink canvas (local frame), supersampled
SS=4; U0,V0,LW,LH = -180,-150,360,240
cv = Image.new('L',(LW*SS,LH*SS),0); dr = ImageDraw.Draw(cv)
P = lambda u,v: ((u-U0)*SS,(v-V0)*SS)
us = np.linspace(-108,116,240)
top=[P(u,float(TOP(u))) for u in us]
lo_u = np.linspace(-108,112,200); low=[P(u,float(U(u))+2.5) for u in lo_u]+[P(116,-5.0)]
dr.polygon(top+low[::-1],fill=255)
def lash(u0,ang,L,w0,curl,canvas=dr,base=None,fill=255,pw=0.8):
    v0 = float(TOP(u0))+2.0 if base is None else base
    a=np.radians(ang); c=np.radians(ang+curl)
    p0=np.array([u0,v0]); p2=p0+L*np.array([np.sin(a),-np.cos(a)]); p1=p0+0.5*L*np.array([np.sin(c),-np.cos(c)])
    ts=np.linspace(0,1,36); pts=[(1-t)**2*p0+2*t*(1-t)*p1+t*t*p2 for t in ts]
    L_=[];R_=[]
    for i,t in enumerate(ts):
        d=pts[min(i+1,35)]-pts[max(i-1,0)]; d/=np.linalg.norm(d)+1e-6; n=np.array([-d[1],d[0]])
        w=w0*(1-t)**pw*0.5+0.10
        L_.append(P(*(pts[i]+n*w))); R_.append(P(*(pts[i]-n*w)))
    canvas.polygon(L_+R_[::-1],fill=fill)
    canvas.ellipse((P(u0-w0*0.5,v0-w0*0.5)+P(u0+w0*0.5,v0+w0*0.5)),fill=fill)
ANG=lambda u: np.interp(u,[-90,-50,0,40,75,100,114],[-24,-12,6,26,46,62,74])
LEN=lambda u: np.interp(u,[-90,-60,-20,20,55,85,104,114],[7,10,15,22,31,39,37,31])
WID=lambda u: np.interp(u,[-90,0,60,114],[4.0,6.2,8.4,8.8])
MAIN=[-87,-72,-56,-40,-24,-8,8,24,39,53,66,78,89,99,108,114]
for i,u0 in enumerate(MAIN):
    k=(1.0,0.84,0.94)[i%3]
    a_=float(ANG(u0)); L=float(LEN(u0))*k; w=float(WID(u0))
    cu=26*np.tanh(a_/22)+6
    lash(u0,a_,L,w,cu)
    if i<len(MAIN)-1:                                   # companion: separate root, tip leaning in to meet the main lash
        lash(u0+w*1.05,a_-9,L*0.78,w*0.85,cu)
ink = np.array(cv.resize((LW,LH),Image.LANCZOS)).astype(np.float32)/255
# lower lid: soft line on the outer half and a few fine lashes
cv2_ = Image.new('L',(LW*SS,LH*SS),0); d2 = ImageDraw.Draw(cv2_)
lu=np.linspace(36,116,60)
LB=lambda u: float(B(u))+np.interp(u,[36,100,116],[0,0,-6.0])
d2.polygon([P(u,LB(u)-np.interp(u,[36,116],[0.5,2.0])) for u in lu]+[P(u,LB(u)+np.interp(u,[36,116],[0.5,2.0])) for u in lu[::-1]],fill=255)
ink2 = ndimage.gaussian_filter(np.array(cv2_.resize((LW,LH),Image.LANCZOS)).astype(np.float32)/255,1.6)*0.85
cv3_ = Image.new('L',(LW*SS,LH*SS),0); d3 = ImageDraw.Draw(cv3_)
for u0,ang,L in ((66,146,7),(79,138,9),(91,130,10),(102,122,9)):
    lash(u0,ang,L,3.0,-18,canvas=d3,base=LB(u0)+0.5)
ink3 = ndimage.gaussian_filter(np.array(cv3_.resize((LW,LH),Image.LANCZOS)).astype(np.float32)/255,0.9)*0.7
ink = ndimage.gaussian_filter(ink,0.8)
INKA = np.clip(np.maximum.reduce([ink*0.97,ink2,ink3]),0,1)

def smooth_skin(im, zone, bad):
    mx_=im.max(2); st=(mx_-im.min(2))/np.maximum(mx_,1e-3)
    good=(mx_>0.62)&(st>0.12)&(im[...,0]>im[...,2])&~zone&~bad
    good=ndimage.binary_erosion(good,iterations=2)
    est=None
    for sg in (60,34,20,12):
        num=ndimage.gaussian_filter(im*good[...,None],(sg,sg,0)); den=ndimage.gaussian_filter(good.astype(np.float32),sg)
        e=num/np.maximum(den,1e-4)[...,None]
        if est is None: est=e
        else:
            c=np.clip((den-0.15)/0.35,0,1)[...,None]; est=est*(1-c)+e*c
    return ndimage.gaussian_filter(est,(3,3,0))

def build(im, mask, cx, cy, f):
    x0,x1,y0,y1 = cx-200,cx+200,cy-150,cy+90
    sl=(slice(y0,y1),slice(x0,x1))
    X=xx[sl]; Y=yy[sl]; u=f*(X-cx); v=Y-cy
    loc=im[sl].copy()
    Uv=U(u); Bv=B(u); Tv=TOP(np.clip(u,-108,116))
    # hair in this window
    mx_=loc.max(2); st=(mx_-loc.min(2))/np.maximum(mx_,1e-3)
    if f<0: hair = Y < E(X)+2
    else:   hair = ndimage.binary_dilation(ndimage.binary_opening((st<0.16)&(mx_>0.4)&(u>120),iterations=2),iterations=3)
    keep = 1-np.clip(ndimage.gaussian_filter(hair.astype(np.float32),1.0)*2,0,1) if f>0 else np.clip((Y-E(X)-1)/2.0,0,1)
    # 1) clean skin above the lid and around the outer corner
    zt=Uv-np.interp(u,[-128,-60,20,150],[58,66,84,84])
    zone=((v<Uv+3)&(v>zt)&(u>-128)&(u<150))|((u>96)&(u<150)&(v<Bv+26)&(v>zt))
    zone&=~((v>Bv-2)&(u<112))
    zone|=(u>74)&(u<150)&(v>Bv+1.5)&(v<Bv+30)
    full=np.zeros((H,W),bool); full[sl]=zone; badf=np.zeros((H,W),bool); badf[sl]=hair
    est=smooth_skin(im,full,badf)[sl]
    zw=np.clip(ndimage.gaussian_filter(zone.astype(np.float32),5)*1.5-0.25,0,1)
    zw=np.maximum(zw,np.clip(ndimage.gaussian_filter((zone&(v>Uv-20)&(v<Uv+3)&(u>-112)&(u<118)).astype(np.float32),1.0)*2,0,1))*keep
    loc=loc*(1-zw[...,None])+est*zw[...,None]
    # soft lid crease
    cr=np.exp(-((v-(Tv-13-0.05*np.clip(u+60,0,200)))/2.6)**2)*np.clip((u+92)/40,0,1)*np.clip((118-u)/30,0,1)*0.30*keep
    loc=loc*(1-cr[...,None])+loc*np.array([0.72,0.50,0.50])*cr[...,None]
    # 2) eye opening: her own iris and eye white (outer white taken from the inner side)
    op=(v>Uv)&(v<Bv)&(u>-110)&(u<112)
    def samp(uu,vv):
        return np.stack([ndimage.map_coordinates(src[...,c],[SY+vv,SX-uu],order=1,mode='nearest') for c in range(3)],-1)
    direct=samp(u,v); refl=samp(-(58+np.maximum(np.abs(u)-58,0)*0.55)*np.sign(u+1e-6),v+np.clip(np.abs(u)-58,0,60)*0.12)
    srchair=((SY+v) < E(SX-u)+4)&(u>0)
    wr=np.clip((u-50)/8,0,1); wr=np.maximum(wr,ndimage.gaussian_filter(srchair.astype(np.float32),1.2))
    eye=direct*(1-wr[...,None])+refl*wr[...,None]
    eye=eye*(1-0.10*np.clip((u-58)/50,0,1))[...,None]
    sh=np.clip(1-(v-Uv)/15,0,1)**1.3*0.82                       # lid shadow under the lash line
    eye=np.minimum(eye,eye*(1-sh[...,None])+np.array([0.05,0.035,0.09])*sh[...,None])
    ow=np.clip(ndimage.gaussian_filter(op.astype(np.float32),1.0)*1.3-0.15,0,1)*keep
    if f<0: ow=ow*np.maximum(wr,(np.abs(v-Uv)<16)|0).astype(np.float32) if False else ow
    loc=loc*(1-ow[...,None])+eye*ow[...,None]
    # 3) ink
    ui=np.clip((u-U0).astype(int),0,LW-1); vi=np.clip((v-V0).astype(int),0,LH-1)
    A=INKA[vi,ui]*keep
    loc=loc*(1-A[...,None])+INK*A[...,None]
    im[sl]=loc
    mask[sl]=np.maximum(mask[sl],np.clip(np.maximum.reduce([zw,cr*3,ow,A])*4,0,1))
    return im,mask
im,mask=build(im,mask,1530,698,-1)
im,mask=build(im,mask,2004,690,+1)
mask=ndimage.gaussian_filter(mask,1.0)
np.save('im2.npy',im); np.save('mask2.npy',mask)
o=Image.fromarray((np.clip(im,0,1)*255).astype(np.uint8))
o.crop((1380,560,2200,800)).resize((1640,480),Image.LANCZOS).save('both.png')
o.crop((1850,560,2180,780)).resize((1650,1100),Image.LANCZOS).save('l.png')
o.crop((1380,560,1710,780)).resize((1650,1100),Image.LANCZOS).save('r.png')
o.crop((1330,480,2230,1220)).save('face.png')
# ---- tidy the dark smudges on the hair lock beside her right eye (hair only)
x0,x1,y0,y1=1372,1560,585,735; sl=(slice(y0,y1),slice(x0,x1))
loc=im[sl].copy(); X=xx[sl]; Y=yy[sl]
lock=(Y<E(X)-1)
mx_=loc.max(2); st=(mx_-loc.min(2))/np.maximum(mx_,1e-3)
goodh=lock&(mx_>0.5)&(st<0.2)
dark=lock&(mx_<0.44)&(X<1515)
dark=ndimage.binary_dilation(ndimage.binary_opening(dark,iterations=1),iterations=3)&lock
est=None
for sg in (30,16,9):
    num=ndimage.gaussian_filter(loc*goodh[...,None],(sg,sg,0)); den=ndimage.gaussian_filter(goodh.astype(np.float32),sg)
    e=num/np.maximum(den,1e-4)[...,None]
    if est is None: est=e
    else:
        c=np.clip((den-0.15)/0.35,0,1)[...,None]; est=est*(1-c)+e*c
dw=np.clip(ndimage.gaussian_filter(dark.astype(np.float32),2.5)*1.4,0,1)*np.clip((E(X)-Y)/2,0,1)*np.clip((X-x0)/12,0,1)*np.clip((Y-y0)/12,0,1)
if '--nohair' not in sys.argv:
    im[sl]=loc*(1-dw[...,None])+est*dw[...,None]; mask[sl]=np.maximum(mask[sl],np.clip(dw*3,0,1))
o=Image.fromarray((np.clip(im,0,1)*255).astype(np.uint8))
o.crop((1380,560,2200,800)).resize((1640,480),Image.LANCZOS).save('both2.png')
o.crop((1340,540,1710,780)).resize((1480,960),Image.LANCZOS).save('r2.png')
np.save('im2.npy',im); np.save('mask2.npy',mask)
