#!/usr/bin/env bash
# rescore_m41.sh <oneclass.py> : every dev number since M39 via split partitions (no --apps=)
cd /workspace/hedis-eval; O=${1:-/workspace/hedis-wt-fix/eval/oneclass.py}; S=--split=data/split.m41dev.json
for p in m38f m39f $(ls -d runs/abl_s*/ | xargs -n1 basename | sort) m40f_raw m40f; do
  for part in real_dev dev real_rn085_dev3 real_rn085_v8dev real_rn085_dev10; do
    printf '%s\t%s\t%s\n' $p $part "$(python3 $O $p $part --subsha $S 2>&1 | grep -oE 'F1 .*|Error.*' )"; done; done
