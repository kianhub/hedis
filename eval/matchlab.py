#!/usr/bin/env python3
"""Offline matcher experiments over the exported fingerprint index.
Methods (cumulative unless --only):
  thesis : exact 3-hash match, score = matched/total per package-version, highest-match version (HMV), any match -> predicted
  a_idf  : drop triples shared across >1 package name; package score = IDF-weighted match fraction
  b_thr  : + per-package minimum score threshold (tuned on dev only; global fallback)
  c_tie  : + version choice by F1 between the version's unique triples and the app's triples of that package
           (rewards functions that differ between neighbouring versions instead of raw overlap)
  d_anchor: + among near-tied versions prefer one whose version string is in its own bundle and in the app's strings
Writes data/predictions/<method>__<app>.json in the eval.py format."""
import json, sys, os, re, math, subprocess, collections, numpy as np
sys.path.insert(0, os.path.dirname(__file__))
from export_index import k3
E="/workspace/hedis-eval"; IDX=f"{E}/cache/index"
meta = json.load(open(f"{IDX}-meta.json"))
split = json.load(open(f"{E}/data/split.json"))
UNI = set(json.load(open(f"{E}/data/universe_targets.json")))

class RNIndex:
    def __init__(s, rn):
        z = np.load(f"{IDX}-{rn}.npz"); s.meta = meta[rn]
        s.names = [m[0] for m in s.meta]
        pv, keys = z["pv"], z["keys"]
        s.pvkeys = collections.defaultdict(set)
        for p, k in zip(pv.tolist(), keys.tolist()): s.pvkeys[p].add(k)
        s.bypkg = collections.defaultdict(list)
        for p, m in enumerate(s.meta): s.bypkg[m[0]].append(p)
        df = collections.defaultdict(set)
        for p, ks in s.pvkeys.items():
            for k in ks: df[k].add(s.names[p])
        s.df = {k: len(v) for k, v in df.items()}; s.N = len(s.bypkg)
        s.pkgkeys = {n: set().union(*(s.pvkeys[p] for p in ps)) for n, ps in s.bypkg.items()}
_idx = {}
def idx(rn):
    if rn not in _idx: _idx[rn] = RNIndex(rn)
    return _idx[rn]

def load_app(app):
    lines = open(f"{E}/runs/apps/{app}.triples").read().split("\n")
    rn = lines[0].split()[2]
    A = {k3(*(l.split(" ") + ["", ""])[:3]) for l in lines[1:] if l}
    return rn, A
_strs = {}
def strings(path):
    if path not in _strs:
        try: _strs[path] = set(subprocess.run(["strings", "-n", "3", path], capture_output=True, text=True).stdout.split("\n"))
        except Exception: _strs[path] = set()
    return _strs[path]
VER = re.compile(r"\d+\.\d+\.\d+")
def verstrings(path):
    return {m for s in strings(path) for m in VER.findall(s)}

def score_app(app, method, thr=None):
    rn, A = load_app(app); I = idx(rn); preds = []
    uniq = method != "thesis"
    for name, pvs in I.bypkg.items():
        best = None
        for p in pvs:
            K = I.pvkeys[p]
            if uniq: K = {k for k in K if I.df[k] == 1}
            if not K: continue
            m = len(K & A); sc = m / len(K)
            if uniq:
                w = lambda ks: sum(math.log(1 + I.N / I.df[k]) for k in ks)
                sc = w(K & A) / w(K)
            cand = (sc, m, p)
            if best is None or cand[:2] > best[:2]: best = cand
        if best is None or best[1] == 0: continue
        sc, m, p = best
        if method in ("b_thr", "c_tie", "d_anchor") and thr is not None and sc < thr.get(name, thr["_global"]): continue
        if method in ("c_tie", "d_anchor"):
            Ap = A & I.pkgkeys[name]; scored = []
            for q in pvs:
                K = I.pvkeys[q]; tp = len(K & Ap); fp = len(Ap - K); fn = len(K - Ap)
                scored.append((2 * tp / (2 * tp + fp + fn) if tp else 0, q))
            scored.sort(reverse=True); p = scored[0][1]
            if method == "d_anchor":
                near = [q for f, q in scored if f >= scored[0][0] - 0.02]
                av = verstrings(f"{E}/apps/out/{app}.hbc")
                anch = [q for q in near if I.meta[q][1] in av and I.meta[q][2] and
                        I.meta[q][1] in verstrings(f"{E}/cache/rn0{rn[2:]}/{I.meta[q][2]}.hbc")]
                if len(anch) >= 1: p = max(anch, key=lambda q: dict((b, a) for a, b in scored)[q])
        preds.append({"package": name, "version": I.meta[p][1], "score": sc, "matched_fos": m})
    preds.sort(key=lambda x: -x["score"])
    return {"app": app, "method": method, "rn_detected": rn, "predictions": preds}

def truth(app, kind="prod_closure"):
    g = json.load(open(f"{E}/data/groundtruth/{app}.json"))
    t = g.get(kind) or g.get("direct"); return {n: v for n, v in t.items() if n in UNI} if isinstance(t, dict) else {}

def tune(method, base):
    """per-package threshold on dev (prod_closure truth, package level), global fallback; dev apps only."""
    dev = list(split["dev"]); raw = {a: score_app(a, base) for a in dev}; grid = [0, .01, .02, .05, .1, .15, .2, .3, .4, .5, .6, .7]
    def f1(rows, t):
        tp = sum(1 for s, y in rows if s >= t and y); fp = sum(1 for s, y in rows if s >= t and not y); fn = sum(1 for s, y in rows if s < t and y)
        return 2 * tp / (2 * tp + fp + fn) if tp else 0
    rows = collections.defaultdict(list)
    for a, r in raw.items():
        T = truth(a); seen = set()
        for x in r["predictions"]: rows[x["package"]].append((x["score"], x["package"] in T)); seen.add(x["package"])
        for n in T:
            if n not in seen: rows[n].append((-1, True))
    allrows = [r for v in rows.values() for r in v]
    thr = {"_global": max(grid, key=lambda t: (f1(allrows, t), -t))}
    for n, rs in rows.items():
        if len(rs) >= 2: thr[n] = max(grid, key=lambda t: (f1(rs, t), t == thr["_global"], -t))
    return thr

if __name__ == "__main__":
    methods = sys.argv[1:] or ["thesis", "a_idf", "b_thr", "c_tie", "d_anchor"]
    apps = list(split["dev"]) + [a for a in split["test"] if os.path.exists(f"{E}/runs/apps/{a}.triples")]
    thr = None
    for m in methods:
        if m in ("b_thr", "c_tie", "d_anchor") and thr is None:
            thr = tune(m, "a_idf"); json.dump(thr, open(f"{E}/data/thresholds_dev.json", "w"), indent=1)
        for a in apps:
            json.dump(score_app(a, m, thr), open(f"{E}/data/predictions/lab_{m}__{a}.json", "w"), indent=1)
        print("done", m, flush=True)
