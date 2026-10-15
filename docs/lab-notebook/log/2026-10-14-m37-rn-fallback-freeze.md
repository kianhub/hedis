# 2026-10-14 — M37: nearest-RN fallback and freeze

**Frozen matcher:** `359ec3f`. **Freeze record:** `876e043`. **Manifest:** `data/m37_frozen.json`.

## Design choice: nearest RN vs union

We compared fallback strategies while **hiding each evaluation app’s own RN fingerprints** from the matcher (so the app cannot match against its native sandbox slice).

| Strategy | Real-dev F1 | Synthetic F1 |
|----------|-------------|----------------|
| Nearest RN | **0.830** | 0.916 |
| Union of RN slices | 0.813 | **0.929** |

We selected **nearest-RN fallback** for M37 production configuration because it improved real-dev F1 relative to union (**0.830** vs **0.813**), accepting the small synthetic regression (**0.916** vs **0.929**).

## Engineering deliverables

- RN fingerprint coverage for **0.72**, **0.75**, **0.78**, and **0.80** (addresses v5 crash class).
- Hermes bytecode **v98** support in commit **`e9f0e06`**.
- Crash regression test **`0ec9f50`**.

## Development metrics (post-freeze)

| Corpus | F1 |
|--------|-----|
| Real-dev | **0.840** |
| Synthetic | **0.970** |

**Database pin:** `6ad0ea70…` (M37 cutoff).

## Next steps

- [ ] Hold out v6 freeze for a single scored pass (see [2026-10-15-v6-held-out.md](./2026-10-15-v6-held-out.md)).
