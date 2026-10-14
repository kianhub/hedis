#!/usr/bin/env bash
# dump_app.sh <app>: apps/out/<app>.hbc -> runs/apps_rel/<app>.triples (RN forced from apps/out/<app>.rnversion major.minor)
set -euo pipefail; cd /workspace/hedis-eval; a=$1
rn=$(grep -o '"0\.[0-9]*' apps/out/$a.rnversion | tr -d '"')
MONGO_CONNECTION_STRING=mongodb://127.0.0.1:27017 HEDIS_PACKAGES_DB=hedis_rel bin/hedis-norm-hits packages -d -a apps/out/$a.hbc --rn-version $rn --dump-app runs/apps_rel/$a.triples --dump-only > logs/dump_$a.log 2>&1
head -1 runs/apps_rel/$a.triples; wc -l < runs/apps_rel/$a.triples
