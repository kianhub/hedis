#!/bin/bash
# Rainbow v1.9.70 release Hermes bundle + source map. GraphQL codegen needs private API keys, so
# src/graphql/__generated__/* are stubbed (getSdk proxy only); this changes app code only, not node_modules.
A=/workspace/hedis-eval/apps; cd $A/rainbow
export COREPACK_HOME=$HOME/.cache/corepack COREPACK_ENABLE_DOWNLOAD_PROMPT=0
[ -f .env ] || cp .env.example .env
[ -d node_modules/react-native ] || YARN_ENABLE_SCRIPTS=0 corepack yarn install --mode=skip-build || exit 1
[ -d patches ] && npx --yes patch-package; [ -f src/config/debug.ts ] || cp src/config/defaultDebug.ts src/config/debug.ts
npx --yes rn-nodeify --hack "crypto,buffer,react-native-randombytes,vm,stream,http,https,os,url,net,fs,process"
mkdir -p src/graphql/__generated__
for n in arc arcDev ens metadata metadataPOST; do
  echo "export const getSdk = (requester: any) => new Proxy({}, { get: (_t, op) => (vars?: any, opts?: any) => requester(String(op), vars, opts) });" > src/graphql/__generated__/$n.ts
done
npx react-native bundle --platform ios --dev false --minify true --entry-file index.js --bundle-output $A/out/rainbow-rel.js --sourcemap-output $A/out/rainbow-rel.js.map || exit 2
node_modules/react-native/sdks/hermesc/linux64-bin/hermesc -O -emit-binary -out $A/out/rainbow-rel.hbc $A/out/rainbow-rel.js -w || exit 3
echo RB_OK
