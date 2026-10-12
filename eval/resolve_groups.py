#!/usr/bin/env python3
"""Group-resolution step on top of the Go matcher predictions (g3_ab).
For a predicted version inside a multi-version class whose members do NOT all share one bundle, split the class into
sub-groups by bundle SHA and pick the sub-group whose specific features (present in it, absent from some other
sub-group) best match the app: score = |spec ∩ app| - |spec \\ app|. Feature types: ins (per-function instruction hashes
with operands), str (referenced strings incl. short). Types are enabled only if verified on dev.
Also computes the real ceiling (credit 1/|bundle-identical sub-group| of the true version).
usage: resolve_groups.py <types comma> <out_prefix>"""
import sys, os, json, collections, subprocess
types, outp = sys.argv[1].split(","), sys.argv[2]
sys.argv = [sys.argv[0], "t"]; sys.path.insert(0, "/workspace/hedis/eval"); import matchlab3 as L
E_ = L.E_; G = json.load(open(f"{E_}/data/groups.json")); FEAT = f"{E_}/cache/feat"
grp = {}
for g in G:
    for v, s in zip(g["versions"], g["shas"]): grp[(g["rn"], g["pkg"], v)] = (g, s)
def F(s): return json.load(open(f"{FEAT}/{s}.json"))
def appF(a):
    p = f"{FEAT}/app_{a}.json"
    if not os.path.exists(p): subprocess.run(["python3", "/workspace/hedis/eval/hbc_features.py", f"{E_}/apps/out/{a}.hbc", p])
    return json.load(open(p))
def rn_of(a): return "0.74" if "rn074" in a else "0.76"
def resolve(a, x, AF):
    k = (rn_of(a), x["package"], x["version"])
    if k not in grp or grp[k][0]["byte_identical"]: return x["version"], False
    g, _ = grp[k]; subs = collections.defaultdict(list)
    for v, s in zip(g["versions"], g["shas"]): subs[s].append(v)
    if len(subs) < 2: return x["version"], False
    feats = {s: F(s) for s in subs}; best = None
    for s in subs:
        sc = 0
        for t in types:
            mine = set(feats[s][t]); others = [set(feats[o][t]) for o in subs if o != s]
            spec = mine - set.intersection(*others) if others else set()
            sc += len(spec & set(AF[t])) - len(spec - set(AF[t]))
        cand = (sc, x["version"] in subs[s], s)
        if best is None or cand > best: best = cand
    vs = subs[best[2]]; return (x["version"] if x["version"] in vs else sorted(vs)[0]), True
split = json.load(open(f"{E_}/data/split.json"))
apps = {p: [a for a in split[p] if a.startswith("synth-")] for p in ("dev", "test")}
changed = collections.Counter()
for p, aa in apps.items():
    for a in aa:
        r = json.load(open(f"{E_}/data/predictions/g3_ab_{a}.json")); AF = appF(a) if types != [""] else None
        for x in r["predictions"]:
            if AF is None: continue
            nv, touched = resolve(a, x, AF)
            if touched: changed[p] += 1
            if nv != x["version"]: changed[p + "_changed"] += 1; x["version"] = nv
        json.dump(r, open(f"{E_}/data/predictions/{outp}_{a}.json", "w"))
print("resolution touched/changed:", dict(changed))
# real ceiling on bundled truth: true version's bundle-identical subgroup size
for p, aa in apps.items():
    tp = fp = fn = 0.0; n = 0
    for a in aa:
        I = L.idx(rn_of(a)); g = json.load(open(f"{E_}/data/groundtruth/{a}.json"))["bundled"]
        for name, vs in g.items():
            if name not in L.UNI: continue
            for v in vs:
                n += 1
                if I.cls.get((name, v)) is None: fn += 1; continue
                k = (rn_of(a), name, v)
                if k in grp:
                    gg, s = grp[k]; m = sum(1 for t in gg["shas"] if t == s)
                else: m = 1
                tp += 1 / m; fp += 1 - 1 / m; fn += 1 - 1 / m
    P = tp / (tp + fp); R = tp / (tp + fn); print(p, "real ceiling (bundle-identical) strict F1", round(2 * P * R / (P + R), 3), "pairs", n)
