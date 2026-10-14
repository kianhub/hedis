#!/usr/bin/env python3
"""hits_feats.py <M> <apps...>: per predicted package, features from runs/<M>/<app>.hits.json + app triples.
 nu: hits on package-unique keys (df==1); cl: largest run of those hits with gaps <= G app functions; clf = cl/nu;
 gdf: fraction of unique-key hits whose key occurs in >= DFMIN of the DF reference apps (generic code), computed
 from the .hbc dumps of the reference apps only (no labels). Output jsonl to stdout."""
import sys, json, os
E_="/workspace/hedis-eval"; M=sys.argv[1]; apps=sys.argv[2:]
G=int(os.environ.get("G","64")); REF=os.environ.get("DFREF","").split(); DFMIN=int(os.environ.get("DFMIN","3"))
def keys(a):
    L=open(f"{E_}/runs/apps_rel/{a}.triples").read().split("\n")[1:]
    out=[]
    for l in L:
        f=l.split(" ")
        if len(f)>=4 and not l.startswith("#"): out.append(f[3]+"|"+(f[4] if len(f)>=5 else ""))
    return out
dfc={}
for r in REF:
    for k in set(keys(r)): dfc[k]=dfc.get(k,0)+1
for a in apps:
    for x in json.load(open(f"{E_}/runs/{M}/{a}.hits.json")) or []:
        h=[y for y in (x["hits"] or []) if y["df"]==1]; idx=sorted({y["i"] for y in h})
        best=cur=0; prev=None
        for i in idx:
            cur = cur+1 if prev is not None and i-prev<=G else 1; prev=i; best=max(best,cur)
        # generic: key present in >= DFMIN reference apps other than this one
        gen=[y for y in h if dfc.get(y["k"],0)-(1 if a in REF else 0)>=DFMIN]
        print(json.dumps({"app":a,"package":x["package"],"version":x["version"],"nuniq":x["NUniq"] if "NUniq" in x else x["nuniq"],
          "nu":len(idx),"cl":best,"clf":best/len(idx) if idx else 0,"gen":len({y['i'] for y in gen}),"nh":len(x["hits"] or [])}))
