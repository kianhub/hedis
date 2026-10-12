#!/usr/bin/env bash
# run_norm.sh <method_prefix> <apps...> -- <hedis --norm flags>   (Go normalized matcher -> predictions)
set -euo pipefail; cd /workspace/hedis-eval
M=$1; shift; APPS=(); EXTRA=()
while [ $# -gt 0 ]; do if [ "$1" = "--" ]; then shift; EXTRA=("$@"); break; fi; APPS+=("$1"); shift; done
export MONGO_CONNECTION_STRING=mongodb://127.0.0.1:27017 HEDIS_PACKAGES_DB=hedis_rel
mkdir -p runs/$M
for a in "${APPS[@]}"; do
  bin/hedis-norm packages -d --from-dump runs/apps_rel/$a.triples -o runs/$M/$a.txt "${EXTRA[@]}" > runs/$M/$a.log 2>&1
  python3 scripts/to_predictions.py runs/$M/$a.txt.json "$a" "$M" > data/predictions/${M}_$a.json
done
