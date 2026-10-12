# Dataset registry

This registry lists datasets referenced in the thesis and datasets present in the Hedis repository. Sizes and counts reflect the repository snapshot unless noted. We mark thesis-only artifacts that are **not** shipped with this repo.

## Summary table

| ID | Name | Source | Version / snapshot | Size (approx.) | License | Preprocessing in Hedis |
|----|------|--------|-------------------|----------------|---------|------------------------|
| D1 | React Native Directory catalog | [reactnative.directory](https://reactnative.directory/) API | API as harvested into repo | 1,717 packages; ~79 MB JSON (`js/get-packages/react-native-directory-packages.json`) | Directory terms; per-package npm licenses | Fetched with `js/get-packages/index.js`; iOS libraries, tools/templates skipped |
| D2 | GitHub Advisory Database (npm) | GitHub `advisories` API | Harvest date in commit history | Part of `vulnerable-packages.json` (~5.4 MB) | GitHub API terms | Paginated fetch; intersected with D1 package names |
| D3 | Vulnerable RN package–version list | Derived (D1 ∩ D2 + npm semver) | Repo snapshot | 446 packages with advisory metadata | Same as D1/D2 | `npm` version lists resolved per advisory range |
| D4 | npm registry metadata | registry.npmjs.org | Per package at pipeline time | Network | npm terms of use | `npm install <pkg>@<ver>` in RN sandboxes |
| D5 | RN baseline template apps | Generated via `pipeline/get_rn_versions.sh` | RN 0.69.12–0.79.3 (11 envs) | Large `node_modules` per `pipeline/react-natives/rn*` | Per dependency | Empty app + `baseline_entry.js` → Metro bundle → Hermes → fingerprints → `baselines_v3` |
| D6 | RN fingerprint pipeline packages | Same as D3, installed into D5 | Per package@version × RN version | MongoDB `hashes` / `hashes_ghsa` | Per package | Baseline subtraction; three SHA-256 hashes per FO |
| D6b | Thesis refined fingerprint targets | RN Directory ∩ GHSA (thesis §4.3.1, §6.2) | Snapshot **2025-07-10**; **1,220** package versions processed (**952** packages succeeded) | **16.06 GB** MongoDB (thesis); not in repo | CC BY 4.0 (thesis) | 76 h pipeline; see Table 5.6 in [baseline-thesis.md](./baseline-thesis.md) |
| D7 | Thesis MongoDB fingerprint DB | Thesis experiments | **Not in repo** | 16,579,257 FO records | N/A | Required for full thesis rerun |
| D8 | Rainbow Wallet (evaluation) | GitHub / App Store IPA | **v1.9.70** (thesis; lockfile scoring in lab log) | IPA not in repo | OSS app license | `main.jsbundle` Hermes extraction; test reference `Rainbow-1_9_70_app_test_bundle.hbc` in analyzer tests |
| D9 | Mattermost Mobile (evaluation) | GitHub / App Store | **v2.29.1** (lab log lockfile scoring) | IPA not in repo | Apache-2.0 (mobile client) | Same pipeline as D8 |
| D10 | Discord (evaluation) | iOS App Store | Production build (thesis) | IPA not in repo; Apple login required | Proprietary | IPA download + Hermes bundle extraction |
| D11 | Microsoft Outlook (evaluation) | iOS App Store | Production build (thesis) | IPA not in repo | Proprietary | Same as D10 |
| D12 | Hermes opcode definitions | [facebook/hermes](https://github.com/facebook/hermes) tags | Bytecode v61–v96 | `pkg/utils/defversions/` | MIT | `pkg/utils/download_all.sh`; `genopcodes` |
| D13 | Eval harness (external) | `run_eval.sh` (optimizer agent) | 2026-10-12 | Outside repo | — | 60/20/20 split; stratified by RN version; leakage rules |
| D14 | Planned eval corpus | Synthetic + OSS apps | Target ≥30 apps | In progress (2026-10-12 decision) | TBD | Rainbow/Mattermost reserved for **dev** only |

## Per-dataset notes

### D1 — React Native Directory

We use the Directory as a practical universe of React Native–relevant npm packages. The harvest script requests `https://reactnative.directory/api/libraries?ios=true&skipTools=true&skipTemplates=true` with pagination (30 libraries per page).

**Thesis (Section 4.3.1):** After filtering tools and templates, the thesis retained **1,717** iOS-relevant packages. Expanding via the npm registry yielded **73,293** distinct package versions before the security-focused refinement described under D6b.

### D2–D3 — Advisories and vulnerable versions

`js/get-packages/README.md` documents the workflow: intersect Directory packages with npm ecosystem advisories, expand affected version ranges via semver, and write `vulnerable-packages.json`. This file drives which package@version pairs enter the MongoDB fingerprint pipeline (`maintain-database -p`).

**Thesis (Section 5.2.2):** The GitHub Advisory Database snapshot on **July 10, 2025** contained **289,277** distinct npm packages with documented vulnerabilities. Intersecting with the Directory yielded **17** packages with security advisories, which fed the refined **1,220**-version fingerprint target set (D6b). The open-repo `vulnerable-packages.json` (**446** packages) is a **broader** harvest and is not identical to the thesis evaluation corpus.

### D5 — Multi-version React Native sandboxes

| Directory | `react-native` version |
|-----------|------------------------|
| `pipeline/react-natives/rn069` | 0.69.12 |
| `pipeline/react-natives/rn070` | 0.70.15 |
| `pipeline/react-natives/rn071` | 0.71.19 |
| `pipeline/react-natives/rn072` | 0.72.17 |
| `pipeline/react-natives/rn073` | 0.73.11 |
| `pipeline/react-natives/rn074` | 0.74.7 |
| `pipeline/react-natives/rn075` | 0.75.5 |
| `pipeline/react-natives/rn076` | 0.76.9 |
| `pipeline/react-natives/rn077` | 0.77.2 |
| `pipeline/react-natives/rn078` | 0.78.2 |
| `pipeline/react-natives/rn079` | 0.79.3 |

Each sandbox bundles `baseline_entry.js` for framework-only baselines and `package_entry.js` for dependency installs.

### D6–D7 — Fingerprint database

MongoDB collections (see root `README.md`): `packages`, `hashes`, `hashes_ghsa`, `baselines_v3`. The thesis-scale database dump is **not** vendored in git; reproducing thesis tables requires rebuilding via `hermes-decompiler maintain-database` or restoring a dump.

### D8–D11 — Application evaluation sets

| App | Version | RN | Hermes BC | Ground-truth notes (thesis §5.3) |
|-----|---------|-----|-----------|----------------------------------|
| Rainbow | v1.9.70 | 0.74.3 declared; 0.74 inferred | v96 | 246 top-level deps; 7 with fingerprints in DB; Table 5.1 |
| Mattermost | v2.29.1 | 0.76.9 | — | Table 5.2 |
| Discord | 288.0 | 0.74 | v96 | Table 5.4; no public ground truth |
| Outlook | 4.2527.0 | 0.71 | v90 | Table 5.5; no public ground truth |

IPAs are not vendored in this repository; Table 5.3 lists extracted metadata for production apps.

### D13–D14 — Post-thesis evaluation target

As of 2026-10-12, we target **micro-F1 ≥ 0.95** at **package+version** granularity on a frozen held-out test set (≥30 applications, 60/20/20 train/validation/test, stratified by RN version, with explicit leakage rules). Rainbow and Mattermost remain in the development split only.

## Preprocessing chain (reference)

1. `npm install` target package into one RN sandbox (D5).
2. Metro bundle → Hermes compile (`hermesc` from that RN version).
3. Disassemble HBC → function objects (FOs) → structural + content IR hashes.
4. Subtract hashes present in `baselines_v3` for that RN version.
5. Store surviving hashes under `package@version` keys for matching.
