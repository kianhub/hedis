#!/usr/bin/env bash
cd /workspace/hedis-eval; R=/workspace/hedis/eval
APPS=$(python3 -c "import json;s=json.load(open('data/split.json'));print(' '.join([a for a in list(s['dev'])+list(s['test']) if a!='rainbow-1.9.70']))")
for K in nsnc ns; do
  [ -f data/predictions/g_${K}_a_synth-09-rn076.json ] || $R/run_norm.sh g_${K}_a $APPS -- --norm $K --thresholds data/thr0.json && echo "done a $K"
  python3 $R/tune_thresholds.py g_${K}_a data/thresholds_g_$K.json
  $R/run_norm.sh g_${K}_ab $APPS -- --norm $K --thresholds data/thresholds_g_$K.json && echo "done ab $K"
  $R/run_norm.sh g_${K}_abc $APPS -- --norm $K --thresholds data/thresholds_g_$K.json --tiebreak && echo "done abc $K"
done
echo ALLDONE
