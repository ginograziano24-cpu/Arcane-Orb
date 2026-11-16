import numpy as np, sys
from PIL import Image
from scipy.ndimage import gaussian_filter
N=1024; rng=np.random.default_rng(5)
def pnoise(scale,beta=2.0):
    # band-limited periodic noise; scale = feature size as a fraction of N
    fr=np.sqrt(np.fft.fftfreq(N)[:,None]**2+np.fft.fftfreq(N)[None,:]**2)*N
    c=float(scale)
    fr[0,0]=1
    amp=fr**(-beta/2)*(fr>=c*0.5)*(fr<=c*4)
    n=np.real(np.fft.ifft2(np.fft.fft2(rng.standard_normal((N,N)))*amp))
    n=(n-n.min())/(n.max()-n.min()); return n.astype(np.float32)
def col(*c): return np.array(c,np.float32)/255
yy,xx=np.mgrid[0:N,0:N].astype(np.float32)
P=N/2  # 2x2 plates
fx=(xx%P)/P; fy=(yy%P)/P
pid=(xx//P).astype(int)+2*(yy//P).astype(int)
ex=np.minimum(fx,1-fx)*P; ey=np.minimum(fy,1-fy)*P; edge=np.minimum(ex,ey)
big=pnoise(2); mid=pnoise(8); fine=pnoise(60)
base=col(70,66,64)*(0.85+0.3*big[...,None])*(0.92+0.16*mid[...,None])
# diamond tread on plates 0 and 3
u=(xx+yy)/22.0; v=(xx-yy)/22.0
du=np.abs((u%1)-0.5); dv=np.abs((v%1)-0.5)
tread=np.clip(1-np.maximum(np.abs(du-0.25)*8,np.abs(dv-0.25)*2.2),0,1)
tread=np.where((pid==0)|(pid==3),tread,0)
img=base*(1+0.22*tread[...,None])
img*=(1-0.12*np.roll(tread,2,0)[...,None])
# plates 1 and 2: grating slots
gs=np.where((pid==1)|(pid==2),np.clip(1-np.abs(((yy%P)%40)-20)/5,0,1)*(np.minimum(ex,ey)>30),0)
img*=(1-0.7*gs[...,None])
# plate seams + bevel + rivets
seam=np.clip(1-edge/3,0,1); bev=np.clip(1-edge/14,0,1)
img*=(1-0.65*seam[...,None]); img*=(1+0.12*bev[...,None]*np.where(fx+fy<1,1,-1)[...,None])
for k in range(0,N,64):
    for (cx,cy) in [(k+32,14),(k+32,P-14),(k+32,P+14),(k+32,N-14),(14,k+32),(P-14,k+32),(P+14,k+32),(N-14,k+32)]:
        d=np.hypot(xx-cx,yy-cy); r=np.clip(1-d/7,0,1)
        img=img*(1+0.35*r[...,None]*np.clip((cy-yy+cx-xx)/7,-1,1)[...,None])
# rust blooms and oil stains (periodic noise -> tileable)
rust=np.clip((pnoise(6)*0.5+pnoise(18)*0.5-0.6)*6,0,1)*(0.5+0.5*fine)
img=img*(1-rust[...,None]*0.5)+col(140,70,36)*rust[...,None]*0.5
oil=np.clip((pnoise(4)*0.5+pnoise(12)*0.5-0.64)*7,0,1)
img*=(1-0.45*oil[...,None])
img+=oil[...,None]*col(30,24,40)*0.25*fine[...,None]
img*=(0.94+0.12*fine[...,None])
img=np.clip(img*1.05,0,1)
Image.fromarray((img*255+0.5).astype(np.uint8)).save(sys.argv[1],quality=90)
