# quick software preview of an .orbm with the game's style of flat lighting
import sys,zlib,struct,numpy as np
from PIL import Image, ImageDraw
def load(p):
    raw=zlib.decompress(open(p,'rb').read()[4:]); n=struct.unpack_from('<I',raw,4)[0]
    P=np.zeros((n,3,3)); C=np.zeros((n,3)); E=np.zeros(n)
    for i in range(n):
        o=8+i*42; P[i]=np.array(struct.unpack_from('<9f',raw,o)).reshape(3,3); C[i]=np.frombuffer(raw[o+36:o+39],np.uint8)/255; E[i]=raw[o+40]/255*2
    return P,C,E
def render(p,yaw,size=420,tint=None):
    P,C,E=load(p); a=np.radians(yaw)
    R=np.array([[np.cos(a),0,np.sin(a)],[0,1,0],[-np.sin(a),0,np.cos(a)]]); Q=P@R.T
    # camera looks along +z from in front, slightly above (game camera looks down the arena)
    tilt=np.radians(14); T=np.array([[1,0,0],[0,np.cos(tilt),-np.sin(tilt)],[0,np.sin(tilt),np.cos(tilt)]]); Q=Q@T.T
    n=np.cross(Q[:,1]-Q[:,0],Q[:,2]-Q[:,0]); n/=np.linalg.norm(n,axis=1,keepdims=True)+1e-12
    L=np.array([-0.35,0.85,-0.4]); L/=np.linalg.norm(L)
    vis=n[:,2]<0
    sh=0.42+0.58*np.clip(n@L,0,1)
    col=np.clip(C*sh[:,None]+C*E[:,None]*0.6,0,1)
    xs=Q[...,0]; ys=Q[...,1]; lo=np.array([xs.min(),ys.min()]); span=max(xs.max()-xs.min(),ys.max()-ys.min())*1.1
    img=Image.new('RGB',(size,size),(18,20,40)); d=ImageDraw.Draw(img)
    order=np.argsort(-Q[...,2].mean(1))
    for i in order:
        if not vis[i]: continue
        pts=[((x-(xs.min()+xs.max())/2)/span*size+size/2, size-((y-lo[1])/span*size+size*0.05)) for x,y,_ in Q[i]]
        d.polygon(pts,fill=tuple((col[i]*255).astype(int)))
    return img
if __name__=='__main__':
    p=sys.argv[1]; out=sys.argv[2]
    ims=[render(p,y) for y in (0,90,200)]
    s=Image.new('RGB',(len(ims)*420,420))
    for i,im in enumerate(ims): s.paste(im,(i*420,0))
    s.save(out)
