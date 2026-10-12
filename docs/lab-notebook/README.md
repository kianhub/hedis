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
| [related-work.md](./related-work.md) | Prior art and newer verified references |
| [paper-outline.md](./paper-outline.md) | Outline for an extended paper beyond the thesis |
| [log/](./log/) | Chronological experiment and decision log |

## Repository entry points

- Go CLI: `go/hermes-decompiler/` (`hermes-decompiler analyze`, `maintain-database`, `disassemble`)
- Fingerprint pipeline: `pipeline/react-natives/rn069`–`rn079` (11 React Native template apps)
- Package/advisory harvesting: `js/get-packages/`

## Citation

Pasani, K. (2025). *Evaluating Dependency Vulnerabilities in React Native Applications*. Bachelor thesis, Technische Universität Darmstadt. DOI: [10.26083/tuprints-00031438](https://doi.org/10.26083/tuprints-00031438). License: CC BY 4.0.
