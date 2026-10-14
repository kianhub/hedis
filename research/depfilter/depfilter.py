#!/usr/bin/env python3
"""Dependency-graph precision filter for hedis predictions (research, M31 candidate).
A prediction is kept if strong (evidence >= thr) or if its package is 'reachable':
  root:      in app root package.json dependencies (prod only)
  closure:   reachable from root deps via installed node_modules (Node resolution, prod+optional deps)
  reg:       required (deps.json registry edges, any version of the confident parent) by a strong prediction
Inputs read: predictions (go_m23v3 etc.), app package.json / node_modules, data/deps.json. Never ground truth."""
import json, os, sys
E_ = "/workspace/hedis-eval"
APPDIR = {"mattermost-rel": "apps/mattermost", "mattermost-2.20.0": "apps/oss/mattermost-2.20.0",
          "bluewallet-7.0.9": "apps/oss/bluewallet-7.0.9", "leather-mobile-1.7.0": "apps/oss/leather-mobile-1.7.0/apps/mobile",
          "rainbow-rel": "apps/rainbow", "bluesky-1.96.0": "apps/oss/bluesky-1.96.0", "inaturalist-1.0.7": "apps/oss/inaturalist-1.0.7", "valora-1.110.0": "apps/oss/valora-1.110.0"}
def appdir(a): return f"{E_}/" + APPDIR.get(a, f"apps/synth/{a}")
def pj(d):
    try: return json.load(open(os.path.join(d, "package.json")))
    except Exception: return {}
def resolve(frm, name):
    d = frm
    while True:
        c = os.path.join(d, "node_modules", name)
        if os.path.isfile(os.path.join(c, "package.json")): return os.path.realpath(c)
        p = os.path.dirname(d)
        if p == d: return None
        d = p
_cache = {}
def manifest(a):
    if a in _cache: return _cache[a]
    root = appdir(a); rp = pj(root); direct = set(rp.get("dependencies", {}))
    clos, seen, st = set(), set(), [(root, n) for n in direct]
    while st:
        frm, n = st.pop(); q = resolve(frm, n)
        if not q or q in seen: continue
        seen.add(q); m = pj(q); clos.add(m.get("name", n))
        for dn in list(m.get("dependencies", {})) + list(m.get("optionalDependencies", {})): st.append((q, dn))
    clos |= direct; _cache[a] = (direct, clos); return _cache[a]
DEPS = json.load(open(f"{E_}/data/deps.json"))
def reg_children(pkg, ver):
    vs = DEPS.get(pkg, {}); return set(vs.get(ver, [])) if ver in vs else set().union(*vs.values()) if vs else set()
def filt(a, preds, metric, thr, mode):
    ev = lambda x: x[metric] if metric != "frac" else x["matched_fos"] / max(1, x["total_fos"])
    strong = [x for x in preds if ev(x) >= thr]
    direct, clos = manifest(a)
    allowed = set()
    if "root" in mode: allowed |= direct
    if "clos" in mode: allowed |= clos
    if "reg" in mode:
        for x in strong: allowed |= reg_children(x["package"], x["version"])
    floor = float(os.environ.get("DF_FLOOR", "-1"))
    keep = [x for x in preds if ev(x) >= floor and (ev(x) >= thr or x["package"] in allowed)]
    drop = [x for x in preds if x not in keep]
    return keep, drop
if __name__ == "__main__":
    src, dst, metric, thr, mode, apps = sys.argv[1], sys.argv[2], sys.argv[3], float(sys.argv[4]), sys.argv[5], sys.argv[6:]
    os.makedirs(dst, exist_ok=True)
    for a in apps:
        d = json.load(open(f"{src}/{a}.json")); keep, drop = filt(a, d["predictions"], metric, thr, mode)
        d["predictions"] = keep; d["depfilter"] = {"metric": metric, "thr": thr, "mode": mode, "dropped": drop}
        json.dump(d, open(f"{dst}/{a}.json", "w"), indent=1)
