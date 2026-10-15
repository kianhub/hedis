# v7 RESULT (scored once, 2026-10-15, after freeze adbc5e5)
Frozen split `data/split.v7.json` sha256 2d6cbe8c553a2a3f19911e8831bb0f5b405eb302f74e083b335eb202a3c3c38b; 13 apps. Nothing was retuned. All numbers come from this one run.

Methods (same 13 apps for all):
- **M37:** bin/hedis-m37 647bfead…, HEDIS_MAX_OID=6ad0ea70…, frozen run_m37 flags (`--norm nsnc … --floor 0.2 --vset-margin 0.003 --vset-samekeys`), no `--unsupported-cov`. The binary has no such flag.
- **M38:** bin/hedis-m38 cc3f2b3e…, HEDIS_MAX_OID=6ad13ed0…, same flags + `--unsupported-cov 0.40`.
- **t35a / t35b:** bin/hedis-vset 2c863251… + thesis_vset.py on the frozen .hbc, at the M37 pin (a) and the M38 pin (b).
- HEDIS_RN_FALLBACK=nearest for all. Runner scripts: score_v7.sh, t35_v7.sh.
- Covered RN minors are identical at both pins: 0.72, 0.74, 0.75, 0.76, 0.78, 0.79, 0.80.

## PRIMARY: oneclass.py --subsha, exact F1 (10 apps; the RN 0.85 group is excluded per the pre-registered rule)
| subset | M37 | M38 | t35a | t35b |
|---|---|---|---|---|
| all 10 (exact) | P .455 R .489 **F1 .471** (45/54/47) | P .566 R .609 **F1 .586** (56/43/36) | F1 .173 (30/225/62) | F1 .176 (31/229/61) |
| fill (4) | F1 .346 (9/19/15) | **F1 .549** (14/13/10) | F1 .101 | F1 .129 |
| control (3) | F1 .703 (26/12/10) | F1 .703 (26/12/10) | F1 .283 | F1 .262 |
| fallback (3) | F1 .308 (10/23/22) | F1 .485 (16/18/16) | F1 .136 | F1 .133 |
| t35 fair subset (9; no yoroi, where t35 crashed) | F1 .533 | F1 .598 | F1 .180 | F1 .183 |
(tp/fp/fn)

## Fill effect, per app (M37 → M38). Locked limit: fill = 3×0.72 + 1×0.80; do not generalize to RN 0.75/0.78
| app | RN | M37 F1 | M38 F1 | t35a | t35b |
|---|---|---|---|---|---|
| jitsi-9.1.0 | 0.72 | .105 (1/9/8) | **.421** (4/6/5) | .105 | .158 |
| xaman-2.8.0 | 0.72 | 0 (0/3/0) | 0 (0/2/0) | 0 (0/29/0) | 0 (0/29/0) |
| daimo-audit-1 | 0.72 | .143 (1/6/6) | **.429** (3/4/4) | 0 | .056 |
| gsy-4.0.2 | 0.80 | .875 (7/1/1) | .875 (7/1/1) | .270 | .270 |
- M38 improves 2 of 4 fill apps (jitsi, daimo; both 0.72) and ties on gsy (0.80). The fill-group gain (.346 → .549) comes entirely from those two 0.72 apps.
- xaman has no in-scope truth packages under --subsha (fn = 0), so it adds only FPs. Its GT has 157 bundled packages, but none are in the M34 release-identity tables.
- Control is identical between M37 and M38 on every app. Fallback differs only on yoroi (RN 0.71 → nearest 0.72): .077 → .519. readany and ogam are identical.

## RN 0.85 / unsupported_hbc98: own table, excluded from exact F1
Rule: an unsupported_bytecode / unsupported flag counts as a correct abstention.
| app | M37 | M38 | t35a/b |
|---|---|---|---|
| blink-beta-3.0.31 (clean dump) | rc0, 0 predictions, no flag (silent empty) | **unsupported flag** (max nsnc cov 0.070 < 0.40) ✓ | crash (HBC 98 disassembly EOF) |
| breathly-ios-2.2-18 (clean dump) | rc0, 0 predictions, no flag (silent empty) | **unsupported flag** (cov 0.035 < 0.40) ✓ | crash (HBC 98) |
| bbplayer-2.5.2 (dumper failure) | no triples → no run | no triples → no run | crash (HBC 98) |
- Correct abstentions: M37 0/2, M38 **2/2**, t35 0/2. Counted over the 2 clean dumps; bbplayer is not scorable by either matcher (dumper failure; package.json RN never fed to the matcher).

## SECONDARY: m35_eval.py (vulnerability level)
| subset | metric | M37 | M38 | t35a | t35b |
|---|---|---|---|---|---|
| exact (10) | strict F1 | .471 | .586 | .173 | .176 |
| exact (10) | set F1 | .565 | .796 | .329 | .375 |
| exact (10) | adv_strict F1 | .915 | .992 | .175 | .171 |
| fill (4) | set F1 | .385 | .863 | .158 | .300 |
| fill (4) | adv_strict F1 | 0 (tp0 fn2) | 1.0 (tp2) | 0 | .013 |
| t35 fair (9) | adv_strict F1 | .970 | .986 | .188 | .183 |
Full JSON is in result/m35_*.json.

## Caveats
- safe-1.54.0 was built with an entry shim (see MANIFEST). Mixed dumpers: hedis-m37 for HBC 98 apps; the cross-check on jitsi was byte-identical.
- t35 crashed on yoroi (RN 0.71: "$in needs an array") and on all HBC 98 apps.
- M37 and M38 differ in pin, binary and flag at once. The 0.72 fill gain cannot be attributed to the pin alone vs the binary from this run.

## Lab Notebook addendum (exact decimals, vuln table, fallback audit)
Committed after the one-shot score at `b416975` (no re-score). All figures recomputed from the frozen predictions via `m35_eval.py` / `oneclass --subsha` equivalence.

### Primary exact F1 — full P / R / F1 and tp/fp/fn (10 apps; RN 0.85 excluded)

| method | P | R | F1 | tp | fp | fn |
|---|---|---|---|---|---|---|
| M37 | 0.455 | 0.489 | **0.471** | 45 | 54 | 47 |
| M38 | 0.566 | 0.609 | **0.586** | 56 | 43 | 36 |
| t35a (M37 pin) | 0.118 | 0.326 | **0.173** | 30 | 225 | 62 |
| t35b (M38 pin) | 0.119 | 0.337 | **0.176** | 31 | 229 | 61 |

Subset exact (same scorer; tp/fp/fn):

| subset | M37 P/R/F1 (tp/fp/fn) | M38 P/R/F1 (tp/fp/fn) | t35a P/R/F1 (tp/fp/fn) | t35b P/R/F1 (tp/fp/fn) |
|---|---|---|---|---|
| fill (4) | .321/.375/**.346** (9/19/15) | .519/.583/**.549** (14/13/10) | .061/.292/**.101** (7/108/17) | .078/.375/**.129** (9/107/15) |
| control (3) | .684/.722/**.703** (26/12/10) | .684/.722/**.703** (26/12/10) | .202/.472/**.283** (17/67/19) | .186/.444/**.262** (16/70/20) |
| fallback (3) | .303/.312/**.308** (10/23/22) | .471/.500/**.485** (16/18/16) | .107/.188/**.136** (6/50/26) | .103/.188/**.133** (6/52/26) |
| t35-fair (9; no yoroi) | .512/.557/**.533** (44/42/35) | .576/.620/**.598** (49/36/30) | .118/.380/**.180** (30/225/49) | .119/.392/**.183** (31/229/48) |

### Vulnerability F1 and app vulnerable-or-clean (exact-10; aligns with v5/v6 tables)

Truth: **3** of 10 exact apps have ≥1 vulnerable universe dependency (`apps_vulnerable_truth=3`).

| method | adv F1 (set) | adv_strict F1 | app_acc (set) | app_acc_strict | apps_vulnerable_truth |
|---|---|---|---|---|---|
| M37 | 0.885 (54/9/5) | **0.915** (54/5/5) | 7/10 | 9/10 | 3 |
| M38 | 0.967 (59/4/0) | **0.992** (59/1/0) | 8/10 | 9/10 | 3 |
| t35a | 0.155 (32/323/27) | 0.175 (32/274/27) | 2/10 | 2/10 | 3 |
| t35b | 0.160 (34/332/25) | 0.171 (33/293/26) | 2/10 | 2/10 | 3 |

(tp/fp/fn under adv / adv_strict.)

### Why fallback rose 0.308 → 0.485 (per app)

`HEDIS_RN_FALLBACK=nearest` (from `runs/M37v7/rn_fallback.log` and `runs/M38v7/rn_fallback.log`, identical mapping):

| app | detected RN | nearest covered | M37 exact F1 (tp/fp/fn) | M38 exact F1 (tp/fp/fn) | Δ |
|---|---|---|---|---|---|
| yoroi-4.12.0 | 0.71.6 | **0.72** | 0.077 (1/12/12) | **0.519** (7/7/6) | +0.442 |
| readany-1.3.4 | 0.81.5 | **0.80** | 0.167 (2/10/10) | 0.167 (2/10/10) | 0 |
| ogam-0.0.49 | 0.83.1 | **0.80** | 0.933 (7/1/0) | 0.933 (7/1/0) | 0 |

**Confirmed:** the entire fallback-group gain is **yoroi** picking up the M38 fill on **RN 0.72**. The two apps that fall back to **0.80** (readany, ogam) are unchanged — consistent with gsy (native 0.80 fill app) also flat at 0.875. Notebook's guess that fallback apps pick up filled 0.72/0.80 versions holds for **0.72 only** on this set; **0.80 fallback did not move**.

### Optimizer note on the M38 confound (recorded, not a second v7 score)
On real_dev/synth, Optimizer reports M38 (new binary + new cutoff + `--unsupported-cov`) matched M37 prediction-for-prediction. None of the 10 non-0.85 v7 apps were flagged unsupported. That attributes the 0.471→0.586 exact-F1 gain on those 10 apps to the **DB cutoff / fill**, pending an optional cheap check (M38 binary @ M37 cutoff on **dev only**) if Notebook wants it in the record. v7 stays burned / unscored again.
