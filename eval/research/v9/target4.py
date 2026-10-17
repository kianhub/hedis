#!/usr/bin/env python3
"""target4.py: coverage check for the 4 M40-targeted packages on v9. Same TP/FP/FN and same --subsha release identity as oneclass.py --apps (per-package restriction)."""
import json,sys,subprocess
E="/workspace/hedis-eval"; PK=["react-native-gesture-handler","react-native-safe-area-context","react-native-screens","react-native-reanimated"]
sys.argv=["oneclass.py","X","real_test","--subsha"]; src=open(f"{E}/scripts/oneclass.py").read().split("indb = ")[0]; exec(src)  # loads sha table only
apps=list(json.load(open(f"{E}/data/split.v9.json"))["real_test"])
print("## truth"); 
for a in apps:
    T=json.load(open(f"{E}/data/groundtruth/{a}.json"))["bundled"]; print(a, {p:T[p] for p in PK if p in T} or "none")
for M in ("M40v9","t35v9","M39v9"):
    tot=[0,0,0]; rows=[]
    for a in apps:
        rn="rn0"+open(f"{E}/runs/apps_rel/{a}.triples").readline().split()[2].split(".")[1]
        T=json.load(open(f"{E}/data/groundtruth/{a}.json"))["bundled"]; Pr={x["package"]:x["version"] for x in json.load(open(f"{E}/data/predictions/{M}_{a}.json"))["predictions"]}
        same=lambda n,v,w: v==w or (sha.get((n,v,rn)) and sha.get((n,v,rn))==sha.get((n,w,rn)))
        for n in PK:
            vs=T.get(n); p=Pr.get(n)
            if not vs and not p: continue
            if vs and p and any(same(n,v,p) for v in vs): r=(1,0,len(vs)-1)
            elif vs: r=(0,1 if p else 0,len(vs))
            else: r=(0,1,0)
            tot=[x+y for x,y in zip(tot,r)]; rows.append(f"  {a} {n} truth={vs} pred={p} tp/fp/fn={r}")
    print(f"## {M} target-4 tp {tot[0]} fp {tot[1]} fn {tot[2]}"); print("\n".join(rows))
