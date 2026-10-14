# depfilter FROZEN configs (2026-10-14 ~15:05 CT). Not yet scored on split.v4.
Code: depfilter.py sha256 5290c1c29835a8ac4d79a80f932544b0a3585ccca91f22bfeaa312e484c4a440 ; scorer score.py sha256 d9a3e0aef06043e50e8f0f40bc968f20a1ca36852cf63839a621f0c94c01b3a7 (oneclass.py copy, --subsha, preds from <dir>/<app>.json)
Tuning split: data/split.v3dev.json sha256 100cf8b8d68bc63443871c76624133818ed5f800039886b3c494e2d61e7c4531 (real_dev = all 8 v3 real apps; synth dev). data/deps.json sha256 0c647d3ffdddec9ea0ba1bf50bcd8aec65941f9a1fe70d44dde7e0c444c92863
Input predictions: frozen go_m23v3 (real) / go_m23 (synth) matcher output; filter is post-hoc.

## (1) HEADLINE: manifest-free (works from .hbc alone)
Rule: drop prediction if score < 0.20. No root/clos/reg.
Command: DF_FLOOR=-1 python3 depfilter.py <src> <dst> score 0.2 none <apps...>
real_dev: P .815 R .776 F1 0.795 (tp97 fp22 fn28) | synth dev: F1 0.970 (tp144 fp4 fn5), unchanged from baseline
Manifest-free two-stage (floor f + reg edges from deps.json between f and upper u) was tried and lost:
f in {0.1,0.15,0.2}, u=0.5: F1 0.786 (92/17/33), synth 0.948; u=0.7: F1 0.736 (81/14/44), synth 0.915. The floor had no effect.

## (2) SECONDARY: with manifest (needs the app's package.json)
Rule: drop if score < 0.15; keep if score >= 0.70; in between keep only if the package is a root package.json prod dep
or a deps.json child of a prediction with score >= 0.70.
Command: DF_FLOOR=0.15 python3 depfilter.py <src> <dst> score 0.7 root+reg <apps...>
real_dev: P .822 R .776 F1 0.798 (tp97 fp21 fn28) | synth dev: F1 0.970 (144/4/5)

Baseline go_m23v3 real_dev: P .530 R .784 F1 0.632 (tp98 fp87 fn27).
