#!/usr/bin/env python3
"""Build 'bundled' truth from Metro source maps: packages with >=1 module under node_modules/<pkg>/ in the bundle.
Version = the version in the app's node_modules/<pkg>/package.json at that path (nested copies -> all versions).
usage: bundled_truth.py <app> <map> <app_dir>   -> adds "bundled" to data/groundtruth/<app>.json"""
import json, sys, os, re
E_ = "/workspace/hedis-eval"; app, mp, d = sys.argv[1:4]
src = json.load(open(mp))["sources"]; pk = {}
for s in src:
    m = re.search(r"^(.*node_modules/((?:@[^/]+/)?[^/]+))/", s)
    if not m: continue
    root, name = m.group(1), m.group(2)
    # Expo maps use project-root-relative paths that start with "/"
    # Expo maps use server-root-relative paths that start with "/"; in a monorepo the server root is an ancestor of the app dir
    if not root.startswith("/") or not os.path.exists(root):
        for up in ("", "..", "../..", "../../.."):
            r2 = os.path.normpath(os.path.join(d, up, root.lstrip("/")))
            if os.path.exists(os.path.join(r2, "package.json")): root = r2; break
        else: root = os.path.join(d, root.lstrip("/"))
    try: v = json.load(open(os.path.join(root, "package.json")))["version"]
    except Exception: continue
    pk.setdefault(name, set()).add(v)
g = json.load(open(f"{E_}/data/groundtruth/{app}.json"))
g["bundled"] = {n: sorted(v) for n, v in sorted(pk.items())}; g["bundled_modules"] = sum(1 for s in src if "node_modules/" in s)
json.dump(g, open(f"{E_}/data/groundtruth/{app}.json", "w"), indent=1)
print(app, len(pk), "bundled packages")
