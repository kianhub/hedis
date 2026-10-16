#!/usr/bin/env bash
# run_m40.sh <prefix> <apps...>: M39 frozen flags + bin/hedis-m40 + --floors data/floors_m40.json (per-package floor),
# then eval/m40/vsel.py version-diff Copeland re-selection (<prefix>_raw -> <prefix>). DB pin only if MAXOID is set.
# Resumable: runs/<prefix>/<app>.done
set -uo pipefail; cd /workspace/hedis-eval
M=$1; shift; mkdir -p runs/$M
export MONGO_CONNECTION_STRING=mongodb://127.0.0.1:27017 HEDIS_PACKAGES_DB=hedis_rel HEDIS_RN_FALLBACK=nearest HEDIS_RN_LOG=runs/$M/rn_fallback.log
if [ -n "${MAXOID:-}" ]; then export HEDIS_MAX_OID=$MAXOID; else unset HEDIS_MAX_OID; fi
unset HEDIS_EXCLUDE_RN
VS=${VSEL:---min=2 --date=off --pkgs=react-native-safe-area-context,react-native-screens,react-native-reanimated,react-native-gesture-handler}
for a in "$@"; do
  [ -f runs/$M/$a.done ] && continue
  rc=0
  HEDIS_HITS_OUT=runs/$M/$a.hits.json bin/hedis-m40 packages -d --from-dump runs/apps_rel/$a.triples -o runs/$M/$a.txt \
    --norm nsnc --thresholds data/thresholds_m4f_dlr.json --lowuniq 2 --lowdf 2 --dep 0.6 --deps data/deps.json --rootver \
    --floor 0.2 --floors data/floors_m40.json --vset-margin 0.003 --vset-samekeys --unsupported-cov 0.40 > runs/$M/$a.log 2>&1 || rc=$?
  echo "$a rc=$rc" >> runs/$M/status.log
  if [ $rc = 0 ]; then
    python3 scripts/to_predictions.py runs/$M/$a.txt.json "$a" "${M}_raw" > data/predictions/${M}_raw_$a.json
    python3 /workspace/hedis-wt-m40/eval/m40/vsel.py ${M}_raw $M $VS $a >> runs/$M/vsel.log
  else
    for p in ${M}_raw $M; do printf '{"app": "%s", "method": "%s", "status": "matcher_exit_%s", "predictions": []}\n' "$a" "$p" "$rc" > data/predictions/${p}_$a.json; done
  fi
  touch runs/$M/$a.done
done
touch runs/$M.done
