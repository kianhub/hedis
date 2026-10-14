# depfilter (dependency-graph precision filter), 2026-10-14
Eval on split.v3dev.json real_dev (all 8 v3 real apps, revised 15:05 CT), scored with score.py (oneclass.py copy, --subsha, reads <dir>/<app>.json).
Input: go_m23v3 predictions copied to base/. Filter: depfilter.py <src> <dst> <metric score|matched_fos|frac> <thr> <mode none|root|clos|reg|combos>.
Baseline 0.632 (98/87/27). Best overall: score-only thr 0.2 = 0.795 (97/22/28). Best dep: root+reg thr 0.7 = 0.789 (97/24/28).
Synth dev: root+reg@0.7 unchanged (0.970); none@0.7 drops to 0.915.
Caveat: inaturalist/valora have no node_modules, so clos falls back to root for them.
