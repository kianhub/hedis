#!/usr/bin/env python3
"""vdiag.py <pred_prefix> <apps...>: for every in-universe GT package of the app, show |A ∩ (T\\P)|/|T\\P| and |A ∩ (P\\T)|/|P\\T|
on nsnc keys (root rows, app RN or nearest), T = GT version, P = predicted. Diagnosis only (reads ground truth)."""
import sys, json, pymongo, collections
E_="/workspace/hedis-eval"; P=sys.argv[1]; apps=sys.argv[2:]
db=pymongo.MongoClient().hedis_rel; U=set(json.load(open(f"{E_}/data/universe_targets.json")))
pk={}
for d in db.packages.find({},{'package_name':1,'package_version':1}): pk[(d['package_name'],d['package_version'])]=d['_id']
cache={}
def keys(n,v,rn):
    k=(n,v,rn)
    if k not in cache:
        i=pk.get((n,v)); s=set()
        if i is not None:
            for h in db.hashes.find({'package_id':i,'react_native_version':rn,'sub':{'$ne':True}},{'ns':1,'nc':1,'_id':0}): s.add((h.get('ns'),h.get('nc')))
        cache[k]=s
    return cache[k]
for a in apps:
    L=open(f"{E_}/runs/apps_rel/{a}.triples").read().splitlines(); rn=L[0].split()[2]
    A={tuple(l.split(' ')[3:5]) for l in L[1:] if l and not l.startswith('#')}
    T={n:v for n,v in json.load(open(f"{E_}/data/groundtruth/{a}.json"))["bundled"].items() if n in U}
    Pr={x["package"]:x["version"] for x in json.load(open(f"{E_}/data/predictions/{P}_{a}.json"))["predictions"]}
    for n,vs in T.items():
        p=Pr.get(n)
        for t in vs:
            if p==t: continue
            kt=keys(n,t,rn); kp=keys(n,p,rn) if p else set()
            d1=kt-kp; d2=kp-kt
            print(f"{a:22s} {n:32s} T={t:8s} P={str(p):8s} |T|={len(kt)} A∩T={len(kt&A)}  T\\P {len(d1&A)}/{len(d1)}  P\\T {len(d2&A)}/{len(d2)}")
