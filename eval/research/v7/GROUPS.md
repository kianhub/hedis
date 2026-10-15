# v7 GROUPS (frozen before any matcher run)

split: data/split.v7.json (copy in this dir). Covered RN minors in hedis_rel at pin 6ad0ea70… (M37) and 6ad13ed0… (M38): **0.72, 0.74, 0.75, 0.76, 0.78, 0.79, 0.80** (identical at both pins; 0.71/0.81/0.83/0.85 not covered).

| app | RN (detected) | HBC | group | dump binary | notes |
|---|---|---|---|---|---|
| jitsi-9.1.0 | 0.72.9 | 94 | fill | hedis-norm-hits |  |
| xaman-2.8.0 | 0.72.7 | 94 | fill | hedis-norm-hits |  |
| daimo-audit-1 | 0.72.4 | 94 | fill | hedis-norm-hits |  |
| gsy-4.0.2 | 0.80.2 | 96 | fill | hedis-norm-hits | gitignored ignoreConfig.js filled with README placeholder |
| safe-1.54.0 | 0.76.3 | 96 | control | hedis-norm-hits | built with entry shim (import expo-router/entry); first attempt dropped, drop reversed |
| jfaudio-2.4.4 | 0.74.3 | 96 | control | hedis-norm-hits |  |
| vega-3.1.1 | 0.74.1 | 96 | control | hedis-norm-hits |  |
| blink-beta-3.0.31 | 0.85.3 | 98 | unsupported_hbc98 | hedis-m37 |  |
| breathly-ios-2.2-18 | 0.85.3 | 98 | unsupported_hbc98 | hedis-m37 |  |
| bbplayer-2.5.2 | — | 98 | unsupported_hbc98 | — (failed) | **dumper failure**: empty detected rnversion, no valid triples; package.json 0.85.3 = group label only (never passed to dump/matcher) |
| yoroi-4.12.0 | 0.71.6 | 90 | fallback | hedis-norm-hits |  |
| readany-1.3.4 | 0.81.5 | 96 | fallback | hedis-norm-hits |  |
| ogam-0.0.49 | 0.83.1 | 96 | fallback | hedis-norm-hits |  |

Sizes: fill 4 (3×0.72 + 1×0.80), control 3 (2×0.74 + 1×0.76), unsupported_hbc98 3 (2 clean dumps + bbplayer dumper failure), fallback 3. Total 13.

Fill limit (locked): fill effect is reported per app; no generalization to RN 0.75/0.78 (no unburned public app found on those minors).

RN 0.85 rule: unsupported_bytecode / unsupported flag = correct abstention; own table; excluded from exact F1.
