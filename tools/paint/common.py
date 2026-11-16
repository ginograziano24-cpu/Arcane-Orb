import numpy as np
from scipy.ndimage import zoom, gaussian_filter
def fbm(rng,w,h,base,oct=5,pers=0.5):
    out=np.zeros((h,w),np.float32); amp=1; tot=0
    for o in range(oct):
        gw=max(2,int(w/base*(2**o))+2); gh=max(2,int(h/base*(2**o))+2)
        g=rng.random((gh,gw)).astype(np.float32)
        z=zoom(g,(h/(gh-1),w/(gw-1)),order=1)[:h,:w]
        if z.shape!=(h,w): z=np.pad(z,((0,h-z.shape[0]),(0,w-z.shape[1])),mode='edge')
        out+=z*amp; tot+=amp; amp*=pers
    return out/tot
def col(*c): return np.array(c,np.float32)/255
def lerp(a,b,t): return a+(b-a)*t
