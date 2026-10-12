#!/usr/bin/env bash
# score_split.sh <prefix>  -> pkg & pkg+ver P/R/F1 on dev and test (prod_closure + direct), universe = targets
set -euo pipefail; E=/workspace/hedis-eval; P=$1; cd $E
for s in dev test; do
  d=$(mktemp -d); for a in $(python3 -c "import json;print(' '.join(json.load(open('$E/data/split.json'))['$s']))"); do
    [ -f $E/data/predictions/${P}_$a.json ] && cp $E/data/predictions/${P}_$a.json $d/; done
  for T in prod_closure direct; do
    printf "%s %-4s %-12s " "$P" $s $T; python3 $E/scripts/eval.py --pred-dir $d --prefix ${P}_ --truth $T --universe $E/data/universe_targets.json | sed -n 3p
  done; rm -rf $d
done
