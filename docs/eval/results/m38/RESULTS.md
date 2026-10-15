# M38 results — frozen-flag rescore after missing-version fill (2026-10-15 16:35 CT)

- Fill: 690 pkg@ver (non-leaky rule: 30 packages already in DB x stable npm X.Y.Z published 2022-01-01..2026-10-01 UTC
  not in DB) x RN 0.72/0.75/0.78/0.80 via eval/fp_m38.sh; root ok 674/651/651/647 of 690.
- DB pin: HEDIS_MAX_OID=6ad13ed00000000000000000 (21:00 UTC / 16:00 CT; last hash 6ad13c3b33cbac53b0656c01 at 15:48 CT).
  hashes 34,300,538 -> 43,376,040; packages 1,520 -> 2,210; fp_done 19,236 -> 24,619.
- Matcher: bin/hedis-m38 (7db3c19, go build -buildvcs=false, sha256 cc3f2b3e39fc12de16e088dd578b6e5506c8528fe8441ca8b08c729ea269877c),
  frozen M37 flags + --unsupported-cov 0.40, HEDIS_RN_FALLBACK=nearest.
- Driver: eval/run_m38.sh m38f; metric eval/oneclass.py --subsha --split=data/split.v3dev.json (now also reads
  fp-rel38/fp-sub38 stats; m37f scores unchanged by this).

| partition | P | R | F1 | tp | fp | fn | dF1 vs M37 |
|---|---|---|---|---|---|---|---|
| real_dev | 0.840 | 0.840 | 0.840 | 105 | 20 | 20 | 0.000 |
| dev synth | 0.973 | 0.966 | 0.970 | 144 | 4 | 5 | 0.000 |

- Predictions identical to m37f on all 23 apps. Dev apps are RN 0.74/0.76 (exact rows, no fallback) and real_dev has
  no "version not in DB" FNs, so the 0.72/0.75/0.78/0.80 fill cannot move these partitions; its target is
  fallback-RN apps (held-out, not scored here).
- unsupported_bytecode: none (all 23 apps exit 0).
- Go tests: go test ./... OK (TestUnsupportedBytecodeError, TestMaxPackageNSNCCoverage PASS).
- Run settings: effort high, model Grok Bot, default model. No CI. v6/zeus untouched.
