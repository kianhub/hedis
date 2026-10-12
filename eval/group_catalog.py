#!/usr/bin/env python3
"""Catalogue identical-fingerprint version groups (classes) in hedis_rel: which evidence differs between members.
Writes data/groups.json: per (rn, class) members, bundle shas, byte_identical, differing feature types."""
import sys, os, json, collections, subprocess, re
from multiprocessing import Pool
sys.argv = [sys.argv[0], "t"]; sys.path.insert(0, "/workspace/hedis/eval"); import matchlab3 as L
E_ = L.E_; FEAT = f"{E_}/cache/feat"
sha = {}
for l in open(f"{E_}/logs/fp-rel-stats.jsonl"):
    d = json.loads(l)
    if d["ok"] and d.get("BundleSHA"): sha[(d["Name"], d["Version"], "0." + d["RN"][3:])] = d["BundleSHA"]
def feat(job):
    rn, s = job; out = f"{FEAT}/{s}.json"
    if not os.path.exists(out): subprocess.run(["python3", "/workspace/hedis/eval/hbc_features.py", f"{E_}/cache-rel/rn0{rn[2:]}/{s}.hbc", out])
    return out
groups = []
for rn in ("0.74", "0.76"):
    I = L.idx(rn); by = collections.defaultdict(list)
    for (n, v), c in I.cls.items(): by[c].append(v)
    for c, vs in by.items():
        if len(vs) > 1: groups.append({"rn": rn, "cls": c, "pkg": c.split("#")[0], "versions": vs, "shas": [sha.get((c.split("#")[0], v, rn)) for v in vs]})
jobs = {(g["rn"], s) for g in groups for s in g["shas"] if s}
with Pool(6) as p: p.map(feat, sorted(jobs))
VER = re.compile(r"^\d+\.\d+\.\d+")
for g in groups:
    F = [json.load(open(f"{FEAT}/{s}.json")) if s else None for s in g["shas"]]
    g["byte_identical"] = len(set(g["shas"])) == 1
    diff = set()
    if not g["byte_identical"] and all(F):
        if len({f["fcount"] for f in F}) > 1: diff.add("fcount")
        for t in ("str", "num", "ins"):
            if len({json.dumps(f[t], sort_keys=True) for f in F}) > 1:
                diff.add(t)
                if t == "str":
                    keys = [set(f["str"]) for f in F]; sym = set.union(*keys) - set.intersection(*keys)
                    if any(len(x) <= 2 for x in sym): diff.add("str_short")
                    if any(x in g["versions"] or VER.match(x) for x in sym): diff.add("str_version")
                    if any(len(x) > 2 for x in sym): diff.add("str_long")
                    if not sym: diff.add("str_counts_only")
    g["differs"] = sorted(diff) or (["none_but_js_differs"] if not g["byte_identical"] else [])
json.dump(groups, open(f"{E_}/data/groups.json", "w"), indent=1)
c = collections.Counter(); ct = collections.Counter()
for g in groups:
    c["byte_identical" if g["byte_identical"] else "differs"] += 1
    for t in g["differs"]: ct[t] += 1
print("groups", len(groups), dict(c)); print("feature types differing (count of groups):", dict(ct.most_common()))
