# v9 held-out freeze — RESULT (scored once, 2026-10-16 ~19:10 CT)

- Freeze: research/v9-freeze 4ee41d9 + amendment 6b8fec1. split.v9.json sha256 `a32b9b14f1c05dba7515355554904b407ee54fddefecd8ecabfa9e59f173125c` (verified OK before scoring). All 9 triples, gt.json and hbc sha256 verified OK against split.v9.json in score_v9.sh.
- M40: v2 @ 81d5493 (contains M40 merge 2e928c8). Built in a detached worktree (`go build` in go/hermes-decompiler) → bin/hedis-m40v9 sha256 `3c8d03cfc19058f6626851b8d4cad90e948845be652d13c015c34fe6beb0c1c8`. (This differs from the existing bin/hedis-m40, sha256 88245dbe…f2ec5ca; that binary was not used.) Flags are per m40_frozen.json: M39 flags + `--floors data/floors_m40.json` (sha256 a3fd0d3c…) `--unsupported-cov 0.40`, followed by eval/m40/vsel.py @81d5493 `--min=2 --date=off --pkgs=safe-area-context,screens,reanimated`. DB hedis_rel, HEDIS_MAX_OID=6ad2b1a20000000000000000, HEDIS_RN_FALLBACK=nearest.
- t35: bin/hedis-vset sha256 `2c863251…9685e4` + /workspace/hedis/eval/thesis_vset.py on frozen/<app>.hbc, same pin (t35_v9.sh, a mirror of t35_v8.sh).
- M39 reference: bin/hedis-m39 sha256 `8b475e7c…2ee297`, v8 flags (incl. --unsupported-cov 0.40), HEDIS_MAX_OID=6ad180000000000000000000 (m39_v9.sh). It was run on all 9 apps; only the controls are part of the criteria.
- Scorer: scripts/oneclass.py --subsha --split=data/split.v9.json, always with `--apps=` (RN taken from the triples header). Without `--apps`, oneclass maps every non-0.74 real app to rn076, which mis-scores 0.85 apps (it turns fast-opencv screens 4.26.2 into an FN). Those first outputs were discarded and are not used anywhere here.

## Headline (exact pkg+version, --subsha)
| set | method | P | R | F1 | tp | fp | fn |
|---|---|---|---|---|---|---|---|
| **rn085 (5)** | **M40** | 0.786 | 1.000 | **0.880** | 11 | 3 | 0 |
| **rn085 (5)** | **t35** | 0.000 | 0.000 | **0.000** | 0 | 0 | 11 |
| rn085 (5) | M39 (ref) | 0.786 | 1.000 | 0.880 | 11 | 3 | 0 |
| rn085 excl. secure-webview (4) | M40 | 0.786 | 1.000 | 0.880 | 11 | 3 | 0 |
| rn085 excl. secure-webview (4) | t35 | 0.000 | 0.000 | 0.000 | 0 | 0 | 11 |
| all 9 | M40 | 0.674 | 0.879 | 0.763 | 29 | 14 | 4 |
| all 9 | t35 | 0.078 | 0.273 | 0.121 | 9 | 107 | 24 |
| all 9 | M39 | 0.690 | 0.879 | 0.773 | 29 | 13 | 4 |

- **t35 crash rate on rn085: 5/5.** hedis-vset cannot disassemble HBC 98 (e.g. "failed to read exception handler: unexpected EOF"), so it produces empty predictions. **t35 fair (non-crash) rn085 subset: empty, so its F1 is undefined.** The only t35 non-crash apps are the 4 controls: F1 0.130 (tp 9 fp 107 fn 13).
- Excluding secure-webview-0.1.0 changes nothing. Its truth (like system-navigat-3.0.1's) has **no universe-target packages**, and both M40 and M39 predict 0 packages there (tp/fp/fn 0/0/0). The rn085 headline therefore rests on 3 apps: fast-opencv, trading-charts, quick-base64. 11 truth items is a small n.
- **M40 = M39 on rn085.** vsel.py changed no prediction on any v9 app (M40v9_raw == M40v9), and the floors change did not alter any rn085 prediction.

## Per-app (P / R / F1, tp/fp/fn)
| app | group | M40 | t35 | M39 |
|---|---|---|---|---|
| fast-opencv-1.0.1 | rn085 | 0.714/1.000/0.833 (5/2/0) | crash 0.000 (0/0/5) | 0.833 (5/2/0) |
| quick-base64-3.0.1 | rn085 | 0.000/–/0.000 (0/1/0) | crash 0.000 (0/0/0) | 0.000 (0/1/0) |
| secure-webview-0.1.0 | rn085 | no targets (0/0/0) | crash (0/0/0) | (0/0/0) |
| system-navigat-3.0.1 | rn085 | no targets (0/0/0) | crash (0/0/0) | (0/0/0) |
| trading-charts-1.1.0 | rn085 | 1.000/1.000/1.000 (6/0/0) | crash 0.000 (0/0/6) | 1.000 (6/0/0) |
| aemobileapp-1.0.3 | control | 0.875/1.000/0.933 (7/1/0) | 0.069/0.286/0.111 (2/27/5) | 0.933 (7/1/0) |
| engrave-SDDFinalRelease | control | 0.429/0.750/**0.545** (3/4/1) | 0.069/0.500/0.121 (2/27/2) | 0.600 (3/3/1) |
| kaku-0.3.1 | control | 0.429/0.600/0.500 (3/4/2) | 0.103/0.600/0.176 (3/26/2) | 0.500 (3/4/2) |
| waifu-walls-0.1.1 | control | 0.714/0.833/0.769 (5/2/1) | 0.069/0.333/0.114 (2/27/4) | 0.769 (5/2/1) |

**Control drop (reported):** on engrave-SDDFinalRelease, M40 scores F1 0.545 against M39's 0.600. The cause is one new FP, react-native-gesture-handler 2.13.4, which is not bundled. That package is let through by M40's gesture-handler floor of 0.02. The other 3 controls are unchanged. Control aggregate: M40 0.706 (18/11/4), M39 0.720 (18/10/4), t35 0.130 (9/107/13).

## M40 unsupported-flag rate
**0/9** (0/5 rn085). Every run exited rc0, and no "unsupported" appears in any log or output. The 2 empty apps (secure-webview, system-navigat) are "0 packages predicted", not abstains.

## Error breakdown (M40, all 9)
- FN wrong version (4): engrave @react-navigation/native truth 7.0.13 pred 6.1.18; kaku @react-navigation/native 7.1.6→6.1.18; kaku react-native-svg 15.8.0→15.3.0; waifu-walls react-native-gesture-handler 2.20.2→2.17.1. All 4 are in controls; rn085 has none.
- FN miss (package not predicted): 0. FN version not in DB: 0.
- FP not bundled (10): react-native-url-polyfill 2.0.0 ×5 (aemobileapp, engrave, kaku, quick-base64, waifu-walls); semver ×3 (engrave 7.6.0, kaku 7.6.0, fast-opencv 7.3.6); react-native-mmkv 4.0.1 (fast-opencv); react-native-gesture-handler 2.13.4 (engrave, new vs M39).
- The 4 wrong-version FNs are also counted as FPs, giving 14 FPs in total.
- t35: 94 FP not-bundled, 13 FN wrong version, 11 FN missed (all from the rn085 crashes). Full lists are in result/t35v9.all.txt.

## Coverage check: M40-targeted packages (gesture-handler, safe-area-context, screens, reanimated)
Bundled truth:
| app | GH | SAC | screens | reanimated |
|---|---|---|---|---|
| aemobileapp-1.0.3 | 2.17.1 | 4.10.5 | 3.31.1 | 3.10.1 |
| engrave-SDDFinalRelease | – | 4.12.0 | 4.1.0 | – |
| fast-opencv-1.0.1 | – | 5.8.0 | 4.26.0 | 4.5.1 |
| kaku-0.3.1 | – | 4.12.0 | 4.4.0 | – |
| quick-base64 / secure-webview / system-navigat | – | – | – | – |
| trading-charts-1.1.0 | – | 5.8.0 | 4.24.0 | – |
| waifu-walls-0.1.1 | 2.20.2 | 4.12.0 | 4.4.0 | 3.16.7 |

17 truth items in total, 5 of them on rn085 (fast-opencv 3, trading-charts 2). **There is no gesture-handler in any rn085 app**, so M40's GH floor is not tested on 0.85 here.
| method | tp | fp | fn |
|---|---|---|---|
| M40 | 16 | 2 | 1 |
| M39 | 16 | 1 | 1 |
| t35 | 3 | 13 | 14 |
On rn085 specifically, both M40 and M39 get 5/5 TP. Several of these TPs come from --subsha byte-identical releases (e.g. SAC 4.12.0 predicted as 5.4.1, screens 4.4.0 as 4.5.0, screens 4.26.0 as 4.26.2). M40's only change on these 4 packages is the extra GH FP on engrave. Detail: result/target4.txt.

## Files
score_v9.sh (M40), t35_v9.sh, m39_v9.sh, target4.py; result/ (scorer outputs, status); data/predictions/{M40v9,t35v9,M39v9}_<app>.json. Raw runs are in /workspace/hedis-eval/runs/{M40v9,t35v9,M39v9}. Procedure deviation: none in the matcher. t35_v9.sh fixes v8's stdout bookkeeping (thesis_vset stdout now goes straight to the prediction file).
