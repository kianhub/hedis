#!/usr/bin/env python3
"""vdiff.py <src_prefix> <dst_prefix> [--pkgs=a,b|all] [--min=2] [--mode=hop|nbr] <apps...>
M40 version re-selection with version-diff discriminative keys. Package decisions are kept from <src_prefix>;
for each predicted package in --pkgs, start at the predicted version and hill-climb over neighbouring versions
(semver order, skipping neighbours with an identical root key set) using only keys that differ between the two
versions: move p -> w when |A ∩ (Kw \\ Kp)| - |A ∩ (Kp \\ Kw)| >= --min. Keys: root nsnc rows for the app RN
(runs/m40/keys_<rn>.pkl; nearest cached RN otherwise). Does not read ground truth."""
import sys, json, pickle, re
E_="/workspace/hedis-eval"
opt={a.split('=')[0][2:]:a.split('=',1)[1] for a in sys.argv if a.startswith('--') and '=' in a}
args=[a for a in sys.argv[1:] if not a.startswith('--')]; src,dst,apps=args[0],args[1],args[2:]
PK=opt.get('pkgs','react-native-safe-area-context,react-native-screens,react-native-reanimated').split(',')
MIN=int(opt.get('min','2')); MODE=opt.get('mode','hop')
def vkey(v):
    m=re.match(r'(\d+)\.(\d+)\.(\d+)(.*)',v); return (int(m[1]),int(m[2]),int(m[3]),m[4]=='' ,m[4]) if m else (0,0,0,False,v)
C={}
def load(rn):
    if rn not in C:
        try: C[rn]=pickle.load(open(f"{E_}/runs/m40/keys_{rn}.pkl",'rb'))
        except FileNotFoundError: C[rn]=None
    return C[rn]
for a in apps:
    L=open(f"{E_}/runs/apps_rel/{a}.triples").read().splitlines(); rn=L[0].split()[2]
    K=load(rn) or load({'0.72':'0.74','0.75':'0.76','0.78':'0.76','0.79':'0.76','0.80':'0.85'}.get(rn,'0.76'))
    ids=K['ids']; A={ids[t] for l in L[1:] if l and not l.startswith('#') for t in [tuple(l.split(' ')[3:5])] if t in ids}
    pr=json.load(open(f"{E_}/data/predictions/{src}_{a}.json")); pr['method']=dst; log=[]
    for x in pr['predictions']:
        n=x['package']
        if PK!=['all'] and n not in PK: continue
        vs=sorted([v for (m,v) in K['pv'] if m==n and not re.search(r'-(nightly|alpha|beta|rc)',v)],key=vkey)
        if x['version'] not in vs: continue
        R=lambda v:K['pv'][(n,v)][0]
        p=x['version']; seen={p}
        if MODE=='copeland':
            # every distinct root key set is a node; v beats w when A hits more of Kv\\Kw than of Kw\\Kv (by >= MIN)
            reps={}
            for v in vs: reps.setdefault(R(v),[]).append(v)
            nodes=list(reps); sc={}
            for ki,kv in enumerate(nodes):
                w_=l_=0
                for kw in nodes:
                    if kw is kv: continue
                    d=len(A&(kv-kw))-len(A&(kw-kv))
                    if d>=MIN: w_+=1
                    elif d<=-MIN: l_+=1
                sc[ki]=(w_-l_)
            top=max(sc.values()); cand=[v for ki,kv in enumerate(nodes) if sc[ki]==top for v in reps[kv]]
            if p not in cand:
                q=min(cand,key=lambda v:abs(vs.index(v)-vs.index(p))); log.append((n,p,q,top)); p=q
            x['m40_tied']=cand
        while MODE=='hop':
            i=vs.index(p); best=None; bd=MIN-1
            for step in (-1,1):
                j=i+step
                while 0<=j<len(vs) and R(vs[j])==R(p): j+=step
                if not 0<=j<len(vs): continue
                w=vs[j]; d=len(A&(R(w)-R(p)))-len(A&(R(p)-R(w)))
                if d>bd: best,bd=w,d
            if best is None or best in seen: break
            seen.add(best); log.append((n,p,best,bd)); p=best
        if p!=x['version']: x['m40_from']=x['version']; x['version']=p
    pr['m40_log']=log
    json.dump(pr,open(f"{E_}/data/predictions/{dst}_{a}.json",'w'),indent=1)
    if log: print(a,log)
