#!/bin/bash
# M29: RN 0.79 coverage: root release fingerprints then subpath fingerprints for all targets on envs/rn079 (RN 0.79.7)
cd /workspace/hedis-eval; export HEDIS_PACKAGES_DB=hedis_rel
bin/hedis-sub3 fingerprint --release --targets db/targets_ext.json --rns rn079 --envs-root envs --workers 3 --cache cache-rel --stats logs/fp-rel079-stats.jsonl && echo ROOT_DONE
bin/hedis-sub3 fingerprint --release --subpaths --targets db/targets_ext.json --rns rn079 --envs-root envs --workers 3 --cache cache-rel --stats logs/fp-sub079-stats.jsonl && echo SUB_DONE
