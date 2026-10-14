#!/usr/bin/env python3
"""thesis_vset.py <matcher.txt.json> <app> <method> [margin]: thesis matcher output (score for every package@version)
-> predictions with HMV version plus version_set = versions within margin of the top score (ties ordered like to_predictions)."""
import json, sys
src, app, method = sys.argv[1:4]; m = float(sys.argv[4]) if len(sys.argv) > 4 else 0.003
r = json.load(open(src)); by = {}
for x in r["results"]: by.setdefault(x["package"], []).append(x)
preds = []
for p, xs in by.items():
    b = max(xs, key=lambda x: (x["score"], x["matched_fos"]))
    if b["matched_fos"] == 0: continue
    s = sorted({x["version"] for x in xs if x["score"] >= b["score"] - m - 1e-12})
    preds.append({"package": p, "version": b["version"], "score": b["score"], "matched_fos": b["matched_fos"], "total_fos": b["total_fos"], "version_set": s})
print(json.dumps({"app": app, "method": method, "rn_detected": r["rn_version"], "predictions": sorted(preds, key=lambda x: -x["score"])}, indent=1))
