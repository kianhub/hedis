# Related work

We summarize prior art cited in the thesis and add newer work we can verify independently. The thesis bibliography is authoritative for in-corpus citations; bracket numbers below mirror common thesis ordering where known from the style guide, not a full re-export of the reference list.

## Supply chain risk in npm and JavaScript

Zimmermann et al. characterize the npm ecosystem as densely connected, which creates systemic weak spots for supply-chain attacks[^zimmermann]. This motivates bytecode-level fingerprinting when `package.json` is unavailable (e.g., App Store IPAs).

**Limit:** Ecosystem graphs and registry metadata do not by themselves reveal which packages are embedded in a Hermes-compiled React Native bundle.

## React Native security plugins and manifest-based tools

Falvo (2023) discusses plugin-oriented approaches to React Native security[^falvo]. **While** such work advances runtime and configuration checks, **in contrast**, our work (and the thesis) targets **production iOS binaries** where JavaScript source and manifests are absent.

## Hermes bytecode tooling

- **P1Sec hermes-dec** — Disassembly inspiration for Hedis; human-readable Hermes reverse engineering[^hermesdec].
- **ReuNify (Liu et al., ASE 2023; journal extension ACM TOSEM 2024)** — Converts Hermes bytecode (via tooling such as hbctool/Hermeser) into Soot Jimple for whole-program static analysis on **Android** RN apps[^reunify][^tosem]. **While** ReuNify improves callgraph and taint coverage, **it does not** solve npm package attribution or GHSA-level vulnerability matching on iOS App Store builds.
- **Hermes engine documentation** — AOT bytecode is default in modern RN releases[^hermesdoc], which is why fingerprinting operates on `.hbc` rather than plain JavaScript.

## Dependency and vulnerability scanners (traditional)

Tools that require `package.json`, lockfiles, or source (npm audit, Snyk, OWASP Dependency-Check, etc.) remain the audit-grade baseline **when** manifests exist. The thesis positions Hedis for the **inverse** problem: coarse dependency inference from Hermes output only.

## Mobile app store and binary analysis

iOS IPA extraction and `main.jsbundle` recovery follow standard mobile forensics practice (thesis footnotes cite App Store URLs and GitHub release tags). **Android** RN static analysis has received additional attention (ReuNify, TOSEM 2024 cohort studies on RN Android malware opcode usage).

## Newer verified references (post-thesis corpus)

| Work | Venue / year | Relevance to Hedis |
|------|----------------|-------------------|
| Liu et al., *ReuNify: A Step Towards Whole Program Analysis for React Native Android Apps* | ASE 2023; arXiv:2309.03524 | Hermes → IR for static analysis; complementary, not competitive for npm fingerprint DB |
| *Demystifying React Native Android Apps for Static Analysis* (ReuNify journal paper) | ACM TOSEM, DOI [10.1145/3702977](https://dl.acm.org/doi/10.1145/3702977) | Large-scale RN Android evaluation; opcode and malware statistics |
| React Native Hermes docs | reactnative.dev, current | Confirms Hermes as default engine; bundling → bytecode in release builds |

We have **not** yet surveyed 2025–2026 commercial SBOM-on-mobile products in depth; any future entry should cite primary vendor documentation or peer-reviewed evaluations.

## Positioning statement (for papers)

**While** whole-program RN analyzers extend taint and privacy analysis, **our** line of work (Hedis) optimizes for **scalable npm package fingerprint matching** against GHSA-oriented reference data on Hermes bundles, accepting coarse version resolution in exchange for store-deployable inputs. **However**, as the thesis and 2026-10-12 lockfile scoring show, audit-grade package+version F1 requires richer version coverage and rigorous held-out evaluation—areas we treat as active engineering rather than solved research.

## Bibliography (partial)

[^zimmermann]: Zimmermann et al., npm ecosystem security (thesis [39]; full bibliographic record in thesis PDF).
[^falvo]: Falvo (2023), React Native security plugin work (thesis [12]).
[^hermesdec]: P1Sec, hermes-dec, https://github.com/P1Sec/hermes-dec
[^reunify]: Liu et al., ReuNify, ASE 2023, https://arxiv.org/abs/2309.03524
[^tosem]: ACM TOSEM, Demystifying React Native Android Apps for Static Analysis, https://dl.acm.org/doi/10.1145/3702977
[^hermesdoc]: React Native docs, Using Hermes, https://reactnative.dev/docs/hermes
