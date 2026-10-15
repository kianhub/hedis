# M38 unsupported-bytecode coverage threshold

**Date:** 2026-10-15  
**Threshold:** `DefaultUnsupportedMinCov = 0.40` (`--unsupported-cov`)  
**Metric:** max over DB package versions with ≥20 nsnc keys of `|fp_keys ∩ app_keys| / |fp_keys|` for the selected/fallback RN.

## Tuning set (ONLY — no zeus/v6)

| group | apps / dumps | evidence |
|---|---|---|
| healthy | all `real_dev` + all `synth*` in `data/split.v3dev.json` via `runs/m37f/*.txt.json` | predicted max IDF scores: min **0.938**, median **0.998** |
| healthy raw check | bluesky `@sentry/react-native@5.24.3` @ RN 0.76; mattermost-rel `safe-area-context@4.10.5` @ 0.74 | raw nsnc coverage **0.661**, **0.796** |
| unsupported | self-compiled `/workspace/rn085/app.hbc` (`runs/rn085_diag/app.triples`) | axios **0.071**, zod **0.126**, safe-area-context **0.068** (max **0.126**) — same as `hbc98_rn085_diag.md` |

**Excluded from threshold choice:** zeus and every other v6 held-out app.

0.40 sits between unsupported ceiling (~0.13) and healthy raw max (≥0.66). Go tests: `TestMaxPackageNSNCCoverage`, `TestUnsupportedBytecodeError`.
