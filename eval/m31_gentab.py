#!/usr/bin/env python3
"""m31_gentab.py SRC apps... > gentab.json: for each (package, key) the number of reference apps where the key was hit
for that package while the package's frozen score was < 0.2 (label-free: uses only matcher output on the REF apps)."""
import sys, json, collections
E_="/workspace/hedis-eval"; SRC=sys.argv[1]; c=collections.Counter()
for a in sys.argv[2:]:
    sc={x["package"]:x["score"] for x in json.load(open(f"{E_}/data/predictions/{SRC}_{a}.json"))["predictions"]}
    for h in json.load(open(f"{E_}/runs/{SRC}/{a}.hits.json")) or []:
        if sc.get(h["package"],1)<0.2:
            for k in {y["k"] for y in (h["hits"] or []) if y["df"]==1}: c[h["package"]+" "+k]+=1
json.dump(dict(c),open("/dev/stdout","w"))
