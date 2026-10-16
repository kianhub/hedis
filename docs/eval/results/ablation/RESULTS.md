# Per-change ablation, t35 -> M39 (2026-10-16, CT)

Run settings: effort high, model 'Grok Bot, default model'. Dev only. DB pin HEDIS_MAX_OID=6ad180000000000000000000 (M39 cutoff), HEDIS_RN_FALLBACK=nearest,
input = dumped triples (runs/apps_rel), metric eval/oneclass.py --subsha (split.v3dev). Driver eval/run_abl.sh (cumulative steps, done markers),
scores eval/abl_score.sh -> scores.tsv. Binary bin/hedis-abl (this branch). It adds a thesis exact-3-hash matcher on --from-dump, plus eval-only switches:
HEDIS_NO_SUB (root rows only), HEDIS_ABL_PLAIN (no IDF and no unique-key filter), and HEDIS_EXCLUDE_RN=0.85 (RN 0.85 rows hidden until s10).

| step (cumulative) | real_dev F1 (tp/fp/fn) | synthetic F1 | RN 0.85 dev x3 F1 | 0.85 unsupported |
|---|---|---|---|---|
| s00 t35 thesis exact-3-hash + vset | 0.312 (55/172/70) | 0.345 (100/330/49) | 0.035 (2/83/28) | 0/3 |
| s01 + normalized minified fingerprints (nsnc) | 0.591 (104/123/21) | 0.484 (140/290/9) | 0.070 (4/81/26) | 0/3 |
| s02 + rarity-weighted unique matching + thresholds | 0.632 (98/87/27) | 0.899 (125/4/24) | 0.086 (3/37/27) | 0/3 |
| s03 + lowuniq 2 / lowdf 2 | 0.667 (105/85/20) | 0.956 (140/4/9) | 0.086 | 0/3 |
| s04 + dep inference 0.6 | 0.671 (106/85/19) | 0.970 (144/4/5) | 0.086 | 0/3 |
| s05 + rootver | 0.671 (106/85/19) | 0.970 | 0.086 | 0/3 |
| s06 + floor 0.2 | 0.837 (105/21/20) | 0.970 | 0.000 (0/0/30) | 0/3 |
| s07 + vset-margin 0.003 / samekeys | 0.837 | 0.970 | 0.000 | 0/3 |
| s08 + subpath fingerprints | 0.840 (105/20/20) | 0.970 | 0.000 | 0/3 |
| s09 + unsupported-cov 0.40 | 0.840 | 0.970 | 0.000 (refused) | 3/3 |
| s10 + RN 0.85 fingerprints (= M39) | 0.840 (105/20/20) | 0.970 (144/4/5) | 0.885 (27/4/3) | 0/3 |

Notes: s10 reproduces the frozen M39 numbers exactly. Before s10 the 0.85 apps fall back to RN 0.80 fingerprints; the floor removes the
spurious low-score matches (0.086 -> 0), and unsupported-cov then refuses all three apps. rootver and vset do not change exact F1
(vset changes only the reported version set). Where no package passed, to_predictions.py wrote nothing; eval/m40 fix_empty.py
records those apps as empty predictions. t35 was run from the dumped triples (the .hbc files are lost) with the same RN baseline filtering as the dump.
