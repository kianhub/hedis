# 2026-10-15 — M38 version fill (feat/m38-version-fill)

**Feature branch:** `feat/m38-version-fill` at **`7db3c19`**. Root fill jobs finished **15:48 CT**; **M38 frozen ~16:30 CT**.

## Fill rule (fixed before execution)

For **every** package already present in the database, we enqueue **each missing stable `X.Y.Z` release** published between **2022-01-01** and **2026-10-01**, fingerprinted across **four** React Native versions (**0.72**, **0.75**, **0.78**, **0.80**). The Cartesian scope is **690 packages × 4 RN versions**.

## Pipeline status at fill completion (15:48 CT)

| Stage | RN 0.72 | RN 0.75 | RN 0.78 | RN 0.80 |
|-------|---------|---------|---------|---------|
| Root fingerprint jobs (of 690) | **674** ok | **651** ok | **651** ok | **647** ok |
| Subpath jobs | **690/690** | **690/690** | **690/690** | **690/690** |

The last fill hash timestamp was **15:48:59 CT**.

## Freeze, merge, and database cutoff

| Item | Value |
|------|--------|
| M38 freeze merge into `v2` | **`c463fd2`** (freeze) |
| Recorded hash commit | **`8e8176e`** |
| `main` (untouched) | **`66fcd2f`** |
| `HEDIS_MAX_OID` cutoff | **`6ad13ed00000000000000000`** (pinned **16:00 CT**) |

**Database growth at cutoff:**

| Metric | After M38 | Delta |
|--------|-----------|-------|
| Hashes | **43,376,040** | **+9,075,502** |
| Packages | **2,210** | **+690** |

## Matcher binary

We built **`bin/hedis-m38`** from **`7db3c19`**.

| Field | Value |
|-------|--------|
| SHA-256 | `cc3f2b3e…877c` (full digest in `docs/eval/results/m38/`) |
| Flags | M37 frozen behavior plus **`--unsupported-cov 0.40`** |

## Rescore (development only, oneclass `--subsha`)

We rescored real-dev and synthetic splits once after cutoff. No held-out panel was rescored; **v6 remains burned**.

| Split | P | R | F1 | TP | FP | FN |
|-------|---|---|-----|-----|-----|-----|
| Real-dev | **0.840** | **0.840** | **0.840** | 105 | 20 | 20 |
| Synthetic | **0.973** | **0.966** | **0.970** | 144 | 4 | 5 |

Predictions were **byte-identical to M37 on every development application**. **No** development app received **`status=unsupported_bytecode`**.

## Interpretation

All **23** development applications use React Native **0.74** or **0.76** only. The M38 rows target **0.72**, **0.75**, **0.78**, and **0.80**, so the matcher **never consults** the new fingerprints on this split. **While** the database grew by **9.08M** hashes, **we cannot** attribute any dev-metric change to M38; the rescore merely confirms parity with M37.

We expect M38 to matter only on a **fresh held-out freeze (v7)** that includes **fallback-RN** applications (the class that failed or degraded under v5/v6). v6 stays burned for tuning.

## Verification and artifacts

- `go test ./...` **passes** at freeze time.
- **Files:** `docs/eval/data/m38_frozen.json`, `docs/eval/results/m38/RESULTS.md`, `eval/run_m38.sh`.

## Next steps

- [ ] Define and freeze **v7** held-out apps with RN **0.72/0.75/0.78/0.80** (or mixed fallback) before scoring M38 against M37.
- [ ] Keep **`--unsupported-cov 0.40`** in the held-out harness to avoid `'$in needs an array'` crashes when coverage is low.
