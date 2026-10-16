#!/usr/bin/env python3
"""vsel.py <src_prefix> <dst_prefix> [--min=2] [--conf=2] [--date=off|max|med] [--pkgs=...|all] <apps...>
M40 version selection (no ground truth read):
 1. version-diff Copeland: per predicted package, nodes = distinct root nsnc key sets over stable versions at the app RN;
    v beats w when |A ∩ (Kv\\Kw)| - |A ∩ (Kw\\Kv)| >= --min. Tie set = nodes with the best (wins - losses).
    Target packages (--pkgs) move to the tied version closest to the M39 pick when the M39 pick is outside the tie set.
 2. dependency-version (release-time) consistency: anchor date D = max (or median) npm publish date of the picks whose
    tie set has <= --conf versions (all predicted packages); a target whose tie set is larger takes its newest tied
    version published on/before D (unchanged if none)."""
import sys, json, pickle, re, os
E_="/workspace/hedis-eval"
opt={a.split('=')[0][2:]:a.split('=',1)[1] for a in sys.argv if a.startswith('--') and '=' in a}
args=[a for a in sys.argv[1:] if not a.startswith('--')]; src,dst,apps=args[0],args[1],args[2:]
PK=opt.get('pkgs','react-native-safe-area-context,react-native-screens,react-native-reanimated').split(',')
MIN=int(opt.get('min','2')); CONF=int(opt.get('conf','2')); DATE=opt.get('date','max')
def vkey(v):
    m=re.match(r'(\d+)\.(\d+)\.(\d+)(.*)',v); return (int(m[1]),int(m[2]),int(m[3]),m[4]=='',m[4]) if m else (0,0,0,False,v)
NPM={}
def pub(n,v):
    if n not in NPM:
        f=f"{E_}/runs/m40/npm/{n.replace('/','_')}.json"; NPM[n]=json.load(open(f)).get('time',{}) if os.path.exists(f) else {}
    return NPM[n].get(v,'')[:10]
C={}
def load(rn):
    if rn not in C:
        try: C[rn]=pickle.load(open(f"{E_}/runs/m40/keys_{rn}.pkl",'rb'))
        except FileNotFoundError: C[rn]=None
    return C[rn]
def copeland(K,n,A):
    vs=sorted([v for (m,v) in K['pv'] if m==n and not re.search(r'-',v)],key=vkey)
    reps={}
    for v in vs: reps.setdefault(K['pv'][(n,v)][0],[]).append(v)
    nodes=list(reps); sc=[]
    for kv in nodes:
        s=0
        for kw in nodes:
            if kw is kv: continue
            d=len(A&(kv-kw))-len(A&(kw-kv)); s+=(d>=MIN)-(d<=-MIN)
        sc.append(s)
    if not sc: return vs,[]
    t=max(sc); return vs,[v for kv,s in zip(nodes,sc) if s==t for v in reps[kv]]
for a in apps:
    L=open(f"{E_}/runs/apps_rel/{a}.triples").read().splitlines(); rn=L[0].split()[2]
    K=load(rn) or load({'0.72':'0.74','0.75':'0.76','0.78':'0.76','0.79':'0.76','0.80':'0.85'}.get(rn,'0.76'))
    ids=K['ids']; A={ids[t] for l in L[1:] if l and not l.startswith('#') for t in [tuple(l.split(' ')[3:5])] if t in ids}
    pr=json.load(open(f"{E_}/data/predictions/{src}_{a}.json")); pr['method']=dst; log=[]; ties={}
    for x in pr['predictions']:
        n=x['package']; vs,cand=copeland(K,n,A)
        if x['version'] not in vs or not cand: continue
        ties[n]=cand
        if (PK==['all'] or n in PK) and x['version'] not in cand:
            q=min(cand,key=lambda v:abs(vs.index(v)-vs.index(x['version']))); log.append((n,x['version'],q,'vdiff')); x['m40_from']=x['version']; x['version']=q
    if DATE!='off':
        conf=sorted(d for y in pr['predictions'] if len(ties.get(y['package'],[1]*99))<=CONF for d in [pub(y['package'],y['version'])] if d)
        if conf:
            D=conf[-1] if DATE=='max' else conf[len(conf)//2]; pr['m40_anchor']=D
            for x in pr['predictions']:
                n=x['package']
                if (PK==['all'] or n in PK) and len(ties.get(n,[]))>CONF:
                    ok=[v for v in ties[n] if pub(n,v) and pub(n,v)<=D]
                    if ok and max(ok,key=vkey)!=x['version']:
                        q=max(ok,key=vkey); log.append((n,x['version'],q,'date '+D)); x.setdefault('m40_from',x['version']); x['version']=q
    pr['m40_log']=log; json.dump(pr,open(f"{E_}/data/predictions/{dst}_{a}.json",'w'),indent=1)
    if log: print(a,log)
