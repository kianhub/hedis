#!/usr/bin/env bash
# run_abl.sh: per-change ablation on dev (real_dev 8 + synth dev 15 + RN 0.85 dev 3), DB pin M39 cutoff, triples input.
# Steps are cumulative (s00 thesis t35 ... s10 = M39). Resumable: runs/abl_<step>/<app>.done ; runs/abl_<step>.done
set -uo pipefail; cd /workspace/hedis-eval
export MONGO_CONNECTION_STRING=mongodb://127.0.0.1:27017 HEDIS_PACKAGES_DB=hedis_rel HEDIS_MAX_OID=6ad180000000000000000000 HEDIS_RN_FALLBACK=nearest
APPS="$(ls data/predictions | grep '^m38f_' | sed 's/^m38f_//;s/\.json$//' | grep -vE 'blink|breathly|zeus' | tr '\n' ' ') blink-beta-3.0.31 breathly-ios-2.2-18 zeus-13.1.3"
B="--norm nsnc --unsupported-cov 0"
declare -A F E
F[s01_norm]="$B";                                        E[s01_norm]="HEDIS_ABL_PLAIN=1 HEDIS_NO_SUB=1 HEDIS_EXCLUDE_RN=0.85"
F[s02_rare_thr]="$B --thresholds data/thresholds_m4f_dlr.json"; E[s02_rare_thr]="HEDIS_NO_SUB=1 HEDIS_EXCLUDE_RN=0.85"
F[s03_lowuniq]="${F[s02_rare_thr]} --lowuniq 2 --lowdf 2";  E[s03_lowuniq]="${E[s02_rare_thr]}"
F[s04_dep]="${F[s03_lowuniq]} --dep 0.6 --deps data/deps.json"; E[s04_dep]="${E[s02_rare_thr]}"
F[s05_rootver]="${F[s04_dep]} --rootver";               E[s05_rootver]="${E[s02_rare_thr]}"
F[s06_floor]="${F[s05_rootver]} --floor 0.2";           E[s06_floor]="${E[s02_rare_thr]}"
F[s07_vset]="${F[s06_floor]} --vset-margin 0.003 --vset-samekeys"; E[s07_vset]="${E[s02_rare_thr]}"
F[s08_subpath]="${F[s07_vset]}";                        E[s08_subpath]="HEDIS_EXCLUDE_RN=0.85"
F[s09_unsupcov]="${F[s07_vset]/--unsupported-cov 0/--unsupported-cov 0.40}"; E[s09_unsupcov]="HEDIS_EXCLUDE_RN=0.85"
F[s10_rn085]="${F[s09_unsupcov]}";                      E[s10_rn085]=""
for S in ${STEPS:-s00_t35 s01_norm s02_rare_thr s03_lowuniq s04_dep s05_rootver s06_floor s07_vset s08_subpath s09_unsupcov s10_rn085}; do
  [ -f runs/abl_$S.done ] && continue
  M=abl_$S; mkdir -p runs/$M; echo "$S start $(date +%T)"
  for a in $APPS; do
    [ -f runs/$M/$a.done ] && continue
    rc=0
    if [ $S = s00_t35 ]; then
      env HEDIS_NO_SUB=1 HEDIS_EXCLUDE_RN=0.85 bin/hedis-abl packages -d --from-dump runs/apps_rel/$a.triples -o runs/$M/$a.txt > runs/$M/$a.log 2>&1 || rc=$?
      [ $rc = 0 ] && [ -s runs/$M/$a.txt.json ] && python3 /workspace/hedis-wt-abl/eval/thesis_vset.py runs/$M/$a.txt.json $a $M > data/predictions/${M}_$a.json || rc=${rc/0/9}
    else
      env ${E[$S]} bin/hedis-abl packages -d --from-dump runs/apps_rel/$a.triples -o runs/$M/$a.txt ${F[$S]} > runs/$M/$a.log 2>&1 || rc=$?
      [ $rc = 0 ] && python3 scripts/to_predictions.py runs/$M/$a.txt.json "$a" "$M" > data/predictions/${M}_$a.json
    fi
    [ $rc = 0 ] || printf '{"app": "%s", "method": "%s", "status": "matcher_exit_%s", "predictions": []}\n' "$a" "$M" "$rc" > data/predictions/${M}_$a.json
    echo "$a rc=$rc" >> runs/$M/status.log; touch runs/$M/$a.done
  done
  touch runs/abl_$S.done; echo "$S done $(date +%T)"
done
echo ABL_ALL_DONE
