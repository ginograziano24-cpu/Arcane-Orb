# vertex-clustering simplify for .orbm (keeps silhouette, averages face colours)
import sys,zlib,struct,numpy as np,io
def load(p):
    raw=zlib.decompress(open(p,'rb').read()[4:]); n=struct.unpack_from('<I',raw,4)[0]
    rec=np.frombuffer(raw,np.uint8,count=n*42,offset=8).reshape(n,42)
    P=np.frombuffer(rec[:,:36].tobytes(),np.float32).reshape(n,3,3).astype(np.float64)
    return P,rec[:,36:42].copy()
def save(p,P,A):
    o=io.BytesIO(); o.write(struct.pack('<II',1,len(P)))
    for t,a in zip(P.astype(np.float32),A): o.write(t.tobytes()); o.write(a.tobytes())
    open(p,'wb').write(b'ORBM'+zlib.compress(o.getvalue(),9))
def simplify(P,A,target):
    lo=P.reshape(-1,3).min(0); hi=P.reshape(-1,3).max(0)
    best=None
    for cells in np.geomspace(60,6,40):
        cs=(hi-lo).max()/cells
        key=np.floor((P.reshape(-1,3)-lo)/cs).astype(np.int64)
        k=key[:,0]*1000003+key[:,1]*1009+key[:,2]
        uk,inv=np.unique(k,return_inverse=True)
        cent=np.zeros((len(uk),3)); np.add.at(cent,inv,P.reshape(-1,3)); cnt=np.bincount(inv); cent/=cnt[:,None]
        tri=inv.reshape(-1,3)
        ok=(tri[:,0]!=tri[:,1])&(tri[:,1]!=tri[:,2])&(tri[:,0]!=tri[:,2])
        # merge duplicate faces, averaging colour
        tk=np.sort(tri[ok],1); fk=tk[:,0]*(len(uk)**2)+tk[:,1]*len(uk)+tk[:,2]
        uf,first,finv=np.unique(fk,return_index=True,return_inverse=True)
        if len(uf)<=target:
            idx=np.nonzero(ok)[0]
            col=np.zeros((len(uf),3)); np.add.at(col,finv,A[idx,:3].astype(float)); col/=np.bincount(finv)[:,None]
            NP=cent[tri[idx[first]]]; NA=A[idx[first]].copy(); NA[:,:3]=np.clip(col+0.5,0,255).astype(np.uint8)
            return NP,NA
    return P,A
P,A=load(sys.argv[1]); NP,NA=simplify(P,A,int(sys.argv[3])); save(sys.argv[2],NP,NA)
print(f"{len(P)} -> {len(NP)} triangles")
