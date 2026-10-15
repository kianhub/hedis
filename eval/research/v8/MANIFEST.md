# v8 real-app held-out set — MANIFEST (pre-registration; never matched or scored)

## Status
Freeze intent recorded 2026-10-15 (~18:25 CT) **before** any v8 build finished and **before** Optimizer posts an RN 0.85 fingerprint cutoff. No matcher run, no scoring.

## Hypothesis
Measure RN **0.85** fingerprint matching once Optimizer posts the 0.85 fill cutoff. Until that cutoff exists, this set is frozen and closed to Optimizer (same discipline as v7).

## Pre-registered success criteria (before any 0.85 cutoff exists)
Locked before scoring; score **once** only after Optimizer posts the 0.85 cutoff; **never retune on v8**.

1. **0.85 group (primary):** exact F1 via `oneclass.py --subsha` vs **t35 baseline** on the **same** apps. Also report **unsupported-flag rate** (how often the matcher still abstains with unsupported / unsupported_bytecode).
2. **Controls (RN 0.74 / 0.76):** report exact F1 vs M38 / t35 as usual; expect little change from a cutoff that only adds 0.85 fingerprints.
3. Optional small fallback group may be kept for coverage but does not count toward the minima below.
4. After MANIFEST + checksums freeze: Optimizer must **not** open v8 apps/dumps until scoring.

## Target group sizes (locked before finishing builds)
- **rn085** (RN 0.85, HBC 98): **≥5** apps that are **NEW** (never in v3–v7 held-outs / burned lists)
- **control** (RN 0.74 or 0.76): **≥3** apps, also new / not burned
- Optional small fallback OK but not required

Do not score until these minima are met (or MANIFEST records an explicit shortfall waiver after pool exhaustion).

## Exclusions (burned / never held-out for v8)
All apps in splits v3–v7 (including families), Notesnook and Uniswap forever.

### v7 held-out (burned for v8 held-out)
jitsi, xaman, daimo, gsy, safe, jfaudio, vega, yoroi, readany, ogam, blink-beta, breathly, bbplayer (and earlier v6/v5/v4/v3 families).

### 0.85 DEV only (burned for held-out; Optimizer may tune on these + synth only)
- `blink-beta-3.0.31`
- `breathly-ios-2.2-18`
- `zeus-13.1.3`

These **must NOT** appear in v8 held-out.

### Family burn list (abbrev.)
mattermost, rainbow, bluesky, inaturalist/seek, valora, bluewallet, leather, alby, karakeep, trezor, expensify, rabby, edge, eigen, chatwoot, tooot, converse, rocketchat, graysky, subwallet, standardnotes, bitpay, showtime, pillar, ecency, spacedrive, alephium, minds, hexa, zeus, zingo, cozy, enuts, keeper, lxmusic, mangareader, pixelfed, pubky, jitsi, xaman, daimo, gsy, safe, jfaudio, vega, yoroi, readany, ogam, blink, breathly, bbplayer, notesnook, uniswap. Also avoid v7-attempted families where possible: memmy, blixt, cherry, olas, berty, jellyfin, backpack, bcwallet, station, temple, metamask, bookcars.

## Selection rule (fixed before builds)
Candidate pool = OSS RN apps never in v3–v7 held-outs, one fixed tag per app.
- Prefer confirmed `package.json` / lock RN **0.85.x** for the rn085 group; **0.74.x / 0.76.x** for controls.
- Deterministic order: sort by `sha256("<github repo>@<tag>")` ascending. Attempt builds in that order.
- If the pool is exhausted under target, new candidates may only be appended (each with a fixed tag) and are attempted in their own sha256 order after the current queue; nothing already attempted is re-ranked.
- Env-only build workarounds (as v5–v7) are allowed; app logic is never changed. Each workaround is logged in the drop/audit table.

## Build method
Same as v7: `/workspace/hedis/eval/build_oss.sh` (clone at tag, lockfile install, release Metro bundle + source map, app's own hermesc -O), truth = `bundled_truth.py` from the source map, triples via `bin/hedis-norm-hits` (HBC 98 apps via `bin/hedis-m37`).

## RN 0.85 dumper rule (pre-registered)
Empty `rnversion` at dump = **dumper failure**. Do **not** patch from `package.json` into matcher input / dump `--rn-version`. `package.json` RN may label the group in this MANIFEST / GROUPS only.

## Scoring config (placeholders until Optimizer posts 0.85 cutoff)
- Matcher / DB pin / flags: **TBD — wait for Optimizer’s 0.85 cutoff announcement.** Do not invent a cutoff.
- PRIMARY scorer: `oneclass.py --subsha`. SECONDARY: `m35_eval.py` (vulnerability-level only) if used as in v7.
- Baseline: thesis method **t35** on the same apps (fair subset if t35 crashes).
- Groups: **rn085** / **control** / optional **fallback**. Group sizes written here and in GROUPS.md before scoring.
- Run once; no retuning.

## Candidate queue (sha256 order; locked 2026-10-15 ~18:30 CT before builds)
| rank | sha256(repo@tag)[:16] | app | repo | tag | package.json RN | intended group |
|---|---|---|---|---|---|---|
| 1 | 3a7830fe0022717c | marlin-1.3.0 | fwille/marlin | v1.3.0 | 0.85.3 | rn085 |
| 2 | 5435358af0c63f61 | imagetopdf-1.1 | nhankun2006/Image-2-PDF | release_v1.1 | 0.85.3 | rn085 |
| 3 | 89e30f01982f878d | rnrestart-0.0.29 | avishayil/react-native-restart | v0.0.29 | 0.85.3 | rn085 |
| 4 | a7e18e2748e25415 | dothething-1.2.1 | BigDadEnergy39/do-the-thing | v1.2.1 | 0.85.3 | rn085 |
| 5 | ad1c133e9e02d904 | appcues-1.0.12 | appcues/mobile-pattern-showcase-app | 1.0.12 | 0.74.5 | control |
| 6 | ada9d316867932dc | minmobile-1.0.62 | gooseDes/min-mobile | v1.0.62 | ^0.85.3 | rn085 |
| 7 | ca45c1d1678279b4 | wixrnn-8.8.7 | wix/react-native-navigation | 8.8.7 | 0.85.2 | rn085 |
| 8 | d7d951b513af4936 | solana-3.0.0 | solana-mobile/solana-mobile-expo-template | v3.0.0 | 0.76.9 | control |
| 9 | db7f7e03bd82c8cc | paper-5.13.2 | callstack/react-native-paper | v5.13.2 | 0.76.3 | control |
| 10 | f9b28386a938c499 | gorhom-5.1.4 | gorhom/react-native-bottom-sheet | v5.1.4 | 0.76.7 | control |
| 11 | fb4ec4c9e5afa3dd | readtrace-1.0.6 | wberry9813/ReadTrace | v1.0.6 | 0.85.3 | rn085 |
| 12 | fbc189f000ef03c2 | ysclaude-1.0.9 | winter-bit-cry/YSClaude | v1.0.9 | 0.85.3 | rn085 |

Selection note: GaloyMoney/galoy-mobile@3.0.41 is RN 0.85 but **excluded** (same family as burned blink / Galoy→Blink). Library playgrounds (wixrnn, rnrestart Example, paper/gorhom example, solana template) are allowed as OSS Hermes apps with fixed tags; package.json RN labels the intended group only.


## Dropped candidates (auditable)
*(every non-kept attempt logged here)*

## Build caveats
*(env-only workarounds logged here)*

