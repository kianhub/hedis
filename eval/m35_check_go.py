#!/usr/bin/env python3
"""m35_check_go.py <go_prefix> <proto_src> <floor> <margin> <samekeys> apps...: Go --floor/--vset-* predictions vs the
Python prototype on runs/<proto_src>/<app>.vset.json (all packages, not only the universe). Prints mismatches."""
import json, sys
E_ = "/workspace/hedis-eval"; G, S, fl, m, sk = sys.argv[1], sys.argv[2], float(sys.argv[3]), float(sys.argv[4]), sys.argv[5] == "1"
bad = n = 0
for a in sys.argv[6:]:
    go = {x["package"]: (x["version"], x.get("version_set") or [x["version"]]) for x in json.load(open(f"{E_}/data/predictions/{G}_{a}.json"))["predictions"]}
    pr = {}
    for r in json.load(open(f"{E_}/runs/{S}/{a}.vset.json")) or []:
        if r["score"] < fl: continue
        c = next(v for v in r["versions"] if v["v"] == r["version"]); s = {r["version"]}
        for v in r["versions"]:
            if v["sc"] >= c["sc"] - m - 1e-12 or (sk and v["hit"] == c["hit"]): s.add(v["v"])
        pr[r["package"]] = (r["version"], sorted(s))
    for p in set(go) | set(pr):
        n += 1
        if p not in go or p not in pr or go[p][0] != pr[p][0] or sorted(go[p][1]) != pr[p][1]: bad += 1; print("MISMATCH", a, p, go.get(p), pr.get(p))
print(f"{n} package predictions compared, {bad} mismatches")
