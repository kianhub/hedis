#!/usr/bin/env python3
"""Which hash component breaks? For each isolated-package function not exactly (S,C1,C2) in the app,
check which component combos still match some app function. usage: diag_components.py pkg.raw app.raw"""
import sys, collections, difflib
def load(p):
    return [tuple((l.rstrip("\n").split("\t") + ["", ""])[:3]) for l in open(p)]
P, A = load(sys.argv[1]), load(sys.argv[2])
full = set(A); idx = {k: collections.defaultdict(list) for k in ("S", "SC1", "SC2", "C1C2")}
for a in A:
    idx["S"][a[0]].append(a); idx["SC1"][(a[0], a[1])].append(a); idx["SC2"][(a[0], a[2])].append(a); idx["C1C2"][(a[1], a[2])].append(a)
c = collections.Counter(); ex = collections.defaultdict(list)
for p in set(P):
    if p in full: c["exact"] += 1; continue
    if (p[0], p[1]) in idx["SC1"]: k = "C2 differs only"; o = idx["SC1"][(p[0], p[1])][0]
    elif (p[0], p[2]) in idx["SC2"]: k = "C1 differs only"; o = idx["SC2"][(p[0], p[2])][0]
    elif (p[1], p[2]) in idx["C1C2"] and (p[1] or p[2]): k = "S differs only"; o = idx["C1C2"][(p[1], p[2])][0]
    elif p[0] in idx["S"]: k = "S matches, C1+C2 differ"; o = idx["S"][p[0]][0]
    else: k = "no component match"; o = None
    c[k] += 1
    if o and len(ex[k]) < 3: ex[k].append((p, o))
n = len(set(P)); print(f"package functions: {n}")
for k, v in c.most_common(): print(f"  {k:28s} {v:5d} {v/n:.1%}")
def d(x, y):
    xs, ys = x.split("|"), y.split("|")
    return [t for t in difflib.ndiff(xs, ys) if t[:1] in "+-"][:8]
for k, es in ex.items():
    print("==", k)
    for p, o in es:
        for i, nm in enumerate("S C1 C2".split()):
            if p[i] != o[i]: print("  ", nm, d(p[i], o[i]))
