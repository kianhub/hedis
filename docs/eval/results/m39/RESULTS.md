# M39: React Native 0.85 / HBC98 fingerprints (2026-10-15, CT)

Run settings: effort high, model 'Grok Bot, default model'. Dev only, no tuning, M38 frozen flags unchanged.

## Build
- Targets: 30 curated packages x all 2210 versions in hedis_rel at the M38 cutoff (docs/eval/data/targets_m39.json).
- Env envs/rn085: RN 0.85.3, react 19.2.3, hermesc from hermes-compiler 250829098.0.10 (HBC 98). Binary bin/hedis-m39 (this branch).
- The first pass lost about 800 builds because npm installs in the work envs removed the hermesc symlink. Fixed in fastpipe (effabf8,
  which falls back to the hermesc in hermes-compiler), then resumed. The stats files here are append logs; the last row per pkg@ver counts.
- Root: **1802/2210 ok**. Failures: better-auth 224 (Metro fails at every RN), zod 58, @sentry/react-native 50, parse 25,
  react-native-webview 23, react-native-svg 14, flash-list 5, axios 4, and a few others.
- Subpath: 409 built ok, 1729 have no subpaths, 72 failed.
- New 0.85 baseline (HBC 98). **Cutoff HEDIS_MAX_OID=6ad180000000000000000000** (last write 6ad17ff9..., 2026-10-15 20:38 CT).

## Scores (eval/run_m39.sh m39f; eval/oneclass.py --subsha; split.v3dev)
| set | M38 (m38f / m38f85) | M39 (m39f) | unsupported flags M38 -> M39 |
|---|---|---|---|
| real_dev | 0.840 (105/20/20) | 0.840 (105/20/20) | 0/8 -> 0/8 |
| synthetic dev | 0.970 (144/4/5) | 0.970 (144/4/5) | 0/15 -> 0/15 |
| RN 0.85 dev (blink-beta, breathly, zeus) | 0.000 (0/0/30) | **0.885** P 0.871 R 0.900 (27/4/3) | 3/3 -> 0/3 |

On all 23 real_dev and synthetic apps, per-app predictions are identical to m38f. 0.85 errors: false positive react-native-mmkv 4.0.1 (blink-beta, not bundled);
breathly has the wrong version for react-freeze, react-native-safe-area-context and react-native-screens.
go test ./... passes. No held-out sets (v6, v7, zeus-as-test, v8) were opened or scored.
