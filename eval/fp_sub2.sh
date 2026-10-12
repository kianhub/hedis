#!/bin/bash
cd /workspace/hedis-eval; export HEDIS_PACKAGES_DB=hedis_rel
bin/hedis-sub2 fingerprint --release --subpaths --targets db/targets_ext.json --rns rn074,rn076 --envs-root envs --workers 3 --cache cache-rel --stats logs/fp-sub-stats.jsonl && echo SUB_DONE
