# Lash set v2: lash line fitted to the SCULPTED upper lid of the model (geo_front.png),
# longer upswept lashes, identical (mirrored) on both eyes.
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
UMIN,UMAX = -112,136
# visible lid line in her painted right eye (only trusted on the nose side, the rest is under hair)
UO = CubicSpline([-112,-110,-95,-80,-65,-50,-35,-20,-5,10,25,40,55,70,85,100,112],
                 [18,14.6,-3.4,-18.8,-31.7,-42.1,-50,-55.6,-58.8,-59.8,-58.5,-55,-49.3,-41.6,-31.8,-19,-6])
Uold = lambda u: UO(np.clip(u,-112,112))
# sculpted upper lid edge of the model (lash line), eye-local coords: u + toward the outer corner, v + down
CC = CubicSpline([-112,-110,-95,-80,-65,-50,-35,-20,-5,10,25,40,60,80,100,116,128,136],
                 [18,14.6,-3.4,-18.8,-31.7,-42.1,-50,-55.6,-58.8,-60.5,-62.5,-64.5,-66,-66,-64,-60,-53,-44])
U = lambda u: CC(np.clip(u,UMIN,UMAX))
qB = np.polyfit([93,-3,-107],[7,26,23],2)
_bu = [-112,-80,-40,0,40,70,93]
BB = CubicSpline(_bu+[112,124,132,136],[float(np.polyval(qB,x)) for x in _bu]+[-4,-18,-32,-44])
B = lambda u: BB(np.clip(u,UMIN,UMAX))
Bold = lambda u: np.polyval(qB,np.clip(u,-112,112))
TH = lambda u: np.interp(u,[-108,-80,-40,0,40,80,118,136],[1.5,3.2,5.0,6.5,8.0,9.5,9.5,4.0])
TOP = lambda u: U(u)-TH(np.clip(u,UMIN,UMAX))
EX=[1380,1400,1420,1445,1470,1492,1530,1580,1630,1680,1720]; EY=[745,722,700,678,658,640,628,614,600,585,574]
E = lambda x: np.interp(x,EX,EY)
# ---- ink canvas (local frame), supersampled
SS=4; U0,V0,LW,LH = -190,-170,400,270
cv = Image.new('L',(LW*SS,LH*SS),0); dr = ImageDraw.Draw(cv)
P = lambda u,v: ((u-U0)*SS,(v-V0)*SS)
us = np.linspace(-108,UMAX,260)
dr.polygon([P(u,float(TOP(u))) for u in us]+[P(u,float(U(u))+1.5) for u in us[::-1]],fill=255)
def lash(u0,ang,L,w0,curl,canvas=dr,base=None,fill=255,pw=0.8):
    v0 = float(TOP(u0))+2.0 if base is None else base
    a=np.radians(ang); c=np.radians(ang+curl)
    p0=np.array([u0,v0]); p2=p0+L*np.array([np.sin(a),-np.cos(a)]); p1=p0+0.5*L*np.array([np.sin(c),-np.cos(c)])
    ts=np.linspace(0,1,40); pts=[(1-t)**2*p0+2*t*(1-t)*p1+t*t*p2 for t in ts]
    L_=[];R_=[]
    for i,t in enumerate(ts):
        d=pts[min(i+1,39)]-pts[max(i-1,0)]; d/=np.linalg.norm(d)+1e-6; n=np.array([-d[1],d[0]])
        w=w0*(1-t)**pw*0.5+0.10
        L_.append(P(*(pts[i]+n*w))); R_.append(P(*(pts[i]-n*w)))
    canvas.polygon(L_+R_[::-1],fill=fill)
    canvas.ellipse((P(u0-w0*0.5,v0-w0*0.5)+P(u0+w0*0.5,v0+w0*0.5)),fill=fill)
K = float(sys.argv[1]) if len(sys.argv)>1 and sys.argv[1][0]!='-' else 1.0     # overall length
ANG=lambda u: np.interp(u,[-105,-60,0,50,90,120,136],[-24,-13,4,19,33,45,56])
LEN=lambda u: np.interp(u,[-105,-70,-30,10,50,90,118,136],[11,18,27,35,43,50,52,42])*K
WID=lambda u: np.interp(u,[-105,0,70,136],[4.2,6.5,8.6,8.6])
MAIN=[-101,-86,-70,-54,-38,-22,-6,10,26,42,57,72,86,99,111,122,131]
for i,u0 in enumerate(MAIN):
    k=(1.0,0.86,0.95)[i%3]
    a_=float(ANG(u0)); L=float(LEN(u0))*k; w=float(WID(u0))
    cu=24*np.tanh(a_/20)+8                      # sweeps out, then curls up
    lash(u0,a_,L,w,cu)
    if i<len(MAIN)-1:
        lash(u0+w*1.05,a_-8,L*0.78,w*0.85,cu)
ink = ndimage.gaussian_filter(np.array(cv.resize((LW,LH),Image.LANCZOS)).astype(np.float32)/255,0.8)
cv2_ = Image.new('L',(LW*SS,LH*SS),0); d2 = ImageDraw.Draw(cv2_)
lu=np.linspace(40,UMAX,70)
d2.polygon([P(u,float(B(u))-np.interp(u,[40,136],[0.5,2.2])) for u in lu]+[P(u,float(B(u))+np.interp(u,[40,136],[0.5,2.2])) for u in lu[::-1]],fill=255)
ink2 = ndimage.gaussian_filter(np.array(cv2_.resize((LW,LH),Image.LANCZOS)).astype(np.float32)/255,1.6)*0.8
cv3_ = Image.new('L',(LW*SS,LH*SS),0); d3 = ImageDraw.Draw(cv3_)
for u0,ang,L in ((74,150,7),(88,142,9),(101,134,10),(113,124,10),(123,112,9)):
    lash(u0,ang,L,3.0,-18,canvas=d3,base=float(B(u0))+0.5)
ink3 = ndimage.gaussian_filter(np.array(cv3_.resize((LW,LH),Image.LANCZOS)).astype(np.float32)/255,0.9)*0.7
INKA = np.clip(np.maximum.reduce([ink*0.97,ink2,ink3]),0,1)

def smooth_skin(im, zone, bad):
    mx_=im.max(2); st=(mx_-im.min(2))/np.maximum(mx_,1e-3)
    good=(mx_>0.62)&(st>0.12)&(im[...,0]>im[...,2])&~zone&~bad
    good=ndimage.binary_erosion(good,iterations=2)
    est=None
    for sg in (70,40,22,12):
        num=ndimage.gaussian_filter(im*good[...,None],(sg,sg,0)); den=ndimage.gaussian_filter(good.astype(np.float32),sg)
        e=num/np.maximum(den,1e-4)[...,None]
        if est is None: est=e
        else:
            c=np.clip((den-0.15)/0.35,0,1)[...,None]; est=est*(1-c)+e*c
    return ndimage.gaussian_filter(est,(3,3,0))
def samp(uu,vv):
    return np.stack([ndimage.map_coordinates(src[...,c],[SY+vv,SX-uu],order=1,mode='nearest') for c in range(3)],-1)
# her own eye-white tone (nose side of her right eye) and the dark top of her iris
_u,_v=np.meshgrid(np.arange(-100,-68),np.arange(-8,16)); _w=samp(_u.astype(np.float32),_v.astype(np.float32)).reshape(-1,3)
WHITE=np.percentile(_w[_w.max(1)>0.85],80,axis=0); IRISDK=samp(np.array([[0.0]]),np.array([[-44.0]]))[0,0]
IRISRIM=samp(np.array([[-57.0]]),np.array([[0.0]]))[0,0]
WHITE=np.clip(WHITE*1.06,0,1)
print('rim',IRISRIM,'white',WHITE,'iris top',IRISDK)
RI=59.0; IRIS_OUT=10.0
def build(im, mask, cx, cy, f):
    x0,x1,y0,y1 = cx-210,cx+210,cy-175,cy+95
    sl=(slice(y0,y1),slice(x0,x1))
    X=xx[sl]; Y=yy[sl]; u=f*(X-cx); v=Y-cy
    loc=im[sl].copy()
    Uv=U(u); Bv=B(u); Tv=TOP(u)
    mx_=loc.max(2); st=(mx_-loc.min(2))/np.maximum(mx_,1e-3)
    if f<0: hair = Y < E(X)+2
    else:   hair = ndimage.binary_dilation(ndimage.binary_opening((st<0.16)&(mx_>0.4)&(u>138),iterations=2),iterations=3)
    keep = 1-np.clip(ndimage.gaussian_filter(hair.astype(np.float32),1.0)*2,0,1) if f>0 else np.ones_like(u)
    # 1) clean skin above the lid and around the outer corner
    zt=Uv-np.interp(u,[-128,-60,20,160],[58,76,92,92])
    zone=((v<Uv+3)&(v>zt)&(u>-128)&(u<(162 if f>0 else 185)))|((u>110)&(u<(162 if f>0 else 185))&(v<Bv+40)&(v>zt))
    zone&=~((v>Bv-2)&(u<UMAX))
    zone|=(u>74)&(u<(162 if f>0 else 185))&(v>Bv+1.5)&(v<Bv+30)
    full=np.zeros((H,W),bool); full[sl]=zone; badf=np.zeros((H,W),bool); badf[sl]=hair
    est=smooth_skin(im,full,badf)[sl]
    zw=np.clip(ndimage.gaussian_filter(zone.astype(np.float32),5)*1.5-0.25,0,1)
    if f<0: zw=np.maximum(zw*(Y>E(X)), np.clip(ndimage.gaussian_filter(zone.astype(np.float32),11)*1.25-0.12,0,1))
    zw=np.maximum(zw,np.clip(ndimage.gaussian_filter((zone&(v>Uv-24)&(v<Uv+3)&(u>-112)&(u<UMAX+4)).astype(np.float32),1.0)*2,0,1))*keep
    loc=loc*(1-zw[...,None])+est*zw[...,None]
    cr=np.exp(-((v-(Tv-12-0.04*np.clip(u+60,0,200)))/2.6)**2)*np.clip((u+92)/40,0,1)*np.clip((UMAX-u)/30,0,1)*0.28*keep
    loc=loc*(1-cr[...,None])+loc*np.array([0.72,0.50,0.50])*cr[...,None]
    # 2) eye opening: her own iris; eye white in her own tone where the old lashes used to cover it
    op=(v>Uv)&(v<Bv)&(u>UMIN+2)&(u<UMAX)
    uo=u; u=u-IRIS_OUT
    r=np.sqrt(u*u+v*v)
    direct=samp(u,v); refl=samp(-u,v)
    src_ok=(v>Uold(u)+2.5)&(v<Bold(u)+3)&(np.abs(u)<110)&~((SY+v)<E(SX-u)+4)      # painted eye visible in the source here
    ref_ok=(v>Uold(-u)+2.5)&(v<Bold(-u)+3)&(np.abs(u)<110)&~((SY+v)<E(SX+u)+4)
    fallback=np.where((r<RI)[...,None],IRISDK[None,None,:],WHITE[None,None,:]*np.ones_like(direct))
    wr=np.clip((u-34)/16,0,1)                                   # outer half: mirror of the nose side
    inir=(r<RI+2)|(u<0)                                        # outer white is one clean tone, not patched
    wd=(1-wr)*src_ok*inir; wf=(wr*ref_ok+(1-wr)*(~src_ok)*ref_ok*(u>0))*inir
    tot=wd+wf
    pick=(direct*wd[...,None]+refl*wf[...,None])/np.maximum(tot,1e-4)[...,None]
    cf=np.clip(ndimage.gaussian_filter(np.clip(tot,0,1),2.5)*1.5-0.35,0,1)*np.clip(tot*4,0,1)
    ext=WHITE[None,None,:]*np.ones_like(direct)
    wi=np.clip((RI+1.2-r)/2.4,0,1)[...,None]
    fallback=IRISRIM[None,None,:]*wi*np.clip((r-RI+7)/6,0,1)[...,None]+fallback*wi*(1-np.clip((r-RI+7)/6,0,1)[...,None])+ext*(1-wi)
    eye=fallback*(1-cf[...,None])+pick*cf[...,None]
    # nose-side white: her own pixels, unshifted, wherever the source shows white there
    ro=np.sqrt(uo*uo+v*v)
    d0=samp(uo,v); ok0=((v>Uold(uo)+2.5)&(v<Bold(uo)+3)&(np.abs(uo)<110)&(ro>RI+5)&(uo<0)&(r>RI+1)).astype(np.float32)
    w0=np.clip(ndimage.gaussian_filter(ok0,1.5)*1.6-0.3,0,1)*ok0
    okb=ok0*(d0.max(2)>0.7)
    n0=ndimage.gaussian_filter(d0*okb[...,None],(7,7,0)); q0=ndimage.gaussian_filter(okb,7)
    e0=np.where((q0>0.03)[...,None],n0/np.maximum(q0,1e-4)[...,None],ext)
    wc=(w0*np.clip((-78-uo)/18,0,1))[...,None]                    # her own pixels only at the inner corner
    scl=d0*wc+ext*(1-wc)
    wi2=np.clip((r-RI+1.2)/2.4,0,1)*np.clip((20-uo)/12,0,1)
    eye=eye*(1-wi2[...,None])+scl*wi2[...,None]
    # shading of the white: rounder toward the outer corner, soft shadow under the lash line
    u=uo
    outer=np.clip((u-75)/65,0,1)*(r>RI)
    eye=eye*(1-0.20*outer[...,None]*np.array([0.85,1.0,1.0]))
    sh=np.clip(1-(v-Uv)/np.interp(u,[-110,0,136],[10,18,22]),0,1)**1.4*0.78
    eye=np.minimum(eye,eye*(1-sh[...,None])+np.array([0.06,0.045,0.10])*sh[...,None])
    ow=np.clip(ndimage.gaussian_filter(op.astype(np.float32),1.0)*1.3-0.15,0,1)*keep
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
o.crop((1380,520,2200,800)).resize((1640,560),Image.LANCZOS).save('both.png')
o.crop((1850,520,2190,780)).resize((1700,1300),Image.LANCZOS).save('l.png')
o.crop((1380,520,1720,780)).resize((1700,1300),Image.LANCZOS).save('r.png')
