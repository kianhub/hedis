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

## Methods and groups (added 2026-10-15 16:40 CT, before any v7 matcher run)

### Three methods, same apps
1. **M37:** `bin/hedis-m37` sha256 `647bfeadac67671d94c9fe2aa1a3bdcea57f1c0263a8d963e426af9ec61c29d7` @ 359ec3f. Config from `data/m37_frozen.json` and research/v6 MANIFEST. `HEDIS_MAX_OID=6ad0ea700000000000000000`, `HEDIS_RN_FALLBACK=nearest`, with **no** `--unsupported-cov`.
2. **M38:** `bin/hedis-m38` sha256 `cc3f2b3e39fc12de16e088dd578b6e5506c8528fe8441ca8b08c729ea269877c` @ 7db3c19. `HEDIS_MAX_OID=6ad13ed00000000000000000`, `HEDIS_RN_FALLBACK=nearest`, M37 flags plus `--unsupported-cov 0.40`.
3. **t35** (thesis baseline): run at both pins, 6ad0ea70… (paired with M37) and 6ad13ed0… (paired with M38), with `HEDIS_RN_FALLBACK=nearest`. Apps where t35 crashes are reported separately as a fair subset.

All three methods run once on exactly the same frozen apps. The **fill effect** is M37 vs M38 on the fill group.

### Groups (stratified; sizes are recorded in GROUPS.md before any matcher run)
1. **fill:** app RN minor is 0.72, 0.75, 0.78 or 0.80, where the M38 fingerprint fill landed.
2. **control:** app RN minor is 0.74 or 0.76 (dev-like RN; isolates app-mix effects).
3. **unsupported_hbc98:** app RN minor is 0.85 (HBC 98). This group is dumped with the HBC 98 path, either `hedis-m37` 647bfead or `hedis-m38`.
4. **fallback:** any other RN minor (not in {0.72, 0.74, 0.75, 0.76, 0.78, 0.79, 0.80} and not 0.85).

Also reported: "covered at pin", meaning the RN minor has fingerprints in `hedis_rel` at each method's pin (queried before scoring).

### RN 0.85 group scoring rule (pre-registered, verbatim)
RN 0.85 group scoring rule (pre-registered): an `unsupported_bytecode` / unsupported flag = **correct abstention**; report in its **own table**; those apps are **excluded from exact F1** (neither inflate nor drag).

## Appended candidates, batch 1 (2026-10-15 16:42 CT; sha256 order, after the initial queue)
| sha256[:12] | app | repo | tag | RN | intended group |
|---|---|---|---|---|---|
| 44518b81b983 | safe-1.54.0 | safe-global/safe-wallet-monorepo | v1.54.0 | 0.76.3 | control |
| 4efaf8ed6c54 | xaman-2.8.0 | XRPL-Labs/xaman-app | v2.8.0 | 0.72.7 | fill |
| 725bdcba4f09 | temple-1.54 | madfish-solutions/templewallet-mobile | 1.54(1476) | 0.72.17 | fill |
| a933e29f9d42 | bbplayer-2.5.2 | bbplayer-app/BBPlayer | v2.5.2 | 0.85.3 | unsupported_hbc98 |

One tag per app: if bbplayer-2.5.2 (0.85) builds, the earlier bbplayer-2.4.4 (0.83) attempt is dropped from the set; the 0.85 group is prioritized per the Optimizer addendum.

## Target group sizes (committed 2026-10-15 before further sourcing; stopping rule locked)
Do not score until these minima are met (or MANIFEST records an explicit shortfall waiver after pool exhaustion):
- **fill** (RN 0.72 / 0.75 / 0.78 / 0.80): **≥4** apps
- **control** (RN 0.74 / 0.76): **≥3** apps
- **unsupported_hbc98** (RN 0.85): **≥2** apps
Fallback apps may be kept for coverage but do not count toward the three minima above.

## Dropped candidates (auditable; every non-kept attempt)
| app | reason |
|---|---|
| memmy-1.1 | git dep gitpkg.now.sh expo-image 402 Payment Required |
| blixt-0.6.9 | protobuf codegen failure |
| cherry-0.2.1 | workspace webview asset not built |
| olas-0.1.9 | pnpm lock outdated |
| berty-2.471.14 | RN 0.64 CLI missing |
| jellyfin-1.8.0.5 | build failed (rc=3 then rc=4 on retry; RNBUNDLE workaround insufficient) |
| bbplayer-2.4.4 | build failed; superseded by bbplayer-2.5.2 if 0.85 group priority |
| backpack-0.5.2 | build failed (rc=3; RNBUNDLE workaround insufficient) |
| bcwallet-1.0.12 | build failed (rc=2; Node 18 workaround insufficient) |
| station-2.0.0 | build failed (rc=3 then rc=4; openssl-legacy insufficient) |
| temple-1.54 | build failed (rc=3; @airgap/beacon-dapp linked to absent node_modules/.cache/null); retry2 with empty stub at node_modules/.cache/null still unresolved by metro → dropped |

## bbplayer-2.5.2 dumper failure (pre-registered measurement)
Build OK (HBC 98). Detected rnversion empty at dump time → **dumper failure** (do not feed package.json RN to the matcher or dump `--rn-version`). `package.json` reports `react-native@0.85.3` and may label the app into **unsupported_hbc98** in this MANIFEST / GROUPS only. Quarantined any triples produced after a package.json rnversion patch (`triples/bbplayer-2.5.2.triples.INVALID_patched_rnversion`). Matcher must detect unsupported bytecode without that label.


## Retry 2 (env-only workarounds, one retry each)
- safe-1.54.0: entry shim `hedis_entry.js` = `import 'expo-router/entry'` (package.json main) → **BUILD_OK** (HBC 96, RN 0.76, control). Removed from drop table.
- temple-1.54: empty stub module at `node_modules/.cache/null` → still exit 3 → dropped (see table).

## Append batch 2 (sourced 2026-10-15; fixed tags; queue in sha256(repo@tag) order; burned apps excluded: bluesky, mattermost, rainbow, expensify, standardnotes, zeus, eigen, rocketchat …)
| sha256(repo@tag)[:16] | app | repo | tag | package.json RN | intended group |
|---|---|---|---|---|---|
| 0c9348e504ac4a83 | gsy-4.0.2 | CarGuo/GSYGithubAPP | v4.0.2 | 0.80.2 | fill |
| 16fa113c8650d62a | metamask-7.46.2 | MetaMask/metamask-mobile | v7.46.2 | 0.72.15 | fill |
| 573465a1d40c5ca3 | breathly-ios-2.2-18 | mmazzarolo/breathly-app | ios-2.2-(18) | 0.85.3 | unsupported_hbc98 |
| 691231e1fa8f9d24 | blink-beta-3.0.31 | GaloyMoney/blink-mobile | beta-3.0.31 | 0.85.3 | unsupported_hbc98 |
| 77d57e0ba669a20c | daimo-audit-1 | daimo-eth/daimo (apps/daimo-mobile) | audit-1 | 0.72.4 | fill |
| c082e9da1bc5b562 | bookcars-7.0 | aelassas/bookcars (mobile) | v7.0 | 0.76.9 | control |
| c480b810cbb46764 | vega-3.1.1 | vega-org/vega-app | v3.1.1 | 0.74.1 | control |
| c81d05e441402407 | jfaudio-2.4.4 | leinelissen/jellyfin-audio-player | v2.4.4 | 0.74.3 | control |

Sourcing note: no non-burned public app found on RN 0.75 or 0.78 (searched GitHub code search + tag scans of ~30 repos; 0.75/0.78 hits were Expensify/standardnotes, both burned). Package.json RN is the intended-group label only; group assignment for scoring uses the build-detected rnversion.
