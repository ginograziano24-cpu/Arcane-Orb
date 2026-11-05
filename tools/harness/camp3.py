import json,statistics as st,sys
for f in sys.argv[1:]:
  d=json.load(open(f))
  archs=[];[archs.append(r['Arch']) for r in d if r['Arch'] not in archs]
  m=lambda x: st.mean(x) if x else float('nan')
  print('==',f.split('/')[-1])
  for act in (1,2,3):
    vals={a:[r for r in d if r['Act']==act and r['Arch']==a and r['Node']=='combat'] for a in archs}
    mu=m([m([r['Time'] for r in v]) for v in vals.values()])
    allt=[r['Time'] for v in vals.values() for r in v]
    spread=st.pstdev([m([r['Time'] for r in v]) for v in vals.values()])/mu*100
    print(' act %d mean %3.0f  med %3.0f  origin spread %4.1f%%  timeouts %2d  lost %.1f | '%(act,mu,st.median(allt),spread,sum(t>=399 for t in allt),m([r['Lost'] for v in vals.values() for r in v]))+'  '.join('%s %+.0f%%'%(a[:4],(m([r['Time'] for r in v])/mu-1)*100) for a,v in vals.items()))
  b=[r for r in d if r['Node']=='boss']; print('  bosses cleared %d/%d'%(sum(r['Cleared'] for r in b),len(b)))
