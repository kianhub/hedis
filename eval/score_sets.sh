#!/usr/bin/env bash
# score_sets.sh <prefix> <truth> <label> <apps...>
cd /workspace/hedis-eval; P=$1; T=$2; L=$3; shift 3; d=$(mktemp -d)
for a in "$@"; do cp data/predictions/${P}_$a.json $d/ 2>/dev/null; done
printf "%-22s %-12s " "$L" $T; python3 scripts/eval.py --pred-dir $d --prefix ${P}_ --truth $T --universe data/universe_targets.json | sed -n 3p; rm -rf $d
