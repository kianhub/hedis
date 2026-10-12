#!/usr/bin/env python3
"""Round 4 (one-class metric, bundled truth). matchlab3 'a+b' plus:
 - thresholds tuned on dev against BUNDLED truth (was prod_closure)
 - dep T : dependency implication. A target package P that a predicted package Q@v declares in its npm
           dependencies (data/deps.json) is added when >= T of P's keys (any df) are in the app; version =
           best coverage. Fixes packages with ~no unique keys (react-freeze inside react-native-screens).
 - vd    : version-variable-key version choice (as matchlab3)
usage: matchlab4.py <tag> [--dep T] [--vd] [--grid ...]"""
import sys, os, json, math, collections, argparse, subprocess
_a = sys.argv[1:]; sys.argv = sys.argv[:1] + ["_lab4"]
import matchlab3 as L
ap = argparse.ArgumentParser(); ap.add_argument("tag"); ap.add_argument("--dep", type=float, default=0)
ap.add_argument("--vd", action="store_true"); ap.add_argument("--fit", action="store_true"); ap.add_argument("--depshared", action="store_true"); ap.add_argument("--rootver", action="store_true"); ap.add_argument("--lowuniq", type=int, default=0); ap.add_argument("--lowdf", type=int, default=2); ap.add_argument("--nothr", action="store_true"); a_ = ap.parse_args(_a)
E_ = L.E_; split = L.split; UNI = L.UNI; DEPS = json.load(open(f"{E_}/data/deps.json"))
def truth(a):
    return {n: v for n, v in json.load(open(f"{E_}/data/groundtruth/{a}.json"))["bundled"].items() if n in UNI}
def vkey(v):
    return [int(x) if x.isdigit() else -1 for x in v.replace("-", ".").split(".")]
_root = {}
def ridx(rn):  # index without subpath rows: used for version choice (subpath rows exist only for some versions)
    if rn not in _root:
        D0 = L.D; L.D = json.load(open(f"{E_}/cache/relidx_root.json")) if not hasattr(L, "_DR") else L._DR; L._DR = L.D
        _root[rn] = L.Idx(rn); L.D = D0
    return _root[rn]
def best_version(I, A, n):
    best = None
    for i in I.bypkg.get(n, []):
        K = [k for k in I.K[i] if I.df[k] <= 1]
        if len(K) <= a_.lowuniq: K = [k for k in I.K[i] if I.df[k] <= a_.lowdf]
        if not K: continue
        w = lambda ks: sum(math.log(1 + I.N / I.df[k]) for k in ks)
        hit = [k for k in K if k in A]; sc = w(hit) / w(K)
        if best is None or (sc, len(hit), vkey(I.meta[i][1])) > best[0]: best = ((sc, len(hit), vkey(I.meta[i][1])), I.meta[i][1])
    return best and best[1]
def raw(a):
    rn, I, A = L.app(a); S = {}
    if not hasattr(I, "pos"): I.pos = {m: j for j, m in enumerate(I.meta)}
    for n, ids in I.bypkg.items():
        best = None
        for i in ids:
            K = [k for k in I.K[i] if I.df[k] <= 1]
            # packages with almost no unique keys (react-freeze: 1, an entry artifact absent from apps):
            # score on keys shared with at most lowdf packages instead
            if len(K) <= a_.lowuniq: K = [k for k in I.K[i] if I.df[k] <= a_.lowdf]
            if not K: continue
            w = lambda ks: sum(math.log(1 + I.N / I.df[k]) for k in ks)
            hit = [k for k in K if k in A]; sc = w(hit) / w(K)
            if best is None or (sc, len(hit), vkey(I.meta[i][1])) > (best[0], best[1], vkey(I.meta[best[2]][1])): best = (sc, len(hit), i)
        if best and best[1] > 0: S[n] = best
    return rn, I, A, S
def predict(a, thr, R):
    rn, I, A, S = R; preds = {}
    for n, (sc, m, i) in S.items():
        if thr is not None and sc < thr.get(n, thr["_global"]): continue
        if a_.vd and I.var[n]:
            V = I.var[n]; AV = A & V
            vs = lambda j: len(I.K[j] & V & AV) - len((I.K[j] & V) - AV) - len(AV - I.K[j])
            i = max(I.bypkg[n], key=lambda j: (vs(j), j == i))
        if a_.fit:  # version = best whole-keyset fit |K∩A| - |K\A| (tie: unique-key score, newest)
            i = max(I.bypkg[n], key=lambda j: (len(I.K[j] & A) - len(I.K[j] - A), j == i, vkey(I.meta[j][1])))
        ver = I.meta[i][1]
        if a_.rootver:
            RI = ridx(rn); ver = best_version(RI, A - RI.base, n) or ver
        preds[n] = (ver, sc, m, "score")
    if a_.dep:
        changed = True
        while changed:
            changed = False
            for q, (v, *_ ) in list(preds.items()):
                for p in DEPS.get(q, {}).get(v, []):
                    if p in preds or p not in I.bypkg: continue
                    if a_.depshared:  # evidence = P's keys that the parent's own fingerprint also contains (parent bundles them)
                        KQ = I.K[I.pos[(q, v)]] if (q, v) in I.pos else frozenset()
                        cs = [(len(I.K[i] & KQ & A) / len(I.K[i] & KQ), len(I.K[i] & A), vkey(I.meta[i][1]), i) for i in I.bypkg[p] if len(I.K[i] & KQ) >= 5]
                        cov = max(cs, key=lambda x: (x[0] >= a_.dep, x[1], x[2]), default=None)
                        if cov: cov = (cov[0], cov[2], cov[3])
                    else:
                        cov = max(((len(I.K[i] & A) / len(I.K[i]), vkey(I.meta[i][1]), i) for i in I.bypkg[p] if I.K[i]), default=None)
                    if cov and cov[0] >= a_.dep:
                        preds[p] = (I.meta[cov[2]][1], cov[0], len(I.K[cov[2]] & A), "dep:" + q); changed = True
    return {"app": a, "rn_detected": rn, "predictions": [{"package": n, "version": v, "score": s, "matched_fos": m, "via": via}
            for n, (v, s, m, via) in sorted(preds.items(), key=lambda x: -x[1][1])]}
def tune(R):
    grid = [0, .01, .02, .05, .1, .15, .2, .3, .4, .5]; rows = collections.defaultdict(list)
    for a in split["dev"]:
        if not a.startswith("synth"): continue
        T = truth(a)
        for n in set(R[a][3]) | set(T): rows[n].append((R[a][3][n][0] if n in R[a][3] else -1, n in T))
    def f1(rs, t):
        tp = sum(s >= t and y for s, y in rs); fp = sum(s >= t and not y for s, y in rs); fn = sum(s < t and y for s, y in rs)
        return 2 * tp / (2 * tp + fp + fn) if tp else 0
    allr = [r for v in rows.values() for r in v]; g = max(grid, key=lambda t: (f1(allr, t), -t)); thr = {"_global": g}
    for n, rs in rows.items():
        if len(rs) >= 2: thr[n] = max(grid, key=lambda t: (f1(rs, t), t == g, -t))
    return thr
if __name__ == "__main__":
    part = os.environ.get("PARTS", "dev").split(",")
    dev = [a for a in split["dev"] if os.path.exists(f"{E_}/runs/apps_rel/{a}.triples")]
    R = {a: raw(a) for a in dev}; thr = None if a_.nothr else tune(R)
    json.dump(thr, open(f"{E_}/data/thresholds_{a_.tag}.json", "w"), indent=1)
    apps = dev + ([a for a in split["test"] if os.path.exists(f"{E_}/runs/apps_rel/{a}.triples")] if "test" in part else [])
    for a in apps:
        json.dump(predict(a, thr, R[a] if a in R else raw(a)), open(f"{E_}/data/predictions/{a_.tag}_{a}.json", "w"))
    for p in part:
        print(subprocess.run(["python3", f"{os.path.dirname(os.path.abspath(__file__))}/oneclass.py", a_.tag, p] + (["--diag"] if os.environ.get("DIAG") else []),
                             capture_output=True, text=True).stdout.strip())
