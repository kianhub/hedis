# 2026-10-12 — Thesis PDF verification (Section 5.3)

We obtained the publisher PDF for Pasani (2025) from TUprints (DOI [10.26083/tuprints-00031438](https://doi.org/10.26083/tuprints-00031438)) after completing the Anubis challenge in a browser session. This entry records which notebook claims we upgraded from **unverified** to **thesis-cited**.

## Observations

- We transcribed **Tables 5.1–5.6** and application metadata into [baseline-thesis.md](../baseline-thesis.md).
- **HMV** is defined in Section 5.3 as the highest matching version by **Matched(%)**, not by raw matched FO count alone.
- Reference database totals: **1,220** targets, **952** packages processed, **16,579,257** FOs, **16.06 GB**, **~76 h** (Table 5.6 narrative).
- Section **6.1** confirms the thesis does **not** report P/R/F1 and states that **no HMV equaled declared versions** in controlled open-source apps, despite package-level detections.

## Dataset registry updates

We aligned [dataset-registry.md](../dataset-registry.md) with thesis counts: **1,717** Directory packages, **73,293** expanded npm versions, GHSA snapshot **2025-07-10** (**289,277** vulnerable npm packages corpus), **17** Directory packages with advisories, and **1,220** refined fingerprint targets.

## Interpretation

This verification does not by itself reproduce the thesis runs in our workspace; it only grounds the notebook prose in primary-source tables. Full rerun still requires the **16.06 GB** database or an equivalent rebuild.

## Next steps

- [ ] Restore or rebuild the thesis-scale MongoDB dump, or complete the reduced eval DB (see [2026-10-12-baseline-reproduction.md](./2026-10-12-baseline-reproduction.md)).
- [ ] Import `run_eval.sh` and map its inputs to D13 in the dataset registry.
