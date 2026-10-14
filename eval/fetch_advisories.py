#!/usr/bin/env python3
"""fetch_advisories.py: npm advisories (OSV.dev public API, which mirrors the GitHub Advisory Database GHSA-* records)
for every universe target package (all packages that can appear in truth-in-universe or predictions).
Writes data/advisories_osv.json {fetched_utc, source, packages: {name: [ {id, aliases, summary, withdrawn, affected: [...]}]}}
and data/advisories_osv.sha256. usage: fetch_advisories.py [extra package names...]"""
import json, sys, hashlib, datetime, urllib.request
E_ = "/workspace/hedis-eval"
pk = sorted(set(json.load(open(f"{E_}/data/universe_targets.json"))) | set(sys.argv[1:]))
def post(url, body):
    r = urllib.request.Request(url, json.dumps(body).encode(), {"Content-Type": "application/json"})
    return json.load(urllib.request.urlopen(r, timeout=60))
out = {}
for n in pk:
    vs, tok = [], None
    while True:
        b = {"package": {"name": n, "ecosystem": "npm"}}
        if tok: b["page_token"] = tok
        d = post("https://api.osv.dev/v1/query", b); vs += d.get("vulns", [])
        tok = d.get("next_page_token")
        if not tok: break
    recs = []
    for v in vs:
        full = json.load(urllib.request.urlopen(f"https://api.osv.dev/v1/vulns/{v['id']}", timeout=60))
        aff = [a for a in full.get("affected", []) if a.get("package", {}).get("ecosystem") == "npm" and a["package"]["name"] == n]
        recs.append({"id": full["id"], "aliases": full.get("aliases", []), "summary": full.get("summary", ""), "withdrawn": full.get("withdrawn"),
                     "modified": full.get("modified"), "affected": [{"ranges": a.get("ranges", []), "versions": a.get("versions", [])} for a in aff]})
    out[n] = sorted(recs, key=lambda r: r["id"]); print(n, len(recs), file=sys.stderr)
doc = {"fetched_utc": datetime.datetime.now(datetime.timezone.utc).isoformat(timespec="seconds"), "source": "https://api.osv.dev/v1 (ecosystem npm; GHSA records from the GitHub Advisory Database)", "packages": out}
b = json.dumps(doc, indent=1, sort_keys=True).encode(); open(f"{E_}/data/advisories_osv.json", "wb").write(b)
h = hashlib.sha256(b).hexdigest(); open(f"{E_}/data/advisories_osv.sha256", "w").write(f"{h}  data/advisories_osv.json\n"); print(h)
