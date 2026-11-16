import numpy as np, sys, math
from PIL import Image, ImageDraw
from scipy.ndimage import gaussian_filter, zoom
from scipy.spatial import cKDTree
W,H=2048,2633
U=W/28.0   # pixels per world unit
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
def paint(variant,out):
    rng=np.random.default_rng(100+variant)
    yy,xx=np.mgrid[0:H,0:W].astype(np.float32)
    # flagstones: jittered grid -> voronoi cells
    cell=U*1.75
    gx,gy=np.meshgrid(np.arange(-1,W/cell+2),np.arange(-1,H/cell+2))
    px=(gx+0.5+(rng.random(gx.shape)-0.5)*0.62)*cell
    py=(gy+0.5+(rng.random(gy.shape)-0.5)*0.62)*cell
    px[::2]+=cell*0.5                      # running bond
    pts=np.stack([px.ravel(),py.ravel()],1)
    tree=cKDTree(pts)
    q=np.stack([xx.ravel(),yy.ravel()],1)
    # chebyshev-ish metric gives squarer slabs
    d,idx=tree.query(q,k=2,p=4)
    d1=d[:,0].reshape(H,W); d2=d[:,1].reshape(H,W); cid=idx[:,0].reshape(H,W)
    edge=(d2-d1)                           # 0 on the joint
    tone=rng.random(len(pts)).astype(np.float32)[cid]
    hue=rng.random(len(pts)).astype(np.float32)[cid]
    grain=fbm(rng,W,H,26,4); blot=fbm(rng,W,H,180,4); big=fbm(rng,W,H,700,3)
    base=col(58,66,96)*(0.78+0.5*tone[...,None])
    base=base*(1-0.10*hue[...,None])+col(70,60,104)*0.10*hue[...,None]*1.2   # a few violet-ish slabs
    stone=base*(0.80+0.42*grain[...,None])*(0.86+0.3*blot[...,None])
    # chiselled bevel: light on the upper-left inner edge, dark on the lower right
    bev=np.clip(1-edge/(U*0.20),0,1)
    gyv,gxv=np.gradient(gaussian_filter(edge,1.5))
    lightdir=(-gxv-gyv)
    stone*= (1+np.clip(lightdir,-1,1)[...,None]*bev[...,None]*0.5)
    # hairline cracks in a few slabs
    cr=fbm(rng,W,H,90,5)
    crack=(np.abs(cr-0.5)<0.004)&(tone>0.80)
    stone[crack]*=0.45
    # joints
    joint=np.clip(1-edge/(U*0.075),0,1)**1.5
    img=stone*(1-joint[...,None])+col(13,15,28)*joint[...,None]
    # moss creeping out of the joints, faintly luminous
    mossn=fbm(rng,W,H,150,5)
    moss=np.clip((mossn-0.50)*5,0,1)*np.clip(1-edge/(U*0.30),0,1)**1.2
    mcol=col(34,96,84)*(0.7+0.6*grain[...,None])
    img=img*(1-moss[...,None]*0.85)+mcol*moss[...,None]*0.85
    # tiny glowing spores on the moss
    sp=(rng.random((H,W))>0.9993)&(moss>0.35)
    spg=gaussian_filter(sp.astype(np.float32),2.2)*30+gaussian_filter(sp.astype(np.float32),0.8)*6
    img+=spg[...,None]*col(120,255,210)
    # ---- inlaid silver-teal runes (kept clear of the brick field's readability: thin, soft)
    S=1
    m=Image.new('L',(W,H),0); d=ImageDraw.Draw(m)
    cx,cy=W/2,H*0.60
    def ring(x,y,r,w=5): d.ellipse([x-r,y-r,x+r,y+r],outline=255,width=w)
    def glyph(x,y,s,k):
        r=np.random.default_rng(k)
        p=[(x+(r.random()-0.5)*s*1.2,y+(r.random()-0.5)*s*1.6) for _ in range(5)]
        d.line(p,fill=255,width=4,joint='curve')
        d.line([(x-s*0.5,y+s*0.9),(x+s*0.5,y+s*0.9)],fill=255,width=3)
    if variant==0:      # the great circle
        for r,w in [(U*6.2,6),(U*5.6,3),(U*3.0,4)]: ring(cx,cy,r,w)
        for k in range(12):
            a=k/12*2*math.pi
            d.line([(cx+math.cos(a)*U*3.0,cy+math.sin(a)*U*3.0),(cx+math.cos(a)*U*5.6,cy+math.sin(a)*U*5.6)],fill=255,width=3)
            glyph(cx+math.cos(a+0.26)*U*4.3,cy+math.sin(a+0.26)*U*4.3,U*0.5,k)
        pts5=[(cx+math.cos(-math.pi/2+k*4*math.pi/5)*U*3.0,cy+math.sin(-math.pi/2+k*4*math.pi/5)*U*3.0) for k in range(6)]
        d.line(pts5,fill=255,width=3)
    elif variant==1:    # moon phases down the middle
        for k in range(7):
            y=H*0.16+k*H*0.115; r=U*1.15
            ring(cx,y,r,4)
            ph=(k-3)/3
            if abs(ph)<0.99:
                ox=ph*r*1.5
                mm=Image.new('L',(W,H),0); dd=ImageDraw.Draw(mm)
                dd.ellipse([cx-r,y-r,cx+r,y+r],fill=255); dd.ellipse([cx-r+ox,y-r,cx+r+ox,y+r],fill=0)
                m.paste(140,mask=mm)
        d.line([(cx-U*4,H*0.08),(cx-U*4,H*0.94)],fill=255,width=3); d.line([(cx+U*4,H*0.08),(cx+U*4,H*0.94)],fill=255,width=3)
    elif variant==2:    # twin vines
        for sgn in (-1,1):
            p=[(cx+sgn*(U*5.2+math.sin(t*5.2)*U*2.2),H*0.04+t*H*0.92) for t in np.linspace(0,1,160)]
            d.line(p,fill=255,width=5,joint='curve')
            for k in range(14):
                t=(k+0.5)/14; x=cx+sgn*(U*5.2+math.sin(t*5.2)*U*2.2); y=H*0.04+t*H*0.92
                a=sgn*(0.9 if k%2 else -0.9)
                x2,y2=x+math.cos(a)*U*1.3*sgn,y-abs(math.sin(a))*U*1.0
                d.line([(x,y),(x2,y2)],fill=255,width=3); ring(x2,y2,U*0.22,3)
    else:               # constellation
        r=np.random.default_rng(5)
        st=[(W*0.12+r.random()*W*0.76,H*0.08+r.random()*H*0.84) for _ in range(16)]
        st.sort(key=lambda p:p[1])
        for a,b in zip(st,st[1:]):
            if math.hypot(a[0]-b[0],a[1]-b[1])<U*9: d.line([a,b],fill=150,width=2)
        for i,(x,y) in enumerate(st):
            rr=U*(0.22+0.2*(i%3))
            d.line([(x-rr*2,y),(x+rr*2,y)],fill=255,width=3); d.line([(x,y-rr*2),(x,y+rr*2)],fill=255,width=3); ring(x,y,rr,3)
    rm=np.asarray(m,np.float32)/255
    # runes are worn: broken up by noise, and hidden under moss
    wear=np.clip((fbm(rng,W,H,60,4)-0.30)*3.5,0,1)
    rm=rm*wear*(1-moss)*(1-joint*0.7)
    core=gaussian_filter(rm,0.9); glow=gaussian_filter(rm,9)
    img=img*(1-core[...,None]*0.85)+col(150,255,232)*core[...,None]*0.85*0.9
    img+=glow[...,None]*col(60,220,200)*0.55
    # ---- moonlight dapple through leaves, slight darkening toward the walls
    dap=np.clip((fbm(rng,W,H,420,4)-0.38)*2.2,0,1)
    img*= (0.80+0.42*dap[...,None]*big[...,None]*1.6)
    img+= dap[...,None]*col(24,34,60)*0.25
    ex=np.minimum(xx,W-xx)/(U*2.2); ey=np.minimum(yy,H-yy)/(U*2.2)
    img*=(0.72+0.28*np.clip(np.minimum(ex,ey),0,1))[...,None]
    img=np.clip(img*1.18,0,1)
    Image.fromarray((img*255+0.5).astype(np.uint8)).save(out,quality=90)
if __name__=='__main__':
    paint(int(sys.argv[1]),sys.argv[2])
