# oneclass.py RN mapping fix (2026-10-16, CT)

Bug: without --apps=, rn = rn074 if the split RN starts with 0.74, else rn076. RN 0.85 apps were checked against rn076 bundle SHAs.
Fix: rn_key()/app_rn() map any RN (split RN, synth name, or triples header); test: eval/test_oneclass.py.
Rescore: eval/rescore_m41.sh with docs/eval/data/split.m41dev.json; buggy vs fixed tables in rescore_split_{buggy,fixed}.tsv.

Changed cells (buggy -> fixed): m39f 0.85 dev3 0.852 -> 0.885, dev10 0.588 -> 0.603; abl_s10 dev3 0.852 -> 0.885;
m40f_raw dev3 0.852 -> 0.885, dev10 0.576 -> 0.590; m40f dev3 0.852 -> 0.918, v8dev 0.436 -> 0.564, dev10 0.619 -> 0.719.
real_dev/synthetic unchanged. Every number reported earlier came from --apps= (RN from triples header) and is reproduced exactly by the fixed split path.
