# v9 real-app held-out set — MANIFEST (pre-registration; never matched or scored)

## Status
**FROZEN 2026-10-16 ~17:20 CT** (split.v9.json sha256 a32b9b14f1c05dba7515355554904b407ee54fddefecd8ecabfa9e59f173125c; rn085 5 / control 4; no matcher run, no scoring). **LOCKED 2026-10-16 ~16:55 CT.** Optimizer named the M40 RN 0.85 DEV set (Hedis v2 room, 2026-10-16): no new apps; DEV = blink-beta, breathly, zeus + 7 v8 apps (marlin, imagetopdf, dothething, minmobile, wixrnn, readtrace, ysclaude), all already burned. M40 gets no cutoff until v9 is frozen. Criteria, sizes and exclusions below are final. Drafted 2026-10-15 ~21:15 CT; no builds started; no freeze commit. No matcher run, no scoring.
Reason: v8 is **burned** (scored once at M39; rn085 F1 0.373; research/v8/RESULT.md, commit a2aab03). Optimizer is starting **M40** (version-choice improvements for safe-area-context, screens, reanimated, gesture-handler), tuning on **DEV only**, and will widen 0.85 DEV beyond blink/breathly/zeus. v9 must be frozen before anyone scores M40.

## Hypothesis
Measure whether M40 version-choice changes improve RN **0.85** (HBC 98) exact matching on never-seen apps, vs t35, without regressing 0.74/0.76 controls.

## Pre-registered success criteria (locked before any build / before M40 cutoff)
Score **once**, only after Optimizer posts the M40 cutoff (matcher commit + DB pin + flags). **Never retune on v9.**
1. **rn085 (primary):** exact F1 via `oneclass.py --subsha` vs **t35** on the same apps. t35 crashes on HBC 98 — report t35 **unsupported/crash rate separately** (and t35 F1 on its fair subset only). Also report M40 unsupported-flag rate (unsupported / unsupported_bytecode abstains).
2. **Controls (RN 0.74 / 0.76):** per-app exact F1 vs **M39** and **t35**; expected ~no change; any per-app drop is reported, not hidden.
3. Optional fallback group allowed for coverage; does not count toward minima.
4. After freeze: Optimizer must **not** open v9 apps/dumps until scoring.

## Target group sizes (locked before builds)
- **rn085** (RN 0.85, HBC 98): **≥5 NEW** apps (never in v3–v8 held-outs, DEV, or burned families)
- **control** (RN 0.74 or 0.76): **≥3 NEW** apps
Do not score until minima are met (or an explicit shortfall waiver is recorded here after pool exhaustion).

## Exclusions (burned for v9 held-out) — see burned.txt
- Everything burned for v8 (v3–v7 held-outs + families, Notesnook, Uniswap forever).
- **v8 held-out (now burned):** dothething, imagetopdf, marlin, minmobile, readtrace, wixrnn, ysclaude, appcues, gorhom, solana.
- **0.85 DEV:** blink-beta (and Galoy/Blink family), breathly, zeus.
- **M40 DEV (named 2026-10-16):** no new apps; v8 rn085 apps reused as DEV (already burned above).
- Prefer to avoid v8 pool failures (rnrestart, paper, streamchat, rnfirebase) — not strictly burned.
- Family burn list (abbrev.): mattermost, rainbow, bluesky, inaturalist/seek, valora, bluewallet, leather, alby, karakeep, trezor, expensify, rabby, edge, eigen, chatwoot, tooot, converse, rocketchat, graysky, subwallet, standardnotes, bitpay, showtime, pillar, ecency, spacedrive, alephium, minds, hexa, zeus, zingo, cozy, enuts, keeper, lxmusic, mangareader, pixelfed, pubky, jitsi, xaman, daimo, gsy, safe, jfaudio, vega, yoroi, readany, ogam, blink, breathly, bbplayer, notesnook, uniswap, memmy, blixt, cherry, olas, berty, jellyfin, backpack, bcwallet, station, temple, metamask, bookcars.

## Selection rule (same as v8)
Candidate pool = OSS RN apps not burned, one fixed tag each. rn085 = package.json/lock RN 0.85.x; control = 0.74.x/0.76.x.
Deterministic order: `sha256("<github repo>@<tag>")` ascending; attempt in that order. Pool exhaustion → append-only batches (fixed tags), own sha256 order, after current queue; nothing re-ranked. Env-only build workarounds allowed and logged; app logic never changed. Pool in pool_meta.tsv, queue in queue.txt.

## Build method
`/workspace/hedis/eval/build_oss.sh` (clone at tag, lockfile install, release Metro bundle + map, app's own hermesc -O); truth = `bundled_truth.py`; triples via `bin/hedis-norm-hits` (HBC 98 via `bin/hedis-m37`). Artifacts → research/v9/frozen + triples; apps/oss clone deleted after each build.

## RN 0.85 dumper rule
Empty `rnversion` at dump = **dumper failure**. Do **not** patch from package.json. package.json RN labels groups only.

## Scoring config
Matcher / DB pin / flags: **TBD — wait for Optimizer's M40 cutoff.** PRIMARY `oneclass.py --subsha`; baseline t35; groups rn085/control/(fallback). Run once.

## Candidate queue
Burn list final; queue = pool sorted by sha256(repo@tag), recorded in queue.txt with its sha256 before the first build.
