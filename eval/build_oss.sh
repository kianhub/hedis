#!/bin/bash
# build_oss.sh <name> <github repo> <tag> [app subdir] [entry]
# Clone at tag, install (lockfile's package manager, scripts off), release Hermes bundle (dev=false, minify=true)
# with source map, bundled truth, then delete node_modules to save disk.
N=$1; R=$2; T=$3; SUB=${4:-.}; ENTRY=$5
A=/workspace/hedis-eval/apps; E=/workspace/hedis-eval; mkdir -p $A/oss; cd $A/oss
[ -d $N ] || git clone -q --depth 1 -b "$T" https://github.com/$R.git $N || exit 1
cd $N
export COREPACK_HOME=$HOME/.cache/corepack COREPACK_ENABLE_DOWNLOAD_PROMPT=0 CI=1
# ssh git deps -> https (process-scoped git config, no global change)
export GIT_CONFIG_COUNT=2 GIT_CONFIG_KEY_0=url.https://github.com/.insteadOf GIT_CONFIG_VALUE_0=ssh://git@github.com/ GIT_CONFIG_KEY_1=url.https://github.com/.insteadOf GIT_CONFIG_VALUE_1=git@github.com:
if [ ! -d node_modules ]; then
  if [ -f pnpm-lock.yaml ]; then corepack pnpm install --frozen-lockfile --ignore-scripts || exit 2
  elif [ -f yarn.lock ]; then
    if grep -q '"packageManager": "yarn@[2-9]' package.json || [ -f .yarnrc.yml ]; then YARN_ENABLE_SCRIPTS=0 corepack yarn install --mode=skip-build || exit 2
    else npx --yes yarn@1.22.22 install --frozen-lockfile --ignore-scripts --ignore-engines --network-timeout 600000 || exit 2; fi
  else npm ci --ignore-scripts --no-audit --no-fund --legacy-peer-deps --engine-strict=false || exit 2; fi
  [ -d patches ] && npx --yes patch-package
fi
cd $SUB
[ -f .env ] || { [ -f .env.example ] && cp .env.example .env; }
[ -n "$PRE" ] && { bash -c "$PRE" || exit 6; }   # per-app codegen step (e.g. lingui compile)
[ -n "$ENTRY" ] || for f in index.js index.ts index.tsx; do [ -f $f ] && ENTRY=$f && break; done
ENTRY=$(realpath $ENTRY)  # absolute: expo export:embed resolves relative entries from the monorepo root
if [ -z "$RNBUNDLE" ] && grep -q '"expo"' package.json && { [ -d node_modules/@expo/cli ] || [ -d ../node_modules/@expo/cli ] || [ -d ../../node_modules/@expo/cli ] || [ -d ../../../node_modules/@expo/cli ]; }; then
  npx expo export:embed --platform ios --dev false --minify true --entry-file $ENTRY --bundle-output $A/out/$N.js --sourcemap-output $A/out/$N.js.map --reset-cache || exit 3
else
  npx react-native bundle --platform ios --dev false --minify true --entry-file $ENTRY --bundle-output $A/out/$N.js --sourcemap-output $A/out/$N.js.map --reset-cache || exit 3
fi
HC=$(ls node_modules/react-native/sdks/hermesc/linux64-bin/hermesc ../node_modules/react-native/sdks/hermesc/linux64-bin/hermesc ../../../node_modules/react-native/sdks/hermesc/linux64-bin/hermesc ../../node_modules/react-native/sdks/hermesc/linux64-bin/hermesc 2>/dev/null | head -1)
$HC -O -emit-binary -out $A/out/$N.hbc $A/out/$N.js -w || exit 4
grep '"version"' $(dirname $(dirname $(dirname $(dirname $HC))))/package.json | head -1 > $A/out/$N.rnversion
[ -f $E/data/groundtruth/$N.json ] || echo '{}' > $E/data/groundtruth/$N.json
python3 /workspace/hedis/eval/bundled_truth.py $N $A/out/$N.js.map $PWD || exit 5
echo BUILD_OK $N
