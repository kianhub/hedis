#!/usr/bin/env bash
# run_vset.sh <method_prefix> <apps...>: frozen go_m23 flags with bin/hedis-vset (= hedis-norm-hits + HEDIS_VSET_OUT
# per-version scores + HEDIS_MAX_OID DB pin). Writes runs/<M>/<app>.{txt.json,hits.json,vset.json} and data/predictions/<M>_<app>.json.
# MAXOID default 6acff32c0000000000000000 = 2026-10-14 16:25 CT (hedis_rel state used by M31-M33, before M34 inserts).
set -euo pipefail; cd /workspace/hedis-eval
M=$1; shift; mkdir -p runs/$M
export MONGO_CONNECTION_STRING=mongodb://127.0.0.1:27017 HEDIS_PACKAGES_DB=hedis_rel HEDIS_MAX_OID=${MAXOID:-6acff32c0000000000000000}
for a in "$@"; do
  HEDIS_HITS_OUT=runs/$M/$a.hits.json HEDIS_VSET_OUT=runs/$M/$a.vset.json bin/hedis-vset packages -d --from-dump runs/apps_rel/$a.triples -o runs/$M/$a.txt \
    --norm nsnc --thresholds data/thresholds_m4f_dlr.json --lowuniq 2 --lowdf 2 --dep 0.6 --deps data/deps.json --rootver ${EXTRA:-} > runs/$M/$a.log 2>&1
  python3 scripts/to_predictions.py runs/$M/$a.txt.json "$a" "$M" > data/predictions/${M}_$a.json
done
