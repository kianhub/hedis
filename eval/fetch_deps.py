#!/usr/bin/env python3
"""data/deps.json: {pkg: {version: [dependency names]}} for every target package version (npm registry docs)."""
import json, urllib.request, urllib.parse
E_ = "/workspace/hedis-eval"; T = json.load(open(f"{E_}/db/targets_ext.json"))["packages"]; out = {}
for n in T:
    d = json.load(urllib.request.urlopen("https://registry.npmjs.org/" + urllib.parse.quote(n, safe="@")))
    out[n] = {v: sorted((m.get("dependencies") or {}).keys()) for v, m in d["versions"].items()}
json.dump(out, open(f"{E_}/data/deps.json", "w"))
print({n: len(v) for n, v in out.items()})
