# Hedis v2 eval — PROGRESS

## 2026-10-12 01:55 CT — M0: state recovery (3rd worker)
- Repo: local `v2` == main (66fcd2f); work sat on unpushed `feat/fast-fingerprint` (4eb1c4b, 96f9ceb, d6f4dc3: fastpipe parallel cached build, hash-set matcher, HEDIS_PACKAGES_DB). origin/v2 == main.
- mongod running (pid 69613, mongo-data 5.5G, db `hedis`).
- targets: db/targets.json = 30 packages, 1496 versions x RN {rn074,rn076} = 3010 jobs.
- Previous build runs (fp-stats*.jsonl): 2353 unique (pkg,ver,rn) ok; build process was dead. Failures incl. better-auth/crypto-js (Metro resolve errors).
- Synthetic apps: rainbow/oss-apps builds failed (RB_EXIT 1/5); only apps/out/mattermost-2.29.1.hbc exists. No held-out split yet.
- Restarted build: nohup bin/hedis fingerprint ... -> logs/fp-build-resume2.log, logs/fp-stats-resume2.jsonl (pid 236275). Cache hits ~2-4 s/job.
- Benchmarks: matcher CompareLegacy 1.10 s/op vs CompareFast 17.5 ms/op (~63x). Build step bench (logs/bench_steps.txt): orig per-job install+metro reset+uninstall ~11.6 s vs fast ~2.6 s (warm metro, no uninstall) ≈4.4x per job, plus 3 workers x 2 RN.

## 2026-10-12 02:10 CT — M1: baselines (partial DB, build still running)
- Thesis transcribed predictions (data/predictions/thesis_*), run_eval.sh thesis_: prod_closure pkg P/R/F1 0.500/1.000/0.667, pkg+ver F1 0.000 (direct: pkg F1 0.545, ver 0.000).
- Thesis matcher (exact 3-hash, highest-match version) on our DB, Mattermost 2.29.1, universe = 30 target pkgs (data/universe_targets.json), DB ~2350/3010 jobs:
  - prod_closure: pkg 0.655/0.950/0.776, pkg+ver 0.345/0.435/0.385
  - direct:       pkg 0.586/0.944/0.723, pkg+ver 0.276/0.444/0.340
  - match wall 87 s (runs/hedisv2_/). Rainbow .hbc not available (bundle fails, logs/rainbow2.log).
- Synthetic apps: scripts/build_synthetic.py 10 running (pid 241545, logs/synth.log); 4/10 OK so far.

## 2026-10-12 02:28 CT — M2: synthetic eval set frozen + baseline on it
- 10 synthetic Hermes apps built (apps/out/synth-0[0-9]-rn07{4,6}.hbc, labels data/groundtruth/synth-*.json, manifest data/synth_manifest.json), 9-12 target pkgs each, alternating rn074/rn076.
- Split frozen in data/split.json (with .hbc sha256): dev = synth-00..04 + mattermost-2.29.1; held-out TEST = synth-05..09 (+ rainbow-1.9.70 when its .hbc builds).
- Thesis matcher baseline (partial DB, universe = 30 targets, threshold 0):
  | split | truth | pkg P | pkg R | pkg F1 | ver P | ver R | ver F1 |
  | dev  | direct       | 0.397 | 0.986 | 0.566 | 0.172 | 0.429 | 0.246 |
  | dev  | prod_closure | 0.466 | 0.988 | 0.633 | 0.207 | 0.379 | 0.268 |
  | test | direct       | 0.317 | 1.000 | 0.482 | 0.172 | 0.543 | 0.262 |
  | test | prod_closure | 0.379 | 1.000 | 0.550 | 0.193 | 0.424 | 0.265 |
- Precision is the failure mode: many FP packages (shared/trivial function hashes), and highest-match version picks wrong among near-identical versions. Far from 0.95.
- Match wall 45-490 s/app (contention with build); dominated by DB fetch, not compare.
- Build resume2: 1475/3010 jobs processed, 1197 ok (pid 236275).

## 2026-10-12 02:45 CT — M3: resume fixed, Mongo index
- Root cause of "resume redoes everything": `fingerprint` read MONGO_DB_NAME (default "clean") while the matcher used HEDIS_PACKAGES_DB=hedis, so resume2/3 built into a separate `clean` DB (6.3M hashes, stray, to be dropped). Fixed: fingerprint honors HEDIS_PACKAGES_DB and logs the DB name (feat/resume-fix).
- resume2 (old binary) was OOM-killed at 02:26 (hedis 6 GB RSS): in-memory bundle cache held all raw IR. Removed (disk .hbc cache stays).
- resume4 (pid 331759, logs/fp-build-resume4.log, logs/fp-stats-resume4.jsonl) skips fp_done jobs instantly (500 skips in 20 s); 2603 rows so far.
- Mongo: covering index cover_rn_pkg_3hash {rn, package_id, S, C1, C2} (built in 76 s). Index alone did not help the $group/$lookup aggregation (80 s); replaced with covered Find + group in Go: db_load 80.2 s -> 23.9 s (3.4x), results identical on synth-00 (1175 pkg-versions). (feat/mongo-indexes)
- Added --dump-app/--dump-only to export filtered app triples for offline experiments (runs/apps/*.triples).

## 2026-10-12 03:19 CT — M4: final DB + re-scored baseline
- Build resume4 (pid 331759) made 0 new successes: the remaining ~657 of 3010 jobs are hard failures (react-native-reanimated 4.x npm ERESOLVE peer deps on RN 0.74/0.76, better-auth Metro ESM resolution, some react-native-screens/axios Metro errors). DB final: hedis = 2353 pkg-version x RN jobs, 8.53M hashes, 1175 pkg-versions on 0.74 / 1178 on 0.76.
- Re-ran thesis matcher (new fast loader) on all 11 apps: 18.6-23.4 s per synth app, 44 s Mattermost (was 45-490 s). Prefix hedisv2f_. Scores unchanged vs M2 (DB did not change):
  dev  prod_closure pkg 0.466/0.988/0.633  ver 0.207/0.379/0.268
  test prod_closure pkg 0.379/1.000/0.550  ver 0.193/0.424/0.265
- Exported covered index to cache/index-*.npz (eval/export_index.py, 52 s + 38 s) for offline experiments (eval/matchlab.py).

## 2026-10-12 03:21 CT — EXP lab_thesis (re-implementation check) + EXP (a) IDF
- matchlab thesis re-implementation reproduces the Go matcher within ±2 TP (dev pc ver F1 0.287 vs 0.268; test 0.269 vs 0.265; diff = only fp_done-complete pkg-versions + set dedup).
- (a) drop triples shared by >1 package name, IDF-weighted match fraction (threshold 0):
  dev  prod_closure pkg 0.756/0.793/0.774  ver 0.349/0.316/0.331
  test prod_closure pkg 0.630/0.836/0.719  ver 0.315/0.348/0.331
  Precision roughly doubles; some recall lost (packages with no unique triples, e.g. thin wrappers).

## 2026-10-12 03:21 CT — EXP (b) per-package min-score threshold (tuned on dev only)
- On top of (a). Tuned on dev package-level F1 (prod_closure); result: global 0.05, parse 0.10, dayjs 0, rest 0.05 (data/thresholds_dev.json -> docs/eval/results). Mostly a global threshold; per-package tuning has too little dev data to differ.
  dev  prod_closure pkg 0.970/0.793/0.872  ver 0.448/0.316/0.370
  test prod_closure pkg 0.939/0.836/0.885  ver 0.469/0.348/0.400
  test direct       pkg 0.898/0.957/0.926  ver 0.469/0.500/0.484

## 2026-10-12 03:22 CT — EXP (c) version choice by F1 of the version's triples vs the app's triples of that package
- On top of (a)+(b). No gain: dev identical (ver F1 0.370), test slightly worse (pc ver 0.383 vs 0.400; direct 0.421 vs 0.484). Diagnosis follows in next entry.

## 2026-10-12 03:23 CT — EXP (d) version-string anchors
- On top of (a)+(b)+(c): among versions within 0.02 F1 of the best, prefer one whose x.y.z string is in its own bundle and in the app's strings. Slightly worse: dev pc ver F1 0.358, test pc 0.365 (direct 0.400). Few packages embed their version; common strings ("1.0.4") create false anchors.
- Version-error diagnosis on dev (direct truth, after b): 28 correct, 13 identical fingerprints (indistinguishable by construction), 20 wrong-but-distinguishable, 9 package misses. Even the TRUE version only matches ~50% of its own unique triples in the app (parse 6.0.0: 1069/2153), so the version signal is drowned out.

## 2026-10-12 03:23 CT — M5: summary table + build-config probe
Build-config probe (parse 6.0.0, RN 0.74, share of the package's non-baseline triples present in synth-00): dev=true/minify=false 0.294; dev=false/minify=false 0.497 (== DB, so DB is dev=false); dev=false/minify=true 0.577. Store apps are minified, the DB is not: rebuilding the DB with minify=true is a cheap +8 pt overlap win. The remaining ~42% mismatch is bundle-context (module ids / inlined requires in C1/C2), which caps exact 3-hash matching; versions need context-robust features (S-only or normalized-content hashes).

| method (cumulative) | split | pkg P | pkg R | pkg F1 | ver P | ver R | ver F1 |
|---|---|---|---|---|---|---|---|
| thesis (Go, hedisv2f_) | dev  | 0.466 | 0.988 | 0.633 | 0.207 | 0.379 | 0.268 |
| thesis (Go, hedisv2f_) | test | 0.379 | 1.000 | 0.550 | 0.193 | 0.424 | 0.265 |
| a IDF/unique | dev  | 0.756 | 0.793 | 0.774 | 0.349 | 0.316 | 0.331 |
| a IDF/unique | test | 0.630 | 0.836 | 0.719 | 0.315 | 0.348 | 0.331 |
| +b threshold | dev  | 0.970 | 0.793 | 0.872 | 0.448 | 0.316 | 0.370 |
| +b threshold | test | 0.939 | 0.836 | 0.885 | 0.469 | 0.348 | 0.400 |
| +c F1 tie-break | dev  | 0.970 | 0.793 | 0.872 | 0.448 | 0.316 | 0.370 |
| +c F1 tie-break | test | 0.939 | 0.836 | 0.885 | 0.449 | 0.333 | 0.383 |
| +d anchors | dev  | 0.970 | 0.793 | 0.872 | 0.433 | 0.305 | 0.358 |
| +d anchors | test | 0.939 | 0.836 | 0.885 | 0.429 | 0.318 | 0.365 |
(prod_closure truth, universe = 30 targets.) Best = a+b. Dropped stray `clean` DB.

## 2026-10-12 03:24 CT — M6: hash-component diagnosis (parse 6.0.0 isolated vs inside synth-00, RN 0.74)
Tools: --dump-app + HEDIS_DUMP_RAW=1 writes raw S/C1/C2; eval/diag_components.py, eval/norm_check.py.
Isolated release build (dev=false, minify=true), 2156 pkg functions: exact 57.7%; S differs only 28.8%; S same but C1+C2 differ 5.4%; C2 only 2.2%; C1 only 0.8%; nothing 5.1%.
Causes (the module-ID hypothesis was wrong: S has no operands and numbers aren't strings):
1. S: operand-width opcode variants depend on the size of the whole bundle's string table / jump distances: GetByIdShort vs GetById, PutNewOwnByIdShort, J*Long, LoadConstUInt8/Int/Zero.
2. C1<->C2 flips: a string is flagged "identifier" bundle-wide if any use is an identifier, so e.g. "fulfilled" moves between C1 and C2 depending on the other code in the bundle.
3. Minification: the DB's unminified build matches only 49.7% (vs 57.7% minified).
Normalized prototype (NS = width-folded opcodes, LoadConst{UInt8,Int,Zero,Double}->LoadConstNum; NC = sorted C1∪C2, strings len<=2 dropped):
  minified pkg: NS+NC 99.8% present in app (orig 57.7%), NS-only 99.8%; unminified pkg: NS+NC 78.4%.

## 2026-10-12 03:27 CT — M7: normalized hashes + release rebuild started
- pkg/normhash: NS (opcode width variants folded, LoadConst numerics unified) and NC (C1∪C2 sorted, len<=2 dropped) SHA-256; stored as hashes.ns / hashes.nc and baselines.hashes.ns/nc; covering index cover_rn_pkg_norm {rn, package_id, ns, nc}. Unit tests in pkg/normhash.
- fingerprint --release: Metro dev=false minify=true (store config), baseline filter on (NS,NC), raw IR not stored (axios 1.6.0 job: 373 funcs, 294 kept vs ~4k in the dev DB).
- Rebuild into new DB hedis_rel: pid 368177, logs/fp-rel.log, logs/fp-rel-stats.jsonl, cache-rel/. ~18 jobs/min (CPU-bound, load 11 on 8 cores) -> ETA ~2.5 h.
- Go matcher port ready: packages -d --norm nsnc|ns [--thresholds f.json] [--tiebreak] (pkg/pipeline/normmatch.go); offline lab eval/matchlab2.py. App norm keys dumped to runs/apps_rel/.

## 2026-10-12 04:35 CT — M8: release DB built
- hedis_rel finished: 3010 jobs, 2353 ok (same coverage as hedis; same hard failures), 8.29M hashes (ns/nc, no raw IR), 1175 pkg-versions on 0.74 / 1178 on 0.76. Exported to cache/relidx.json (matchlab2.py export, 44 s).

## 2026-10-12 04:59 CT — M9: box restore incident + recovery
- Around 04:37 CT the box was restored/restarted: mongod died with an unclean shutdown; WiredTiger sizeStorer.wt is corrupted (checksum error), and `mongod --repair` aborts. node_modules (envs/*, envs/work/*), ~/.npm, /tmp, the .venv and cache/relidx.json were lost. Kept: .hbc caches (cache/, cache-rel/), stats JSONLs, runs/apps_rel dumps, cache/index-*.npz (thesis DB export), git.
- Corrupt data dir preserved as mongo-data (+ copy mongo-data.bak-0450); a fresh mongod now runs on mongo-data-rel.
- Recovery without re-running npm/Metro for 2353 jobs: new `fingerprint --release --import-stats logs/fp-rel-stats.jsonl` re-hashes the cached release .hbc files into hedis_rel (feat/import-from-cache). Only the baselines need Metro, so running npm ci in envs/rn074 and envs/rn076.

## 2026-10-12 05:24 CT — M10: Go normalized matcher scores (hedis_rel, app NS/NC from runs/apps_rel dumps)
- hedis_rel recovered via --import-stats: 2351/2353 jobs, 8.26M hashes, release baselines rebuilt. apps/out/*.hbc were lost in the restore, so the Go matcher ran with --from-dump on the M7 app dumps (same functions; orig-key baseline pre-filtered). Thresholds tuned on dev only (data/thresholds_g_*.json).
```
g_nsnc_a dev  prod_closure  0.00 |  0.984  0.732  0.839 (60, 1, 22) |  0.475  0.305  0.372 (29, 32, 66)
g_nsnc_a dev  direct        0.00 |  0.967  0.843  0.901 (59, 2, 11) |  0.443  0.386  0.412 (27, 34, 43)
g_nsnc_a test prod_closure  0.00 |  1.000  0.764  0.866 (42, 0, 13) |  0.571  0.364  0.444 (24, 18, 42)
g_nsnc_a test direct        0.00 |  1.000  0.913  0.955 (42, 0, 4) |  0.571  0.522  0.545 (24, 18, 22)
g_nsnc_ab dev  prod_closure  0.00 |  1.000  0.732  0.845 (60, 0, 22) |  0.483  0.305  0.374 (29, 31, 66)
g_nsnc_ab dev  direct        0.00 |  0.983  0.843  0.908 (59, 1, 11) |  0.450  0.386  0.415 (27, 33, 43)
g_nsnc_ab test prod_closure  0.00 |  1.000  0.764  0.866 (42, 0, 13) |  0.571  0.364  0.444 (24, 18, 42)
g_nsnc_ab test direct        0.00 |  1.000  0.913  0.955 (42, 0, 4) |  0.571  0.522  0.545 (24, 18, 22)
g_nsnc_abc dev  prod_closure  0.00 |  1.000  0.732  0.845 (60, 0, 22) |  0.467  0.295  0.361 (28, 32, 67)
g_nsnc_abc dev  direct        0.00 |  0.983  0.843  0.908 (59, 1, 11) |  0.433  0.371  0.400 (26, 34, 44)
g_nsnc_abc test prod_closure  0.00 |  1.000  0.764  0.866 (42, 0, 13) |  0.548  0.348  0.426 (23, 19, 43)
g_nsnc_abc test direct        0.00 |  1.000  0.913  0.955 (42, 0, 4) |  0.548  0.500  0.523 (23, 19, 23)
g_ns_a dev  prod_closure  0.00 |  0.949  0.683  0.794 (56, 3, 26) |  0.458  0.284  0.351 (27, 32, 68)
g_ns_a dev  direct        0.00 |  0.932  0.786  0.853 (55, 4, 15) |  0.424  0.357  0.388 (25, 34, 45)
g_ns_a test prod_closure  0.00 |  1.000  0.764  0.866 (42, 0, 13) |  0.524  0.333  0.407 (22, 20, 44)
g_ns_a test direct        0.00 |  1.000  0.913  0.955 (42, 0, 4) |  0.524  0.478  0.500 (22, 20, 24)
```

## 2026-10-12 05:27 CT — M11: ns (structural-only) a+b, a+b+c + error diagnosis
```
g_ns_ab dev  prod_closure  0.00 |  0.982  0.683  0.806 (56, 1, 26) |  0.474  0.284  0.355 (27, 30, 68)
g_ns_ab dev  direct        0.00 |  0.965  0.786  0.866 (55, 2, 15) |  0.439  0.357  0.394 (25, 32, 45)
g_ns_ab test prod_closure  0.00 |  1.000  0.764  0.866 (42, 0, 13) |  0.524  0.333  0.407 (22, 20, 44)
g_ns_ab test direct        0.00 |  1.000  0.913  0.955 (42, 0, 4) |  0.524  0.478  0.500 (22, 20, 24)
g_ns_abc dev  prod_closure  0.00 |  0.982  0.683  0.806 (56, 1, 26) |  0.456  0.274  0.342 (26, 31, 69)
g_ns_abc dev  direct        0.00 |  0.965  0.786  0.866 (55, 2, 15) |  0.421  0.343  0.378 (24, 33, 46)
g_ns_abc test prod_closure  0.00 |  1.000  0.764  0.866 (42, 0, 13) |  0.500  0.318  0.389 (21, 21, 45)
g_ns_abc test direct        0.00 |  1.000  0.913  0.955 (42, 0, 4) |  0.500  0.457  0.477 (21, 21, 25)
```
Diagnosis (g_nsnc_ab, dev, direct truth; eval/diag_versions.py): 27 correct, 11 wrong with an IDENTICAL normalized fingerprint to the true version (indistinguishable by construction), 21 wrong but distinguishable, 10 packages missed, 1 true version not in DB, 1 FP package.
- Distinguishable errors are near-ties: true vs predicted version match almost the same share (e.g. @react-navigation/native 7.3.7: 3506/3752 vs 7.2.6: 3490/3737; zod 4.1.5 1026/1083 vs 4.1.0 1012/1074). Each package-version set is ~1-4k keys because the release baseline is tiny (a ~17 KB entry bundle), so shared React/RN code stays in every version and only a handful of keys differ between neighbours.
- Even the true version only has ~93% of its keys in the app; the missing ~7% (app-context inlining/tree-shaking) is larger than the gap between neighbouring versions, so 'highest match' and F1 tie-break both pick by noise. --tiebreak made it slightly worse.
- Package misses (prod_closure recall 0.73-0.76): mostly transitive packages whose unique keys are in the app at >=50% but fall under the dev-tuned threshold, or whose keys are all shared with another package (dropped by the uniqueness filter).
Next: baseline = full RN+React release bundle (shrinks package sets to package code), score versions only on keys that differ between versions (symmetric difference with neighbours), and collapse indistinguishable versions into equivalence classes when scoring.

## 05:32 CT — M12: full React+RN release baseline (step 1)
- Built a full React+RN release bundle per RN (envs/rnXXX/fullbase_entry.js imports all of react-native + react; 1.3 MB hbc, runs/fullbase/). Its 3119 NS/NC keys are already inside the existing release baselines (3158 keys) and none survive in any filtered package set or app set, so re-scoring is identical (lab3: test strict pkg+ver F1 0.541 both ways).
- Correction to M11: the "1-4k keys per version, shared RN code" observation came from diag_versions.py reading unfiltered Mongo rows; after the baseline filter zod has 226 keys. RN code was not the problem.
- Found a Go bug: --from-dump used strings.Fields, which shifts NS/NC when C1/C2 are empty, so M10 Go scores were understated. Fixed (split on single spaces). Python lab with the same method (a+b, nsnc): test pkg F1 0.900, strict pkg+ver F1 0.541 (eval.py), vs Go M10 0.444.
- matchlab3.py adds a class-level metric (identical-fingerprint versions = one class): test 0.775.

## 05:34 CT — M13: why packages are missed (step 4)
Dev misses with a+b nsnc (15 truth pairs, prod_closure); for the true version: keys after baseline / in app / package-unique / unique in app:
- crypto-js 4.2.0 (synth-00,01,04): 219 / 1-102 / 116 / 0. semver 5.7.2 (synth-00,02,03,04): 85 / 4-5 / 78 / 0. dayjs 1.11.23 (synth-00,04): 55 / 0 / 53 / 0. => the lockfile lists them (transitive), but none of their own code is in the bundle (0 unique keys present). Not a threshold or filter problem: the prod_closure label includes packages that are installed but not bundled.
- react-freeze 1.0.4 (synth-00,01,03,04, mattermost): 55 / 47-53 / 1 / 0. 45 of its 55 keys are shared with ~20 other packages (generic React helper code that the RN baseline does not contain); its single unique key is absent. The uniqueness filter drops it, but nothing is left that identifies react-freeze specifically.
- @gorhom/bottom-sheet (mattermost): not in the DB (outside the 30 targets' versions).
- Threshold is not the cause: every missed package has 0 unique hits, so no threshold value recovers it. Tried a dependency-aware fallback (accept when >=80-90% of ALL keys present): recovers react-freeze but adds many FPs (test pkg P 1.00 -> 0.47-0.53); restricting to low-df keys recovers nothing. Not adopted.
- Fix for the metric: report recall against bundled truth too (packages with >=1 unique key present is circular; needs Metro module lists from the rebuilt synthetic bundles, step 5).

## 05:35 CT — M14: version-diff scoring (step 2)
- vd: on version-variable keys V_p (union minus intersection over the package versions), score(v) = |D_v present| - |D_v absent| - |app keys in V_p not in v|; pick max, tie -> IDF score.
- Result: test identical (strict 0.541, class 0.775); dev slightly worse (strict 0.519 -> 0.494). Not adopted.
- Why: class precision is already 0.956 on test while strict precision is 0.667, i.e. nearly every wrong version has a fingerprint identical to the true one. There is nothing left between versions for any key-based tie-break to use; remaining strict errors are indistinguishable versions.

## 05:38 CT — M15: Go matcher after the --from-dump fix (prod_closure truth, test unless noted)
| method | pkg P/R/F1 | strict pkg+ver P/R/F1 | class pkg+ver P/R/F1 |
|---|---|---|---|
| thesis baseline (old DB) | 0.379/1.000/0.550 | 0.193/0.424/0.265 | n/a |
| Go nsnc a+b, M10 (Fields bug) | 1.000/0.764/0.866 | 0.571/0.364/0.444 | 0.786/0.500/0.611 |
| **Go nsnc a+b, fixed (g3_ab)** | 1.000/0.818/0.900 | **0.667/0.455/0.541** | **0.956/0.652/0.775** |
| + full RN baseline (lab) | same | 0.541 | 0.775 |
| + version-diff vd (lab) | same | 0.541 | 0.775 |
Dev g3_ab: pkg 0.899, strict 0.519, class 0.790. Direct truth, test: pkg F1 0.989, strict 0.659. Go and the Python lab produce identical predictions.
Recall ceiling: prod_closure includes installed-but-unbundled packages (M13), and identical-fingerprint versions cap strict precision.
Synthetic app rebuild running: logs/synth-rebuild.log (frozen labels backed up in data/groundtruth.frozen/).

## 05:43 CT — M16: synthetic apps rebuilt and verified (step 5)
- scripts/build_synthetic.py 10 re-run after the restore: all 10 synthetic .hbc files are byte-identical to the SHA-256 values frozen in data/split.json; direct and prod_closure labels are identical to the backed-up frozen labels (data/groundtruth.frozen/). Mattermost and Rainbow .hbc are still missing.
- Go norm matcher run on the real test .hbc files (not dumps): test strict pkg+ver F1 0.541, pkg F1 0.900, identical to the --from-dump run.

## 05:45 CT — M17: strict pkg+version ceiling from identical-fingerprint classes (step 2)
eval/ceiling.py: an oracle detects every truth package whose true version is in hedis_rel and picks a version inside the true version's identical-fingerprint class. A hash method can't separate class members, so expected strict credit = 1/|class|; "optimistic" = lucky pick (= class metric).
| split | truth | pairs | not in DB | mean class size | singleton share | expected strict ceiling F1 | optimistic |
|---|---|---|---|---|---|---|---|
| dev  | prod_closure | 95 | 1 | 3.16 | 0.638 | 0.730 | 0.995 |
| test | prod_closure | 66 | 0 | 3.00 | 0.576 | **0.695** | 1.000 |
| dev  | direct | 70 | 1 | 3.75 | 0.551 | 0.665 | 0.993 |
| test | direct | 46 | 0 | 3.78 | 0.435 | 0.591 | 1.000 |
=> With this DB, no exact-hash method can expect more than ~0.70 strict pkg+ver F1 on test (current 0.541). The 0.95 strict target is unreachable by hashing alone; it needs non-hash version evidence or a coarser version definition (class-level F1 test 0.775).
