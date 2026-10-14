#!/usr/bin/env python3
"""Matcher JSON (all package@version scores) -> predictions: HMV (highest-scoring version) per package."""
import json, sys
src, app, method = sys.argv[1:4]
r = json.load(open(src)); best = {}
for x in r["results"]:
    b = best.get(x["package"])
    if b is None or (x["score"], x["matched_fos"]) > (b["score"], b["matched_fos"]): best[x["package"]] = x
preds = sorted(({"package": p, "version": x["version"], "score": x["score"], "matched_fos": x["matched_fos"], "total_fos": x["total_fos"], **({"version_set": x["version_set"]} if x.get("version_set") else {})} for p, x in best.items()), key=lambda x: -x["score"])
print(json.dumps({"app": app, "method": method, "rn_detected": r["rn_version"], "timings": r["timings"], "predictions": preds}, indent=1))
