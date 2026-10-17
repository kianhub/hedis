#!/usr/bin/env bash
# t35_v9.sh <M> <pin>: thesis baseline (hedis-vset + thesis_vset.py) on frozen v9 hbc
cd /workspace/hedis-eval; M=$1; mkdir -p runs/$M
export MONGO_CONNECTION_STRING=mongodb://127.0.0.1:27017 HEDIS_PACKAGES_DB=hedis_rel HEDIS_MAX_OID=$2 HEDIS_RN_FALLBACK=nearest
for a in $(python3 -c "import json;print(' '.join(json.load(open('data/split.v9.json'))['real_test']))"); do
  nice -n 10 bin/hedis-vset packages -d -a research/v9/frozen/$a.hbc -o runs/$M/$a.txt > runs/$M/$a.log 2>&1; rc=$?
  if [ -s runs/$M/$a.txt.json ] && python3 /workspace/hedis/eval/thesis_vset.py runs/$M/$a.txt.json $a $M > data/predictions/${M}_$a.json 2> runs/$M/$a.tv.log; then echo "$a ok" >> runs/$M/status.txt
  else echo "$a FAIL rc$rc $(grep -m1 -oiE 'needs an array|not supported|unsupported[^ ]*|bytecode version[^.]*' runs/$M/$a.log)" >> runs/$M/status.txt
    echo "{\"app\":\"$a\",\"method\":\"$M\",\"status\":\"crash\",\"predictions\":[]}" > data/predictions/${M}_$a.json; fi
done
