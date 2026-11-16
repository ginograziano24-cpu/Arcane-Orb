import numpy as np, sys
from PIL import Image
from scipy.ndimage import gaussian_filter, zoom
sys.path.insert(0,'/tmp/paint'); from common import fbm,col,lerp
W,H=2048,1152; HY=int(H*0.72)
def ridge(rng,w,base,oct=6,rough=0.55):
    out=np.zeros(w,np.float32); amp=1; tot=0
    for o in range(oct):
        n=max(3,int(w/base*(2**o))+3); g=rng.random(n).astype(np.float32)
        z=zoom(g,w/(n-1),order=3)[:w]
        if len(z)<w: z=np.pad(z,(0,w-len(z)),mode='edge')
        out+=z*amp; tot+=amp; amp*=rough
    return out/tot
def crystal(img,xx,yy,x,yb,h,w,c,lean):
    # a faceted crystal: two-tone (lit left face, darker right), pointed tip
    u=(xx-x-lean*(yb-yy)/h*w)/w
    v=(yb-yy)/h
    m=(v>0)&(v<1)&(np.abs(u)<np.where(v<0.55,1.0-0.15*v,(1-v)/0.45*0.92))
    lit=np.where(u<-0.15,1.05,np.where(u<0.25,0.82,0.55))
    cc=c[None,None,:]*lit[...,None]*(0.75+0.5*v[...,None])
    return np.where(m[...,None],cc,img),m
def paint(seed,out):
    rng=np.random.default_rng(seed)
    yy,xx=np.mgrid[0:H,0:W].astype(np.float32)
    t=np.clip(yy/HY,0,1)
    img=lerp(col(6,8,20),col(18,26,56),t[...,None]**1.2)
    img=lerp(img,col(30,52,92),(np.clip((t-0.7)/0.3,0,1)**1.5)[...,None])
    # cave ceiling: hanging rock with stalactites
    ceil=0.12*H+ridge(rng,W,300,6)*0.18*H
    stal=np.zeros(W,np.float32)
    for k in range(34):
        x=int(rng.random()*W); w=14+rng.random()*40; L=80+rng.random()*260
        xs=np.arange(max(0,int(x-w)),min(W,int(x+w)))
        stal[xs]=np.maximum(stal[xs],L*(1-np.abs(xs-x)/w))
    cy=ceil+stal
    rock=fbm(rng,W,H,90,5)
    cm=yy<cy[None,:]
    img=np.where(cm[...,None],col(14,16,30)*(0.7+0.6*rock[...,None]),img)
    rim=np.exp(-np.clip(cy[None,:]-yy,0,None)/5)*cm
    img=img+rim[...,None]*col(70,140,200)*0.25
    # shafts of light from cracks in the roof
    for k in range(3):
        x0=W*(0.15+0.7*rng.random()); ang=(rng.random()-0.5)*0.4; wd=40+rng.random()*70
        d=np.abs((xx-x0)-ang*(yy))/ (wd*(1+yy/H))
        sh=np.exp(-d**2)*np.clip(1-yy/(HY*1.05),0,1)*(yy>cy[None,:])
        img=img+sh[...,None]*col(120,200,255)*0.22
    # dust motes in the shafts
    m=np.zeros((H,W),np.float32); px=rng.integers(0,W,900); py=rng.integers(0,HY,900); m[py,px]=rng.random(900)
    img=img+gaussian_filter(m,1.1)[...,None]*col(170,220,255)*3
    # far cavern walls (layered rock, mist between layers)
    for i,(hgt,c0,base) in enumerate([(0.42,col(30,44,82),520),(0.3,col(22,30,60),360),(0.18,col(12,16,36),240)]):
        r=ridge(rng,W,base,6,0.55); r=(r-r.min())/(r.max()-r.min()+1e-9)
        top=HY-(0.02+r*hgt)*HY
        mm=np.clip((yy-top[None,:])/1.5,0,1)
        tex=fbm(rng,W,H,50,4)
        lc=c0*(0.8+0.4*tex[...,None])
        lc=lc+np.exp(-np.clip(yy-top[None,:],0,None)/8)[...,None]*col(80,160,220)*0.2
        mist=np.exp(-np.clip(yy-top[None,:],0,None)/60)*0.0+np.clip((yy-top[None,:])/(HY*0.2),0,1)
        lc=lerp(lc,col(40,70,120),(mist*0.5*(1-i*0.3))[...,None])
        img=img*(1-mm[...,None])+lc*mm[...,None]
        # great crystals rising from this layer
        hues=[col(80,230,255),col(170,110,255),col(90,255,210),col(255,110,220)]
        glow=np.zeros((H,W,3),np.float32)
        for k in range(4+i*2):
            x=rng.random()*W; yb=top[int(x)]+45
            c=hues[rng.integers(0,4)]*(0.55+0.25*i)
            for j in range(rng.integers(3,6)):
                h=(110-i*30)*(0.5+rng.random()*1.3); w=h*(0.07+rng.random()*0.05)
                xj=x+rng.normal(0,h*0.25)
                img,mk=crystal(img,xx,yy,xj,yb,h,w,c,(rng.random()-0.5)*1.2)
                glow+=gaussian_filter(mk.astype(np.float32),22)[...,None]*c*0.6
        img=img+glow*1.6
    # underground lake reflecting the crystals below the horizon line
    g=np.clip((yy-HY)/(H-HY),0,1)
    refl=np.flipud(img[2*HY-H:HY]) if 2*HY-H>=0 else None
    lake=lerp(col(10,18,40),col(6,10,24),g[...,None]**0.6)
    if refl is not None:
        rpart=np.zeros_like(img); rpart[HY:]=refl[:H-HY]
        rip=fbm(rng,W,H,30,3)
        lake=lake+rpart*0.35*(1-g[...,None])*(0.7+0.6*rip[...,None])
    gm=np.clip((yy-HY)/2,0,1)[...,None]
    img=img*(1-gm)+lake*gm
    img=1-np.exp(-np.clip(img,0,None)*1.5)
    img+=(rng.random((H,W,1)).astype(np.float32)-0.5)/255*1.5
    Image.fromarray((np.clip(img,0,1)*255+0.5).astype(np.uint8)).save(out,quality=92)
paint(int(sys.argv[1]),sys.argv[2])
