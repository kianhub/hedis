# Hedis lab notebook

This directory records experiments, datasets, and decisions for the Hedis (Hermes Decompiler/Disassembler) research line that began with the TU Darmstadt bachelor thesis [*Evaluating Dependency Vulnerabilities in React Native Applications*](https://doi.org/10.26083/tuprints-00031438).

## How to use this notebook

We treat each substantial experiment as a dated log entry under [`log/`](./log/). Before running new work, copy [`protocol-template.md`](./protocol-template.md) into `log/` and fill every section. Prose in this tree follows [`STYLE.md`](./STYLE.md).

## Index

| Document | Purpose |
|----------|---------|
| [STYLE.md](./STYLE.md) | Writing conventions (academic *we*, hedging, formatting) |
| [protocol-template.md](./protocol-template.md) | Blank experiment protocol |
| [dataset-registry.md](./dataset-registry.md) | Datasets used in the thesis and in this repository |
| [baseline-thesis.md](./baseline-thesis.md) | Thesis Tables 5.1–5.6, matching procedure, reproduction notes |
| [comparison-thesis-baseline.md](./comparison-thesis-baseline.md) | t35 vs M37/M38 on burned held-out freezes (v5, v6, v7) |
| [related-work.md](./related-work.md) | Prior art and newer verified references |
| [paper-outline.md](./paper-outline.md) | Outline for an extended paper beyond the thesis |
| [log/](./log/) | Chronological experiment and decision log |

### Log entries (chronological)

| Date | Entry |
|------|--------|
| 2026-10-12 | [baseline reproduction](./log/2026-10-12-baseline-reproduction.md), [thesis PDF verification](./log/2026-10-12-thesis-pdf-verification.md), [matcher / NS/NC eval](./log/2026-10-12-eval-matcher-optimizer.md) |
| 2026-10-14 | [M34 Go matcher](./log/2026-10-14-m34-go-matcher.md), [v5 held-out](./log/2026-10-14-v5-held-out.md), [M37 RN fallback freeze](./log/2026-10-14-m37-rn-fallback-freeze.md) |
| 2026-10-15 | [snapshot restore incident](./log/2026-10-15-snapshot-restore-incident.md), [v6 held-out](./log/2026-10-15-v6-held-out.md), [M38 version fill](./log/2026-10-15-m38-version-fill.md), [v7 held-out](./log/2026-10-15-v7-held-out.md) |

## Repository entry points

- Go CLI: `go/hermes-decompiler/` (`hermes-decompiler analyze`, `maintain-database`, `disassemble`)
- Fingerprint pipeline: `pipeline/react-natives/rn069`–`rn079` (11 React Native template apps)
- Package/advisory harvesting: `js/get-packages/`

## Citation

Pasani, K. (2025). *Evaluating Dependency Vulnerabilities in React Native Applications*. Bachelor thesis, Technische Universität Darmstadt. DOI: [10.26083/tuprints-00031438](https://doi.org/10.26083/tuprints-00031438). License: CC BY 4.0.
