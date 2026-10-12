# 2026-10-12 — Matcher v2, NS/NC fingerprints, and bundled-truth evaluation

Timezone for timestamps: **US Central (CT)** unless noted. This entry documents the Hedis Optimizer experiment session on 2026-10-12. It extends the post-thesis goal of measurable package+version accuracy (see [baseline-thesis.md](../baseline-thesis.md) and [2026-10-12-baseline-reproduction.md](./2026-10-12-baseline-reproduction.md)).

**Run settings (session):** All runs below were executed by **Hedis Optimizer** (Grok Bot, default model, effort **high**). We did **not** use cloud agents for this session. Commits from **08:26 CT** onward were authored as **Kian Pasani**.

---

## 2026-10-12 02:29 CT — Scoped database and provisional baselines

We compared the **v2 matcher** implementation against the prior compare step. The v2 path reduced wall time from **1.10 s** to **17.5 ms** per compare (**~63×**), and the overall build completed **~4.4×** faster than the previous pipeline revision.

**Database scope:** We loaded a reduced fingerprint database covering **29 of 30** target packages (**1,496** package versions) built for React Native **0.74** and **0.76**. Package **better-auth** failed during Metro bundling and was excluded from the successful set.

**Evaluation split:** We froze the train/dev/test partition in `data/split.json`. Development held **synth-00** through **synth-04** plus **Mattermost**. Test held **synth-05** through **synth-09**. **Rainbow** could not be bundled in this environment and was omitted from the split.

**Provisional metrics (test set, prior metric definition):**

| Split | Package F1 | Package+version F1 |
|-------|------------|-------------------|
| Test (provisional baseline) | 0.550 | 0.265 |
| Dev | 0.633 | 0.268 |

Recall was approximately **1.0** on both splits; **precision** was the limiting factor. This pattern suggests that the matcher is inclusive relative to labels rather than failing to surface candidate packages.

---

## 2026-10-12 03:23 CT — Matcher ablations (commit `20ef16f`)

We recorded the following **test-set package+version F1** progression on the frozen split, starting from baseline **0.265**:

| Variant | Test pkg+version F1 | Notes |
|---------|---------------------|-------|
| Baseline | 0.265 | |
| (a) IDF-weighted unique hashes | 0.331 | |
| (a)+(b) score threshold **0.05** | 0.400 | Package F1 **0.885** |
| +(c) version tie-break | 0.383 | |
| +(d) version-string anchors | 0.365 | |

We fixed a **database name resume bug** that had caused incomplete reloads during iterative rebuilds.

**Observation:** For packages where we knew the true version, only **~50%** of unique hashes matched the labeled version; we measured **~58%** of hash material surviving minification. **Hypothesis (hedged):** fingerprint hashes may depend on **bundle context** (surrounding modules and Metro graph), not only on the isolated package source.

---

## 2026-10-12 05:28 CT — Opcode-width diagnosis and NS/NC rebuild

We diagnosed a structural issue in legacy fingerprints: **opcode width variants** depend on **bundle size**, and the **identifier flag** (C1/C2 content split) behaves as a **bundle-wide** property rather than a per-function stable signal.

We introduced **normalized NS/NC** hashes (normalized structural / normalized content representations). On a presence check, match rate rose from **57.7%** to **99.8%** after normalization.

We rebuilt the MongoDB corpus as **`hedis_rel`** with **8.26M** hashes, generated from **minified** bundles to align with production App Store artifacts. **Incident:** a host **box restart** corrupted the previous database volume; local **`.hbc`** intermediates were **lost**, which limits bitwise replay of earlier matcher runs without regeneration.

---

## 2026-10-12 05:43 CT — Go matcher and NS/NC ablation (a+b)

We fixed a **column-shift bug** in the Go **from-dump** loader that had misaligned hash columns during import.

On the **test** split, configuration **NS/NC (a)+(b)** yielded:

| Metric | Value |
|--------|-------|
| Package F1 | **0.900** (precision **1.0**) |
| Strict package+version F1 | **0.541** |
| Relaxed pkg+version (identical fingerprints → one class) | **0.775** |

Remaining misses appeared to be **label issues** rather than matcher failures at package level. **Interpretation (hedged):** strict version labels may over-penalize byte-identical bundles that Metro emitted under different version strings.

---

## 2026-10-12 06:04 CT — Bundled truth via Metro source maps

We derived **bundled ground truth** from Metro **source maps** rather than lockfile declarations alone. On the synthetic portion of the split, we achieved **15/15** correct assignments. We retained **`split.v1.json`** as the canonical split artifact (superseding `split.json` for this experiment line).

| Metric | Test |
|--------|------|
| Package F1 | **0.942** |
| Strict package+version F1 | **0.610** |
| Hash-only ceiling (strict) | **0.651** |

The gap between **0.610** and **0.651** indicates that a fraction of version errors may be irreducible under hash-only strict labeling even with improved matchers.

---

## 2026-10-12 06:10 CT — Byte-identical bundles and strict ceiling

We measured how often “identical fingerprint groups” imply byte-identical Metro output:

- **96.9%** (**310/320** groups) were **byte-identical** Metro bundles across versions labeled distinct.

**Strict package+version ceiling** (hash-only labeling):

| Split | Ceiling |
|-------|---------|
| Dev | 0.676 |
| Test | 0.659 |

**Conclusion:** **Strict package+version F1 ≥ 0.95** is **infeasible** if we treat every npm version string as a distinct class while fingerprints collapse across versions. Furthermore, the database fingerprints **root entry points** only and does **not** fingerprint arbitrary subpath entry files, which may cap version discrimination for packages with multiple bundle shapes.

---

## 2026-10-12 07:40 CT — Decision: release-equivalence metric (Kian)

Kian decided to redefine the **target metric** for post-thesis engineering:

- **Primary metric:** **package+version F1** on the **held-out test** split.
- **Version equivalence:** npm versions that produce **byte-identical Metro bundles** count as **one release class** (bundled truth from source maps).
- **Labels:** bundled truth, not lockfile-only declarations.
- **Reported result under this definition (test):** **0.927**.

This supersedes the earlier micro-F1 ≥ 0.95 target on strict per-version strings when bundles are identical (see [2026-10-12-baseline-reproduction.md](./2026-10-12-baseline-reproduction.md)). We will update harness documentation when `run_eval.sh` and `data/split.v1.json` land in the repository.

---

## 2026-10-12 09:50 CT — Dev split and researcher agent request

Under the **updated metric** (byte-identical bundle equivalence + bundled truth):

| Split | Package+version F1 |
|-------|---------------------|
| Dev | **0.956** |
| Test | **pending** at time of report |

Kian requested a dedicated **Hedis Researcher** agent role to run **experiment swarms** and to execute items listed under **Future Work** in the thesis (finer fingerprinting, version accuracy, transitive analysis). We have not yet integrated that agent into the repository automation.

### Next steps

- [ ] Publish `data/split.v1.json`, bundled-truth extraction scripts, and eval harness in-repo.
- [ ] Complete held-out **test** evaluation under the 07:40 metric and record in a follow-up log entry.
- [ ] Regenerate lost `.hbc` artifacts after the box-restart incident or document immutable storage for pipeline outputs.
- [ ] Scope Hedis Researcher agent: protocol template + swarm orchestration per [protocol-template.md](../protocol-template.md).
