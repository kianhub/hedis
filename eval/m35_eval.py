#!/usr/bin/env python3
"""m35_eval.py: version-set + vulnerability-level eval (bundled truth, universe = 30 target packages).
usage: m35_eval.py <pred_prefix> <partition> [--split=...] [--proto=SRC,floor,margin,samekeys] [--per-app] [--json=out]
 Predictions: data/predictions/<prefix>_<app>.json; a prediction's set = its "version_set" (Go --vset-margin) or {version}.
 --proto: Python prototype instead: read runs/<SRC>/<app>.vset.json (all per-version selector scores), apply the floor and
          build sets (score >= chosen - margin, or identical matched-key set if samekeys=1). Used to check the Go flags.
Metrics (per partition, summed over apps):
 strict     : pkg+exact version P/R/F1 with byte-identical-bundle versions as one release (= oneclass.py --subsha)
 set        : same counting but a prediction hits if any truth version is in its set (same-SHA aware); set_hit_rate =
              hits / truth packages; mean_set = mean set size over predicted packages
 adv        : true advisories = OSV/GHSA advisories (data/advisories_osv.json, withdrawn excluded) affecting any bundled truth
              version of a universe package; predicted = advisories affecting any version in a predicted set; P/R/F1 over
              (app, advisory) pairs. adv_strict uses only the chosen version.
 app        : per app 'has any vulnerable universe dep' (truth vs predicted sets) accuracy."""
import sys, json, re, collections
E_ = "/workspace/hedis-eval"; P, part = sys.argv[1], sys.argv[2]
arg = lambda k, d=None: next((a.split("=", 1)[1] for a in sys.argv if a.startswith(f"--{k}=")), d)
UNI = set(json.load(open(f"{E_}/data/universe_targets.json"))); split = json.load(open(arg("split", f"{E_}/data/split.json")))
ADV = json.load(open(f"{E_}/data/advisories_osv.json"))["packages"]
sha = {}
for f in ("fp-rel-stats.jsonl", "fp-sub-stats.jsonl", "fp-sub3-stats.jsonl"):
    for l in open(f"{E_}/logs/{f}"):
        d = json.loads(l); k = (d["Name"], d["Version"], d["RN"])
        if f.startswith("fp-rel"):
            if d["ok"] and d.get("BundleSHA"): sha[k] = (d["BundleSHA"], "")
        elif d.get("skipped") != "already-in-db" and k in sha: sha[k] = (sha[k][0], d.get("BundleSHA", "") if d["ok"] else "")
def vkey(v):
    m = re.match(r"^v?(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:-([0-9A-Za-z.-]+))?", v.strip())
    if not m: return ((0, 0, 0), (0,))
    pre = m.group(4)
    pk = (1,) if pre is None else (0,) + tuple((0, int(x), "") if x.isdigit() else (1, 0, x) for x in pre.split("."))
    return ((int(m.group(1)), int(m.group(2) or 0), int(m.group(3) or 0)), pk)
def affected(n, v):
    out = set(); kv = vkey(v)
    for a in ADV.get(n, []):
        if a.get("withdrawn"): continue
        hit = False
        for af in a["affected"]:
            if v in af.get("versions", []): hit = True
            for r in af.get("ranges", []):
                if r.get("type") not in ("SEMVER", "ECOSYSTEM"): continue
                intro = None
                for e in r["events"]:  # OSV: sorted events; affected if introduced <= v < fixed (or <= last_affected)
                    if "introduced" in e: intro = e["introduced"]
                    elif intro is not None and ("fixed" in e or "last_affected" in e or "limit" in e):
                        lo = intro == "0" or vkey(intro) <= kv
                        if "fixed" in e: up = kv < vkey(e["fixed"])
                        elif "limit" in e: up = kv < vkey(e["limit"])
                        else: up = kv <= vkey(e["last_affected"])
                        if lo and up: hit = True
                        intro = None
                if intro is not None and (intro == "0" or vkey(intro) <= kv): hit = True
        if hit: out.add(a["id"])
    return out
proto = arg("proto")
if proto:
    SRC, pfl, pm, psk = proto.split(","); pfl, pm, psk = float(pfl), float(pm), psk == "1"
def preds(a):
    if not proto:
        return {x["package"]: (x["version"], x.get("version_set") or [x["version"]]) for x in json.load(open(f"{E_}/data/predictions/{P}_{a}.json"))["predictions"] if x["package"] in UNI}
    out = {}
    for r in json.load(open(f"{E_}/runs/{SRC}/{a}.vset.json")) or []:
        if r["package"] not in UNI or r["score"] < pfl: continue
        c = next((v for v in r["versions"] if v["v"] == r["version"]), None); s = {r["version"]}
        if c:
            for v in r["versions"]:
                if v["sc"] >= c["sc"] - pm - 1e-12 or (psk and v["hit"] == c["hit"]): s.add(v["v"])
        out[r["package"]] = (r["version"], sorted(s))
    return out
C = collections.Counter(); sizes = []; per = []
for a in [x for x in split[part] if x.startswith("synth") or part.startswith("real")]:
    rn = ("rn074" if split[part][a]["rn"].startswith("0.74") else "rn076") if part.startswith("real") else ("rn074" if "rn074" in a else "rn076")
    T = {n: vs for n, vs in json.load(open(f"{E_}/data/groundtruth/{a}.json"))["bundled"].items() if n in UNI}
    Pr = preds(a); same = lambda n, v, w: v == w or (sha.get((n, v, rn)) and sha.get((n, v, rn)) == sha.get((n, w, rn)))
    for mode in ("strict", "set"):
        for n, vs in T.items():
            cand = [Pr[n][0]] if (n in Pr and mode == "strict") else (Pr[n][1] if n in Pr else [])
            hit = [v for v in vs if any(same(n, v, w) for w in cand)]
            if hit: C[mode + "_tp"] += 1; C[mode + "_fn"] += len(vs) - 1
            else:
                C[mode + "_fn"] += len(vs)
                if n in Pr: C[mode + "_fp"] += 1
        C[mode + "_fp"] += sum(1 for n in Pr if n not in T)
    C["truth_pkgs"] += len(T); C["set_hits"] += sum(1 for n, vs in T.items() if n in Pr and any(same(n, v, w) for v in vs for w in Pr[n][1]))
    sizes += [len(s) for _, s in Pr.values()]
    TA = set().union(*[affected(n, v) for n, vs in T.items() for v in vs]) if T else set()
    for mode, PA in (("adv", set().union(*[affected(n, v) for n, (_, s) in Pr.items() for v in s]) if Pr else set()),
                     ("advstrict", set().union(*[affected(n, v) for n, (v, _) in Pr.items()]) if Pr else set())):
        C[mode + "_tp"] += len(TA & PA); C[mode + "_fp"] += len(PA - TA); C[mode + "_fn"] += len(TA - PA)
        if mode == "adv": C["app_n"] += 1; C["app_ok"] += (bool(TA) == bool(PA)); C["app_vuln_true"] += bool(TA); per.append((a, len(TA), len(PA), len(TA & PA), sorted(TA - PA), sorted(PA - TA)))
        else: C["appstrict_ok"] += (bool(TA) == bool(PA))
def prf(m):
    tp, fp, fn = C[m + "_tp"], C[m + "_fp"], C[m + "_fn"]; p = tp / (tp + fp) if tp + fp else 0; r = tp / (tp + fn) if tp + fn else 0
    return {"P": round(p, 3), "R": round(r, 3), "F1": round(2 * p * r / (p + r) if p + r else 0, 3), "tp": tp, "fp": fp, "fn": fn}
res = {"prefix": P if not proto else "proto:" + proto, "partition": part, "strict": prf("strict"), "set": prf("set"),
       "set_hit_rate": round(C["set_hits"] / C["truth_pkgs"], 3), "mean_set": round(sum(sizes) / len(sizes), 2) if sizes else 0,
       "adv": prf("adv"), "adv_strict": prf("advstrict"), "app_acc": f'{C["app_ok"]}/{C["app_n"]}', "app_acc_strict": f'{C["appstrict_ok"]}/{C["app_n"]}', "apps_vulnerable_truth": C["app_vuln_true"]}
print(json.dumps(res))
if "--per-app" in sys.argv:
    for x in per: print("  ", x)
if arg("json"): json.dump(res, open(arg("json"), "w"), indent=1)
