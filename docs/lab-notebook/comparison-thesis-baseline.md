# Comparison: thesis baseline (t35) and Hedis matchers (M37 / M38)

This page synthesizes held-out evaluations logged under [`log/`](./log/). **We do not introduce new measurements here**; every number below is copied from [v5](./log/2026-10-14-v5-held-out.md), [v6](./log/2026-10-15-v6-held-out.md), [v7](./log/2026-10-15-v7-held-out.md), [M37](./log/2026-10-14-m37-rn-fallback-freeze.md), [M38](./log/2026-10-15-m38-version-fill.md), and related incident notes. Earlier informal comparison drafts in the repository did not include a standalone v4 document; **this file supersedes any ad hoc v4 notes** if they appear elsewhere.

## 1. Setup

### Thesis baseline (t35)

We implement the thesis matching policy as **t35**: **exact triple-hash** agreement (structural, content IR1, content IR2), **no score threshold**, **highest-matching version (HMV)** per package name, against the **same MongoDB fingerprint corpus** as the Hedis runs at each freeze cutoff. Configurations **t35a** and **t35b** on v7 are two harness variants logged at `research/v7-freeze` **`7e9b4f7`**.

### Hedis matchers (M37 / M38)

**M37** adds production engineering around the thesis core: **nearest-RN fallback** (frozen at **`359ec3f`**, see M37 log), baseline subtraction, and Go matcher binaries pinned per freeze.

**M38** extends M37 with database cutoff **`6ad13ed0…`** (version fill on RN **0.72/0.75/0.78/0.80**), binary **`bin/hedis-m38`**, and **`--unsupported-cov 0.40`** for abstention when fingerprint coverage is insufficient.

### Primary metric

**Exact package+version F1** on **bundled ground truth**, with npm versions that yield **byte-identical Metro bundles** collapsed to a single release class (post–v6 engineering target; v7 scored under this definition at **`7e9b4f7`**). **Vulnerability F1** uses **strict advisory** alignment where logged. **App status** denotes applications that **completed** scoring without crash or without incorrect forced scoring on RN **0.85** (v7 strict table).

### Held-out protocol

Each freeze (**v5**, **v6**, **v7**) was **preregistered or manifest-tagged**, then **scored once**; **v6** and **v7** are **burned** (no rescoring for tuning).

| Freeze | Apps (design) | Apps in pooled exact table | Source log |
|--------|---------------|----------------------------|------------|
| v5 | 12 | 12 (6 crashed) | [2026-10-14-v5-held-out](./log/2026-10-14-v5-held-out.md) |
| v6 | 12 | 12 | [2026-10-15-v6-held-out](./log/2026-10-15-v6-held-out.md) |
| v6 fair subset | 8 | 8 | same (`03de73d`) |
| v7 | 13 preregistered | **10** (RN **0.85** excluded from pool) | [2026-10-15-v7-held-out](./log/2026-10-15-v7-held-out.md) |

## 2. Results tables

### v5 — twelve applications (single Hedis pipeline; pre–M37 freeze)

The v5 log reports **one** Hedis pipeline only. **t35**, **M37**, and **M38** were **not** scored on v5 in the notebook.

| Method | Exact P | Exact R | Exact F1 | Vuln F1 | App status (completed) |
|--------|---------|---------|----------|---------|-------------------------|
| Hedis (v5 harness) | 0.720 | 0.362 | **0.482** | **0.578** | **6 / 12** (6 crashed; see §4) |
| t35 | — | — | — | — | — |
| M37 | — | — | — | — | — |
| M38 | — | — | — | — | — |

*Subset of six non-crashed apps:* exact F1 **0.705**, vuln F1 **0.905** (v5 log).

### v6 — twelve applications

| Method | Exact P | Exact R | Exact F1 | Vuln F1 | App status (completed) |
|--------|---------|---------|----------|---------|-------------------------|
| **M37** | 0.486 | 0.458 | **0.472** | **0.883** | **11 / 12** |
| **t35** | — | — | **0.245** | **0.161** | **5 / 12** |
| M38 | — | — | — | — | — |

### v6 — fair eight-application subset (`03de73d`)

Apps both pipelines could score under fair comparison rules.

| Method | Exact P | Exact R | Exact F1 | Vuln F1 | App status (completed) |
|--------|---------|---------|----------|---------|-------------------------|
| **M37** | 0.607 | 0.618 | **0.613** | **0.919** | **7 / 8** |
| **t35** | — | — | **0.278** | **0.182** | **3 / 8** |
| M38 | — | — | — | — | — |

### v7 — ten applications (pooled exact; `7e9b4f7`)

| Method | Exact P | Exact R | Exact F1 | Vuln F1 (strict) | App status (strict) |
|--------|---------|---------|----------|------------------|---------------------|
| **M37** | — | — | **0.471** (45/54/47) | **0.915** | **9 / 10** |
| **M38** | — | — | **0.586** (56/43/36) | **0.992** | **9 / 10** |
| **t35a** | 0.118 | 0.326 | **0.173** (30/225/62) | **~0.17** | — |
| **t35b** | 0.119 | 0.337 | **0.176** (31/229/61) | **~0.17** | — |

**Vulnerability prevalence (ground truth):** **3 / 10** v7 apps contain at least one vulnerable package (v7 log). We did **not** log separate per-app “clean vs vulnerable” classification accuracy beyond completion status.

## 3. Where the gain comes from

### v6 fair subset (M37 vs t35)

**Set-hit rate** was **0.648** for **both** M37 and t35. M37 exact F1 rose **0.278 → 0.613** while recall stayed near parity (**0.618** vs t35 implicit recall from F1). The dominant mechanism logged is **precision**: false positives fell **181 → 44** (v6 log). **While** this supports an engineering improvement over t35 ranking noise, **we do not** claim recall gains on that subset.

### v7 (M38 vs M37)

Aggregate exact F1 moved **0.471 → 0.586**. Per-group exact F1:

| Group | M37 | M38 |
|-------|-----|-----|
| Fill | 0.346 | 0.549 |
| Control | 0.703 | 0.703 |
| Fallback | 0.308 | 0.485 |

The **fallback** delta **0.308 → 0.485** is **entirely yoroi** (RN **0.71**, nearest **0.72**): **0.077 → 0.519**; **readany** and **ogam** (nearest **0.80**) were **unchanged**. Fill-group lifts on **jitsi** and **daimo** (RN **0.72**) plus **yoroi** account for the pooled M38 gain; **gsy** (**0.80**) stayed **0.875** for both matchers. **We therefore attribute the observed v7 M38 improvement to the RN 0.72 fill**, not to the **0.80** slice.

## 4. Robustness

### t35 crashes

| Freeze | Failure mode |
|--------|----------------|
| v6 | Crashed on **hexa**, **lxmusic**, **minds**, **zeus** (5/12 apps completed for t35) |
| v7 | Crashed on **HBC98** for RN **0.85** apps (**blink-beta**, **breathly**) |

### M37 nearest-RN fallback

Introduced to score applications whose native RN fingerprint slice is missing (v6 **zeus**: coverage **0.07–0.13** vs **~0.94** control; exact F1 **0**). v7 **yoroi** benefits when nearest **0.72** fill exists.

### M38 unsupported abstention (RN 0.85)

On v7, M38 **abstained 2/2** RN **0.85** apps (**correct** under preregistered rules). M37 scored **0/2** on that abstention criterion; t35 **crashed** on **HBC98**.

### v5 infrastructure failures

Six of twelve v5 apps crashed with MongoDB **`'$in needs an array'`** because RN **0.72/0.75/0.78/0.80** had **no fingerprints** at that cutoff (**58** of **118** exact FNs attributed to crashes in the v5 log).

## 5. Limitations

**Sample size:** held-out panels are **12**, **12**, and **13** preregistered (**10** pooled for v7 exact F1). **We cannot** treat these as large-scale benchmarks.

**Database coverage:** On v6, **62 of 84** exact false negatives were versions **missing from the database**, not matcher ranking errors alone.

**Version fill evidence:** M38 **0.80** fill is **not demonstrated** on v7 (**gsy**, **readany**, **ogam** flat). RN **0.75** and **0.78** have **no public unburned** held-out apps in the v7 preregistration (v7 log shortfall).

**Ground-truth caveats:** **Pixelfed** labels grew **62 → 236** after a **StoryCarousel** stub ([incident log](./log/2026-10-15-snapshot-restore-incident.md)). **safe** uses an **expo-router** entry shim (**`2ef2ee6`**). **xaman** has **157** GT packages absent from the **subsha** set (v7 log).

**Provenance and incidents:** Binary **`36d63cb0…`** could **not** be reproduced ([incident log](./log/2026-10-15-snapshot-restore-incident.md)). A **2026-10-15 ~10:29 CT** snapshot restore destroyed v6 freeze artifacts (re-frozen at **`91a5069`**).

**Confound (M38 on v7):** Development rescoring showed **identical** M37 vs M38 predictions on all **23** dev apps (RN **0.74/0.76** only); v7 gain is **primarily attributed to DB cutoff** until a dev ablation (M38 binary at M37 cutoff) or a **v8** freeze isolates components (v7 log).

## References

| Topic | Log entry |
|-------|-----------|
| Thesis tables (historical) | [baseline-thesis.md](./baseline-thesis.md) |
| M37 freeze | [2026-10-14-m37-rn-fallback-freeze.md](./log/2026-10-14-m37-rn-fallback-freeze.md) |
| M38 fill + dev parity | [2026-10-15-m38-version-fill.md](./log/2026-10-15-m38-version-fill.md) |
