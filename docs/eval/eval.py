#!/usr/bin/env python3
"""HEDIS matching eval harness.
Usage: python3 scripts/eval.py --pred-dir data/predictions --prefix thesis_ [--threshold 0.0]
       [--truth direct|prod_closure] [--universe data/universe.json] [--sweep]
Prediction file: {"app":..., "predictions":[{"package","version","score"}...]}
Ground truth:    data/groundtruth/<app>.json (from build_groundtruth.py)
Scoring is restricted to the reference-DB universe (packages the matcher can
possibly name); apps without ground truth are skipped. Micro-averaged P/R/F1
at package level and package+version level."""
import json, glob, os, argparse
ap = argparse.ArgumentParser()
ap.add_argument("--pred-dir", default="data/predictions"); ap.add_argument("--prefix", default="thesis_")
ap.add_argument("--gt-dir", default="data/groundtruth"); ap.add_argument("--threshold", type=float, default=0.0)
ap.add_argument("--truth", default="prod_closure", choices=["direct", "prod_closure", "bundled"])
ap.add_argument("--universe", default=None, help="JSON list of package names in the reference DB; default = union of predicted names")
ap.add_argument("--sweep", action="store_true"); ap.add_argument("--json", action="store_true")
a = ap.parse_args()

def prf(tp, fp, fn):
    p = tp / (tp + fp) if tp + fp else 0.0; r = tp / (tp + fn) if tp + fn else 0.0
    return p, r, (2 * p * r / (p + r) if p + r else 0.0)

def load():
    runs = []
    for f in sorted(glob.glob(os.path.join(a.pred_dir, a.prefix + "*.json"))):
        pr = json.load(open(f)); g = os.path.join(a.gt_dir, pr["app"] + ".json")
        if os.path.exists(g): runs.append((pr, json.load(open(g))))
    return runs

def score(runs, thr):
    uni = set(json.load(open(a.universe))) if a.universe else set()
    agg = {"pkg": [0, 0, 0], "ver": [0, 0, 0]}; per = []
    for pr, gt in runs:
        U = uni or {x["package"] for x in pr["predictions"]}
        truth = gt[a.truth]
        tv = {(n, v) for n, vs in truth.items() if n in U for v in ([vs] if isinstance(vs, str) else vs)}
        tp_ = {n for n, _ in tv}
        sel = [x for x in pr["predictions"] if x["score"] >= thr and x["package"] in U]
        pp = {x["package"] for x in sel}; pv = {(x["package"], x["version"]) for x in sel}
        r = {"app": pr["app"]}
        for k, P, T in (("pkg", pp, tp_), ("ver", pv, tv)):
            c = (len(P & T), len(P - T), len(T - P)); r[k] = c
            for i in range(3): agg[k][i] += c[i]
        per.append(r)
    return agg, per

runs = load()
if not runs: raise SystemExit("no prediction files with matching ground truth")
thrs = [i / 100 for i in range(0, 61, 5)] if a.sweep else [a.threshold]
out = []
for thr in thrs:
    agg, per = score(runs, thr)
    out.append({"threshold": thr, "truth": a.truth, "per_app": per,
                "package": dict(zip("PRF", prf(*agg["pkg"])), tp_fp_fn=agg["pkg"]),
                "package_version": dict(zip("PRF", prf(*agg["ver"])), tp_fp_fn=agg["ver"])})
if a.json: print(json.dumps(out, indent=1)); raise SystemExit
print(f"truth={a.truth} apps={[r[0]['app'] for r in runs]}")
print(f"{'thr':>5} | {'pkg P':>6} {'pkg R':>6} {'pkg F1':>6} (tp/fp/fn) | {'ver P':>6} {'ver R':>6} {'ver F1':>6} (tp/fp/fn)")
for o in out:
    p, v = o["package"], o["package_version"]
    print(f"{o['threshold']:5.2f} | {p['P']:6.3f} {p['R']:6.3f} {p['F']:6.3f} {tuple(p['tp_fp_fn'])} | {v['P']:6.3f} {v['R']:6.3f} {v['F']:6.3f} {tuple(v['tp_fp_fn'])}")
if not a.sweep:
    for r in out[0]["per_app"]: print("  ", r)
