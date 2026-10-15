#!/bin/bash
# M37: fingerprints for RN 0.72/0.75/0.78/0.80 (envs/rnXXX with .npmrc legacy-peer-deps + peer-pins), same universe as M34/M35
# (db/targets_ext.m34.json), bin/hedis-sub4, root then subpaths, one RN at a time (memory), 2 workers.
cd /workspace/hedis-eval; export MONGO_CONNECTION_STRING=mongodb://127.0.0.1:27017 HEDIS_PACKAGES_DB=hedis_rel
for rn in ${RNS:-rn072 rn075 rn078 rn080}; do
  [ -f logs/fp-m37-$rn.done ] && continue
  bin/hedis-sub4 fingerprint --release --targets db/targets_ext.m34.json --rns $rn --envs-root envs --workers 2 --cache cache-rel --stats logs/fp-rel37-$rn-stats.jsonl && echo ROOT_DONE $rn $(date +%H:%M)
  bin/hedis-sub4 fingerprint --release --subpaths --targets db/targets_ext.m34.json --rns $rn --envs-root envs --workers 2 --cache cache-rel --stats logs/fp-sub37-$rn-stats.jsonl && echo SUB_DONE $rn $(date +%H:%M) && touch logs/fp-m37-$rn.done
done
echo FP_ALL_DONE $(date +%s)
