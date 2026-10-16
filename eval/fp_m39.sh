#!/bin/bash
# M39: React Native 0.85 / Hermes bytecode 98 fingerprints (root + subpath) for the 30 curated universe packages
# x every version already in hedis_rel.packages at the M38 cutoff (db/targets_m39.json, 2210 pkg@ver).
# Env envs/rn085: RN 0.85.3 + react 19.2.3; hermesc from hermes-compiler 250829098.0.10 symlinked into
# node_modules/react-native/sdks/hermesc/linux64-bin (RN 0.85 no longer ships sdks/hermesc). Binary bin/hedis-m39
# (feat/m39-rn085, includes the M37 bcv98 disassembler). Disk guard: aborts when / has < 5 GB free.
cd /workspace/hedis-eval
export MONGO_CONNECTION_STRING=mongodb://127.0.0.1:27017 HEDIS_PACKAGES_DB=hedis_rel
( while sleep 30; do f=$(df --output=avail -B1G / | tail -1); if [ "$f" -lt 5 ]; then echo "DISK_LOW ${f}G $(date)"; pkill -f 'hedis-m39 fingerprint'; exit; fi; done ) & G=$!
rn=rn085
bin/hedis-m39 fingerprint --release --targets db/targets_m39.json --rns $rn --envs-root envs --workers ${W:-3} --cache cache-rel --stats logs/fp-rel39-$rn-stats.jsonl && echo ROOT_DONE $rn $(date +%H:%M)
bin/hedis-m39 fingerprint --release --subpaths --targets db/targets_m39.json --rns $rn --envs-root envs --workers ${W:-3} --cache cache-rel --stats logs/fp-sub39-$rn-stats.jsonl && echo SUB_DONE $rn $(date +%H:%M) && touch logs/fp-m39-$rn.done
kill $G; echo FP_ALL_DONE $(date +%s)
