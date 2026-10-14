#!/usr/bin/env python3
"""m31_filter.py: post-hoc precision filters on frozen go_m23 predictions, from the .hbc dump alone (no package.json).
Inputs per app: data/predictions/<SRC>_<app>.json and runs/<SRC>/<app>.hits.json (bin/hedis-norm-hits), runs/apps_rel/<app>.triples.
Filters (each optional):
 floor  t     : drop predictions with score < t
 gen    m,d   : generic-function down-weighting. A hit is generic if its key occurs >= m times in this app (tiny shared
                helper) or was hit for the same package in >= d reference apps where that package scored < 0.2 (keys that
                fire without the package; table built from REF apps only, frozen). score_g = score * (non-generic unique hits / unique hits).
                Packages with < 3 unique-key hits keep their score (no evidence either way).
 clus   c,f   : module clustering. The package's unique-key hits are sorted by app function index and split into runs
                (gap <= G functions; Hermes emits a Metro module's functions contiguously). Keep if largest run >= c or
                (largest run >= 3 and largest run / unique hits >= f). Packages with < 10 unique keys in the DB, or < MINN unique hits, are exempt.
usage: m31_filter.py SRC DST floor gen_m gen_d clus_c clus_f apps...   (use 0 / -1 to disable)"""
import sys, json, os, collections
E_="/workspace/hedis-eval"
SRC,DST=sys.argv[1],sys.argv[2]; floor=float(sys.argv[3]); gm=int(sys.argv[4]); gd=int(sys.argv[5]); cc=int(sys.argv[6]); cf=float(sys.argv[7]); apps=sys.argv[8:]
G=int(os.environ.get("G","64")); REF=json.load(open(os.environ.get("REFTAB",f"{E_}/research/m31/gentab.json")))
MINN=int(os.environ.get("MINN","1"))  # unique hits needed before clustering applies (0 hits = dep-implied / non-unique evidence)
REFAPPS=set(os.environ.get("REFAPPS","").split())  # leave-one-out: an app does not count its own hits
def mult(a):
    c=collections.Counter()
    for l in open(f"{E_}/runs/apps_rel/{a}.triples").read().split("\n")[1:]:
        f=l.split(" ")
        if len(f)>=4 and not l.startswith("#"): c[f[3]+"|"+(f[4] if len(f)>=5 else "")]+=1
    return c
for a in apps:
    P=json.load(open(f"{E_}/data/predictions/{SRC}_{a}.json")); H={x["package"]:x for x in (json.load(open(f"{E_}/runs/{SRC}/{a}.hits.json")) or [])}
    mc=mult(a) if gm>0 else {}; keep=[]
    for x in P["predictions"]:
        h=H.get(x["package"]); u=[y for y in (h["hits"] or []) if y["df"]==1] if h else []
        idx=sorted({y["i"] for y in u}); n=len(idx); sc=x["score"]
        if n>=3 and (gm>0 or gd>0):
            g={y["i"] for y in u if (gm>0 and mc[y["k"]]>=gm) or (gd>0 and REF.get(x["package"]+" "+y["k"],0)-(1 if (a in REFAPPS and x["score"]<0.2) else 0)>=gd)}
            sc=sc*(n-len(g))/n
        if sc<floor: continue
        nuq=h["nuniq"] if h else 0
        if cc>0 and nuq>=10 and n>=MINN:
            best=cur=0; prev=None
            for i in idx: cur=cur+1 if prev is not None and i-prev<=G else 1; prev=i; best=max(best,cur)
            if not (best>=cc or (best>=3 and best/n>=cf)): continue
        keep.append(x)
    P["predictions"]=keep; P["method"]=DST
    json.dump(P,open(f"{E_}/data/predictions/{DST}_{a}.json","w"))
