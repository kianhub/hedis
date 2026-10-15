# v6 groups (assigned before scoring)

Pin: `HEDIS_MAX_OID=6ad0ea700000000000000000`.

Query (read-only): `hedis_rel.baselines.distinct("react_native_version", {_id: {$lt: pin}})` → `0.72, 0.74, 0.75, 0.76, 0.78, 0.79, 0.80`. Confirmed each has at least one `hashes` row at the pin.

Rules (Lab Notebook):
- **covered** = app RN minor is in that set
- **zeus** = always its own group (`zeus-13.1.3` only), even though 0.85 has no fingerprints
- **fallback** = everything else

| app | RN | RN minor | group |
|---|---|---|---|
| alephium-1.0.2 | 0.72.3 | 0.72 | covered |
| minds-4.37.0 | 0.71.7 | 0.71 | fallback |
| hexa-2.4.2 | 0.71.13 | 0.71 | fallback |
| zeus-13.1.3 | 0.85.3 | 0.85 | zeus |
| zingo-2.0.24 | 0.81.6 | 0.81 | fallback |
| cozy-1.1.39 | 0.77.3 | 0.77 | fallback |
| enuts-0.3.0 | 0.72.10 | 0.72 | covered |
| keeper-2.5.9 | 0.77.3 | 0.77 | fallback |
| lxmusic-1.0.2 | 0.71.4 | 0.71 | fallback |
| mangareader-0.7.7 | 0.72.10 | 0.72 | covered |
| pixelfed-2.0.0 | 0.76.9 | 0.76 | covered |
| pubky-ring-0.0.16 | 0.78.1 | 0.78 | covered |

Counts: covered=5 fallback=6 zeus=1
