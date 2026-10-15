# 2026-10-15 — M38 version fill (feat/m38-version-fill)

**Branch:** `feat/m38-version-fill` at **`7db3c19`**. **Completion:** **15:48 CT** on 2026-10-15.

## Fill rule (fixed before execution)

For **every** package already present in the database, we enqueue **each missing stable `X.Y.Z` release** published between **2022-01-01** and **2026-10-01**, fingerprinted across **four** React Native versions (**0.72**, **0.75**, **0.78**, **0.80**). The Cartesian scope is **690 packages × 4 RN versions**.

## Pipeline status at finish

| Stage | RN 0.72 | RN 0.75 | RN 0.78 | RN 0.80 |
|-------|---------|---------|---------|---------|
| Root fingerprint jobs (of 690) | **674** ok | **651** ok | **651** ok | **647** ok |
| Subpath jobs | **690/690** | **690/690** | **690/690** | **690/690** |

Failures on root jobs plausibly reflect Metro or Hermes compile errors on individual package versions; subpath completion at **690/690** indicates the subpath worker queue drained successfully.

## Pending work

- [ ] Pin database **cutoff** after ingest stabilizes.
- [ ] **Rescore** real-dev and synthetic splits only (no new held-out burn until a future freeze).

## Hypothesis

**While** v6 showed **62/84** misses from missing versions, filling stable releases through 2026-10-01 may raise exact recall **if** bundle context hashes align with production apps. We have not yet measured post-fill dev F1.
