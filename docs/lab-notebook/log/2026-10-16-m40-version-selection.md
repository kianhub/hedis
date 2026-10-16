# 2026-10-16 — M40: version selection (development only)

**Source:** Hedis Optimizer report, 18:59 CT. Merge `2e928c8` on v2. **Cutoff:** `HEDIS_MAX_OID=6ad2b1a20000000000000000`, set 18:22 CT, after the v9 freeze (`4ee41d9`, 17:21 CT). `go test` passes; research/v9 and main untouched (as reported).

**Hypothesis.** Most remaining errors on 0.85 applications are wrong versions; targeted version selection reduces them.

**Method.** (a) Per-package score floor 0.02 for react-native-gesture-handler. (b) Pairwise version-diff selection for safe-area-context, screens, reanimated.

| Set | M39 | M40 |
|---|---|---|
| real_dev | 0.840 | 0.845 |
| synthetic | 0.970 | 0.970 |
| blink-beta, breathly, zeus | 0.885 | 0.918 |
| 7 v8 applications | 0.373 | 0.564 |
| all 10 0.85 development applications | 0.603 | 0.719 |

Unsupported flags 0/33. **All values in-sample:** the target packages and both changes were chosen from v8 errors.

**Rejected variants (development):** date-based dependency-version consistency (synthetic 0.956); version-diff on gesture-handler (real_dev 0.797); version-diff on all packages (0.760); neighbouring-version steps only (0.691).

**Residual errors.** safe-area-context 4.7.2–5.10.1 and screens 4.25–4.26 have near-identical code; gesture-handler on marlin and ysclaude predicted 2.33.0 instead of 2.31.2.

**Threats to validity.** (1) Package-specific rules selected on observed errors carry a high overfitting risk; v9 is the only valid test. (2) Five variants were tried on the same development data, so the selected one is optimistically biased. (3) v9's 0.85 apps may not contain the four targeted packages; report which v9 apps do.
