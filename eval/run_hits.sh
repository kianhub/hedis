#!/usr/bin/env bash
# run_hits.sh <method_prefix> <apps...>: frozen go_m23 flags with bin/hedis-norm-hits; also writes runs/<M>/<app>.hits.json (HEDIS_HITS_OUT)
set -euo pipefail; cd /workspace/hedis-eval
M=$1; shift; mkdir -p runs/$M
export MONGO_CONNECTION_STRING=mongodb://127.0.0.1:27017 HEDIS_PACKAGES_DB=hedis_rel
for a in "$@"; do
  HEDIS_HITS_OUT=runs/$M/$a.hits.json bin/hedis-norm-hits packages -d --from-dump runs/apps_rel/$a.triples -o runs/$M/$a.txt \
    --norm nsnc --thresholds data/thresholds_m4f_dlr.json --lowuniq 2 --lowdf 2 --dep 0.6 --deps data/deps.json --rootver > runs/$M/$a.log 2>&1
  python3 scripts/to_predictions.py runs/$M/$a.txt.json "$a" "$M" > data/predictions/${M}_$a.json
done
