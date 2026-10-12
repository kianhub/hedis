# Experiment protocol (template)

> Copy this file to `log/YYYY-MM-DD-short-title.md` and complete every section before starting a run. See [STYLE.md](./STYLE.md).

## Metadata

| Field | Value |
|-------|-------|
| **Date** | YYYY-MM-DD (timezone) |
| **Authors** | |
| **Research question / milestone** | e.g., RQ2 version precision, eval harness v2 |
| **Git commit** | `git rev-parse HEAD` |
| **Run settings** | Agent or human runner, model, effort level, host (e.g., Hedis Optimizer on Grok Bot, effort high; no cloud agents); commit author if delegated |
| **Related log entries** | |

## Hypothesis

We hypothesize that …

## Setup

- **Environment:** OS, `OS_HERMES`, MongoDB URI (if any), Node/Go versions
- **Input artifacts:** IPA/HBC paths, lockfiles, database snapshot id
- **Tooling flags:** `hermes-decompiler` command line
- **Controls:** baseline filtering on/off, fuzzy matching threshold

## Dataset

Which rows from [dataset-registry.md](./dataset-registry.md) apply? Include RN version, Hermes bytecode version, and app or package identifiers.

## Configuration

```bash
# Commands, .env keys (redact secrets), pipeline batch ids
```

## Metrics

Define primary and secondary metrics before the run (e.g., package-level F1, package+version F1, match rate on structural / content IR hashes).

## Results

Tables and observations only. Do not interpret here.

| Metric | Value | Notes |
|--------|-------|-------|
| | | |

## Interpretation

We note that … (hedged). This may indicate …

## Decision

We decided to … because …

## Next steps

- [ ] …

## Artifacts

Paths to outputs, screenshots, or external stores (S3, Mongo dump name).
