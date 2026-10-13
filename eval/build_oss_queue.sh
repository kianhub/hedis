#!/bin/bash
# sequential OSS builds; node_modules removed after each successful build (disk)
cd /workspace/hedis-eval
while read N R T SUB ENTRY; do
  [ -z "$N" ] && continue
  [ -f apps/out/$N.hbc ] && { echo "SKIP $N"; continue; }
  echo "=== $N $R $T $(date)"
  timeout 3600 scripts/build_oss.sh $N $R $T $SUB $ENTRY > logs/oss_$N.log 2>&1; rc=$?
  echo "RC $N $rc"
  [ $rc -eq 0 ] && rm -rf apps/oss/$N/node_modules apps/oss/$N/*/node_modules apps/oss/$N/packages/*/node_modules
done < scripts/oss_apps.txt
echo QUEUE_DONE
