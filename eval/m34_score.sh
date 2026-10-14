#!/bin/bash
# M34: rescore real_dev (split.v3dev) + synth dev with frozen go_m23 (eval/run_hits.sh) + floor 0.2 (m31_filter.py) on hedis_rel after M34 inserts
set -euo pipefail; cd /workspace/hedis-eval; M=${1:-h34b}
DEV=$(python3 -c "import json;s=json.load(open('data/split.v3dev.json'));print(' '.join(list(s['real_dev'])+[a for a in s['dev'] if a.startswith('synth')]))")
bash /workspace/hedis/eval/run_hits.sh $M $DEV
python3 /workspace/hedis/eval/m31_filter.py $M ${M}floor 0.2 0 0 0 0 $DEV
mkdir -p /workspace/hedis/docs/eval/results/m34
for V in $M ${M}floor; do
  python3 /workspace/hedis/eval/oneclass.py $V real_dev --subsha --split=data/split.v3dev.json > /workspace/hedis/docs/eval/results/m34/$V.real_dev.txt
  python3 /workspace/hedis/eval/oneclass.py $V dev --subsha --split=data/split.v3dev.json > /workspace/hedis/docs/eval/results/m34/$V.dev.txt
done
