#!/usr/bin/env bash
# m31_grid.sh: grid over floor / gen / clus on synth dev + real_dev only (split.v3dev.json); prints F1s
cd /workspace/hedis-eval; S=data/split.v3dev.json
APPS=$(python3 -c "import json;s=json.load(open('$S'));print(' '.join([a for a in s['dev'] if a.startswith('synth')]+list(s['real_dev'])))")
export REFAPPS="$APPS"
while read fl gm gd cc cf; do
  python3 /workspace/hedis/eval/m31_filter.py h31 g31 $fl $gm $gd $cc $cf $APPS
  d=$(python3 /workspace/hedis/eval/oneclass.py g31 dev --subsha --split=$S | sed 's/.*P /P /'); r=$(python3 /workspace/hedis/eval/oneclass.py g31 real_dev --subsha --split=$S | sed 's/.*P /P /')
  echo "fl=$fl gm=$gm gd=$gd cc=$cc cf=$cf | dev $d | real_dev $r"
done
