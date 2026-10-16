# Baseline: thesis method and reproduction

This page records the evaluation baseline described in Pasani (2025)[^thesis] and our status reproducing it from the Hedis repository. Thesis numbers below were transcribed from Section 5.3 and Table 5.1–5.6 (PDF verified 2026-10-12 via TUprints). Lockfile reconciliation metrics come from the 2026-10-12 optimizer run in [`log/2026-10-12-baseline-reproduction.md`](./log/2026-10-12-baseline-reproduction.md).

[^thesis]: Pasani, K. (2025). *Evaluating Dependency Vulnerabilities in React Native Applications*. TU Darmstadt. DOI [10.26083/tuprints-00031438](https://doi.org/10.26083/tuprints-00031438).

## Thesis matching procedure (baseline definition)

As described in Chapter 4 and Section 5.3, the baseline detector operates as follows:

1. **Reference database:** For each npm `package@version`, we store fingerprints of Hermes bytecode **function objects (FOs)** across eleven React Native (RN) build environments (0.69–0.79).
2. **Fingerprint triple:** Each FO yields three SHA-256 hashes—structural (S-AR), content IR1 (C1-AR), and content IR2 (C2-AR).
3. **Application analysis:** We disassemble the target `main.jsbundle`, extract FOs, subtract framework hashes using per-RN baselines, then match against the reference database filtered to the inferred RN version.
4. **Match rule:** An FO counts toward a package when **all three** hashes match exactly for a given `package@version` entry.
5. **Version selection (HMV):** For each package name, we report the **highest matching version (HMV)** based on the **Matched(%)** column (matched FOs divided by total FOs for that package in the reference slice).
6. **Reported metrics:** The thesis publishes per-application match tables (Tables 5.1–5.5) and reference-database statistics (Table 5.6). It does **not** report precision, recall, or F1. Section 6.1 states that, in controlled apps, **no matched HMV corresponded exactly to declared versions** in `package.json`, although several top-level dependencies were detected at package level.

## Reference database (Table 5.6, Section 5.3)

Following pipeline execution across **1,220** distinct package-version targets, we successfully processed **952** distinct packages, generating **16,579,257** aggregate FO entries. Processing required approximately **76 hours** and produced **16.06 GB** uncompressed MongoDB data.

| RN version | Has C1-AR | Has C2-AR | Only S-AR | Total #FO | Total #packages |
|------------|-----------|-----------|-----------|-----------|-----------------|
| 0.69 | 450,698 | 1,277,692 | 16,511 | 1,295,873 | 947 |
| 0.70 | 457,731 | 1,284,855 | 16,511 | 1,303,036 | 947 |
| 0.71 | 449,148 | 1,336,894 | 16,449 | 1,355,076 | 947 |
| 0.72 | 486,127 | 1,407,531 | 16,922 | 1,426,943 | 949 |
| 0.73 | 503,318 | 1,475,737 | 16,575 | 1,494,973 | 949 |
| 0.74 | 575,300 | 1,562,279 | 17,428 | 1,581,685 | 949 |
| 0.75 | 581,999 | 1,563,124 | 17,652 | 1,582,754 | 949 |
| 0.76 | 585,680 | 1,572,537 | 16,607 | 1,591,122 | 949 |
| 0.77 | 584,601 | 1,573,562 | 16,424 | 1,592,284 | 949 |
| 0.78 | 598,190 | 1,613,221 | 16,424 | 1,631,946 | 949 |
| 0.79 | 628,523 | 1,704,692 | 16,572 | 1,723,565 | 949 |

## Application metadata and runtime (Section 5.3)

| Application | App version | RN (inferred / declared) | Hermes BC | Total FOs | FOs after baseline removal | Analysis duration |
|-------------|-------------|--------------------------|-----------|-----------|----------------------------|-------------------|
| Rainbow v1.9.70 | — | 0.74 (inferred; app uses 0.74.3) | v96 | 96,538 | 85,968 | 15m 59s |
| Mattermost v2.29.1 | — | 0.76.9 (declared) | — | 53,709 | 45,158 | 5m 9s |
| Discord | 288.0 | 0.74 | 96 | 109,067 | 98,418 | 24m 40s |
| Outlook | 4.2527.0 | 0.71 | 90 | 28,336 | 25,757 | 2m 50s |

Rainbow matched against **1,581,685** package FOs spanning **949** packages for RN 0.74 after version filtering (Section 5.3 narrative).

## Per-application detection tables (thesis-reported)

Asterisk (*) marks packages listed in the app `package.json`. Columns: **HMV**, matched FOs, total package FOs in the reference slice, **Matched(%)**.

### Table 5.1 — Rainbow

| Package | HMV | Matched FOs | Total FOs | Matched(%) |
|---------|-----|-------------|-----------|------------|
| zod* | 1.11.16 | 397 | 676 | 58.73% |
| @sentry/react-native* | 5.19.0 | 3294 | 6651 | 49.53% |
| axios | 1.7.5 | 179 | 428 | 41.82% |
| immer* | 9.0.3 | 72 | 196 | 36.73% |
| react-native-fast-image* | 1.0.0 | 1053 | 2922 | 36.04% |
| react-native-mmkv* | 2.10.2 | 1056 | 2943 | 35.88% |
| react-native-webview* | 1.0.2 | 1028 | 2901 | 35.44% |
| react-native-keys | 0.7.11 | 1008 | 2884 | 34.95% |
| react-native-document-picker | 7.1.0 | 1010 | 2894 | 34.90% |
| react-native-reanimated* | 2.7.0 | 1482 | 4497 | 32.96% |
| semver | 7.3.5 | 76 | 258 | 29.46% |
| @nozbe/watermelondb | 0.8.0 | 725 | 2764 | 26.23% |
| i18next | 2.0.10 | 62 | 301 | 20.60% |
| better-auth | 0.0.1 | 6 | 44 | 13.64% |
| crypto-js | 4.1.1 | 22 | 317 | 6.94% |
| parse | 1.1.5 | 29 | 562 | 5.16% |

### Table 5.2 — Mattermost

| Package | HMV | Matched FOs | Total FOs | Matched(%) |
|---------|-----|-------------|-----------|------------|
| zod | 1.11.12 | 328 | 676 | 48.52% |
| react-native-fast-image | 5.0.2 | 1090 | 2924 | 37.28% |
| @sentry/react-native* | 5.19.0 | 2454 | 6655 | 36.87% |
| react-native-webview | 1.0.1 | 1058 | 2904 | 36.43% |
| react-native-mmkv | 1.4.0 | 1062 | 2924 | 36.32% |
| react-native-keys | 0.7.11 | 1041 | 2886 | 36.07% |
| react-native-document-picker* | 8.2.0 | 1044 | 2896 | 36.05% |
| react-native-reanimated* | 2.4.1 | 1479 | 4478 | 33.03% |
| @nozbe/watermelondb* | 0.1.24 | 763 | 2625 | 29.07% |
| semver* | 7.2.1 | 43 | 169 | 25.44% |
| axios | 1.7.3 | 79 | 428 | 18.46% |
| i18next | 2.0.10 | 30 | 301 | 9.97% |
| better-auth | 0.0.1 | 3 | 44 | 6.82% |
| immer | 7.0.0 | 9 | 189 | 4.76% |
| parse | 1.8.1 | 38 | 872 | 4.36% |
| crypto-js | 4.1.1 | 8 | 317 | 2.52% |

### Table 5.4 — Discord

| Package | HMV | Matched FOs | Total FOs | Matched(%) |
|---------|-----|-------------|-----------|------------|
| zod | 1.11.15 | 231 | 676 | 34.17% |
| @sentry/react-native | 5.19.0 | 1657 | 6651 | 24.91% |
| semver | 7.3.6 | 60 | 298 | 20.13% |
| @nozbe/watermelondb | 0.8.0 | 427 | 2764 | 15.45% |
| react-native-mmkv | 1.3.3 | 10 | 70 | 14.29% |
| react-native-fast-image | 8.1.8 | 369 | 2889 | 12.77% |
| react-native-document-picker | 6.1.1 | 369 | 2893 | 12.75% |
| react-native-webview | 7.1.1 | 375 | 2943 | 12.74% |
| react-native-keys | 0.0.13 | 366 | 2882 | 12.70% |
| i18next | 2.0.10 | 38 | 301 | 12.62% |
| axios | 1.7.5 | 54 | 428 | 12.62% |
| react-native-reanimated | 2.3.0 | 540 | 4480 | 12.05% |
| immer | 7.0.0 | 13 | 189 | 6.88% |
| better-auth | 0.0.1 | 2 | 44 | 4.55% |
| parse | 1.6.8 | 29 | 645 | 4.50% |
| crypto-js | 4.1.1 | 6 | 317 | 1.89% |

### Table 5.5 — Outlook

| Package | HMV | Matched FOs | Total FOs | Matched(%) |
|---------|-----|-------------|-----------|------------|
| zod | 1.11.14 | 262 | 646 | 40.56% |
| react-native-webview | 7.0.7 | 444 | 2510 | 17.69% |
| react-native-document-picker | 8.0.0 | 430 | 2462 | 17.47% |
| react-native-fast-image | 8.1.6 | 426 | 2454 | 17.36% |
| react-native-keys | 0.0.13 | 423 | 2447 | 17.29% |
| react-native-mmkv | 1.4.0 | 425 | 2483 | 17.12% |
| react-native-reanimated | 1.12.0 | 499 | 3050 | 16.36% |
| @nozbe/watermelondb | 0.8.0 | 415 | 2781 | 14.92% |
| @sentry/react-native | 5.16.0 | 636 | 5695 | 11.17% |
| i18next | 2.2.0 | 23 | 272 | 8.46% |
| semver | 5.3.0 | 5 | 89 | 5.62% |
| axios | 1.8.1 | 17 | 397 | 4.28% |
| parse | 1.6.8 | 18 | 620 | 2.90% |
| immer | 7.0.0 | 4 | 154 | 2.60% |
| crypto-js | 4.1.1 | 4 | 288 | 1.39% |
| better-auth | 0.0.1 | 0 | 15 | 0.00% |

## Lockfile reconciliation baseline (2026-10-12, not thesis-reported)

We scored thesis-style package lists against lockfiles for **Rainbow v1.9.70** and **Mattermost v2.29.1** to quantify package-level accuracy. These metrics are **external** to the thesis tables above.

| Comparison | Precision | Recall | F1 | Notes |
|------------|-----------|--------|-----|-------|
| Detected vs **declared** dependencies | 0.375 | 1.0 | 0.545 | |
| Detected vs **all shipped** dependencies | — | — | 0.667 | Includes transitive lockfile entries |
| With **0.25 cutoff** tuned on same apps | — | — | 0.789 | Optimistic; in-sample tuning |
| **Package+version** F1 | — | — | **0** | Repo DB snapshot held vulnerable versions only |

**Interpretation (hedged):** Thesis Section 6.1 already notes that HMV values did not match declared versions in controlled apps; the zero package+version F1 on our 2026-10-12 scoring may additionally reflect incomplete version coverage in the open-source database build rather than a failure of triple-hash matching alone.

## Reproduction from this repository

### Prerequisites

- Go 1.23+, Node.js, MongoDB
- `.env` in `go/hermes-decompiler/` (`MONGO_CONNECTION_STRING`, `MONGO_DB_NAME`, `OS_HERMES`, `GITHUB_TOKEN`)
- `pipeline/react-natives/rn*` environments with intact `node_modules`

### Build reference fingerprints

```bash
cd go/hermes-decompiler
go build -o hermes-decompiler .

hermes-decompiler maintain-database -b
hermes-decompiler maintain-database -g
hermes-decompiler maintain-database -p
```

### Analyze an application bundle

```bash
hermes-decompiler analyze -s \
  -p path/to/package.bundle.hbc \
  -a path/to/app/main.jsbundle.hbc \
  -o analysis.txt
```

### Remaining blockers (2026-10-12)

| Blocker | Impact |
|---------|--------|
| **16.06 GB** thesis MongoDB dump not in git | Cannot bitwise-replay thesis runs without ~76 h rebuild or restore |
| Production IPAs not vendored | Discord/Outlook bundles require App Store acquisition |
| Open-repo `vulnerable-packages.json` (446 packages) ≠ thesis refined **1,220**-version target set | Metrics diverge if we fingerprint the wider GHSA harvest without thesis filtering |

## Target beyond thesis baseline

**2026-10-12 00:10 CT (initial):** micro-F1 ≥ 0.95 at strict **package+version** on a held-out test set (≥30 apps, 60/20/20, stratified by RN version); Rainbow and Mattermost in dev only.

**2026-10-12 07:40 CT (current):** **package+version F1** on held-out **test** with **bundled truth** (Metro source maps) and **byte-identical Metro bundles** counted as one release class. Test **0.927** reported; dev **0.956** under the updated metric ([log](./log/2026-10-12-eval-matcher-optimizer.md)). Harness: `run_eval.sh`, `data/split.v1.json` (external at time of writing).

## Decision log

[`log/2026-10-12-baseline-reproduction.md`](./log/2026-10-12-baseline-reproduction.md), [`log/2026-10-12-thesis-pdf-verification.md`](./log/2026-10-12-thesis-pdf-verification.md).
