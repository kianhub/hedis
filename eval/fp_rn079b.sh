#!/bin/bash
# M35: RN 0.79 resume: re-attempt the failed rn079 root jobs with bin/hedis-sub4 + envs/rn079/peer-pins.txt (legacy-peer-deps env never
# installs peers, so bottom-sheet / reanimated 4 / mmkv 4 failed to resolve them), plus the 2 versions added in M34; then subpaths.
cd /workspace/hedis-eval; export MONGO_CONNECTION_STRING=mongodb://127.0.0.1:27017 HEDIS_PACKAGES_DB=hedis_rel
bin/hedis-sub4 fingerprint --release --targets db/targets_ext.m34.json --rns rn079 --envs-root envs --workers 3 --cache cache-rel --stats logs/fp-rel079b-stats.jsonl && echo ROOT_DONE
bin/hedis-sub4 fingerprint --release --subpaths --targets db/targets_ext.m34.json --rns rn079 --envs-root envs --workers 3 --cache cache-rel --stats logs/fp-sub079b-stats.jsonl && echo SUB_DONE
