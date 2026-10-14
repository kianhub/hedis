#!/usr/bin/env python3
"""Primary metric: pkg+version F1 where versions with byte-identical bundles (same Metro bundle SHA per RN in the
release fingerprint run) count as one release; bundled truth; universe = target packages.
Also prints per-category FN/FP diagnosis with --diag.
usage: oneclass.py <pred_prefix> <dev|test|real> [--diag] [--subsha] [--split=data/split.v3.json]"""
import sys, json, collections
E_ = "/workspace/hedis-eval"; P, part = sys.argv[1], sys.argv[2]; diag = "--diag" in sys.argv
UNI = set(json.load(open(f"{E_}/data/universe_targets.json"))); SPL = next((a.split("=",1)[1] for a in sys.argv if a.startswith("--split=")), f"{E_}/data/split.json"); split = json.load(open(SPL))
sha = {}  # release identity = root bundle SHA; --subsha (primary since M24) also the subpath bundle SHA
# M34: fp-rel34/fp-sub34 = ERESOLVE-failed jobs rerun (--force + peer pins) and added real_dev versions (attempt2 rows hold the SHAs of jobs skipped later)
REL = ("fp-rel-stats.jsonl", "fp-rel34-stats.attempt2.jsonl", "fp-rel34-stats.jsonl", "fp-rel079b-stats.jsonl")
SUB = ("fp-sub-stats.jsonl", "fp-sub3-stats.jsonl", "fp-sub34-stats.attempt2.jsonl", "fp-sub34-stats.jsonl", "fp-sub079b-stats.jsonl")
for f in REL + (SUB if "--subsha" in sys.argv else ()):
    try:
        for l in open(f"{E_}/logs/{f}"):
            d = json.loads(l); k = (d["Name"], d["Version"], d["RN"])
            if f.startswith("fp-rel"):
                if d["ok"] and d.get("BundleSHA"): sha[k] = (d["BundleSHA"], "")
            elif d.get("skipped") != "already-in-db" and k in sha:
                # later runs override; an ok row without a bundle is the no-subpath marker ("")
                sha[k] = (sha[k][0], d.get("BundleSHA", "") if d["ok"] else "")
    except FileNotFoundError: pass
indb = {(n, v) for n, v, r in sha}
tp = fp = fn = 0; cat = collections.Counter(); ex = collections.defaultdict(list)
for a in [x for x in split[part] if x.startswith("synth") or part.startswith("real")]:
    rn = ("rn074" if split[part][a]["rn"].startswith("0.74") else "rn076") if part.startswith("real") else ("rn074" if "rn074" in a else "rn076")
    T = {n: vs for n, vs in json.load(open(f"{E_}/data/groundtruth/{a}.json"))["bundled"].items() if n in UNI}
    Pr = {x["package"]: x["version"] for x in json.load(open(f"{E_}/data/predictions/{P}_{a}.json"))["predictions"] if x["package"] in UNI}
    same = lambda n, v, w: v == w or (sha.get((n, v, rn)) and sha.get((n, v, rn)) == sha.get((n, w, rn)))
    for n, vs in T.items():
        if n in Pr and any(same(n, v, Pr[n]) for v in vs):
            tp += 1; fn += len(vs) - 1
            for v in vs:
                if not same(n, v, Pr[n]): cat["FN extra nested version"] += 1
            continue
        for v in vs:
            fn += 1
            c = "FN version not in DB" if (n, v) not in indb else ("FN wrong version" if n in Pr else "FN package missed")
            cat[c] += 1; ex[c].append((a, n, v, Pr.get(n)))
        if n in Pr: fp += 1
    for n in Pr:
        if n not in T: fp += 1; cat["FP package not bundled"] += 1; ex["FP package not bundled"].append((a, n, Pr[n]))
p = tp / (tp + fp) if tp + fp else 0; r = tp / (tp + fn) if tp + fn else 0
print(f"{P} {part} one-class(bundle) P {p:.3f} R {r:.3f} F1 {2*p*r/(p+r) if p+r else 0:.3f} (tp {tp} fp {fp} fn {fn})")
if diag:
    print(dict(cat))
    for c, es in ex.items():
        print(" ", c); [print("    ", e) for e in es[:40]]
