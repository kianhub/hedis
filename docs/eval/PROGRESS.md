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
