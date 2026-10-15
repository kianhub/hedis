# v7 real-app held-out set — MANIFEST (pre-registration; never matched or scored)

## Status
Freeze intent recorded 2026-10-15 (CT) before any v7 build finished. No matcher run, no scoring.

## Scoring config (pre-registered; M38 frozen 2026-10-15)
- Matcher: `bin/hedis-m38` sha256 `cc3f2b3e39fc12de16e088dd578b6e5506c8528fe8441ca8b08c729ea269877c` @ `7db3c19` (v2 merge c463fd2 / 8e8176e).
- DB pin: `HEDIS_MAX_OID=6ad13ed00000000000000000`.
- Env: `HEDIS_RN_FALLBACK=nearest`.
- Flags: M37 frozen flags plus `--unsupported-cov 0.40`, i.e. `packages -d --norm nsnc --thresholds data/thresholds_m4f_dlr.json --lowuniq 2 --lowdf 2 --dep 0.6 --deps data/deps.json --rootver --floor 0.2 --vset-margin 0.003 --vset-samekeys --unsupported-cov 0.40`.
- PRIMARY scorer: `oneclass.py --subsha` (release-identity tables). SECONDARY: `m35_eval.py` (vulnerability-level only).
- Baseline: thesis method t35, same pin + `HEDIS_RN_FALLBACK=nearest`; apps where t35 crashes reported separately (fair subset) as in v6.
- Groups: **covered** = app RN minor has fingerprints in `hedis_rel` at the pin above; **fallback** = rest; HBC 98 apps form their own group. Group sizes are written here before scoring.
- Run once; no retuning.

### Selection bias for M38 (added 2026-10-15 16:36 CT, before any scoring)
v7 is meant to measure M38, whose fingerprint fill landed on RN minors 0.72, 0.75, 0.78, 0.80. Candidates appended after the initial queue are restricted to apps on those minors (or whose nearest fallback lands on them), each with a fixed tag, attempted in sha256(repo@tag) order after the initial queue. Already-built apps (yoroi 0.71, readany 0.81) stay.

## Exclusions (burned / never held-out)
All apps in split.v3 (real + real_dev), v4 real_test, v5 real_test, v6 real_test, and their app families; Notesnook and Uniswap forever.

## Selection rule (fixed before builds)
Candidate pool = OSS RN apps never in v3–v6 splits, one fixed tag per app chosen for RN-minor diversity (`pool.txt`).
Deterministic order: sort by `sha256("<github repo>@<tag>")` ascending (`queue.txt`). Attempt builds in that order and keep every BUILD_OK until 12 (minimum 8 if disk forces).
If the pool is exhausted with < 12, new candidates may only be appended (each with a fixed tag) and are attempted in their own sha256 order after the current queue; nothing already attempted is re-ranked.
Env-only build workarounds (as v5/v6) are allowed; app logic is never changed. Each workaround is logged.

## Build method
Same as v5/v6: `/workspace/hedis/eval/build_oss.sh` (clone at tag, lockfile install, release Metro bundle + source map, app's own hermesc -O), truth = `bundled_truth.py` from the source map, triples via `bin/hedis-norm-hits` a29d0be7 (HBC 98 apps via `bin/hedis-m37` 647bfead, with a byte-identical cross-check on one non-HBC98 app if mixed).

## Candidate queue (sha256 order)
| rank | sha256(repo@tag)[:12] | app | repo | tag | subdir |
|---|---|---|---|---|---|
| 1 | 142eaf28a69a | memmy-1.1 | Memmy-App/memmy | 1.1 | . |
| 2 | 296bf6af2fe1 | blixt-0.6.9 | hsjoberg/blixt-wallet | v0.6.9-420 | . |
| 3 | 5fb1afbb556b | cherry-0.2.1 | CherryHQ/cherry-studio-app | v0.2.1 | . |
| 4 | 7fe79b5725a1 | jitsi-9.1.0 | jitsi/jitsi-meet | android-sdk-9.1.0 | . |
| 5 | 861708d9e753 | jellyfin-1.8.0.5 | jellyfin/jellyfin-expo | v1.8.0.5 | . |
| 6 | 8acac29ffa91 | olas-0.1.9 | pablof7z/olas | 0.1.9 | apps/mobile |
| 7 | 8cd0c69a438d | bbplayer-2.4.4 | bbplayer-app/BBPlayer | v2.4.4 | apps/mobile |
| 8 | b3bae656dc27 | backpack-0.5.2 | coral-xyz/backpack | v0.5.2 | packages/app-mobile |
| 9 | be17045c91f1 | bcwallet-1.0.12 | bcgov/bc-wallet-mobile | v1.0.12 | app |
| 10 | ce0cfda74bff | yoroi-4.12.0 | Emurgo/yoroi | v4.12.0 | apps/wallet-mobile |
| 11 | daab3a88d0e4 | ogam-0.0.49 | off-grid-ai/OGAM | v0.0.49 | . |
| 12 | e189e878e341 | berty-2.471.14 | berty/berty | v2.471.14 | js |
| 13 | ea273dc86d6c | readany-1.3.4 | codedogQBY/ReadAny | v1.3.4 | packages/app-expo |
| 14 | f714bfc5be1d | station-2.0.0 | terra-money/station-mobile | v2.0.0 | . |
