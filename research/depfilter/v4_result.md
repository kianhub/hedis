# depfilter SECONDARY (with manifest) on split.v4 real_test: single scoring run (2026-10-14 ~16:30 CT)
Config, frozen in FROZEN.md @ af2546d: `DF_FLOOR=0.15 python3 depfilter.py <src> <dst> score 0.7 root+reg <apps>`
Checks run before scoring:
- data/split.v4.json sha256 168a81f84e2c2ec293de21ef952d23fdfe30267a32782edb3e5d1fcf9ac78626 matches split.v4.sha256 (M32).
- depfilter.py sha256 5290c1c29835a8ac4d79a80f932544b0a3585ccca91f22bfeaa312e484c4a440 matches FROZEN.md and the af2546d blob.
Input: the unfiltered go_m23 v4 predictions data/predictions/h32_<app>.json, the same files Optimizer scored in M33 (sha256s in v4src.sha256).
Scorer: score.py, which is oneclass.py with only the prediction path changed. Run as `real_test --subsha --split=data/split.v4.json`.

| variant | P / R / F1 | tp/fp/fn |
|---|---|---|
| go_m23 unfiltered (h32, Optimizer M33) | .436 / .607 / 0.508 | 82/106/53 |
| floor 0.2 (h32floor, Optimizer M33) | .672 / .593 / 0.630 | 80/39/55 |
| **depfilter SECONDARY root+reg, floor .15, upper .7** | **.698 / .548 / 0.614** | **74/32/61** |

Per app, depfilter SECONDARY (floor-0.2 F1 from M33 in brackets):
alby-go .526 (5/3/6) [.571] · karakeep .588 (5/2/5) [.600] · trezor .643 (9/4/6) [.733] · expensify .828 (12/2/3) [.828]
rabby .627 (16/5/14) [.627] · edge .759 (11/3/4) [.733] · eigen .400 (7/7/14) [.368] · chatwoot .545 (9/6/9) [.629]

Packages in the truth with score >= 0.2 that this filter dropped (the floor-0.2 baseline keeps them): react-freeze ×5
(alby, karakeep, trezor, eigen, chatwoot; it is transitive via react-native-screens, which is not a root dep), @react-navigation/native 6.1.18
(alby, karakeep; wrong version anyway), zod 3.24.4 (trezor), axios 0.28.1 (edge), immer 9.0.6 (eigen), react-native-url-polyfill (chatwoot).
Diag: FN package missed 14 (floor 0.2: 6), FP not bundled 2 (floor 0.2: 4). Full output is in v4_score.txt and v4_per_app.txt.

Verdict: on held-out apps the with-manifest filter is worse than the plain 0.2 floor (0.614 vs 0.630). It gains precision (+.026)
but loses more recall (-.045), because a band of 0.2–0.7 scores holds real transitive deps that are neither root deps nor reachable through the sparse
deps.json edges. The real_dev gain (+0.003) did not generalise. The manifest-free floor 0.2 stays the headline.

Deviations:
1. Harness: the frozen depfilter.py has a hard-coded APPDIR with only the 8 v3 apps. For unknown apps it would look in apps/synth/<app>, which does not exist,
   and get an empty root set. run_v4.py imports the unchanged frozen module and only adds the 8 v4 app dirs (monorepo app subdirs:
   karakeep apps/mobile, trezor suite-native/app, rabby apps/mobile, as with leather in v3), then runs its main block. The filter logic is unchanged.
2. Every v4 app has a package.json, so no root fallback was needed. alby, karakeep, trezor and expensify have no node_modules on the box,
   but root+reg does not use node_modules.
3. Per-app numbers come from re-scoring the same single filter output one app at a time. The filter was not re-run and no parameters changed.
