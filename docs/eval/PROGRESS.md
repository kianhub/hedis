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

## 05:50 CT — M18: bundled truth from Metro source maps (step 1)
- Re-bundled synth-00..09 with --sourcemap-output (JS identical to the evaluated bundles apart from the sourceMappingURL comment); eval/bundled_truth.py adds "bundled" = packages with >=1 module under node_modules/<pkg>/ in the source map, version from that path's package.json. 29-64 bundled packages per app. eval.py accepts --truth bundled. Labels copied to docs/eval/groundtruth/.
- Bundled vs prod_closure: e.g. synth-04 closure-only {crypto-js, dayjs, semver} (installed, not bundled), bundled-only {@react-navigation/native} (nested dependency). Mattermost has no source map (no .hbc/app build), so dev here = synth-00..04.
- Go nsnc a+b (g3_ab) on synthetic apps:
| split | truth | pkg P/R/F1 | strict pkg+ver P/R/F1 |
|---|---|---|---|
| dev  | bundled      | 1.000/0.875/0.933 | 0.633/0.554/0.590 |
| dev  | prod_closure | 1.000/0.790/0.883 | 0.633/0.431/0.512 |
| test | bundled      | 1.000/0.918/**0.957** | 0.667/0.612/**0.638** |
| test | prod_closure | 1.000/0.818/0.900 | 0.667/0.455/0.541 |
- Ceiling on bundled truth (expected, uniform pick in class): dev 0.656, test 0.616 (mean class size 3.6; 47-54% singletons). The matcher's 0.638 is above the uniform-pick expectation because it picks deterministically within classes and got lucky on some ties; optimistic ceiling 1.0.

## 06:04 CT — M19: eval set grown to 30 synthetic apps (step 3)
- build_synthetic.py 20 10 (seeds 1010-1029, now with --sourcemap-output and a start index): 20/20 OK, 166 picks, 131 distinct pkg@version over 29 packages. Bundled truth from their source maps (mean 36.8 bundled packages/app).
- Split extended under the same frozen rule (data/split.json, v1 kept as data/split.v1.json): synth-10..19 -> dev, synth-20..29 -> test, sha256 of each .hbc recorded. Size now dev = 15 synth + mattermost, test = 15 synth + rainbow (no .hbc).
- Go nsnc a+b (unchanged thresholds from the v1 dev tuning):
| set | truth | pkg P/R/F1 | strict pkg+ver P/R/F1 |
|---|---|---|---|
| dev 15 synth  | bundled      | 1.000/0.866/0.928 | 0.620/0.537/0.576 |
| test 15 synth | bundled      | 1.000/0.891/**0.942** | 0.648/0.577/**0.610** |
| dev 15 synth  | prod_closure | 1.000/0.777/0.875 | 0.620/0.402/0.488 |
| test 15 synth | prod_closure | 1.000/0.787/0.881 | 0.648/0.418/0.508 |
| test new 10 only | bundled   | 1.000/0.875/0.933 | 0.636/0.557/0.594 |
- Strict ceiling, bundled truth: dev15 0.667, test15 0.651 (mean class size 3.3-3.6, 137 test pairs).
- RN 0.79: not cheap. It needs a new rn079 env plus ~1500 more fingerprint jobs (~1 h+ CPU) before any 0.79 app can be matched; not done.

## 06:07 CT — M20: catalogue of identical-fingerprint version groups
eval/hbc_features.py (hedis disassemble -> referenced strings incl. short ones, numeric-constant operands, per-function full-instruction hashes with operands, function count); eval/group_catalog.py over every multi-version class in hedis_rel (both RNs). Output docs/eval/results/groups.json.
- 320 groups with >1 version. **310 (96.9%) are byte-identical**: every member compiled from an identical Metro bundle (same bundle SHA-256), so no feature of the compiled output can tell them apart.
- Spot check with npm pack: react-native-safe-area-context 4.10.0 vs 4.10.1 differ only in ios/RNCSafeAreaProvider.m and package.json; zod 3.25.34 vs 3.25.35 differ only under dist/*/v4 (the zod/v4 subpath, not reached by `import 'zod'`). Version bumps that touch only native code, types, docs or unbundled subpaths are invisible in the JS bundle. package.json fields are not inlined (no version strings differ in any group).
- 10 groups have different bundles: 6 differ in instruction operands (ins; immer 11.1.x), 2 also in short strings (immer 11.1.2/11.1.3), 4 differ only in JS text with identical bytecode features (i18next 23.12.0/1, react-native-screens 3.35-3.37; e.g. comment or ordering changes). No group differs in numeric constants, function count or version-string constants.
- Caveat: the DB fingerprints only the package root entry; apps importing subpaths (e.g. zod/v4) are not covered.

## 06:09 CT — M21: group-resolution step + real ceiling (bundled truth, 15+15 synthetic apps)
- eval/resolve_groups.py: for a predicted version in a group whose members have different bundles, split it by bundle SHA and pick the sub-group whose specific features (instruction-with-operand hashes, referenced strings incl. short) best match the app's disassembly. Verified on dev: it fires on 3 dev and 2 test predictions and leaves all of them unchanged (the matcher's pick is already in the right sub-group), so it neither helps nor hurts. Adopted as a no-op safeguard; other feature types (numeric constants, function counts, version strings, inlined package.json) differ in 0 groups, so there was nothing to verify for them.
| set | method | pkg F1 | strict pkg+ver P/R/F1 | one-class pkg+ver P/R/F1 |
|---|---|---|---|---|
| dev15  | g3_ab | 0.928 | 0.620/0.537/0.576 | 0.992/0.859/0.921 |
| dev15  | + group resolution | 0.928 | 0.620/0.537/0.576 | 0.992/0.859/0.921 |
| test15 | g3_ab | 0.942 | 0.648/0.577/**0.610** | 0.984/0.876/**0.927** |
| test15 | + group resolution | 0.942 | 0.648/0.577/**0.610** | 0.984/0.876/**0.927** |
- Real ceiling (credit 1/|bundle-byte-identical sub-group| of the true version, perfect package detection): dev **0.676**, test **0.659** strict F1 (149 / 137 truth pairs). 310/320 DB groups (96.9%) are byte-identical, so the real ceiling is almost the class ceiling (0.651 test).

## M22 — one-class metric, dev error diagnosis, recall fixes (2026-10-12)
Primary metric (Kian): pkg+version F1, byte-identical bundles = one release, bundled truth (eval/oneclass.py; release identity = root bundle SHA + subpath bundle SHA).
Baseline g3_ab: dev 0.914 (P .984 R .852, tp127 fp2 fn22), test 0.927.
Dev errors by category (g3_ab): FN package missed 19 (react-freeze 13, crypto-js 4, @react-navigation/native 6.1.18 2), FN wrong version 2, FN version not in DB 1 (crypto-js 4.1.1); FP 2 (both wrong-version).
Root causes:
- react-freeze: its only package-unique key (df=1) is never present in apps (bundle artifact); remaining 54 keys are shared. Same for @react-navigation/native 6.1.x (1 unique key, absent).
- crypto-js: apps get it via parse/react-native-keys which require subpaths (crypto-js/aes, core, md5, enc-*); DB had only the root entry, and only version 4.2.0.
- Note: runs/apps_rel dumps for synth-10..29 were lost in the restore; regenerated from apps/out (synth-00 dump re-check byte-identical).
Fixes (eval/matchlab4.py + Go fingerprint --subpaths):
- thresholds tuned on dev against bundled truth.
- --lowuniq 2: packages with <=2 unique keys score on keys with df<=2.
- --dep 0.6: dependency implication from npm dependencies (data/deps.json via eval/fetch_deps.py).
- fingerprint --subpaths: one extra bundle per version importing every subpath entry (exports map, else top-level .js files); rows tagged sub; bad subpaths dropped one by one on Metro failure. Logs: logs/fp-sub-stats.jsonl.
- Added crypto-js 3.1.6..4.2.0 and react-freeze 1.0.0..1.0.4 (db/targets_ext.json).
Dev (old index, before subpath rows): m4_base 0.899; +dep 0.930; +dep+lowuniq 0.956 (P .972 R .940, tp140 fp4 fn9). --vd / --fit version choice: 0.942 (worse, dropped).

## M23: subpath rows, dep + lowuniq ported to Go, frozen config, test scored once (2026-10-12)
Run settings: worker effort level high (default).
- History: v2 rewritten so all 69 commits are authored and committed by Kian Pasani, with no Co-authored-by trailers. A local backup ref is kept as backup/v2-pre-rewrite. Merged remote feat/* branches were deleted. Commit refs cited above (66fcd2f main, 4eb1c4b/96f9ceb/d6f4dc3) were Kian's and are unchanged. Every other bot-era commit hash changed.
- Subpath fingerprinting (fingerprint --subpaths) over db/targets_ext.json, both RNs:
  - 287 version/RN jobs added subpath rows (crypto-js, parse, semver, axios, permissions, zod v3/v4-locales, ...).
  - 1037 had no subpaths and 514 failed.
  - better-auth, bottom-sheet, fast-image and reanimated fail mostly because their exports need babel plugins that RN 0.74/0.76 lack (e.g. zod/v4 classic export-namespace-from). Apps on these RNs can't import those subpaths either.
  - Probing now skips subpaths that already failed in 2+ versions and only probes when there are 50 or fewer subpaths.
- Metric: one-class identity is the root bundle SHA, as Kian defined it. `oneclass.py --subsha` is optional.
- Subpath rows exist only for some versions, which biased version choice (parse 6.0.0 vs 8.0.1, zod 4.1.x). The fix is --rootver: version choice uses root-entry keys only (cache/relidx_root.json via `matchlab2.py export-root`; rows tagged sub in Go).
- Dep rule (--depshared): the evidence is the dependency's keys that the parent's own fingerprint also contains (min 5), with coverage of at least 0.6. This fixes crypto-js pulled in via parse, whose crypto-js keys are not unique.
- Go port in pkg/pipeline/normmatch.go adds the flags --lowuniq/--lowdf, --dep/--deps, and --rootver. Go matches Python exactly on dev (0.970, same errors).
- Frozen config (data/frozen_m23.json and .sha256): `--norm nsnc --thresholds data/thresholds_m4f_dlr.json --lowuniq 2 --lowdf 2 --dep 0.6 --deps data/deps.json --rootver`.

| one-class F1 (P/R) | dev | test |
|---|---|---|
| g3_ab (M21) | 0.914 (.984/.852) | 0.927 (.984/.876) |
| m4 dep+lowuniq, old index | 0.956 (.972/.940) | – |
| + subpath rows (no rootver) | 0.940 (.940/.940) | – |
| **frozen go_m23** | **0.970 (.973/.966)** | **0.975 (.971/.978)**, scored once |

Remaining dev errors (go_m23):
- FN missed (1): crypto-js 4.2.0 in synth-00.
- Wrong version (4): screens 3.35 vs 3.37 and immer 11.1.2 vs 11.1.3 have identical NS/NC keysets but different bundles; screens 3.33 vs 3.37 and fast-image 9.0.1 vs 9.0.2 have version-distinguishing keys absent from the app.
Test errors:
- Wrong version (3): i18next 23.12.0 vs .1, fast-image 8.23 vs 8.25, immer 11.1.14 vs .15.
- FP (1): @react-navigation/native 6.1.18 in synth-24 (not bundled).

## M24: primary metric = root AND subpath bundle SHA (--subsha), frozen go_m23 rescored (2026-10-12)
Run settings: worker effort level high (default).
Kian's decision: two versions count as one release only if both the root bundle SHA and the subpath bundle SHA are identical (`oneclass.py --subsha`). Matcher, thresholds and predictions are unchanged (frozen go_m23, data/frozen_m23.json). This is only a rescore.

| go_m23 one-class P / R / F1 | dev | test |
|---|---|---|
| root SHA only (M23) | .973 / .966 / 0.970 (tp144 fp4 fn5) | .971 / .978 / 0.975 (tp134 fp4 fn3) |
| **root + subpath SHA (primary)** | **.959 / .953 / 0.956** (tp142 fp6 fn7) | **.964 / .971 / 0.967** (tp133 fp5 fn4) |

Errors under --subsha:

Dev:
- FN missed (1): crypto-js 4.2.0 in synth-00.
- Wrong version (6). These 4 carry over from M23:
  - screens 3.35 vs 3.37
  - screens 3.33 vs 3.37
  - fast-image 9.0.1 vs 9.0.2
  - immer 11.1.2 vs 11.1.3

  These 2 are new because the subpath bundles differ while the root bundles are identical:
  - flash-list 2.0.2 vs 2.0.3: different subpath SHAs.
  - permissions 5.2.5 vs 5.3.0: 5.3.0 has no subpaths.

Test:
- Wrong version (4). The 3 from M23 carry over (i18next 23.12.0 vs .1, fast-image 8.23 vs 8.25, immer 11.1.14 vs .15). The new one:
  - gesture-handler 2.27.0 vs 2.27.1: the 2.27.0 rn076 subpath job failed in Metro, so it has no subpath SHA and doesn't match the identical root bundle. This error comes from the failed job, not from a real difference between versions.
- FP (1): @react-navigation/native 6.1.18 in synth-24.

## M25: track 1, the 514 failed subpath fingerprint jobs (2026-10-12)
Run settings: worker effort level high (default).

Triage of the 514 failed subpath jobs (logs/fp-sub-stats.jsonl):

| cause | jobs | fix |
|---|---|---|
| root bundle also failed (package-version not in the DB: bottom-sheet npm install errors; better-auth / reanimated Metro errors) | 359 | none needed for identity (no root SHA, so never matched) |
| top-level tooling files taken as subpaths (app.plugin.js in fast-image 70 / share 46, metro.js in @sentry 9, jestSetup.js in gesture-handler 2) | 127 | `toolingFile()` excludes app.plugin, metro, babel, cli, scripts, jest*, mock*, eslint/webpack/rollup from subpath candidates |
| zod v4 subpaths: Babel on RN 0.74/0.76 lacks export-namespace-from (TransformError), plus all-probes-failed (zod 24, keys 2, permissions 2) | 28 | probing for any subpath count; skip subpaths that already failed in 2+ versions; when nothing bundles, record the **no-subpath marker** (ok, no SHA, fp_done no_subpath) instead of an error |

Note: no Metro/Babel config change. Adding a Babel plugin only for fingerprinting would make our bundles differ from what an RN 0.74/0.76 app produces.

The 287 earlier ok jobs whose subpath list held tooling files were re-run so their identity is consistent.

| | before | after (logs/fp-sub3-stats.jsonl) |
|---|---|---|
| root-ok jobs: subpath rows / no-subpath / error | 629 / 1591 / 155 | 350 / 2025 / **0** |

- oneclass.py --subsha now merges fp-sub then fp-sub3 (later rows override; marker = empty subpath SHA).
- Rescore of frozen go_m23 (same flags, rerun on the fixed DB as go_m23r; predictions identical):

| --subsha P / R / F1 | dev | test |
|---|---|---|
| M24 (spurious splits) | .959 / .953 / 0.956 | .964 / .971 / 0.967 |
| **M25** | **.973 / .966 / 0.970** (tp144 fp4 fn5) | **.971 / .978 / 0.975** (tp134 fp4 fn3) |

The 3 spurious M24 splits are gone: flash-list 2.0.2/2.0.3, permissions 5.2.5/5.3.0, gesture-handler 2.27.0/2.27.1. The remaining errors equal M23:
- dev: crypto-js missed in synth-00; wrong version for screens ×2, fast-image, immer
- test: wrong version for i18next, fast-image, immer; FP @react-navigation/native in synth-24

## M26: track 2, real-app Hermes builds + bundled truth (2026-10-12)
Run settings: executor effort high, model 'Grok Bot, default model', no cloud agents.
All builds: release Metro bundle (dev=false, minify=true) + source map, then the app's own hermesc -O -> apps/out/<app>.hbc. Bundled truth from the source map (eval/bundled_truth.py; fix: Expo maps use root-relative "/node_modules/..." paths).

| app | RN | bundled pkgs | modules |
|---|---|---|---|
| mattermost-rel (Mattermost 2.29.1) | 0.76.9 | 271 | 4496 |
| rainbow-rel (Rainbow 1.9.70-1) | 0.74.3 | 530 | 7497 |
| mattermost-2.20.0 | 0.74.5 | 252 | 4226 |
| bluesky-1.96.0 (Expo) | 0.76.3 | 246 | 3828 |
| inaturalist-1.0.7 | 0.76.9 | 202 | 6447 |
| valora-1.110.0 | 0.76.5 | 424 | 6060 |
| bluewallet-7.0.9 | 0.76.6 | 264 | 2494 |
| leather-mobile-1.7.0 | 0.74.1 | 59 | 821 |

Not built (blockers):
- metamask-7.47.3: git dep aprock/react-native-tcp tarball returns 404.
- zeus-0.11.4: git dep aprock/react-native-os tarball returns 404.
- joplin-3.3.13: root.tsx uses `import x = require()`, which Metro's Babel TS transform rejects. It needs Joplin's own tsc prebuild.
- bluewallet: the git dep rn-qr-generator repo is gone. Swapped to npm rn-qr-generator@1.4.3 (native QR wrapper) in package.json/lock, then ran the app's postinstall (rn-nodeify).

## M27: track 2, split v3 frozen (2026-10-12 21:40 CT)
- data/split.v3.json (sha256 in data/split.v3.sha256; copy in docs/eval/data/). It keeps v2 dev/test unchanged and adds a `real` held-out set of 8 apps with their hbc sha256 and RN version: mattermost-rel, rainbow-rel, mattermost-2.20.0, bluesky, inaturalist, valora, bluewallet, leather-mobile.
- Frozen before any go_m23 run on these apps. These apps are never used for tuning.
- oneclass.py: new `--split=` flag; partition `real` takes RN from the split entry. Dev reproduces 0.970.

## M28: track 2, frozen go_m23 scored once on split v3 `real` (2026-10-12)
Same frozen flags and DB (hedis_rel), with --subsha identity. App triples were dumped to runs/apps_rel (the existing mattermost-rel/rainbow-rel dumps were byte-identical). Run as go_m23v3, scored once, no tuning.

**real (8 apps): P .530 / R .784 / F1 0.632 (tp98 fp87 fn27)**, vs synth dev 0.970 / test 0.975.

Per app F1:
- mattermost-rel .727
- rainbow-rel .720
- inaturalist .700
- mattermost-2.20.0 .667
- bluewallet .615
- bluesky .529
- valora .524
- leather .421

Errors:
- FP, package not bundled (67) is the dominant error. Precision collapses on large real bundles: parse, immer, i18next, axios, dayjs, @nozbe/watermelondb, react-native-mmkv and react-native-share are predicted in apps that don't bundle them. This looks like shared or generic code matching that synth apps did not exercise.
- FN, wrong version (16): mostly react-native-screens and react-native-svg, plus @react-navigation/native 7.x predicted as 6.1.18.
- FN, version not in DB (9): @gorhom/bottom-sheet ×5 (root fingerprint jobs failed, see M25), fast-image 8.5.11/8.6.3, clipboard 1.14.1, reanimated nightly.
Details: docs/eval/results/go_m23v3/.

## M29: track 3, RN 0.79 env set up and fingerprinting started (2026-10-12)
- envs/rn079 is a copy of the rn076 template with react-native 0.79.7, react 19.0.0, @react-native/* 0.79.7 and CLI 18.0.1. package.json is in docs/eval/envs/rn079.
- It needs `.npmrc legacy-peer-deps=true` in the env and in its envs/work/rn079_w* worker copies, because the react 19 peer conflict blocks every install. Attempts 1 and 2 failed on this; their logs are kept as logs/fp-rel079-stats.attempt{1,2}.jsonl.
- eval/fp_rn079.sh runs root release fingerprints and then subpath fingerprints for all 1518 target package-versions, using bin/hedis-sub3 with the same tooling filter and no-subpath marker as M25. Stats go to logs/fp-rel079-stats.jsonl and logs/fp-sub079-stats.jsonl. The rn079 release baseline is built automatically.
- Early results: 172/202 ok. The failures are mostly parse: Metro 0.82, as shipped with RN 0.79, fails to resolve `ws`. This is real RN 0.79 behaviour, so there is no config workaround.

## M30: track 4, real_dev / real_test split frozen (2026-10-14 15:00 CT)
Run settings: executor effort high, model 'Grok Bot, default model'.
- data/split.v3dev.json (sha256 c5bad439…95bb in data/split.v3dev.sha256; copy in docs/eval/data/). Splits the 8 v3 `real` apps by app family so both Mattermost versions stay together:
  - real_dev (tuning): mattermost-rel, mattermost-2.20.0, bluewallet-7.0.9, leather-mobile-1.7.0
  - real_test (held out, scored once at the end): rainbow-rel, bluesky-1.96.0, inaturalist-1.0.7, valora-1.110.0
- Frozen before any tuning. It is shared with the dependency-graph work, which is owned elsewhere; this track only does DF penalties and module clustering.
- Caveat: M28 already printed aggregate go_m23 errors for all 8 apps. Nothing is tuned on real_test from here on.
- oneclass.py accepts `real_dev` / `real_test` partitions. Baseline go_m23v3 real_dev: P .511 R .852 F1 0.639 (tp46 fp44 fn8).

## M31: track 4, generic-function penalty + module clustering on real_dev (2026-10-14)
Run settings: executor effort high, model 'Grok Bot, default model'. Protocol change: all 8 v3 real apps were already scored in M28, so data/split.v3dev.json now makes all 8 of them **real_dev**. The held-out real_test is a new set of never-scored apps (split.v4, M32). The dependency-graph filter is owned by the Researcher track (research/depfilter) and is not done here.
- bin/hedis-norm-hits = go_m23 matcher plus `HEDIS_HITS_OUT`, which dumps, for each predicted package@version, the app function indices hit by its keys (and each key's package df). Predictions are identical to go_m23: synth dev 0.970, real_dev 0.632.
- eval/m31_filter.py works post hoc on the .hbc dump only:
  - `floor`: drop predictions with score < t.
  - `gen` (generic-function down-weighting): score × (non-generic unique hits / unique hits). A hit is generic if its key occurs >= m times in the same app (shared tiny helper), or (`gd`) if it was hit for the same package in >= d reference apps where that package scored < 0.2. That reference table (gentab) is built leave-one-out from synth dev + real_dev and is label-free.
  - `clus` (module clustering): sort the package's unique-key hits by function index and split them into runs with gap <= 64 functions (Hermes emits a Metro module's functions contiguously). Keep the package if the largest run >= c, or if the largest run >= 3 and largest run / hits >= f. Packages with < 10 unique keys or 0 unique hits are exempt (dep-implied crypto-js would otherwise be lost on synth).
- Grids: docs/eval/results/m31/grid{1,2,3}.txt. Synth dev stays at 0.970 for every variant below.

| variant (frozen in data/m31_variants.json) | real_dev P / R / F1 |
|---|---|
| none (go_m23) | .530 / .784 / 0.632 |
| floor 0.2 (Researcher's) | .815 / .776 / 0.795 |
| clus alone (c40 f0.9) | .748 / .784 / 0.766 |
| gen alone (needs a floor to act; m3 + floor 0.2) | .822 / .776 / **0.798** |
| gen + clus + floor 0.2 | .822 / .776 / 0.798 |
| genref d1 + floor 0.15 | .817 / .784 / 0.800 |

- Reading: almost all of the not-bundled FPs have score < 0.1 with scattered hits. Clustering alone removes 54 of the 87 FPs without losing a TP. Once a 0.2 floor is in place, though, clustering adds nothing, and gen adds only 1 FP removed (+0.003), which is within noise.
- Remaining real_dev errors are mostly wrong versions, which these filters can't fix. The few high-score not-bundled FPs look like code shared under another package name or vendored code: rainbow watermelondb .705, leather url-polyfill .945, valora keychain .818 / webview .740.
- Headline (from the .hbc alone): gen_floor.

## M32: track 4, split v4 real_test frozen (2026-10-14 16:25 CT)
Run settings: executor effort high, model 'Grok Bot, default model'.
- Same pipeline as M26 (eval/build_oss.sh: release Metro bundle dev=false minify=true + source map, app's own hermesc -O, bundled truth from the map). 8 new OSS apps, never scored, disjoint from the 8 v3 apps:

| app | RN | bundled pkgs | modules |
|---|---|---|---|
| alby-go-1.13.1 | 0.76.8 | 156 | 1850 |
| karakeep-mobile-0.25.0 | 0.76.3 | 116 | 3116 |
| trezor-mobile-25.5.3 (suite-native) | 0.76.9 | 364 | 4653 |
| expensify-9.0.91 | 0.76.3 | 214 | 3489 |
| rabby-mobile-0.6.44 | 0.76.9 | 463 | 14437 |
| edge-4.8.0 | 0.74.1 | 283 | 2814 |
| eigen-8.73.0 (Artsy) | 0.76.9 | 307 | 5111 |
| chatwoot-4.3.0 | 0.76.9 | 233 | 2967 |

- data/split.v4.json, partition `real_test` (hbc sha256, RN, truth sha256 per app); sha256 168a81f8…8626 in data/split.v4.sha256; copy + truth in docs/eval/. Frozen before any scoring (only `--dump-only` triples exist).
- Build fixes (env only, no app code changes): bundled_truth.py resolves Expo server-root paths in monorepos; edge built with plain `react-native bundle` (RNBUNDLE=1, its metro config is not Expo); eigen needs relay-compiler + Echo.json fetch + metaflags.json; chatwoot needs an `index.js` registerRootComponent shim (pnpm breaks expo/AppEntry's ../../App) and a hoisted pnpm install (undeclared redux); trezor/rabby need their workspace lib builds (PRE in eval/oss_apps_v4*.txt).
- Not built: notesnook-3.0.31 (git dep yamill/react-native-orientation repo gone), uniswap-mobile-1.46 (expo-barcode-scanner config plugin unresolvable).

## M33: track 4, frozen variants scored once on split v4 real_test (2026-10-14 16:35 CT)
Run settings: executor effort high, model 'Grok Bot, default model'. split.v4.json checked against its sha256 (OK) before the run. A single run, with no tuning: eval/run_hits.sh h32 (frozen go_m23 flags, hedis_rel), then m31_filter.py with the frozen data/m31_variants.json params, then oneclass.py real_test --subsha --split=data/split.v4.json. Outputs are in docs/eval/results/m32/.

| variant | real_test P / R / F1 | tp/fp/fn |
|---|---|---|
| go_m23 unfiltered (h32) | .436 / .607 / 0.508 | 82/106/53 |
| floor 0.2 (h32floor) | .672 / .593 / **0.630** | 80/39/55 |
| gen_floor m3+0.2 (h32genfloor) | .667 / .578 / 0.619 | 78/39/57 |

Per app F1 (unfiltered / floor / gen_floor):
- alby-go .375 / .571 / .571
- karakeep .400 / .600 / .600
- trezor .579 / .733 / .733
- expensify .632 / .828 / .786
- rabby .571 / .627 / .600
- edge .564 / .733 / .733
- eigen .375 / .368 / .368
- chatwoot .524 / .629 / .629

Reading:
- The floor generalises: not-bundled FPs fall from 68 to 4 (watermelondb, url-polyfill again).
- gen adds nothing on held-out apps and costs 2 TPs (expensify, rabby). The M31 +0.003 was noise.
- Once the floor is applied, the remaining errors are wrong versions (25: svg, webview, gesture-handler, and @react-navigation/native 7.x predicted as 6.1.18) and versions not in the DB (16, including @gorhom/bottom-sheet ×5, fast-image).
- Real held-out F1 is 0.63, against 0.80 on real_dev and 0.97 on synth.

## M35: version sets + vulnerability-level eval, frozen on real_dev (2026-10-14 17:10 CT)
Run settings: executor effort high, model 'Grok Bot, default model'.
- DB pin: hedis_rel is read-only here and M34 was inserting rows, so the matcher takes `HEDIS_MAX_OID` (ObjectId cutoff on hash rows and packages; 6acff32c… = 16:25 CT, the M31-M33 state). Check: the pinned rerun (v35) reproduces h31 predictions on all 8 real_dev + 15 synth dev apps (same package, version, score).
- Go (feat/m35-vset, pkg/pipeline/normmatch.go, pkg/cmd/packages.go), new matcher flags:
  - `--floor x`: drop final predictions (incl. dep-implied) with score < x. This is the M31 post-hoc floor, now in the matcher.
  - `--vset-margin m` / `--vset-samekeys`: per predicted package, `version_set` in the JSON output. It holds the chosen version plus every DB version whose version-selector score (the same IDF formula on root keys with dfRoot under --rootver) is >= chosen - m, plus, with samekeys, every version whose set of app-matched selector keys equals the chosen one's.
  - `HEDIS_VSET_OUT` (eval only) dumps every version's selector score and matched keys, for the Python prototype.
  - `HEDIS_MAX_OID` is also honoured by the thesis fast loader.
- Go vs prototype: eval/m35_check_go.py compares Go `--floor 0.2 --vset-margin 0.003 --vset-samekeys` (v35g) with the Python prototype on the v35 dumps: 267 package predictions over real_dev + synth dev, **0 mismatches**. Go strict real_dev 0.795 = M31 floor.
- Advisories: eval/fetch_advisories.py, OSV.dev npm API (GHSA records), all 30 universe packages. Output data/advisories_osv.json (fetched 2026-10-14 ~21:50 UTC, sha256 3fc5e620…7207 in data/advisories_osv.sha256): 100 advisories, 16 packages with any (axios 56, better-auth 22).
- eval/m35_eval.py (bundled truth, universe = 30 targets, same-bundle-SHA versions count as one, like oneclass --subsha):
  - strict: exact-version P/R/F1.
  - set: a hit if any truth version is in the set. Reported as set-hit rate (hits / truth packages), mean set size, and set P/R/F1.
  - adv: (app, advisory) P/R/F1. Truth = advisories affecting any bundled truth version. Predicted = advisories affecting any version in a predicted set (adv_strict uses the chosen version only).
  - app: 'has any vulnerable dep' accuracy.
- Margin grid on real_dev (floor 0.2; set-hit / mean set; no samekeys → samekeys):

| margin | set-hit | mean set | adv P/R |
|---|---|---|---|
| 0 (exact ties) | .846 | 4.82 → 5.92 | 1.0 / .985 |
| 0.001 | .862 | 5.04 → 6.14 | 1.0 / 1.0 |
| 0.002 | .870 | 5.15 → 6.25 | 1.0 / 1.0 |
| **0.003** | **.878** | 5.55 → **6.65** | 1.0 / 1.0 |
| 0.005 | .878 | 6.08 → 7.18 | .985 / 1.0 |
| 0.05 | .878 | 12.1 → 12.6 | .944 / 1.0 (app 6/8) |

- Frozen (data/m35_frozen.json): v35g = go_m23 + --floor 0.2 + --vset-margin 0.003 --vset-samekeys. 0.003 is the smallest margin at the max real_dev set-hit rate. samekeys stays on because it is part of the set definition (+1.1 set size on real_dev, no hit gain). Thesis baseline t35 = exact-3-hash HMV + versions within 0.003 of the top score.
- v35g results:

| | strict P/R/F1 | set-hit | mean set | adv P/R/F1 (set) | adv F1 strict | app acc |
|---|---|---|---|---|---|---|
| real_dev (8) | .815/.776/0.795 | .878 | 6.65 | 1.000/1.000/1.000 (tp67) | 0.992 | 8/8 (2 vulnerable) |
| synth dev (15) | .973/.966/0.970 | .993 | 4.41 | .957/1.000/0.978 (tp315 fp14) | 0.990 | 15/15 (strict 14/15) |

- Reading:
  - Exact ties alone give sets of ~5 versions (many releases have identical fingerprints), and they recover most wrong-version errors (strict .776 → set .846 R).
  - Only 2 of 8 real_dev apps bundle a vulnerable universe version (rainbow, inaturalist: axios), so the real_dev advisory metric is coarse.
  - Sets trade advisory precision for recall on synth (fp14: crypto-js / react-native-keys / document-picker sets that span a fix boundary).
