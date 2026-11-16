import numpy as np, sys
from PIL import Image
from scipy.ndimage import gaussian_filter, zoom
W,H=2048,1152
HY=int(H*0.72)
def fbm(rng,w,h,base,oct=5,pers=0.5):
    out=np.zeros((h,w),np.float32); amp=1; tot=0
    for o in range(oct):
        gw=max(2,int(w/base*(2**o))+2); gh=max(2,int(h/base*(2**o))+2)
        g=rng.random((gh,gw)).astype(np.float32)
        z=zoom(g,(h/(gh-1),w/(gw-1)),order=3)[:h,:w]
        if z.shape!=(h,w):
            z=np.pad(z,((0,h-z.shape[0]),(0,w-z.shape[1])),mode='edge')
        out+=z*amp; tot+=amp; amp*=pers
    return out/tot
def ridge(rng,w,base,oct=6,rough=0.55):
    out=np.zeros(w,np.float32); amp=1; tot=0
    for o in range(oct):
        n=max(3,int(w/base*(2**o))+3)
        g=rng.random(n).astype(np.float32)
        z=zoom(g,w/(n-1),order=3)[:w]
        if len(z)<w: z=np.pad(z,(0,w-len(z)),mode='edge')
        out+=z*amp; tot+=amp; amp*=rough
    return out/tot
def lerp(a,b,t): return a+(b-a)*t
def col(*c): return np.array(c,np.float32)/255
def paint(seed,out):
    rng=np.random.default_rng(seed)
    yy,xx=np.mgrid[0:H,0:W].astype(np.float32)
    img=np.zeros((H,W,3),np.float32)
    # ---- sky gradient: deep indigo -> violet -> teal glow at the horizon
    t=np.clip(yy/HY,0,1)
    top,mid,low,hor=col(6,7,26),col(22,18,62),col(40,52,110),col(74,120,150)
    sky=np.where(t[...,None]<0.5,lerp(top,mid,(t[...,None]/0.5)**1.2),
        np.where(t[...,None]<0.85,lerp(mid,low,((t[...,None]-0.5)/0.35)),lerp(low,hor,np.clip((t[...,None]-0.85)/0.15,0,1)**1.5)))
    img[:]=sky
    # ---- milky band + nebula tint
    neb=fbm(rng,W,H,520,5); neb2=fbm(rng,W,H,300,5)
    band=np.exp(-((yy-(0.18*H+0.22*xx*H/W+60*np.sin(xx/400)))/(0.13*H))**2)
    img+=band[...,None]*(neb[...,None]**2.2)*col(70,60,130)*1.3*(1-t[...,None]**3)
    img+=band[...,None]*(np.clip(neb2-0.5,0,1)[...,None]*2)*col(30,70,90)*0.8*(1-t[...,None]**3)
    # ---- stars
    n=1100
    sx=rng.integers(0,W,n); sy=(rng.random(n)**1.3*HY*0.92).astype(int)
    sb=rng.random(n)**4
    stars=np.zeros((H,W),np.float32); stars[sy,sx]=np.maximum(stars[sy,sx],0.18+sb*0.8)
    big=np.zeros((H,W),np.float32); k=sb>0.45; big[sy[k],sx[k]]=sb[k]
    glow=gaussian_filter(big,2.2)*14+gaussian_filter(big,0.8)*3
    tint=lerp(col(200,215,255),col(255,230,210),rng.random((H,W,1)).astype(np.float32))
    fade=(1-t**2.5)[...,None]
    img+=(gaussian_filter(stars,0.55)[...,None]*2.2+glow[...,None])*tint*fade
    # ---- aurora curtains
    aur=np.zeros((H,W,3),np.float32)
    for i in range(3):
        ph=rng.random()*10; base=HY*(0.30+0.13*i+rng.random()*0.06)
        wob=ridge(rng,W,700,4)*HY*0.22+np.sin(np.arange(W)/W*np.pi*(1.2+i*0.6)+ph)*HY*0.06
        yc=base+wob-HY*0.11
        d=(yy-yc[None,:])
        streak=ridge(rng,W,26,3,0.6)**1.5
        strength=ridge(rng,W,500,3)
        strength=np.clip((strength-0.35)*2.5,0,1)
        up=np.exp(np.clip(d,None,0)/(HY*(0.16+0.05*i)))*(d<0)   # long fade upward
        dn=np.exp(-np.clip(d,0,None)/(HY*0.018))*(d>=0)         # sharp lower edge
        a=(up+dn)*(0.35+0.65*streak[None,:])*strength[None,:]
        c1,c2=[(col(60,255,190),col(120,90,255)),(col(80,230,255),col(200,90,255)),(col(110,255,170),col(70,130,255))][i]
        g=np.clip(-d/(HY*0.2),0,1)[...,None]
        aur+=a[...,None]*lerp(c1,c2,g)*(0.5-0.1*i)
    aur=gaussian_filter(aur,(3,1.2,0))
    img+=aur
    # ---- moon
    mx=W*(0.24+0.5*rng.random()); my=HY*(0.36+0.1*rng.random()); mr=H*0.098
    d=np.hypot(xx-mx,yy-my)
    halo=np.exp(-(d/(mr*6.5))**1.0)*0.5+np.exp(-(d/(mr*1.7))**2)*0.7
    img+=halo[...,None]*col(150,190,255)
    disc=np.clip((mr-d)/1.6,0,1)
    mn=fbm(rng,W,H,90,5); mn2=fbm(rng,W,H,30,3)
    nx,ny=(xx-mx)/mr,(yy-my)/mr
    limb=np.sqrt(np.clip(1-nx*nx-ny*ny,0,1))
    surf=(0.80+0.2*limb)*(0.86+0.3*(mn-0.5)+0.12*(mn2-0.5))
    maria=np.clip((fbm(rng,W,H,160,3)-0.52)*6,0,1)*0.16
    moonc=np.clip(surf-maria,0,1.3)[...,None]*col(246,250,255)*1.45
    img=img*(1-disc[...,None])+moonc*disc[...,None]
    # thin cloud wisps across the moon region / sky
    cl=fbm(rng,W,H//3,640,5); cl=zoom(cl,(3,1),order=1)[:H]
    cl=np.clip((cl-0.5)*3.2,0,1)*np.exp(-((yy-HY*0.52)/(HY*0.2))**2)
    lit=0.25+0.9*np.exp(-(d/(mr*4))**1.2)
    img=img*(1-cl[...,None]*0.55)+cl[...,None]*0.55*col(86,104,160)*lit[...,None]
    # ---- mountain layers with mist
    layers=[(0.46,col(58,74,138),760,0.52),(0.34,col(38,48,104),540,0.56),(0.22,col(22,28,70),380,0.6)]
    for i,(hgt,c,base,rough) in enumerate(layers):
        r=ridge(rng,W,base,7,rough)
        r=(r-r.min())/(r.max()-r.min())
        r=r**1.5
        top_y=HY-(0.03+r*hgt)*HY*(0.75 if i else 0.9)
        m=np.clip((yy-top_y[None,:])/1.5,0,1)
        depth=np.clip((yy-top_y[None,:])/(HY*0.16),0,1)
        tex=fbm(rng,W,H,60,4)
        # moonlit rim on slopes facing the moon
        slope=np.gradient(top_y)
        face=np.clip(-slope*np.sign(np.arange(W)-mx)*0.5,0,1)
        rim=np.exp(-np.clip(yy-top_y[None,:],0,None)/14)*(0.3+face[None,:])*0.9
        lc=c[None,None,:]*(0.8+0.4*tex[...,None])+rim[...,None]*col(120,150,220)
        mist=lerp(col(52,84,130),c,0.25)
        lc=lerp(lc,mist[None,None,:],(depth**1.4)[...,None]*0.85)
        img=img*(1-m[...,None])+lc*m[...,None]
    # ---- wizard tower on a crag
    tx=W*(0.72 if mx<W*0.5 else 0.26)+rng.integers(-60,60)
    crag=np.exp(-((xx-tx)/150)**2)*HY*0.17+np.exp(-((xx-tx-40)/60)**2)*HY*0.05
    cn=ridge(rng,W,40,4,0.6)*26
    cy=HY-crag-cn[None,:]*np.exp(-((xx-tx)/230)**2)
    m=np.clip((yy-cy)/1.5,0,1)*(np.abs(xx-tx)<520)
    dk=col(9,10,30)
    img=img*(1-m[...,None])+dk*m[...,None]
    base_y=HY-HY*0.17-10
    def rect(x0,x1,y0,y1,c):
        m=((xx>=x0)&(xx<=x1)&(yy>=y0)&(yy<=y1))[...,None]
        return np.where(m,c,img)
    th=150; tw=17
    taper=tw+ (yy-(base_y-th))/th*7
    m=((np.abs(xx-tx)<taper)&(yy>=base_y-th)&(yy<=base_y+30))[...,None]
    img=np.where(m,dk,img)
    # battlement + conical roof
    m=((np.abs(xx-tx)<tw+6)&(yy>=base_y-th-8)&(yy<=base_y-th+4))[...,None]; img=np.where(m,dk,img)
    m=((np.abs(xx-tx)<(yy-(base_y-th-62))*0.36)&(yy>=base_y-th-62)&(yy<=base_y-th-8))[...,None]; img=np.where(m,dk,img)
    # side turret
    m=((np.abs(xx-(tx+30))<8)&(yy>=base_y-70)&(yy<=base_y+30))[...,None]; img=np.where(m,dk,img)
    m=((np.abs(xx-(tx+30))<(yy-(base_y-98))*0.36)&(yy>=base_y-98)&(yy<=base_y-70))[...,None]; img=np.where(m,dk,img)
    # lit windows + glow
    wins=[(tx,base_y-th+22,3,6),(tx-4,base_y-th+60,2.5,5),(tx+5,base_y-60,2.5,5),(tx+30,base_y-52,2,4)]
    wg=np.zeros((H,W),np.float32)
    for (wx,wy,ww,wh) in wins:
        wm=((np.abs(xx-wx)<=ww)&(np.abs(yy-wy)<=wh)); wg[wm]=1
    img+=gaussian_filter(wg,7)[...,None]*col(255,170,70)*5+gaussian_filter(wg,1)[...,None]*col(255,220,150)*1.6
    # the orb above the tower
    ox,oy=tx,base_y-th-84
    od=np.hypot(xx-ox,yy-oy)
    img+=(np.exp(-(od/26)**1.3)*0.9+np.exp(-(od/4.5)**2)*2.2)[...,None]*col(150,120,255)
    # ---- two conifer forest layers, the far one mist-blue, the near one dark
    crowns=[]
    for li,(fc,hscale,dens,soft) in enumerate([(col(20,30,70),0.75,0.5,0.6),(col(6,8,22),1.0,0.42,0.0)]):
        fr=ridge(rng,W,260,5,0.55); fr=(fr-fr.min())/(fr.max()-fr.min())
        fy=HY-(0.02+0.05*fr+0.03*(1-li))*HY
        trees=np.zeros(W,np.float32)
        x=-40
        while x<W:
            tw_=rng.integers(34,74)*hscale; th_=tw_*(1.5+rng.random()*1.3)
            if rng.random()<0.12: th_*=1.5
            cx=x+tw_/2
            xs=np.arange(max(0,int(x-4)),min(W,int(x+tw_+4)))
            u=np.clip(1-np.abs(xs-cx)/(tw_/2),0,1)
            tiers=5+int(th_/40)
            saw=(u*tiers-np.floor(u*tiers))*0.55/tiers          # drooping bough tiers
            prof=np.clip(u**0.85+saw-0.02,0,1)*th_
            trees[xs]=np.maximum(trees[xs],prof)
            if li==1: crowns.append((cx,th_,tw_))
            x+=tw_*(dens+rng.random()*0.5)
        ty=fy-trees
        m=np.clip((yy-ty[None,:])/1.2,0,1)
        rimf=np.exp(-np.clip(yy-ty[None,:],0,None)/4)*0.3
        lc=fc[None,None,:]+rimf[...,None]*col(80,120,190)
        if soft: lc=lerp(lc,col(52,84,130)[None,None,:],soft*np.clip((yy-ty[None,:])/(HY*0.1),0,1)[...,None])
        img=img*(1-m[...,None])+lc*m[...,None]
    # soft bioluminescent glow pooled under a few trees, a few lantern blooms, fireflies
    spec=np.zeros((H,W,3),np.float32); pool=np.zeros((H,W,3),np.float32)
    hues=[col(90,255,220),col(120,220,255),col(170,120,255),col(255,110,220)]
    for (cx,th_,tw_) in crowns:
        if rng.random()<0.3:
            c=hues[rng.choice(4,p=[0.45,0.3,0.17,0.08])]
            px=int(np.clip(cx,0,W-1)); py=int(np.clip(fy[px]-th_*(0.1+0.25*rng.random()),0,H-1))
            pool[py,px]+=c*(0.6+rng.random())
            for k in range(rng.integers(2,6)):
                qx=int(np.clip(cx+rng.normal(0,tw_/3.2),0,W-1)); qy=int(np.clip(fy[qx]-rng.random()*th_*0.6,0,H-1))
                spec[qy,qx]+=c*(0.4+rng.random()*0.6)
    for k in range(110):
        px=rng.integers(0,W); py=int(HY-rng.random()**2*HY*0.16)
        spec[min(py,H-1),px]+=col(200,255,170)*(0.3+rng.random()*0.7)
    img+=gaussian_filter(pool,(26,34,0))*1500+gaussian_filter(spec,(2.6,2.6,0))*16+gaussian_filter(spec,(0.9,0.9,0))*4
    # low mist at the horizon
    mist=fbm(rng,W,H//4,500,4); mist=zoom(mist,(4,1),order=1)[:H]
    ma=np.exp(-((yy-HY+6)/(HY*0.045))**2)*(0.35+0.5*mist)
    img=img*(1-ma[...,None]*0.6)+ma[...,None]*0.6*col(70,110,150)
    # ---- ground below the horizon: dark, moon-washed, fading to the outer colour
    g=np.clip((yy-HY)/(H-HY),0,1)
    gn=fbm(rng,W,H,120,4)
    gcol=lerp(col(18,26,58),col(14,10,34),(g**0.6)[...,None])*(0.85+0.3*gn[...,None])
    refl=np.exp(-((xx-mx)/(W*0.16))**2)*np.exp(-g*5)*0.22
    gcol+=refl[...,None]*col(120,160,230)
    gm=np.clip((yy-HY)/2,0,1)[...,None]
    img=img*(1-gm)+gcol*gm
    # ---- finish: gentle vignette, filmic curve, dither
    vig=1-0.22*(((xx-W/2)/(W/2))**2+((yy-H*0.45)/(H*0.75))**2)
    img*=vig[...,None]
    img=1-np.exp(-img*1.25)
    img=np.clip(img,0,1)**(1/1.08)
    img+= (rng.random((H,W,1)).astype(np.float32)-0.5)/255*1.5
    Image.fromarray((np.clip(img,0,1)*255+0.5).astype(np.uint8)).save(out,quality=92)
if __name__=='__main__':
    paint(int(sys.argv[1]),sys.argv[2])
