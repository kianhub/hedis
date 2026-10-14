#!/bin/bash
# M34: rerun root + subpath release fingerprints (rn074, rn076) for ERESOLVE-failed jobs and missing real_dev versions
# bin/hedis-sub4 = hedis-sub3 + ERESOLVE retry with --force plus env peer-pins.txt; pinned peers are bundled as empty modules (resolver) so their code stays out of the keys. Earlier attempts: 1 legacy-peer-deps pruned metro, 2 --force pulled gesture-handler 2.33 (TransformError), 3 pins bundled into bottom-sheet keys (synth dev regressed to 0.949)
cd /workspace/hedis-eval; export MONGO_CONNECTION_STRING=mongodb://127.0.0.1:27017 HEDIS_PACKAGES_DB=hedis_rel
bin/hedis-sub4 fingerprint --release --targets db/targets_m34.json --rns rn074,rn076 --envs-root envs --workers 3 --cache cache-rel --stats logs/fp-rel34-stats.jsonl && echo ROOT_DONE
bin/hedis-sub4 fingerprint --release --subpaths --targets db/targets_m34.json --rns rn074,rn076 --envs-root envs --workers 3 --cache cache-rel --stats logs/fp-sub34-stats.jsonl && echo SUB_DONE
