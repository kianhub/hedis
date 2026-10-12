#!/usr/bin/env python3
"""Classify version/package errors of a prediction prefix on DEV (direct + prod_closure) using hedis_rel norm keys."""
import json, sys, collections
from pymongo import MongoClient
E_ = "/workspace/hedis-eval"; P = sys.argv[1]; split = json.load(open(f"{E_}/data/split.json"))
UNI = set(json.load(open(f"{E_}/data/universe_targets.json"))); db = MongoClient()["hedis_rel"]
pk = {(p["package_name"], p["package_version"]): p["_id"] for p in db.packages.find()}
done = {d["_id"] for d in db.fp_done.find({}, {"_id": 1})}
_k = {}
def keys(n, v, rn):
    if (n, v, rn) not in _k:
        i = pk.get((n, v)); _k[(n, v, rn)] = None if i is None or f"{i}|{rn}" not in done else \
            frozenset((d.get("ns"), d.get("nc")) for d in db.hashes.find({"package_id": i, "react_native_version": rn}, {"ns": 1, "nc": 1}))
    return _k[(n, v, rn)]
for kind in ("direct", "prod_closure"):
    c = collections.Counter(); ex = collections.defaultdict(list)
    for a in split["dev"]:
        g = json.load(open(f"{E_}/data/groundtruth/{a}.json")); T = {n: v for n, v in g[kind].items() if n in UNI}
        r = json.load(open(f"{E_}/data/predictions/{P}_{a}.json")); rn = r["rn_detected"]
        pred = {x["package"]: x["version"] for x in r["predictions"]}
        app = {tuple((l.rstrip("\n").split(" ") + ["", ""])[3:5]) for l in open(f"{E_}/runs/apps_rel/{a}.triples") if not l.startswith("#")}
        for n, v in T.items():
            kt = keys(n, v, rn)
            if kt is None: k = "true version not in DB"
            elif n not in pred: k = "pkg missed: true keys in app %s" % ("0%" if not kt else f"{len(kt & app)/len(kt):.0%}" if len(kt & app)/len(kt) < .5 else ">=50%")
            elif pred[n] == v: k = "correct"
            else:
                kp = keys(n, pred[n], rn)
                k = "wrong: identical norm fingerprint" if kp == kt else "wrong: distinguishable"
                if k.endswith("distinguishable") and len(ex[k]) < 6: ex[k].append((a, n, v, pred[n], len(kt), len(kt & app), len(kp), len(kp & app)))
            c[k if not k.startswith("pkg missed") else "pkg missed"] += 1; c["  " + k] += k.startswith("pkg missed")
        for n in pred:
            if n not in T: c["fp package"] += 1
    print(f"== {P} dev {kind}:", dict(sorted(c.items())))
    for e in ex.get("wrong: distinguishable", []): print("   ", e)
