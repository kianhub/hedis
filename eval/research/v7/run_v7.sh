#!/bin/bash
# v7 builds in sha256 order; stop at 12 BUILD_OK. Copy artifacts to research/v7/frozen, dump triples, delete clone.
E_=/workspace/hedis-eval; cd $E_; L=research/v7/PROGRESS.log
ok=$(ls research/v7/frozen/*.hbc 2>/dev/null | wc -l)
while IFS="|" read N R T SUB ENTRY PRE RNB ENVX; do
  [ -z "$N" ] && continue
  [ "$ok" -ge ${CAP:-12} ] && { echo "TARGET ${CAP:-12} reached"; break; }
  [ -f research/v7/frozen/$N.hbc ] && { echo "SKIP $N"; continue; }
  [ -z "$RETRY" ] && grep -q "^$N " research/v7/attempted.txt 2>/dev/null && { echo "SKIP attempted $N"; continue; }
  export PRE; if [ -n "$RNB" ]; then export RNBUNDLE=1; else unset RNBUNDLE; fi
  free=$(df --output=avail -BG / | tail -1 | tr -dc 0-9); [ "$free" -lt 6 ] && { echo "STOP low disk ${free}G before $N"; echo "$(date '+%F %H:%M') STOP low disk ${free}G" >> $L; break; }
  echo "=== $N $R $T $(date '+%H:%M')"
  ( [ -n "$ENVX" ] && eval "export $ENVX"; timeout 3600 nice -n 15 ionice -c3 bash /workspace/hedis/eval/build_oss.sh "$N" "$R" "$T" "${SUB:-.}" "$ENTRY" > logs/oss_$N.log 2>&1 ); rc=$?
  msg=""
  if [ $rc -eq 0 ] && grep -q BUILD_OK logs/oss_$N.log; then
    cp -p apps/out/$N.hbc apps/out/$N.js apps/out/$N.js.map apps/out/$N.rnversion research/v7/frozen/
    cp -p data/groundtruth/$N.json research/v7/frozen/$N.gt.json
    ver=$(python3 -c "import struct;print(struct.unpack('<QI',open('apps/out/$N.hbc','rb').read(12))[1])")
    rn=$(grep -o '"0\.[0-9]*' apps/out/$N.rnversion | tr -d '"')
    D=bin/hedis-norm-hits; [ "$ver" -ge 98 ] && D=bin/hedis-m37
    nice -n 15 env MONGO_CONNECTION_STRING=mongodb://127.0.0.1:27017 HEDIS_PACKAGES_DB=hedis_rel $D packages -d -a apps/out/$N.hbc --rn-version $rn --dump-app runs/apps_rel/$N.triples --dump-only > logs/dump_$N.log 2>&1
    if [ -s runs/apps_rel/$N.triples ]; then cp -p runs/apps_rel/$N.triples research/v7/triples/; msg="hbc$ver rn$rn dump=$(basename $D) ok"; else msg="hbc$ver rn$rn dump=$(basename $D) FAIL"; fi
    ok=$((ok+1))
  fi
  rm -rf apps/oss/$N ~/.cache/yarn/v6 2>/dev/null
  echo "$N rc=$rc" >> research/v7/attempted.txt
  echo "$(date '+%F %H:%M') RC $N $rc $(grep -o 'BUILD_OK.*' logs/oss_$N.log | head -1) $msg ok=$ok free=$(df --output=avail -BG / | tail -1 | tr -dc 0-9)G" >> $L
done < ${Q:-research/v7/queue.txt}
echo QUEUE_DONE ok=$ok
