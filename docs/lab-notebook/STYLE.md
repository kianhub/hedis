# Lab notebook writing style

This notebook follows the research writing conventions documented in the authoritative guide at the repository root of this effort: **Kian Pasani, Research Writing Style Guide (for Hedis Lab Notebook)**, derived from the bachelor thesis *Evaluating Dependency Vulnerabilities in React Native Applications* (TU Darmstadt, DOI [10.26083/tuprints-00031438](https://doi.org/10.26083/tuprints-00031438)).

## Quick rules

- **Language:** English only.
- **Person:** Academic **we** for our actions and findings; never **I**. Use impersonal or passive phrasing for background mechanisms.
- **Tone:** Sober, technical, and self-critical. We state limitations plainly and do not oversell. Negative results stay in the record with the reason we abandoned or deferred an approach.
- **Sentences:** Medium length (roughly 20–35 words), often with a qualifying clause (*While…*, *Although…*, *, which…*). Embed concrete versions, counts, and dates in the prose.
- **Acronyms:** Define once, then use the short form (e.g., React Native (RN), function object (FO), ahead-of-time (AOT)).
- **Hedging:** Observations are stated flatly; interpretations use *may*, *could*, *potentially*, *plausible*, *appear*, *indicate*.
- **Formatting:** Bold run-in labels in lists (`**Structural (S-AR):** …`). Tables for per-app or per-metric results. Code blocks for commands, bytecode snippets, and config. Explicit cross-links to other notebook pages.
- **Avoid:** Contractions, emoji, exclamation marks, rhetorical questions, and casual asides.

## Per-entry checklist

Each log entry should include:

1. **Hypothesis or goal** (which research question or milestone we address).
2. **Setup** (RN/Hermes versions, commit hash, paths, tooling).
3. **Observations** (what we measured or built).
4. **Interpretation** (hedged).
5. **Decision** and **next steps**.

When citing the thesis, prefer section and table references over paraphrase. Mark any metric we have not re-run in this repository as **unverified**.
