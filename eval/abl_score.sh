#!/usr/bin/env bash
# abl_score.sh: score finished ablation steps (oneclass --subsha) -> docs/eval/results/ablation/scores.tsv
cd /workspace/hedis-eval; O=/workspace/hedis-wt-v2/eval/oneclass.py; [ -f /workspace/hedis-wt-abl/eval/oneclass.py ] && O=/workspace/hedis-wt-abl/eval/oneclass.py
python3 /workspace/hedis-wt-m40/eval/m40/fix_empty.py $(ls runs | grep -oE '^abl_s[0-9]+_[a-z0-9_]+$' | sort -u) >&2
printf 'step\treal_dev\tsynth\trn085x3\tunsupported_rn085\n'
for d in $(ls -d runs/abl_s*/ | sort); do S=$(basename $d); [ -f runs/$S.done ] || continue
  r=$(python3 $O $S real_dev --subsha --split=data/split.v3dev.json | grep -oE 'P.*'); s=$(python3 $O $S dev --subsha --split=data/split.v3dev.json | grep -oE 'P.*')
  z=$(python3 $O $S apps --subsha --apps=blink-beta-3.0.31,breathly-ios-2.2-18,zeus-13.1.3 | grep -oE 'P.*'); u=$(grep -E 'blink|breathly|zeus' runs/$S/status.log | grep -vc 'rc=0')
  printf '%s\t%s\t%s\t%s\t%s/3\n' $S "$r" "$s" "$z" $u; done
