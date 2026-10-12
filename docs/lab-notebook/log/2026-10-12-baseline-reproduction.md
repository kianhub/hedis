# 2026-10-12 — Baseline reproduction and evaluation target

Timezone for timestamps: **US Central (CT)**. This entry documents work by the optimizer agent and decisions confirmed by Kian. It contributes to the post-thesis goal of measurable package+version accuracy (see [baseline-thesis.md](../baseline-thesis.md)).

---

## 2026-10-12 00:07 CT — Work started

We began baseline reproduction for the Hedis thesis matching pipeline. The immediate objectives were threefold: (1) restate the thesis **baseline definition** in executable terms, (2) fix a **target metric** for post-thesis engineering, and (3) confirm whether an **evaluation harness** already exists outside this repository.

**Setup note:** We did not yet restore the thesis MongoDB fingerprint dump (~16 GB) in this environment.

---

## 2026-10-12 00:09 CT — Method restatement, blockers, and lockfile scoring

### Thesis matching procedure (baseline)

We restated the thesis detector as follows: application Hermes bytecode **function objects (FOs)** are matched against npm **`package@version`** fingerprints stored as **three SHA-256 hashes** (structural, content IR1, content IR2). A hit requires an **exact match on all three** hashes for a given reference entry; package version is chosen by the **highest-match** version among candidates for that package name. The thesis reports per-application detection tables but **does not** report precision, recall, or F1 (**verified** during this review).

### Full rerun blockers

We could not execute a full thesis rerun in this workspace for two reasons:

1. The **~16 GB MongoDB fingerprint database** from the thesis experiments is **not** present in the repository.
2. **App Store application bundles** (Discord, Microsoft Outlook, and similar) require an **Apple login** and are not vendored as IPAs in git.

Consequently, any numeric comparison to thesis tables remains **unverified** until we restore or rebuild the database and obtain IPAs.

### Lockfile reconciliation (Rainbow and Mattermost)

We scored thesis-style package detections against lockfile ground truth for **Rainbow v1.9.70** and **Mattermost v2.29.1**:

| Metric | Declared dependencies | All shipped deps | Notes |
|--------|----------------------|------------------|-------|
| Precision | 0.375 | — | |
| Recall | 1.0 | — | |
| F1 | 0.545 | 0.667 | “All shipped” includes transitive lockfile entries |
| F1 with **0.25 cutoff** tuned on the same apps | — | — | **0.789** (optimistic; in-sample tuning) |
| **Package+version F1** | — | — | **0** — reference DB held **vulnerable versions only**, not full version histories |

**Interpretation (hedged):** The package-level recall of 1.0 may indicate that fingerprint matching is inclusive relative to lockfile declarations, while precision 0.375 suggests that many detected package names do not correspond to declared direct dependencies. Package+version F1 of zero is plausibly explained by the restricted version coverage in the database rather than by a failure of the triple-hash matcher itself.

### Evaluation target (confirmed by Kian)

We recorded the following engineering target:

- **Metric:** micro-F1 **≥ 0.95** at **package+version** granularity.
- **Test set:** frozen held-out split with **≥ 30 applications** (open source plus synthetic), **60/20/20** train/validation/test, **stratified by RN version**, with explicit **leakage rules**.
- **Development holdout:** **Rainbow** and **Mattermost** remain in the **development** split only (not in held-out test).

### Eval harness

An evaluation harness **`run_eval.sh`** exists **outside this repository** (optimizer agent, 2026-10-12). We have not yet imported it into `hedis`; reproduction steps should reference that script once published.

---

## 2026-10-12 00:10 CT — Decision

Kian decided to pursue a **smaller fingerprint database rebuild** that includes **full version histories** for approximately **30 evaluation-relevant packages**, combined with **synthetic test applications**, rather than attempting to restore the full thesis-scale dump first.

We **started** that rebuild as of this timestamp. The rebuild is intended to unblock package+version F1 measurement and the micro-F1 ≥ 0.95 held-out goal.

### Next steps

- [ ] Publish or vendor `run_eval.sh` and document invocation in [baseline-thesis.md](../baseline-thesis.md).
- [ ] Complete the reduced MongoDB build with non-vulnerable versions for eval packages.
- [ ] Generate synthetic apps covering diverse RN/Hermes versions for the 60/20/20 split.
- [x] Transcribe thesis Tables 5.1–5.6 — completed 2026-10-12; see [2026-10-12-thesis-pdf-verification.md](./2026-10-12-thesis-pdf-verification.md).
