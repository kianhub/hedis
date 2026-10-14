#!/usr/bin/env bash
# run_thesis_vset.sh <method> <apps...>: thesis exact-3-hash HMV matcher (no --norm) on apps/out/<app>.hbc, DB pinned with
# HEDIS_MAX_OID like run_vset.sh, then eval/thesis_vset.py (HMV + top-score near-tie set, same 0.003 margin).
set -euo pipefail; cd /workspace/hedis-eval; M=$1; shift; mkdir -p runs/$M
export MONGO_CONNECTION_STRING=mongodb://127.0.0.1:27017 HEDIS_PACKAGES_DB=hedis_rel HEDIS_MAX_OID=${MAXOID:-6acff32c0000000000000000}
for a in "$@"; do
  bin/hedis-vset packages -d -a apps/out/$a.hbc -o runs/$M/$a.txt > runs/$M/$a.log 2>&1
  python3 /workspace/hedis-wt-m35/eval/thesis_vset.py runs/$M/$a.txt.json $a $M > data/predictions/${M}_$a.json
done
