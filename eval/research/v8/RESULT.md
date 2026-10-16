# v8 held-out freeze — RESULT (scored once, 2026-10-15 ~21:10 CT)

- Freeze commit: cffd75f. Split: data/split.v8.json sha256 b81c8e1e…3946 (matches GROUPS.md).
- Cutoff: HEDIS_MAX_OID=6ad180000000000000000000 (M39). Env: hedis_rel @127.0.0.1:27017, HEDIS_RN_FALLBACK=nearest.
- Matcher: bin/hedis-m39 sha256 `8b475e7c9fc3a64c676827938b95d068506504e3718af4ac94b96669ef2ee297`; flags: --norm nsnc --thresholds data/thresholds_m4f_dlr.json --lowuniq 2 --lowdf 2 --dep 0.6 --deps data/deps.json --rootver --floor 0.2 --vset-margin 0.003 --vset-samekeys --unsupported-cov 0.40
- Baseline t35: bin/hedis-vset sha256 `2c8632519eced124db60be49024294ee5f58a57a486846c658d221336f9685e4` + /workspace/hedis/eval/thesis_vset.py on research/v8/frozen/<app>.hbc, same cutoff.
- Triples: all 10 runs/apps_rel/<app>.triples verified OK vs split.v8.json triples_sha256 before scoring.
- Scorer: scripts/oneclass.py --subsha --split=data/split.v8.json (real_test; subsets via --apps=).

## Headline
| set | method | P | R | F1 | tp | fp | fn |
|---|---|---|---|---|---|---|---|
| all 10 | M39 | 0.458 | 0.519 | 0.486 | 27 | 32 | 25 |
| all 10 | t35 | 0.070 | 0.115 | 0.087 | 6 | 80 | 46 |
| **rn085 (7)** | **M39** | 0.359 | 0.389 | **0.373** | 14 | 25 | 22 |
| **rn085 (7)** | **t35** | 0.000 | 0.000 | **0.000** | 0 | 0 | 36 |

Controls: per-app only (no group headline, per Lab Notebook).

## Per-app (P / R / F1, tp/fp/fn)
| app | group | M39 | t35 |
|---|---|---|---|
| dothething-1.2.1 | rn085 | P 0.250 R 0.333 F1 0.286 (tp 1 fp 3 fn 2) | P 0.000 R 0.000 F1 0.000 (tp 0 fp 0 fn 3) |
| imagetopdf-1.1 | rn085 | P 0.250 R 0.333 F1 0.286 (tp 1 fp 3 fn 2) | P 0.000 R 0.000 F1 0.000 (tp 0 fp 0 fn 3) |
| marlin-1.3.0 | rn085 | P 0.286 R 0.286 F1 0.286 (tp 2 fp 5 fn 5) | P 0.000 R 0.000 F1 0.000 (tp 0 fp 0 fn 7) |
| minmobile-1.0.62 | rn085 | P 0.300 R 0.300 F1 0.300 (tp 3 fp 7 fn 7) | P 0.000 R 0.000 F1 0.000 (tp 0 fp 0 fn 10) |
| readtrace-1.0.6 | rn085 | P 1.000 R 1.000 F1 1.000 (tp 1 fp 0 fn 0) | P 0.000 R 0.000 F1 0.000 (tp 0 fp 0 fn 1) |
| wixrnn-8.8.7 | rn085 | P 0.750 R 1.000 F1 0.857 (tp 3 fp 1 fn 0) | P 0.000 R 0.000 F1 0.000 (tp 0 fp 0 fn 3) |
| ysclaude-1.0.9 | rn085 | P 0.333 R 0.333 F1 0.333 (tp 3 fp 6 fn 6) | P 0.000 R 0.000 F1 0.000 (tp 0 fp 0 fn 9) |
| appcues-1.0.12 | control | P 0.667 R 0.800 F1 0.727 (tp 4 fp 2 fn 1) | P 0.036 R 0.200 F1 0.061 (tp 1 fp 27 fn 4) |
| gorhom-5.1.4 | control | P 0.556 R 0.714 F1 0.625 (tp 5 fp 4 fn 2) | P 0.103 R 0.429 F1 0.167 (tp 3 fp 26 fn 4) |
| solana-3.0.0 | control | P 0.800 R 1.000 F1 0.889 (tp 4 fp 1 fn 0) | P 0.069 R 0.500 F1 0.121 (tp 2 fp 27 fn 2) |

## Unsupported-flag rate (rn085)
- M39: **0/7** unsupported; all 7 rc0 with non-empty predictions (status.txt; no "unsupported" in logs).
- t35: **7/7** failed — hedis-vset cannot disassemble HBC 98 ("failed to read large function header: EOF"), empty predictions (status crash). This is why t35 rn085 F1 = 0.000.

## Failed-fingerprint-build–tagged misses (M39)
Failed-build set (M39 RESULTS): better-auth (all), zod 58, @sentry/react-native 50, parse 25, react-native-webview 23, react-native-svg 14, @shopify/flash-list 5, axios 4, …
- [FAILED-BUILD] minmobile-1.0.62 react-native-svg truth 15.15.4, pred 15.7.1 (wrong version)
- [FAILED-BUILD] ysclaude-1.0.9 react-native-svg truth 15.15.4, pred 12.5.1 (wrong version)
- [FAILED-BUILD] gorhom-5.1.4 @shopify/flash-list truth 1.7.3, pred 1.7.6 (wrong version)
Other 22 FNs are not in the failed-build set: 19 wrong-version FNs (safe-area-context, screens, reanimated, semver, mmkv, @react-navigation/native, gesture-handler) and 3 gesture-handler misses (marlin, minmobile, ysclaude). Full list: result/M39v8.all.txt.

## Notes
- blink-beta-3.0.31 / breathly-ios-2.2-18 / zeus-13.1.3 are RN 0.85 DEV only, NOT in this held-out set. The earlier 0.885 was in-sample, so it is not comparable to these numbers.
- Procedure deviation (bookkeeping only, not a rescore): t35_v8.sh (a literal mirror of t35_v7.sh) sends thesis_vset.py stdout to <app>.tv.log, so the 3 control apps had no prediction file at first. Those tv.log outputs were copied as-is to data/predictions/t35v8_<app>.json, then oneclass was run. The matcher was not re-run.
- Outputs: data/predictions/{M39v8,t35v8}_<app>.json; runs/M39v8/, runs/t35v8/; scorer outputs in research/v8/result/.
