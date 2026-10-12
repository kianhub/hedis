#!/usr/bin/env python3
"""Normalized-hash experiments on the release DB (hedis_rel).
Keys: nsnc = (NS, NC) normalized pair; ns = NS only (structural fallback); hyb = nsnc, but a version's
NS-only keys are used for functions whose (NS,NC) never matched (structural fallback).
Methods: thesis-style HMV on norm keys (n0), (a) unique+IDF, (a)+(b) dev-tuned threshold, (a)+(b)+(c) version F1 tie-break.
usage: matchlab2.py export | run <keymode>"""
import sys, os, json, math, time, collections
from pymongo import MongoClient
E = "/workspace/hedis-eval"; IDX = f"{E}/cache/relidx"
M = (1 << 63) - 1
def h(x): return int(x[:16], 16) if x else 0
def key(ns, nc, mode): return h(ns) if mode == "ns" else (h(ns) ^ (h(nc) * 3)) & M

def export():
    db = MongoClient("mongodb://127.0.0.1:27017")["hedis_rel"]
    pk = {p["_id"]: (p["package_name"], p["package_version"]) for p in db.packages.find({}, {"package_name": 1, "package_version": 1})}
    done = {tuple(d["_id"].split("|")) for d in db.fp_done.find({}, {"_id": 1})}
    out = {"base": {}, "pv": {}}
    for b in db.baselines.find():
        out["base"][b["react_native_version"]] = [[x.get("ns", ""), x.get("nc", "")] for x in b["hashes"]]
    for rn in ("0.74", "0.76"):
        t = time.time(); g = collections.defaultdict(list)
        for d in db.hashes.find({"react_native_version": rn}, {"_id": 0, "package_id": 1, "ns": 1, "nc": 1}, hint="cover_rn_pkg_norm", batch_size=50000):
            if (str(d["package_id"]), rn) in done: g[d["package_id"]].append([d.get("ns", ""), d.get("nc", "")])
        out["pv"][rn] = [[*pk[p], v] for p, v in g.items()]
        print(rn, len(g), sum(len(v) for v in g.values()), f"{time.time()-t:.1f}s", flush=True)
    json.dump(out, open(f"{IDX}.json", "w"))

split = json.load(open(f"{E}/data/split.json")); UNI = set(json.load(open(f"{E}/data/universe_targets.json")))
def truth(app):
    g = json.load(open(f"{E}/data/groundtruth/{app}.json")); return {n: v for n, v in g["prod_closure"].items() if n in UNI}

class Idx:
    def __init__(s, D, rn, mode):
        s.mode = mode; s.base = {key(a, b, mode) for a, b in D["base"][rn]}
        s.meta, s.K, s.KS = [], [], []
        for n, v, hs in D["pv"][rn]:
            s.meta.append((n, v)); s.K.append({key(a, b, mode) for a, b in hs} - s.base)
            s.KS.append({h(a) for a, b in hs})
        s.bypkg = collections.defaultdict(list)
        for i, (n, v) in enumerate(s.meta): s.bypkg[n].append(i)
        df = collections.defaultdict(set)
        for i, ks in enumerate(s.K):
            for k in ks: df[k].add(s.meta[i][0])
        s.df = {k: len(v) for k, v in df.items()}; s.N = len(s.bypkg)
        s.pk = {n: set().union(*(s.K[i] for i in ids)) for n, ids in s.bypkg.items()}

def app_keys(app, mode, I):
    L = [l.split() for l in open(f"{E}/runs/apps_rel/{app}.triples") if not l.startswith("#")]
    return {key(x[3], x[4] if len(x) > 4 else "", mode) for x in L} - I.base

def score(app, I, method, thr=None):
    A = app_keys(app, I.mode, I); preds = []
    for n, ids in I.bypkg.items():
        best = None
        for i in ids:
            K = I.K[i] if method == "n0" else {k for k in I.K[i] if I.df[k] == 1}
            if not K: continue
            m = len(K & A)
            if method == "n0": sc = m / len(K)
            else:
                w = lambda ks: sum(math.log(1 + I.N / I.df[k]) for k in ks); sc = w(K & A) / w(K)
            if best is None or (sc, m) > best[:2]: best = (sc, m, i)
        if not best or best[1] == 0: continue
        sc, m, i = best
        if thr is not None and sc < thr.get(n, thr["_global"]): continue
        if method == "c":
            Ap = A & I.pk[n]; f = []
            for j in ids:
                K = I.K[j]; tp = len(K & Ap); f.append(((2 * tp / (2 * tp + len(Ap - K) + len(K - Ap))) if tp else 0, sc == 0, j))
            i = max(f)[2]
        preds.append({"package": n, "version": I.meta[i][1], "score": sc, "matched_fos": m})
    return {"app": app, "predictions": sorted(preds, key=lambda x: -x["score"])}

def tune(I_of, apps):
    grid = [0, .01, .02, .05, .1, .15, .2, .3, .4, .5]; rows = collections.defaultdict(list)
    for a in apps:
        T = truth(a); P = {x["package"]: x["score"] for x in score(a, I_of(a), "a")["predictions"]}
        for n in set(P) | set(T): rows[n].append((P.get(n, -1), n in T))
    def f1(rs, t):
        tp = sum(s >= t and y for s, y in rs); fp = sum(s >= t and not y for s, y in rs); fn = sum(s < t and y for s, y in rs)
        return 2 * tp / (2 * tp + fp + fn) if tp else 0
    allr = [r for v in rows.values() for r in v]; g = max(grid, key=lambda t: (f1(allr, t), -t)); thr = {"_global": g}
    for n, rs in rows.items():
        if len(rs) >= 2: thr[n] = max(grid, key=lambda t: (f1(rs, t), t == g, -t))
    return thr

if __name__ == "__main__":
    if sys.argv[1] == "export": export(); sys.exit()
    mode = sys.argv[2]; methods = sys.argv[3:] or ["n0", "a", "b", "c"]
    D = json.load(open(f"{IDX}.json")); cache = {}
    def I_of(app):
        rn = open(f"{E}/runs/apps_rel/{app}.triples").readline().split()[2]
        if rn not in cache: cache[rn] = Idx(D, rn, mode)
        return cache[rn]
    dev = list(split["dev"]); test = [a for a in split["test"] if os.path.exists(f"{E}/runs/apps_rel/{a}.triples")]
    thr = tune(I_of, dev); json.dump(thr, open(f"{E}/data/thresholds_rel_{mode}.json", "w"), indent=1)
    for m in methods:
        for a in dev + test:
            r = score(a, I_of(a), {"b": "a"}.get(m, m), thr if m in ("b", "c") else None)
            json.dump(r, open(f"{E}/data/predictions/rel_{mode}_{m}__{a}.json", "w"))
        print("done", mode, m, flush=True)
