#!/usr/bin/env python3
"""Ceiling for strict pkg+version F1 given identical-fingerprint version classes in hedis_rel.
Oracle: detects every truth package that has its true version in the DB, and picks a version inside the
true version's class. A hash method cannot separate members of a class, so its expected strict credit is 1/|class|
(uniform pick); 'optimistic' credits 1 (lucky pick), i.e. the class-level metric.
usage: ceiling.py <truth_kind> <apps...>"""
import sys, json
sys.argv, args = [sys.argv[0], "t"], sys.argv[1:]
sys.path.insert(0, "/workspace/hedis/eval"); import matchlab3 as L
kind, apps = args[0], args[1:]
tp = fp = fn = 0.0; notdb = 0; tot = 0; ksz = []
for a in apps:
    g = json.load(open(f"{L.E_}/data/groundtruth/{a}.json")); rn = open(f"{L.E_}/runs/apps_rel/{a}.triples").readline().split()[2] if __import__("os").path.exists(f"{L.E_}/runs/apps_rel/{a}.triples") else ("0.74" if "rn074" in a else "0.76")
    I = L.idx(rn)
    for n, vs in g[kind].items():
        if n not in L.UNI: continue
        vs = vs if isinstance(vs, list) else [vs]
        for v in vs:
            tot += 1; c = I.cls.get((n, v))
            if c is None: fn += 1; notdb += 1; continue
            k = sum(1 for x in I.cls.values() if x == c); ksz.append(k)
            tp += 1 / k; fp += 1 - 1 / k; fn += 1 - 1 / k
P = tp / (tp + fp) if tp + fp else 0; R = tp / (tp + fn) if tp + fn else 0
Po, Ro = (tot - notdb) / (tot - notdb), (tot - notdb) / tot
print(json.dumps({"truth": kind, "apps": len(apps), "truth_pairs": tot, "version_not_in_db": notdb,
  "mean_class_size": round(sum(ksz) / len(ksz), 2), "singleton_share": round(sum(k == 1 for k in ksz) / len(ksz), 3),
  "expected_ceiling_PRF": [round(P, 3), round(R, 3), round(2 * P * R / (P + R), 3)],
  "optimistic_ceiling_PRF": [round(Po, 3), round(Ro, 3), round(2 * Po * Ro / (Po + Ro), 3)]}))
