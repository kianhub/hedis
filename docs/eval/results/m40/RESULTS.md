# M40: version selection (2026-10-16, CT)

Run settings: effort high, model 'Grok Bot, default model'. Dev only: real_dev (8), synthetic dev (15), RN 0.85 dev =
blink-beta, breathly, zeus + the 7 v8 apps (marlin, imagetopdf, dothething, minmobile, wixrnn, readtrace, ysclaude).
Tuned without a DB pin. The DB had no rows after the M39 cutoff, so the results equal the pinned results. research/v9 was not read.

## Method (final)
1. bin/hedis-m40 = M39 matcher + `--floors data/floors_m40.json` (per-package floor; react-native-gesture-handler 0.02).
   Diagnosis: on 0.85 apps, GH's best unique-key fraction is 0.000 when GH is absent and 0.025-0.83 when present, so the global 0.2 floor dropped it.
2. eval/m40/vsel.py, version-diff Copeland for safe-area-context, screens and reanimated: the nodes are the distinct root nsnc key sets
   of a package's stable versions at the app's RN. Version v beats w when |A∩(Kv\Kw)| - |A∩(Kw\Kv)| >= 2. Only keys that differ between versions are compared.
   When the M39 pick is outside the best tie set, the tied version nearest to it is chosen.
Driver: eval/run_m40.sh m40f <apps> (matcher -> m40f_raw -> vsel -> m40f).

## Results (oneclass --subsha), F1 (tp/fp/fn)
| dev set | M39 | M40 raw (floors only) | **M40** |
|---|---|---|---|
| real_dev | 0.840 (105/20/20) | 0.845 (106/20/19) | **0.845 (106/20/19)** |
| synthetic dev | 0.970 (144/4/5) | 0.970 | **0.970 (144/4/5)** |
| 0.85 dev: blink/breathly/zeus | 0.885 (27/4/3) | 0.885 | **0.918 (28/3/2)** |
| 0.85 dev: 7 v8 apps | 0.373 (14/25/22) | 0.359 (14/28/22) | **0.564 (22/20/14)** |
| all 10 0.85 dev | 0.603 (41/29/25) | 0.590 | **0.719 (50/23/16)** |
No dev set is worse than M39. Unsupported flags: 0/33.

## Tried and rejected
- Hill-climb over neighbouring versions (vdiff hop): 0.85-10 F1 0.691. Copeland is better (0.735 before floors).
- Dependency-version (release-time) consistency: the newest tied version published on or before an anchor date taken from
  confidently resolved packages (npm publish times). It helps real_dev (0.856 at max/conf2) and 0.85 (0.779), but **synthetic drops to 0.956**
  (synthetic apps pin random versions, so release time is no signal there). Rejected under the no-worse rule.
- vsel also on gesture-handler: real_dev 0.797. vsel on all packages: real_dev 0.760. Both rejected.
- Remaining errors: safe-area-context 4.7.2-5.10.1 have nearly identical JS (43 versions in one tie set), and screens 4.25.0-4.26.2 tie. GH on marlin/ysclaude is now
  detected but at 2.33.0 (truth 2.31.2), which turns an FN into an FP+FN.
go test ./... passes.
