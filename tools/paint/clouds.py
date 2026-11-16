import numpy as np, sys
from PIL import Image
from scipy.ndimage import gaussian_filter
sys.path.insert(0,'/tmp/paint'); from common import fbm,col,lerp
W,H=2048,1152; HY=int(H*0.72)
def paint(seed,out):
    rng=np.random.default_rng(seed)
    yy,xx=np.mgrid[0:H,0:W].astype(np.float32)
    t=np.clip(yy/HY,0,1)
    sx=W*(0.3+0.4*rng.random()); sy=HY*0.25
    # clear high sky: deep blue -> soft azure -> pearly horizon
    sky=lerp(col(54,110,196),col(124,178,236),t[...,None]**1.2)
    sky=lerp(sky,col(226,232,248),(np.clip((t-0.65)/0.35,0,1)**1.4)[...,None])
    img=sky
    d=np.hypot(xx-sx,yy-sy)
    img=img+np.exp(-(d/(W*0.28))**1.1)[...,None]*col(255,246,220)*0.45+np.clip(1-d/30,0,1)[...,None]*col(255,255,240)*1.6
    # towering cumulus: a puff density field, shaded from its own gradient
    D=np.zeros((H,W),np.float32)
    for k in range(8):
        cx=W*(k+0.5)/8+rng.normal(0,W*0.03); base=HY*(0.99+0.04*rng.random()); top=HY*(0.22+0.38*rng.random())
        h=base-top; w=h*(0.45+0.3*rng.random())
        for j in range(int(60+h/6)):
            f=rng.random()**0.9
            py=base-f*h; px=cx+rng.normal(0,w*0.3*(1-0.5*f)); r=(18+rng.random()*34)*(1.15-0.5*f)*(0.6+h/700)
            D+=np.exp(-(((xx-px)/r)**2+((yy-py)/(r*0.85))**2)*1.6)
    D+= (fbm(rng,W,H,26,4)-0.5)*0.5
    dens=np.clip((D-0.55)*4,0,1)
    Ds=gaussian_filter(D,3)
    gy,gx=np.gradient(Ds)
    n=np.sqrt(gx*gx+gy*gy)+1e-6
    sunv=np.array([sx-W/2,sy-HY*0.6]); sunv=sunv/np.linalg.norm(sunv)
    ndl=-(gx*sunv[0]+gy*sunv[1])/n            # facing the sun -> bright
    thick=np.clip(Ds/ (Ds.max()+1e-6),0,1)
    light=np.clip(0.55+0.45*ndl,0,1)*(1-0.35*thick)+0.25
    shadow=col(168,182,220); bright=col(255,253,248)
    c=lerp(shadow,bright,np.clip(light,0,1)[...,None])
    silver=np.clip(1-dens*3,0,1)*dens*4
    c=c+silver[...,None]*col(255,250,235)*0.3
    img=img*(1-dens[...,None])+c*dens[...,None]
    # distant isles in the clear gaps, hazy, one castle each
    placed=0; tries=0
    while placed<2 and tries<60:
        tries+=1
        ix=W*(0.06+0.88*rng.random()); iy=HY*(0.42+0.25*rng.random()); iw=55+rng.random()*55
        if dens[int(iy)-60:int(iy)+40,int(max(0,ix-iw)):int(min(W,ix+iw))].mean()>0.05: continue
        placed+=1
        haze=col(176,198,232)
        u=(xx-ix)/iw
        rock=(np.abs(u)<1)&(yy>iy)&(yy<iy+iw*0.7*np.clip(1-np.abs(u),0,1)**1.3)
        grass=(np.abs(u)<1)&(yy>iy-5)&(yy<=iy)
        img=np.where(rock[...,None],lerp(col(110,104,120),haze,0.5),img)
        img=np.where(grass[...,None],lerp(col(96,160,96),haze,0.45),img)
        for s_ in range(3):
            tx=ix+(s_-1)*iw*0.35; th=(34 if s_==1 else 20)+rng.random()*14; tw=5+(3 if s_==1 else 0)
            tower=(np.abs(xx-tx)<tw)&(yy>iy-th)&(yy<=iy)
            roof=(np.abs(xx-tx)<tw*1.35*np.clip((yy-(iy-th-22))/22,0,1))&(yy>iy-th-22)&(yy<=iy-th)
            img=np.where(tower[...,None],lerp(col(244,244,252),haze,0.4),img)
            img=np.where(roof[...,None],lerp(col(60,100,200),haze,0.4),img)
        wall=(np.abs(xx-ix)<iw*0.4)&(yy>iy-10)&(yy<=iy)
        img=np.where(wall[...,None],lerp(col(236,236,246),haze,0.4),img)
    # god rays from the sun through gaps
    ang=np.arctan2(yy-sy,xx-sx); rays=np.clip(np.sin(ang*31+1.7)*np.sin(ang*13),0,1)**2
    img=img+(rays*np.exp(-d/(W*0.3))*(1-dens)*0.04)[...,None]*col(255,248,225)
    # the cloud deck: a soft rolling sea of cloud filling the bottom
    deck=fbm(rng,W,H,160,5)
    surf=HY-40+deck[int(HY),:]*70
    dm=np.clip((yy-surf[None,:])/18,0,1)
    roll=fbm(rng,W,H,60,4)
    dc=lerp(col(214,222,244),col(250,250,255),np.clip(roll*1.2-0.1,0,1)[...,None])
    dc=dc*(1-np.clip((yy-HY)/(H-HY),0,1)[...,None]*0.1)
    img=img*(1-dm[...,None])+dc*dm[...,None]
    img=1-np.exp(-np.clip(img,0,None)*1.6); img=np.clip(img*1.06,0,1)
    img+=(rng.random((H,W,1)).astype(np.float32)-0.5)/255*1.5
    Image.fromarray((np.clip(img,0,1)*255+0.5).astype(np.uint8)).save(out,quality=92)
paint(int(sys.argv[1]),sys.argv[2])
