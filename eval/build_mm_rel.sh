#!/bin/bash
# Mattermost v2.29.1: release Hermes bundle (dev=false, minify=true) + source map
A=/workspace/hedis-eval/apps; cd $A/mattermost
[ -d node_modules/react-native ] || npm ci --ignore-scripts --no-audit --no-fund || exit 1
npx --yes patch-package; bash ./scripts/postinstall.sh
npx react-native bundle --platform ios --dev false --minify true --entry-file index.ts --bundle-output $A/out/mattermost-rel.js --sourcemap-output $A/out/mattermost-rel.js.map || exit 2
node_modules/react-native/sdks/hermesc/linux64-bin/hermesc -O -emit-binary -out $A/out/mattermost-rel.hbc $A/out/mattermost-rel.js -w || exit 3
echo MM_OK
