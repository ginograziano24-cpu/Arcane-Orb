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
def paint(seed,out):
    rng=np.random.default_rng(seed)
    yy,xx=np.mgrid[0:H,0:W].astype(np.float32)
    t=np.clip(yy/HY,0,1)
    sx=W*(0.3+0.4*rng.random()); sy=HY*0.86
    # golden-hour sky: deep blue -> lavender -> peach -> gold at the horizon
    sky=lerp(col(52,84,150),col(140,130,180),t[...,None]**1.3)
    sky=lerp(sky,col(250,170,120),(np.clip((t-0.55)/0.45,0,1)**1.6)[...,None])
    sky=lerp(sky,col(255,214,140),(np.clip((t-0.85)/0.15,0,1)**1.4)[...,None])
    img=sky
    d=np.hypot(xx-sx,(yy-sy)*1.25)
    img=img+np.exp(-(d/(W*0.22))**1.2)[...,None]*col(255,170,90)*0.55+np.exp(-(d/60)**2)[...,None]*col(255,240,200)*1.2
    # sun rays through the clouds
    ang=np.arctan2(yy-sy,xx-sx); rays=0.5+0.5*np.sin(ang*23+rng.random()*6)*np.sin(ang*7+1)
    rays*=0.5
    img=img+(np.clip(rays,0,1)**3*np.exp(-d/(W*0.35))*(yy<sy))[...,None]*col(255,210,150)*0.18
    # painted cumulus heaps: each a cluster of soft puffs, lit from below and the sun side
    dens=np.zeros((H,W),np.float32); litacc=np.zeros((H,W),np.float32)
    for k in range(14):
        cx=rng.random()*W; cy=HY*(0.12+0.45*rng.random()); s_=0.6+0.8*rng.random()*(1-cy/HY)
        for j in range(rng.integers(8,16)):
            px=cx+rng.normal(0,110*s_); py=cy-abs(rng.normal(0,34*s_)); r=(36+rng.random()*60)*s_
            dd=((xx-px)/r)**2+((yy-py)/(r*0.8))**2
            puff=np.clip(1-dd,0,1)**0.6
            dens=np.maximum(dens,puff*(yy<cy+r*0.35))   # flat bases
            litacc=np.maximum(litacc,puff*np.clip((yy-(py-r))/(1.6*r),0,1))
    edge=fbm(rng,W,H,40,4)
    dens=np.clip(dens*1.6-0.25+(edge-0.5)*0.5,0,1); dens=gaussian_filter(dens,1.5)
    sunw=np.exp(-np.abs(xx-sx)/(W*0.4))
    c=lerp(col(126,104,160),col(255,206,170),np.clip(litacc*0.7+sunw*0.5,0,1)[...,None])
    rimc=np.clip(dens-np.roll(dens,6,0),0,1)
    c=c+rimc[...,None]*col(255,226,170)*0.9*sunw[...,None]
    img=img*(1-dens[...,None]*0.95)+c*dens[...,None]*0.95
    # distant blue mountains, then rolling green hills with tree lines
    for i,(hgt,c,base,haze) in enumerate([(0.17,col(126,128,170),640,0.6),(0.10,col(96,120,110),420,0.45),(0.065,col(78,116,64),300,0.25)]):
        r=ridge(rng,W,base,6,0.5); r=(r-r.min())/(r.max()-r.min()+1e-9)
        top=HY-(0.015+r*hgt)*HY
        m=np.clip((yy-top[None,:])/1.4,0,1)
        tex=fbm(rng,W,H,50,4)
        lc=c*(0.85+0.3*tex[...,None])
        rim=np.exp(-np.clip(yy-top[None,:],0,None)/7)*np.exp(-np.abs(xx-sx)/(W*0.4))
        lc=lc+rim[...,None]*col(255,190,120)*0.35
        lc=lerp(lc,col(240,180,140),haze*np.exp(-np.clip(yy-top[None,:],0,None)/40)[...,None])
        img=img*(1-m[...,None])+lc*m[...,None]
        if i==2:  # round tree clumps on the nearest hills, sun-lit on one side
            for j in range(140):
                x=int(rng.random()*W); y=top[x]; rr=5+rng.random()*10
                m2=((xx-x)**2+((yy-(y-rr*0.9))*1.1)**2)<rr*rr
                side=np.clip((xx-x)*np.sign(sx-x)/rr,0,1)
                tc=col(44,72,42)+side[...,None]*col(90,80,30)*0.6
                img=np.where(m2[...,None],tc,img)
    # a windmill and farmhouse silhouette on a far ridge
    wx=W*(0.15 if sx>W/2 else 0.82); wy=HY-HY*0.06
    m=(np.abs(xx-wx)<(9+(yy-(wy-90))*0.05))&(yy>wy-90)&(yy<wy+5); img=np.where(m[...,None],col(70,60,70),img)
    a=np.arctan2(yy-(wy-92),xx-wx); r=np.hypot(xx-wx,yy-(wy-92))
    sail=(r<62)&(r>6)&(np.mod(a*4/np.pi+0.3,2)<0.22); img=np.where(sail[...,None],col(80,70,80),img)
    # ground below horizon: meadow green with flower-dot texture
    g=np.clip((yy-HY)/(H-HY),0,1); gn=fbm(rng,W,H,70,5)
    gcol=lerp(col(104,150,72),col(70,118,54),g[...,None]**0.6)*(0.85+0.3*gn[...,None])
    gm=np.clip((yy-HY)/2,0,1)[...,None]; img=img*(1-gm)+gcol*gm
    img=1-np.exp(-np.clip(img,0,None)*1.35); img=np.clip(img*1.08,0,1)
    img+=(rng.random((H,W,1)).astype(np.float32)-0.5)/255*1.5
    Image.fromarray((np.clip(img,0,1)*255+0.5).astype(np.uint8)).save(out,quality=92)
paint(int(sys.argv[1]),sys.argv[2])
