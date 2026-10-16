#!/usr/bin/env bash
# run_m39.sh (M39: bin/hedis-m39, cutoff after RN 0.85 fp build) -- was run_m38.sh <prefix> <apps...>: frozen M37 flags (run_m37.sh) with bin/hedis-m39 (feat/m38-version-fill) +
# --unsupported-cov 0.40 (M38 unsupported RN/bytecode signal). Env: MAXOID (default M38 cutoff 6ad180000000000000000000),
# FALLBACK=nearest|union. An app the matcher refuses (exit != 0, e.g. status=unsupported_bytecode) is logged to
# runs/<prefix>/status.log and scored with an empty prediction list.
set -euo pipefail; cd /workspace/hedis-eval
M=$1; shift; mkdir -p runs/$M
export MONGO_CONNECTION_STRING=mongodb://127.0.0.1:27017 HEDIS_PACKAGES_DB=hedis_rel HEDIS_MAX_OID=${MAXOID:-6ad180000000000000000000}
export HEDIS_RN_FALLBACK=${FALLBACK:-nearest} HEDIS_RN_LOG=runs/$M/rn_fallback.log
unset HEDIS_EXCLUDE_RN
for a in "$@"; do
  rn=$(head -1 runs/apps_rel/$a.triples | awk '{print $3}')
  echo "$a rn=$rn" >> runs/$M/rn_fallback.log
  rc=0
  HEDIS_HITS_OUT=runs/$M/$a.hits.json bin/hedis-m39 packages -d --from-dump runs/apps_rel/$a.triples -o runs/$M/$a.txt \
    --norm nsnc --thresholds data/thresholds_m4f_dlr.json --lowuniq 2 --lowdf 2 --dep 0.6 --deps data/deps.json --rootver \
    --floor 0.2 --vset-margin 0.003 --vset-samekeys --unsupported-cov 0.40 > runs/$M/$a.log 2>&1 || rc=$?
  echo "$a rc=$rc" >> runs/$M/status.log
  if [ $rc = 0 ]; then
    python3 scripts/to_predictions.py runs/$M/$a.txt.json "$a" "$M" > data/predictions/${M}_$a.json
  else
    printf '{"app": "%s", "method": "%s", "status": "matcher_exit_%s", "predictions": []}\n' "$a" "$M" "$rc" > data/predictions/${M}_$a.json
  fi
done
