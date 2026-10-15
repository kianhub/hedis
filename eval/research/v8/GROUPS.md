# v8 GROUPS (frozen before any matcher run)

split: `data/split.v8.json` / `eval/research/v8/split.v8.json` sha256 `b81c8e1e909fc676081dd9f85bfec79b970ccb5eacc8a1e1819d97858b443946`.

**No matcher run. No scoring.** Wait for Optimizer’s RN 0.85 cutoff.

| app | RN (detected) | HBC | group | dump binary | notes |
|---|---|---|---|---|---|
| dothething-1.2.1 | 0.85.3 | 98 | rn085 | hedis-m37 |  |
| imagetopdf-1.1 | 0.85.3 | 98 | rn085 | hedis-m37 | entry shim expo-router/entry; hermesc PRE |
| marlin-1.3.0 | 0.85.3 | 98 | rn085 | hedis-m37 | entry shim expo-router/entry; hermesc PRE |
| minmobile-1.0.62 | 0.85.3 | 98 | rn085 | hedis-m37 | package.json rn ^0.85.3; detected 0.85.3 |
| readtrace-1.0.6 | 0.85.3 | 98 | rn085 | hedis-m37 |  |
| wixrnn-8.8.7 | 0.85.2 | 98 | rn085 | hedis-m37 | wix react-native-navigation playground |
| ysclaude-1.0.9 | 0.85.3 | 98 | rn085 | hedis-m37 |  |
| appcues-1.0.12 | 0.74.5 | 96 | control | hedis-norm-hits | entry shim expo-router/entry |
| gorhom-5.1.4 | 0.76.0 | 96 | control | hedis-norm-hits | retry1: example npm ci (root yarn skipped example lock); entry shim expo/AppEntry; first attempt dropped |
| solana-3.0.0 | 0.76.9 | 96 | control | hedis-norm-hits | entry shim expo/AppEntry; solana-mobile expo template |

Sizes: rn085 **7** / ≥5 ✓, control **3** / ≥3 ✓. Total 10.

0.85 DEV only (not held-out): blink-beta-3.0.31, breathly-ios-2.2-18, zeus-13.1.3.

Success criteria (pre-registered): rn085 primary = exact F1 vs t35 + unsupported-flag rate; controls = exact F1 vs M38/t35; score once after Optimizer posts 0.85 cutoff.
