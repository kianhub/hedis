#!/bin/bash
# M22: root release fingerprints for added versions, then subpath-entry fingerprints for all targets
cd /workspace/hedis-eval
export HEDIS_PACKAGES_DB=hedis_rel
bin/hedis-sub fingerprint --release --targets db/targets_ext.json --rns rn074,rn076 --envs-root envs --workers 3 --cache cache-rel --stats logs/fp-rel-stats.jsonl && echo ROOT_DONE
bin/hedis-sub fingerprint --release --subpaths --targets db/targets_ext.json --rns rn074,rn076 --envs-root envs --workers 3 --cache cache-rel --stats logs/fp-sub-stats.jsonl && echo SUB_DONE
