import sys,zlib,struct,hashlib,numpy as np
def h(path):
    raw=zlib.decompress(open(path,'rb').read()[4:]); n=struct.unpack_from('<I',raw,4)[0]
    H=hashlib.sha256(); pos=[]; at={}; tri=[]
    for i in range(n):
        o=8+i*42
        P=np.array(struct.unpack_from('<9f',raw,o),np.float32).astype(np.float64)
        H.update(P.astype('<f8').tobytes())
        H.update(bytes([raw[o+36],raw[o+37],raw[o+38],255]))
        emi=np.float32(raw[o+40])/np.float32(255)*np.float32(2)
        H.update(np.array([emi],'<f4').tobytes())
        H.update(bytes([1 if raw[o+41]&1==0 else 0]))
        t=[]
        for k in range(3):
            p=tuple(P[k*3:k*3+3])
            if p not in at: at[p]=len(pos); pos.append(p)
            t.append(at[p])
        tri.append(t)
    H.update(np.array(pos,'<f8').tobytes()); H.update(np.array(tri,'<i4').tobytes())
    return H.hexdigest()
for p in sys.argv[1:]: print(p.split('/')[-1], h(p))
