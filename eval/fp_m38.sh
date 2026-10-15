#!/bin/bash
# M38: non-leaky missing-version densification under RN 0.72/0.75/0.78/0.80.
# Universe = fill list only (db/targets_m38.json): packages already in hedis_rel.packages
# × stable npm X.Y.Z in [2022-01-01, 2026-10-01] UTC not already in DB. Not the 62 v6 miss list.
# Same pattern as fp_m37.sh: bin/hedis-m37fp, root then subpaths, one RN at a time, 2 workers.
# Resume: logs/fp-m38-rnXXX.done; optional HEDIS_FP_REDO for stale fp_done markers.
cd /workspace/hedis-eval
export HEDIS_FP_REDO=${HEDIS_FP_REDO:-}
export MONGO_CONNECTION_STRING=mongodb://127.0.0.1:27017 HEDIS_PACKAGES_DB=hedis_rel
for rn in ${RNS:-rn072 rn075 rn078 rn080}; do
  [ -f logs/fp-m38-$rn.done ] && continue
  bin/hedis-m37fp fingerprint --release --targets db/targets_m38.json --rns $rn --envs-root envs --workers 2 --cache cache-rel --stats logs/fp-rel38-$rn-stats.jsonl && echo ROOT_DONE $rn $(date +%H:%M)
  bin/hedis-m37fp fingerprint --release --subpaths --targets db/targets_m38.json --rns $rn --envs-root envs --workers 2 --cache cache-rel --stats logs/fp-sub38-$rn-stats.jsonl && echo SUB_DONE $rn $(date +%H:%M) && touch logs/fp-m38-$rn.done
done
echo FP_ALL_DONE $(date +%s)
