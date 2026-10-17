#!/usr/bin/env bash
# m39_v9.sh <M> <bin> <maxoid> [extra flags]: score_v8.sh mirrored on split.v9 (M39 reference; ONLY="apps" restricts)
cd /workspace/hedis-eval; M=$1; BIN=$2; PIN=$3; shift 3
APPS=$(python3 -c "import json;print(' '.join(json.load(open('data/split.v9.json'))['real_test']))")
python3 - <<'PY' || { echo "ABORT: triples sha256 mismatch"; exit 2; }
import json,hashlib,sys
d=json.load(open('data/split.v9.json'))['real_test']; bad=0
for a,m in d.items():
    try: h=hashlib.sha256(open(f'runs/apps_rel/{a}.triples','rb').read()).hexdigest()
    except Exception as e: print(a,'MISSING',e); bad=1; continue
    ok=h==m['triples_sha256']; print(a,'OK' if ok else f'MISMATCH {h}'); bad|=not ok
sys.exit(bad)
PY
mkdir -p runs/$M
export MONGO_CONNECTION_STRING=mongodb://127.0.0.1:27017 HEDIS_PACKAGES_DB=hedis_rel HEDIS_MAX_OID=$PIN HEDIS_RN_FALLBACK=nearest HEDIS_RN_LOG=runs/$M/rn_fallback.log
unset HEDIS_EXCLUDE_RN
for a in ${ONLY:-$APPS}; do
  HEDIS_HITS_OUT=runs/$M/$a.hits.json nice -n 10 $BIN packages -d --from-dump runs/apps_rel/$a.triples -o runs/$M/$a.txt \
    --norm nsnc --thresholds data/thresholds_m4f_dlr.json --lowuniq 2 --lowdf 2 --dep 0.6 --deps data/deps.json --rootver \
    --floor 0.2 --vset-margin 0.003 --vset-samekeys "$@" > runs/$M/$a.log 2>&1; rc=$?
  if grep -qiE "unsupported|not supported" runs/$M/$a.log && [ $rc -ne 0 ]; then st=unsupported; else st=rc$rc; fi
  echo "$a $st" >> runs/$M/status.txt
  if [ -s runs/$M/$a.txt.json ] && python3 scripts/to_predictions.py runs/$M/$a.txt.json "$a" "$M" > /tmp/p.json 2>/dev/null; then mv /tmp/p.json data/predictions/${M}_$a.json
  else echo "{\"app\":\"$a\",\"method\":\"$M\",\"rn_detected\":null,\"status\":\"$st\",\"predictions\":[]}" > data/predictions/${M}_$a.json; fi
done
