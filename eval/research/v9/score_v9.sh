#!/usr/bin/env bash
# score_v9.sh <M> <bin> <maxoid>: v9 one-shot M40 driver. Mirrors score_v8.sh + eval/run_m40.sh @81d5493
# (M39 flags + --floors data/floors_m40.json, then eval/m40/vsel.py <M>_raw -> <M>). Verifies split, triples, gt sha256 first.
cd /workspace/hedis-eval; M=$1; BIN=$2; PIN=$3; VSEL_PY=${VSEL_PY:-/workspace/hedis-wt-m40v9/eval/m40/vsel.py}
echo "a32b9b14f1c05dba7515355554904b407ee54fddefecd8ecabfa9e59f173125c  data/split.v9.json" | sha256sum -c || { echo "ABORT: split sha256"; exit 2; }
python3 - <<'PY' || { echo "ABORT: triples/gt sha256 mismatch"; exit 2; }
import json,hashlib,sys
d=json.load(open('data/split.v9.json'))['real_test']; bad=0
for a,m in d.items():
    for p,k in ((f'runs/apps_rel/{a}.triples','triples_sha256'),(f'research/v9/frozen/{a}.gt.json','gt_sha256'),(f'research/v9/frozen/{a}.hbc','hbc_sha256')):
        try: h=hashlib.sha256(open(p,'rb').read()).hexdigest()
        except Exception as e: print(a,'MISSING',p); bad=1; continue
        ok=h==m[k]; print(a,k,'OK' if ok else f'MISMATCH {h}'); bad|=not ok
sys.exit(bad)
PY
APPS=$(python3 -c "import json;print(' '.join(json.load(open('data/split.v9.json'))['real_test']))")
mkdir -p runs/$M
export MONGO_CONNECTION_STRING=mongodb://127.0.0.1:27017 HEDIS_PACKAGES_DB=hedis_rel HEDIS_MAX_OID=$PIN HEDIS_RN_FALLBACK=nearest HEDIS_RN_LOG=runs/$M/rn_fallback.log
unset HEDIS_EXCLUDE_RN
for a in $APPS; do
  HEDIS_HITS_OUT=runs/$M/$a.hits.json nice -n 10 $BIN packages -d --from-dump runs/apps_rel/$a.triples -o runs/$M/$a.txt \
    --norm nsnc --thresholds data/thresholds_m4f_dlr.json --lowuniq 2 --lowdf 2 --dep 0.6 --deps data/deps.json --rootver \
    --floor 0.2 --floors data/floors_m40.json --vset-margin 0.003 --vset-samekeys --unsupported-cov 0.40 > runs/$M/$a.log 2>&1; rc=$?
  if grep -qiE "unsupported|not supported" runs/$M/$a.log; then st=unsupported_rc$rc; else st=rc$rc; fi
  echo "$a $st" >> runs/$M/status.txt
  if [ $rc = 0 ] && [ -s runs/$M/$a.txt.json ] && python3 scripts/to_predictions.py runs/$M/$a.txt.json "$a" "${M}_raw" > /tmp/p.json; then
    mv /tmp/p.json data/predictions/${M}_raw_$a.json
    python3 $VSEL_PY ${M}_raw $M --min=2 --date=off --pkgs=react-native-safe-area-context,react-native-screens,react-native-reanimated $a >> runs/$M/vsel.log 2>&1 || echo "$a VSEL_FAIL" >> runs/$M/status.txt
  else for p in ${M}_raw $M; do echo "{\"app\":\"$a\",\"method\":\"$p\",\"status\":\"$st\",\"predictions\":[]}" > data/predictions/${p}_$a.json; done; fi
done
