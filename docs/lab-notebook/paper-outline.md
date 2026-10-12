# Extended paper outline (beyond the bachelor thesis)

Working title: **Bytecode Fingerprints for Supply-Chain Visibility in React Native iOS Applications**

Target venue: security or software-engineering conference with artifact evaluation (e.g., USENIX Security, ASE, ISSTA). We write in academic **we**; see [STYLE.md](./STYLE.md).

## Abstract (draft bullets)

- Problem: App Store RN apps ship Hermes bytecode without manifest access; traditional SBOM tools fail.
- Approach: Multi-RN reference database of FO triple-hashes; baseline subtraction; exact and optional fuzzy matching.
- Results: Reproduced package-level metrics on OSS ground truth; new held-out eval with micro-F1 targets at package+version.
- Limitations: Version precision, false positives, iOS-only pipeline in thesis lineage.

## 1. Introduction

- Motivation: supply-chain attacks on npm transitively affect mobile RN apps.
- Gap: no audit-grade tool for production `main.jsbundle` on iOS.
- Contributions (numbered):
  1. Open pipeline from npm advisory → Hermes fingerprints across 11 RN versions.
  2. Evaluation protocol with leakage-aware splits and synthetic apps.
  3. Measurement of package vs package+version detection quality.
  4. Public artifact (Hedis) and lab notebook methodology.

## 2. Background

- React Native architecture (JSI, Metro, Hermes AOT).
- npm advisories and transitive dependency graphs.
- Hermes bytecode versions v61–v96 (compatibility matrix).

## 3. Threat model and scope

- Attacker: malicious or vulnerable dependency included at build time.
- Defender: analyst with IPA/HBC only.
- Explicit non-goals: native module C++ code, runtime exploitation, Android (future work).

## 4. Approach

### 4.1 Fingerprint construction

- Structural (S-AR), content IR1, content IR2 (thesis terminology).
- Normalization levels and hashing (align with `pkg/hbc/types/functionobject.go`).

### 4.2 Reference database pipeline

- RN sandboxes `rn069`–`rn079`.
- Baseline filtering (`baselines_v3`).
- Advisory-driven vs full version history builds.

### 4.3 Application matching

- Exact triple-hash match; highest-version aggregation.
- Optional fuzzy layer (Levenshtein / MinHash)—thesis notes scale limits.

### 4.4 Version and vulnerability mapping

- Mapping detected `package@version` to GHSA ranges.
- Confidence scoring proposal (extend thesis HMV columns).

## 5. Implementation

- Hedis CLI commands (`disassemble`, `analyze`, `maintain-database`).
- MongoDB schema (`packages`, `hashes`, `hashes_ghsa`).
- Reproducibility: Docker compose for Mongo, pinned RN docker images (artifact).

## 6. Evaluation

### 6.1 Research questions

- **RQ1 (presence):** Can we detect declared npm packages from Hermes bundles with high recall?
- **RQ2 (version):** Can we identify correct package@version at micro-F1 ≥ 0.95 on held-out apps?
- **RQ3 (security):** Given RQ2, what fraction of GHSA-affected versions are flagged in production apps?

### 6.2 Datasets

- Registry [dataset-registry.md](./dataset-registry.md): Directory, GHSA, OSS apps, synthetic generators, production IPAs.

### 6.3 Metrics

- Package-level and package+version P/R/F1; micro-F1 aggregation.
- Per-hash-type match rates (structural vs IR1 vs IR2).
- Runtime and database size.

### 6.4 Baselines

- Lockfile oracle (upper bound).
- Thesis matching procedure (historical baseline).
- Manifest-only scanners (when source available).

### 6.5 Results sections (placeholders)

- Table: per-app results (Package | HMV | Matched FOs | …) — extend thesis format.
- Ablation: baseline filtering, fuzzy off/on, RN version mismatch.

## 7. Discussion

- False positives: shared utilities, incomplete baseline subtraction.
- False negatives: tree-shaking, Hermes version skew.
- Comparison to ReuNify-class tools (complementary).

## 8. Related work

- Condensed from [related-work.md](./related-work.md).

## 9. Conclusion and future work

- Android HBC extraction; richer transitive resolution; integration with binary SBOM standards (CycloneDX).

## Artifact appendix

- Link to Hedis repo, lab notebook, eval harness (`run_eval.sh` when published).
- Threat to validity checklist.
