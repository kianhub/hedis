#!/usr/bin/env bash
# run_m37.sh <prefix> <apps...>: frozen v35g config (go_m23 + --floor 0.2 --vset-margin 0.003 --vset-samekeys) with bin/hedis-m37
# (M37 RN fallback). Env: MAXOID (default 6ad005ec0000000000000000), FALLBACK=nearest|union, LOO=1 hides each app's own RN
# (HEDIS_EXCLUDE_RN = RN in the triples header) to force the fallback. Fallback notes -> runs/<prefix>/rn_fallback.log.
set -euo pipefail; cd /workspace/hedis-eval
M=$1; shift; mkdir -p runs/$M
export MONGO_CONNECTION_STRING=mongodb://127.0.0.1:27017 HEDIS_PACKAGES_DB=hedis_rel HEDIS_MAX_OID=${MAXOID:-6ad005ec0000000000000000}
export HEDIS_RN_FALLBACK=${FALLBACK:-nearest} HEDIS_RN_LOG=runs/$M/rn_fallback.log
for a in "$@"; do
  rn=$(head -1 runs/apps_rel/$a.triples | awk '{print $3}')
  if [ "${LOO:-0}" = 1 ]; then export HEDIS_EXCLUDE_RN=$rn; else unset HEDIS_EXCLUDE_RN; fi
  echo "$a rn=$rn exclude=${HEDIS_EXCLUDE_RN:-}" >> runs/$M/rn_fallback.log
  HEDIS_HITS_OUT=runs/$M/$a.hits.json bin/hedis-m37 packages -d --from-dump runs/apps_rel/$a.triples -o runs/$M/$a.txt \
    --norm nsnc --thresholds data/thresholds_m4f_dlr.json --lowuniq 2 --lowdf 2 --dep 0.6 --deps data/deps.json --rootver \
    --floor 0.2 --vset-margin 0.003 --vset-samekeys > runs/$M/$a.log 2>&1
  python3 scripts/to_predictions.py runs/$M/$a.txt.json "$a" "$M" > data/predictions/${M}_$a.json
done
