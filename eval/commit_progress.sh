#!/usr/bin/env bash
# commit_progress.sh <branch> <msg> <progress-text-file>: append to PROGRESS.md, commit on a feature branch off v2, merge into v2, push both.
set -e; cd /workspace/hedis-eval; cat "$3" >> PROGRESS.md; cp PROGRESS.md /workspace/hedis/docs/eval/
mkdir -p /workspace/hedis/docs/eval/results; cp data/thresholds_*.json /workspace/hedis/docs/eval/results/ 2>/dev/null || true
cd /workspace/hedis; git checkout -q v2; git checkout -q -b "$1"; git add eval docs; git commit -qm "$2"; git checkout -q v2; git merge -q --no-ff "$1" -m "merge $1"; git push -q origin v2 "$1"; echo pushed "$1"
