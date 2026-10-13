#!/bin/bash
# sequential OSS builds; node_modules removed after each successful build (disk)
cd /workspace/hedis-eval
while IFS="|" read N R T SUB ENTRY PRE; do
  export PRE
  [ -z "$N" ] && continue
  [ -f apps/out/$N.hbc ] && { echo "SKIP $N"; continue; }
  echo "=== $N $R $T $(date)"
  timeout 3600 scripts/build_oss.sh "$N" "$R" "$T" "${SUB:-.}" "$ENTRY" > logs/oss_$N.log 2>&1; rc=$?
  echo "RC $N $rc"
  : # node_modules kept for truth re-runs
done < ${QUEUE:-scripts/oss_apps.txt}
echo QUEUE_DONE
