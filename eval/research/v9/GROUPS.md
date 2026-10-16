# v9 GROUPS (frozen before any matcher run)

split: `docs/eval/data/split.v9.json` / `eval/research/v9/split.v9.json` sha256 `a32b9b14f1c05dba7515355554904b407ee54fddefecd8ecabfa9e59f173125c`.

**No matcher run. No scoring.** Wait for Optimizer's M40 cutoff.

| app | repo@tag | RN (detected) | HBC | group | dump binary | notes |
|---|---|---|---|---|---|---|
| fast-opencv-1.0.1 | lukaszkurantdev/react-native-fast-opencv@v1.0.1 | 0.85.0 | 98 | rn085 | hedis-m37 | hermesc PRE (env-only) |
| quick-base64-3.0.1 | craftzdog/react-native-quick-base64@v3.0.1 | 0.85.0 | 98 | rn085 | hedis-m37 | hermesc PRE (env-only) |
| secure-webview-0.1.0 | adnxy/react-native-secure-webview@v0.1.0 | 0.85.0 | 98 | rn085 | hedis-m37 | hermesc PRE (env-only) |
| system-navigat-3.0.1 | kadiraydinli/react-native-system-navigation-bar@v3.0.1 | 0.85.0 | 98 | rn085 | hedis-m37 | hermesc PRE (env-only) |
| trading-charts-1.1.0 | kirill3333/react-native-trading-charts@v1.1.0 | 0.85.0 | 98 | rn085 | hedis-m37 | hermesc PRE (env-only) |
| aemobileapp-1.0.3 | koiNoCirculation/AEMobileApp@1.0.3 | 0.74.3 | 96 | control | hedis-norm-hits | hermesc PRE (env-only) |
| engrave-SDDFinalRelease | shashwot2/Engrave@SDDFinalRelease | 0.76.5 | 96 | control | hedis-norm-hits | entry shim; hermesc PRE (env-only) |
| kaku-0.3.1 | joevtap/kaku@v0.3.1 | 0.76.9 | 96 | control | hedis-norm-hits | entry shim; hermesc PRE (env-only) |
| waifu-walls-0.1.1 | ronitkrshah/waifu-walls@v0.1.1 | 0.76.7 | 96 | control | hedis-norm-hits | hermesc PRE (env-only) |

Sizes: rn085 **5** / ≥5 ✓, control **4** / ≥3 ✓. Total 9.

Failures (queue order, not retried): fiesta-0.7.0 rc3 (@expo/metro-config missing), piano-analytic-0.7.5 rc3 (@react-native/metro-config missing), dynamically-se-3.3.2 rc2 (yarn frozen-lockfile out of date), swc-0.1.5 rc3 (@react-native-swc/core dist not built), daemon-4.7.11 rc2 (pnpm lockfile overrides mismatch), browser-1.1.2 rc3 (RN CLI not resolvable from root install), vocablypro-1.1099.0 rc3 (RN CLI not installed; npx pulled react-native@0.88.0), jigsaw-56.0.4 rc3 (unresolved @draftbit/core/lib build output). Remaining queue entries not attempted (targets met).

Note: system-navigat-3.0.1 and secure-webview-0.1.0 are both create-react-native-library example apps (near-identical bundles); kept per pre-registered order.

0.85/M40 DEV only (not held-out): blink-beta, breathly, zeus + v8 rn085 apps.
