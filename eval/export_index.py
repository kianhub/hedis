#!/usr/bin/env python3
"""Export the hashes collection (covered index scan) to compact numpy arrays per RN:
keys (int64 of the S/C1/C2 triple) and pv (package-version id), plus pv metadata."""
import sys, time, json, numpy as np
from pymongo import MongoClient
M = (1 << 63) - 1
def k3(s, c1, c2):
    f = lambda h: int(h[:16], 16) if h else 0
    return (f(s) ^ (f(c1) * 3) ^ (f(c2) * 7)) & M
if __name__ == "__main__":
    out = sys.argv[1] if len(sys.argv) > 1 else "/workspace/hedis-eval/cache/index"
    db = MongoClient("mongodb://127.0.0.1:27017")["hedis"]
    pk = {p["_id"]: (p["package_name"], p["package_version"]) for p in db.packages.find({}, {"package_name": 1, "package_version": 1})}
    done = {}
    for d in db.fp_done.find():
        pid, rn = d["_id"].split("|"); done[(pid, rn)] = d.get("bundle_sha", "")
    meta = {}
    for rn in ("0.74", "0.76"):
        t = time.time(); ks, pv = [], []; ids = {}
        cur = db.hashes.find({"react_native_version": rn}, {"_id": 0, "package_id": 1, "structural.hash": 1, "ar1.hash": 1, "ar2.hash": 1},
                             hint="cover_rn_pkg_3hash", batch_size=50000)
        for d in cur:
            p = d["package_id"]
            if (str(p), rn) not in done: continue  # only completed jobs
            i = ids.setdefault(p, len(ids))
            ks.append(k3(d["structural"]["hash"], d.get("ar1", {}).get("hash", ""), d.get("ar2", {}).get("hash", ""))); pv.append(i)
        np.savez(f"{out}-{rn}.npz", keys=np.array(ks, dtype=np.int64), pv=np.array(pv, dtype=np.int32))
        meta[rn] = [[*pk[p], done[(str(p), rn)]] for p, _ in sorted(ids.items(), key=lambda x: x[1])]
        print(rn, len(ks), len(ids), f"{time.time()-t:.1f}s", flush=True)
    json.dump(meta, open(f"{out}-meta.json", "w"))
