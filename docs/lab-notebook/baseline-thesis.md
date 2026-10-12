# Baseline: thesis method and reproduction

This page records the evaluation baseline described in Pasani (2025)[^thesis] and our status reproducing it from the Hedis repository. Numbers taken only from the thesis are cited to chapter/section where we have confirmation from the style guide or public abstract; lockfile reconciliation metrics come from the 2026-10-12 optimizer run documented in [`log/2026-10-12-baseline-reproduction.md`](./log/2026-10-12-baseline-reproduction.md).

[^thesis]: Pasani, K. (2025). *Evaluating Dependency Vulnerabilities in React Native Applications*. TU Darmstadt. DOI [10.26083/tuprints-00031438](https://doi.org/10.26083/tuprints-00031438).

## Thesis matching procedure (baseline definition)

As described in the thesis (Chapter 4, fingerprinting and matching sections), we understand the baseline detector as follows:

1. **Reference database:** For each npm `package@version`, we store fingerprints of Hermes bytecode **function objects (FOs)** produced across multiple React Native (RN) build environments.
2. **Fingerprint triple:** Each FO yields three SHA-256 hashes—structural (instruction-sequence / S-AR), content IR1 (non-identifier strings), and content IR2 (identifiers and references). The thesis uses this three-hash representation for matching.
3. **Application analysis:** We disassemble the target app `main.jsbundle` (Hermes bytecode), extract FOs, subtract framework hashes using per-RN **baselines**, then match remaining hashes against the database.
4. **Match rule:** An FO counts as a package hit when **all three** hashes match exactly for a given `package@version` entry.
5. **Version selection:** Among versions of a package name that match, we select the **highest-match** version (by matched FO count or share—per thesis tables with columns *Package | HMV | Matched FOs | Total FOs | Matched(%)*; **exact tie-break rule unverified** without PDF access in this environment).
6. **Reported metrics:** The thesis presents per-application tables of detected packages and qualitative discussion. It does **not** report precision, recall, or F1 at package or package+version level (**confirmed** in lab log 2026-10-12 after thesis review).

## Thesis-reported application results (qualitative)

The thesis validates on four iOS applications (README and thesis abstract):

| Application | Role | Ground truth |
|-------------|------|--------------|
| Rainbow Wallet | Open source | `package.json` / lockfile |
| Mattermost Mobile | Open source | Declared dependencies |
| Discord | Production (App Store) | None (detection only) |
| Microsoft Outlook | Production (App Store) | None (detection only) |

Per-app numeric tables (matched FO counts and percentages per package) appear in the thesis evaluation chapter. We have **not** transcribed those table rows here because the TUprints PDF was blocked by bot protection during notebook setup; they remain **unverified** in this workspace until we obtain the PDF or a restored MongoDB dump.

Qualitative findings from the DataCite abstract (not a substitute for tables): the method identifies many direct and transitive packages, but **imprecise version matching**, **false positives**, and **limited audit-grade confidence** remain open problems.

## Lockfile reconciliation baseline (2026-10-12, optimizer agent)

We scored thesis-style package lists against lockfiles to approximate package-level accuracy where ground truth exists. These numbers are **not** thesis-reported; they quantify the gap before we hit the micro-F1 ≥ 0.95 target.

| Comparison | Precision | Recall | F1 | Notes |
|------------|-----------|--------|-----|-------|
| Detected vs **declared** dependencies | 0.375 | 1.0 | 0.545 | Rainbow v1.9.70 and Mattermost v2.29.1 |
| Detected vs **all shipped** dependencies | — | — | 0.667 | Includes transitive packages in lockfile |
| With **0.25 cutoff** tuned on same apps | — | — | 0.789 | Optimistic; same-app tuning |
| **Package+version** F1 | — | — | **0** | Reference DB contained only **vulnerable** versions, not full version histories |

**Interpretation (hedged):** High recall at package level may indicate that fingerprint matching is permissive or that undeclared transitive packages appear in bytecode, while precision near 0.375 suggests substantial false positives or naming ambiguities. Package+version F1 of zero is expected if the database lacks non-vulnerable versions; this limits any direct replay of thesis version columns without rebuilding the DB.

## Reproduction from this repository

### Prerequisites

- Go 1.23+, Node.js, MongoDB
- `.env` in `go/hermes-decompiler/` (`MONGO_CONNECTION_STRING`, `MONGO_DB_NAME`, `OS_HERMES`, `GITHUB_TOKEN` for advisory harvest)
- `pipeline/react-natives/rn*` environments with intact `node_modules`

### Build reference fingerprints

```bash
cd go/hermes-decompiler
go build -o hermes-decompiler .

# Framework baselines → baselines_v3
hermes-decompiler maintain-database -b

# Vulnerable packages from js/get-packages (after npm install there)
hermes-decompiler maintain-database -g   # download advisories + packages
hermes-decompiler maintain-database -p   # fingerprint pipeline (long-running)
```

### Analyze an application bundle

```bash
# Specific package vs app bundle (supported mode in current CLI)
hermes-decompiler analyze -s \
  -p path/to/package.bundle.hbc \
  -a path/to/app/main.jsbundle.hbc \
  -o analysis.txt

# Optional fuzzy matching
hermes-decompiler analyze -b path/to/app.hbc -f -c 0.8
```

For Rainbow, the repository references `testdata/Rainbow-1_9_70_app_test_bundle.hbc` in `pkg/analyzer/compute_test.go`; the binary may not be committed (check LFS or local artifacts).

### Blockers for full thesis rerun (2026-10-12)

| Blocker | Impact |
|---------|--------|
| ~16 GB MongoDB fingerprint DB missing from repo | Cannot reproduce thesis hash lookups without multi-day rebuild |
| App Store IPAs (Discord, Outlook) require Apple login | No bundled IPAs in git |
| DB covered vulnerable versions only | Package+version metrics fail until full version histories are built |

## Target beyond thesis baseline

Kian confirmed (2026-10-12) the engineering target: **micro-F1 ≥ 0.95** at **package+version** on a frozen held-out test set (≥30 apps, OSS plus synthetic, 60/20/20 split stratified by RN version, leakage rules). Rainbow and Mattermost stay in **development** only. External harness: `run_eval.sh` (not yet in repo).

## Decision log pointer

See [`log/2026-10-12-baseline-reproduction.md`](./log/2026-10-12-baseline-reproduction.md) for the rebuild decision (smaller DB with full version histories for ~30 eval-relevant packages plus synthetic test apps).
