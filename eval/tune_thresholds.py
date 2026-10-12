#!/usr/bin/env python3
"""Tune per-package min-score thresholds (method b) on DEV ONLY from threshold-0 prediction files.
usage: tune_thresholds.py <pred_prefix> <out.json>"""
import json, sys, collections
E_ = "/workspace/hedis-eval"; P, out = sys.argv[1], sys.argv[2]
split = json.load(open(f"{E_}/data/split.json")); UNI = set(json.load(open(f"{E_}/data/universe_targets.json")))
grid = [0, .01, .02, .05, .1, .15, .2, .3, .4, .5]; rows = collections.defaultdict(list)
for a in split["dev"]:
    T = {n for n in json.load(open(f"{E_}/data/groundtruth/{a}.json"))["prod_closure"] if n in UNI}
    S = {x["package"]: x["score"] for x in json.load(open(f"{E_}/data/predictions/{P}_{a}.json"))["predictions"]}
    for n in set(S) | T: rows[n].append((S.get(n, -1), n in T))
def f1(rs, t):
    tp = sum(s >= t and y for s, y in rs); fp = sum(s >= t and not y for s, y in rs); fn = sum(s < t and y for s, y in rs)
    return 2 * tp / (2 * tp + fp + fn) if tp else 0
allr = [r for v in rows.values() for r in v]; g = max(grid, key=lambda t: (f1(allr, t), -t)); thr = {"_global": g}
for n, rs in rows.items():
    if len(rs) >= 2: thr[n] = max(grid, key=lambda t: (f1(rs, t), t == g, -t))
json.dump(thr, open(out, "w"), indent=1); print(thr)
