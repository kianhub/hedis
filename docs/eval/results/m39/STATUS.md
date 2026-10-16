# M39 (RN 0.85 / HBC98) — in progress

Run settings: effort high, model 'Grok Bot, default model'.
- Targets: docs/eval/data/targets_m39.json = 30 curated pkgs x 2210 versions in hedis_rel at the M38 cutoff.
- Env envs/rn085 (RN 0.85.3, react 19.2.3; hermesc from hermes-compiler 250829098.0.10 symlinked to sdks/hermesc/linux64-bin).
- Binary bin/hedis-m39 = this branch (go build -buildvcs=false). Smoke: zod 3.23.8 ok; baseline 0.85 / HBC 98 created.
- Build: `nohup bash eval/fp_m39.sh > logs/fp_m39.out 2>&1 &` (in /workspace/hedis-eval checkout of this script); resumable,
  fp_done markers skip finished pkg@ver; stats logs/fp-{rel,sub}39-rn085-stats.jsonl; done marker logs/fp-m39-rn085.done.
- M38 baseline on 0.85 dev apps (m38f85): blink-beta, breathly, zeus all refused (unsupported_bytecode, max cov 0.070/0.035/0.085);
  exact F1 0.000 (tp 0 fp 0 fn 30), unsupported rate 3/3.
- better-auth fails Metro at every RN (known; 1/225 ok at rn080).
