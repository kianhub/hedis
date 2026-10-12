#!/usr/bin/env python3
"""Build synthetic Hermes apps from the repo's RN templates with known package versions.
Each app: copy of pipeline template rnXXX, `npm install` of K packages at pinned versions (saved, so the
lockfile is the label source), an index.js that imports and *uses* every package, release Metro bundle
(dev=false, minify=true, like a store build), then hermesc -O. Labels -> data/groundtruth/<app>.json.
Deterministic: seeded RNG; versions drawn from db/targets.json (window + shipped)."""
import json, os, random, shutil, subprocess, sys, hashlib
ROOT = "/workspace/hedis-eval"; T = json.load(open(f"{ROOT}/db/targets.json"))["packages"]
# packages that bundle standalone (from fp-stats ok rows) per RN
ok = {}
for l in open(f"{ROOT}/logs/fp-stats.jsonl"):
    x = json.loads(l)
    if x["ok"]: ok.setdefault(x["RN"], set()).add((x["Name"], x["Version"]))
SPECS = int(sys.argv[1]) if len(sys.argv) > 1 else 8
START = int(sys.argv[2]) if len(sys.argv) > 2 else 0
def sh(cmd, cwd):
    r = subprocess.run(cmd, cwd=cwd, shell=True, capture_output=True, text=True)
    if r.returncode: raise RuntimeError(f"{cmd}: {r.stdout[-800:]}{r.stderr[-800:]}")
    return r.stdout
os.makedirs(f"{ROOT}/apps/synth", exist_ok=True); os.makedirs(f"{ROOT}/apps/out", exist_ok=True)
results = []
for i in range(START, START + SPECS):
    rng = random.Random(1000 + i)
    rn = ["rn074", "rn076"][i % 2]
    names = sorted({n for n, _ in ok.get(rn, [])})
    k = rng.randint(6, 12)
    picks = {}
    for n in rng.sample(names, min(k, len(names))):
        vs = sorted(v for m, v in ok[rn] if m == n)
        picks[n] = rng.choice(vs)
    app = f"synth-{i:02d}-{rn}"
    d = f"{ROOT}/apps/synth/{app}"
    try:
        if not os.path.exists(f"{d}/node_modules"):
            shutil.rmtree(d, ignore_errors=True)
            shutil.copytree(f"{ROOT}/envs/{rn}", d, symlinks=True, ignore=shutil.ignore_patterns("*.hbc", "*_out_*", "package_entry*", "bundle-*"))
        sh("npm install --no-audit --no-fund --ignore-scripts --save-exact " + " ".join(f"{n}@{v}" for n, v in picks.items()), d)
        imports, uses = [], []
        for j, n in enumerate(sorted(picks)):
            imports.append(f"import * as P{j} from '{n}';")
            uses.append(f"P{j}")
        open(f"{d}/index.js", "w").write("""import {AppRegistry, Text} from 'react-native';
import React from 'react';
%s
const deps = [%s];
function App() { return React.createElement(Text, null, 'synthetic ' + deps.map(d => Object.keys(d).length).join(',')); }
AppRegistry.registerComponent('%s', () => App);
""" % ("\n".join(imports), ", ".join(uses), app))
        sh(f"npx react-native bundle --platform ios --dev false --minify true --entry-file index.js --bundle-output {ROOT}/apps/out/{app}.js --sourcemap-output {ROOT}/apps/out/{app}.js.map --reset-cache", d)
        sh(f"node_modules/react-native/sdks/hermesc/linux64-bin/hermesc -O -emit-binary -out {ROOT}/apps/out/{app}.hbc {ROOT}/apps/out/{app}.js -w", d)
        pkg = json.load(open(f"{d}/package.json"))
        pkg["dependencies"] = {n: v for n, v in pkg["dependencies"].items() if n in picks or n in ("react", "react-native")}
        json.dump(pkg, open(f"{d}/package.labels.json", "w"))
        sh(f"python3 {ROOT}/scripts/build_groundtruth.py --app {app} --package-json package.labels.json --lock package-lock.json --rn-version {rn} -o {ROOT}/data/groundtruth/{app}.json", d)
        results.append({"app": app, "rn": rn, "picks": picks, "ok": True})
        print("OK", app, picks, flush=True)
    except Exception as e:
        results.append({"app": app, "rn": rn, "picks": picks, "ok": False, "err": str(e)[-500:]})
        print("FAIL", app, str(e)[-500:], flush=True)
json.dump(results, open(f"{ROOT}/apps/synth/manifest-{START}.json", "w"), indent=1)
