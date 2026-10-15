# M38 binary ablation at the M37 cutoff (2026-10-15, CT)

Run settings: effort high, model 'Grok Bot, default model'.
Binary bin/hedis-m38 (sha256 cc3f2b3e...877c), driver eval/run_m38.sh, frozen M38 flags (M37 flags + --unsupported-cov 0.40),
HEDIS_RN_FALLBACK=nearest, **HEDIS_MAX_OID=6ad0ea700000000000000000** (M37 cutoff). Prefix m38a37, 23 dev apps
(8 real_dev + 15 dev synth, split.v3dev). Metric eval/oneclass.py --subsha --split=data/split.v3dev.json.

| partition | P | R | F1 | tp/fp/fn |
|---|---|---|---|---|
| real_dev | 0.840 | 0.840 | 0.840 | 105/20/20 |
| synthetic (dev) | 0.973 | 0.966 | 0.970 | 144/4/5 |

All 23 apps exit 0 (no unsupported_bytecode). Per-app (package, version) prediction sets are **identical to m37f on 23/23 apps**.
So on dev the M38 binary change is a no-op at the M37 DB pin, and so is the M38 version fill (m38f = m37f).
