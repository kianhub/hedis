#!/usr/bin/env python3
"""Prototype of the normalized hashes on raw dumps: NS = opcode names with width variants folded
(…Short/…Long/…LongIndex, LoadConst{UInt8,Int,Zero,Double}->LoadConstNum); NC = all strings (C1 ∪ C2,
identifier flag ignored), sorted, minifier-length (<=2) names dropped. usage: norm_check.py pkg.raw app.raw"""
import re, sys
W = re.compile(r"(Short|LongIndex|Long)$")
NUM = {"LoadConstUInt8", "LoadConstInt", "LoadConstZero", "LoadConstDouble"}
def ns(s):
    t = s.split("|"); out = [t[0]]
    for o in t[1:]:
        if not o: continue
        o = W.sub("", o); out.append("LoadConstNum" if o in NUM else o)
    return "|".join(out)
def nc(c1, c2):
    xs = [x for x in (c1.split("|") + c2.split("|")) if len(x) > 2]
    return "|".join(sorted(xs))
def load(p, f):
    return {f(*(l.rstrip("\n").split("\t") + ["", ""])[:3]) for l in open(p)}
for name, f in [("orig S+C1+C2", lambda s, a, b: (s, a, b)), ("NS only", lambda s, a, b: ns(s)),
                ("NS+NC", lambda s, a, b: (ns(s), nc(a, b))), ("S+NC", lambda s, a, b: (s, nc(a, b)))]:
    P, A = load(sys.argv[1], f), load(sys.argv[2], f)
    print(f"{name:14s} pkg keys {len(P):5d} in app {len(P & A):5d} {len(P & A)/len(P):.1%}")
