#!/usr/bin/env python3
"""Round 3 (normalized keys, hedis_rel export cache/relidx.json):
 base   : RN baseline keys  [+ full React+RN release bundle keys (runs/fullbase/rnXXX.triples) with --fullbase]
 a      : IDF-weighted score on package-unique keys (df==1)     a2: IDF on keys with df<=DF (fix for misses)
 b      : dev-tuned per-package threshold
 vd     : version choice on version-variable keys V_p = union - intersection over the package's versions:
          score(v) = |D_v ∩ A| - |D_v \\ A| - |(A ∩ V_p) \\ K_v|   (D_v = K_v ∩ V_p), tie -> package score
Also writes class map (identical key sets) for the class-level metric.
usage: matchlab3.py <tag> [--fullbase] [--df N] [--vd] """
import sys, os, json, math, collections, argparse
E_ = "/workspace/hedis-eval"
ap = argparse.ArgumentParser(); ap.add_argument("tag"); ap.add_argument("--fullbase", action="store_true")
ap.add_argument("--df", type=int, default=1); ap.add_argument("--vd", action="store_true"); ap.add_argument("--dep", type=float, default=0); ap.add_argument("--depdf", type=float, default=3); a_ = ap.parse_args()
M = (1 << 63) - 1
def h(x): return int(x[:16], 16) if x else 0
def key(ns, nc): return (h(ns) ^ (h(nc) * 3)) & M
split = json.load(open(f"{E_}/data/split.json")); UNI = set(json.load(open(f"{E_}/data/universe_targets.json")))
D = json.load(open(f"{E_}/cache/relidx.json"))
def dump_keys(p):
    return {key(*(l.rstrip("\n").split(" ") + ["", ""])[3:5]) for l in open(p) if not l.startswith("#")}
class Idx:
    def __init__(s, rn):
        s.base = {key(a, b) for a, b in D["base"][rn]}
        if a_.fullbase: s.base |= dump_keys(f"{E_}/runs/fullbase/rn0{rn[2:]}.triples")
        s.meta, s.K = [], []
        for n, v, hs in D["pv"][rn]:
            s.meta.append((n, v)); s.K.append(frozenset({key(x, y) for x, y in hs} - s.base))
        s.bypkg = collections.defaultdict(list)
        for i, (n, v) in enumerate(s.meta): s.bypkg[n].append(i)
        df = collections.defaultdict(set)
        for i, ks in enumerate(s.K):
            for k in ks: df[k].add(s.meta[i][0])
        s.df = {k: len(v) for k, v in df.items()}; s.N = len(s.bypkg)
        s.var = {}
        for n, ids in s.bypkg.items():
            U = frozenset().union(*(s.K[i] for i in ids)); I = frozenset.intersection(*(s.K[i] for i in ids)); s.var[n] = U - I
        s.cls = {}  # (name, version) -> class id (identical key sets)
        for n, ids in s.bypkg.items():
            seen = {}
            for i in ids: s.cls[s.meta[i]] = seen.setdefault(s.K[i], f"{n}#{len(seen)}")
_c = {}
def idx(rn):
    if rn not in _c: _c[rn] = Idx(rn)
    return _c[rn]
def app(a):
    p = f"{E_}/runs/apps_rel/{a}.triples"; rn = open(p).readline().split()[2]; I = idx(rn)
    return rn, I, dump_keys(p) - I.base
def score(a, thr):
    rn, I, A = app(a); preds = []
    for n, ids in I.bypkg.items():
        best = None
        for i in ids:
            K = [k for k in I.K[i] if I.df[k] <= a_.df]
            if not K: continue
            w = lambda ks: sum(math.log(1 + I.N / I.df[k]) for k in ks)
            hit = [k for k in K if k in A]; sc = w(hit) / w(K)
            if best is None or (sc, len(hit)) > best[:2]: best = (sc, len(hit), i)
        # dependency-aware fallback (--dep T): a package whose keys are mostly shared with a dependent package
        # (e.g. react-freeze inside react-native-screens) has ~no unique keys; accept it when >= T of ALL its keys are in the app
        dep = None
        if a_.dep:
            fr = max(((len(I.K[i] & A) / len(I.K[i]), i) for i in ids if len(I.K[i]) >= 20
                      and sum(I.df[k] for k in I.K[i]) / len(I.K[i]) <= a_.depdf), default=(0, None))
            if fr[0] >= a_.dep: dep = fr
        if (not best or best[1] == 0) and not dep: continue
        if not best or best[1] == 0: best = (dep[0], len(I.K[dep[1]] & A), dep[1])
        sc, m, i = best
        if thr is not None and sc < thr.get(n, thr["_global"]) and not dep: continue
        if a_.vd and I.var[n]:
            V = I.var[n]; AV = A & V
            def vs(j):
                Dv = I.K[j] & V; return len(Dv & AV) - len(Dv - AV) - len(AV - I.K[j])
            i = max(ids, key=lambda j: (vs(j), j == i))
        preds.append({"package": n, "version": I.meta[i][1], "score": sc, "matched_fos": m, "cls": I.cls[I.meta[i]]})
    return {"app": a, "rn_detected": rn, "predictions": sorted(preds, key=lambda x: -x["score"])}
def truth(a):
    g = json.load(open(f"{E_}/data/groundtruth/{a}.json"))["prod_closure"]
    return {n: (v if isinstance(v, list) else [v]) for n, v in g.items() if n in UNI}
def tune(raw):
    grid = [0, .01, .02, .05, .1, .15, .2, .3, .4, .5]; rows = collections.defaultdict(list)
    for a in split["dev"]:
        T = truth(a); S = {x["package"]: x["score"] for x in raw[a]["predictions"]}
        for n in set(S) | set(T): rows[n].append((S.get(n, -1), n in T))
    def f1(rs, t):
        tp = sum(s >= t and y for s, y in rs); fp = sum(s >= t and not y for s, y in rs); fn = sum(s < t and y for s, y in rs)
        return 2 * tp / (2 * tp + fp + fn) if tp else 0
    allr = [r for v in rows.values() for r in v]; g = max(grid, key=lambda t: (f1(allr, t), -t)); thr = {"_global": g}
    for n, rs in rows.items():
        if len(rs) >= 2: thr[n] = max(grid, key=lambda t: (f1(rs, t), t == g, -t))
    return thr
def metrics(res, apps):
    out = {}
    for mode in ("pkg", "strict", "class"):
        tp = fp = fn = 0
        for a in apps:
            rn = res[a]["rn_detected"]; I = idx(rn); T = truth(a); P = {x["package"]: x for x in res[a]["predictions"]}
            for n, x in P.items():
                if n not in T: fp += 1; continue
                ok = True if mode == "pkg" else (x["version"] in T[n]) if mode == "strict" else \
                     any(I.cls.get((n, v)) == x["cls"] for v in T[n]) or x["version"] in T[n]
                if ok: tp += 1
                else: fp += 1; fn += 1
            fn += sum(1 if mode == "pkg" else len(T[n]) for n in T if n not in P) + (0 if mode == "pkg" else sum(len(T[n]) - 1 for n in T if n in P))
        p = tp / (tp + fp) if tp + fp else 0; r = tp / (tp + fn) if tp + fn else 0
        out[mode] = (round(p, 3), round(r, 3), round(2 * p * r / (p + r), 3) if p + r else 0)
    return out
if __name__ == "__main__":
    dev = list(split["dev"]); test = [a for a in split["test"] if os.path.exists(f"{E_}/runs/apps_rel/{a}.triples")]
    raw = {a: score(a, None) for a in dev}; thr = tune(raw)
    json.dump(thr, open(f"{E_}/data/thresholds_{a_.tag}.json", "w"), indent=1)
    res = {a: score(a, thr) for a in dev + test}
    for a, r in res.items(): json.dump(r, open(f"{E_}/data/predictions/{a_.tag}_{a}.json", "w"))
    row = {"tag": a_.tag, "args": vars(a_), "dev": metrics(res, dev), "test": metrics(res, test)}
    print(json.dumps(row)); open(f"{E_}/data/lab3_results.jsonl", "a").write(json.dumps(row) + "\n")
