# v6 real-app test set — MANIFEST (pre-registration, never scored)

## Scoring config (pre-registered)
- PRIMARY scorer for v6: `oneclass.py --subsha` (with the M34 release-identity tables).
- `m35_eval.py` is SECONDARY, used for vulnerability-level metrics only.

## Build method
Same as v5: build_oss.sh (clone at tag, lockfile install, release Metro bundle + source map, app's own hermesc -O), bundled_truth.py truth from source maps, dump_app.sh triples. No matcher run, no scoring at build time.

## Pre-registered M37 scoring run (added 2026-10-15 10:26 CT, before any v6 scoring)
- Matcher: hedis binary built from v2 commit 359ec3f5b3df9723a6810ced2b502877e527a280 (M37 freeze), settings in data/m37_frozen.json.
- Env: HEDIS_RN_FALLBACK=nearest HEDIS_MAX_OID=6ad0ea700000000000000000
- Flags: packages -d --norm nsnc --thresholds data/thresholds_m4f_dlr.json --lowuniq 2 --lowdf 2 --dep 0.6 --deps data/deps.json --rootver --floor 0.2 --vset-margin 0.003 --vset-samekeys
- Primary scorer: oneclass.py --subsha (M34 release-identity tables).
- Secondary: m35_eval.py for vulnerability-level metrics only.
- Thesis baseline (added before scoring): run t35 with the same HEDIS_MAX_OID pin and HEDIS_RN_FALLBACK=nearest; report exact-version and vulnerability-level metrics alongside M37.
- Groups: covered / fallback / zeus, determined by a DB query at the pin before scoring (zeus is covered if HBC 98 dumps succeed).
- Run once; no retuning.

## v6 re-freeze (2026-10-15)
The original freeze (`data/split.v6.json` sha256 `99ee8a214ba9921d60af7d5b529730e44a37fd6099c11163969a316cd79b26d7`) was lost to a box restore on 2026-10-15. The same 12 apps were rebuilt unscored from the same tags/commits (never matched or scored). **Original `.hbc` hashes were not recoverable** (the MANIFEST app/hash table was wiped with the restore), so **bytecode identity versus the lost freeze is unverified**. Ground-truth JSON was compared against surviving `data/groundtruth/` copies (see below).

New freeze: `data/split.v6.json` sha256 `1b8e1d195000dc1f9276e1286a3995ed3ce6dddef13840ea18618a602b5ea080`.

### Binaries (recorded at freeze)
| binary | sha256 | source |
|---|---|---|
| `bin/hedis-m37` (matcher, for later scoring) | `647bfeadac67671d94c9fe2aa1a3bdcea57f1c0263a8d963e426af9ec61c29d7` | v2 @ `359ec3f5b3df9723a6810ced2b502877e527a280` |
| `bin/hedis-norm-hits` (triples dumper) | `a29d0be7e692ccbffaf0ce3a4cf2c1d2526ff237bae228037977cf918b73139f` | shared (Optimizer: built from `caf039a`) |

Note: `research/v6/bin/hedis-norm-hits` (647bfead from 359ec3f) was built earlier by mistake and must **not** be used for dumps. Zeus HBC 98 dump **fails** with the caf039a dumper (`failed to read exception handler: unexpected EOF`); HBC 98 support landed in 359ec3f (`e9f0e06`). Zeus stays in the split with `triples_sha256: null`.

### Ground-truth rebuild diffs (vs surviving originals)
| app | result |
|---|---|
| alephium-1.0.2 | IDENTICAL |
| minds-4.37.0 | IDENTICAL |
| hexa-2.4.2 | IDENTICAL |
| zeus-13.1.3 | IDENTICAL |
| zingo-2.0.24 | IDENTICAL |
| cozy-1.1.39 | IDENTICAL |
| enuts-0.3.0 | IDENTICAL |
| keeper-2.5.9 | IDENTICAL |
| lxmusic-1.0.2 | IDENTICAL |
| mangareader-0.7.7 | IDENTICAL |
| pixelfed-2.0.0 | **DIFFERENT** — original 62 pkgs / 873 modules; rebuild 236 pkgs / 2411 modules. Rebuild required a stub `src/components/common/Stories.tsx` (null `StoryCarousel`) because that module is imported but absent at tag v2.0.0; without the stub Metro fails. The surviving original GT appears to come from an incomplete prior bundle. Only-orig packages: buffer@5.7.1, react-is@18.3.1 (rebuild has extra versions of both plus ~174 other packages). |
| pubky-ring-0.0.16 | IDENTICAL |

### Apps (re-frozen; never matched or scored)
| app | repo | tag | commit | RN | HBC | GT pkgs | GT modules | triples |
|---|---|---|---|---|---|---|---|---|
| alephium-1.0.2 | alephium/mobile-wallet | v1.0.2 | `e71931fdcd` | 0.72.3 | 94 | 257 | 3839 | ok |
| minds-4.37.0 | Minds/mobile-native | v4.37.0 | `7e8a4e542c` | 0.71.7 | 90 | 376 | 3759 | ok |
| hexa-2.4.2 | bithyve/hexa | v2.4.2 | `1a58b3c492` | 0.71.13 | 90 | 285 | 2517 | ok |
| zeus-13.1.3 | ZeusLN/zeus | v13.1.3 | `73db3bf97e` | 0.85.3 | 98 | 288 | 3006 | FAIL (HBC98 vs caf039a dumper) |
| zingo-2.0.24 | zingolabs/zingo-mobile | zingo-2.0.24-320 | `8000dc88e6` | 0.81.6 | 96 | 114 | 2947 | ok |
| cozy-1.1.39 | cozy/cozy-flagship-app | 1.1.39 | `62f966fcf9` | 0.77.3 | 96 | 228 | 3098 | ok |
| enuts-0.3.0 | cashubtc/eNuts | v0.3.0 | `faea4d6425` | 0.72.10 | 94 | 139 | 1679 | ok |
| keeper-2.5.9 | bithyve/bitcoin-keeper | v2.5.9 | `ecf9a5b9c5` | 0.77.3 | 96 | 335 | 3745 | ok |
| lxmusic-1.0.2 | lyswhut/lx-music-mobile | v1.0.2 | `151007cb66` | 0.71.4 | 90 | 89 | 901 | ok |
| mangareader-0.7.7 | youniaogu/MangaReader | v0.7.7 | `fbdb0bbc1f` | 0.72.10 | 94 | 146 | 1935 | ok |
| pixelfed-2.0.0 | pixelfed/pixelfed-rn | v2.0.0 | `bbdda1af55` | 0.76.9 | 96 | 236 | 2411 | ok |
| pubky-ring-0.0.16 | pubky/pubky-ring | v0.0.16 | `06114f46c7` | 0.78.1 | 96 | 94 | 3155 | ok |

### Per-artifact sha256
| app | hbc | js | js.map | groundtruth | triples |
|---|---|---|---|---|---|
| alephium-1.0.2 | `e041a8438dd09a3df78d5294b68d1ec2180b96d2d055ed718cd5695e78276030` | `bf496edf40c7682948c848cacf74f097050a73483e4102f76a8812bf1b652634` | `18b756ad31860353f3694b32c3de76f565f90c98b1777b4490a9191817a0b117` | `97978b57e063b6e2f8da305859112370a953a0c6f6c4bc1bb58ca73f96b3a195` | `724ad9b6d9e8ca88c936efa59dd13874f8e042bf1da3f206047b098b62051f0a` |
| minds-4.37.0 | `2aef456f86901b398c5e11faed10eeafa4f7784c4a73418d7616dde68dda44fe` | `7e0925968187f916880716b92d7f9b1558ca22f43988128235e0473b4175186c` | `18675a6c41f3c3d76c9306493fefdd4b6028fbb716cc85734838637c7ab55c34` | `7e2f8ee3b7eecc34413890fbe6368deb44249e8d671eb9ad20f980c4e78f6bac` | `b7de95ed4d414eff51ed15d26025ee997ed72f4380c45382e7e97393d12ef74b` |
| hexa-2.4.2 | `e482600f78dab50f7f96f5bc0cd0241a9b6152a2a48323854cb1f412db566cbb` | `0e110524df630b298e0b0c2e9755b20437df74703712be3e922be731f4774164` | `0917111daca557209d209e4f33ae234c993d25ca659408f1ae23bb7d37848159` | `fc74affd103e3a952560a0bfe7a0f850d1d0ef2d360080bdf449a84e3bebeed2` | `54a036f0778ab7133df2350b70157498e49b596d31172cdb9a0a680a67bda916` |
| zeus-13.1.3 | `88b88145447954bf6699fa5b47f70788b7a8f36ae89c6b566ea701800f825944` | `dcd435c2b10643f81f89339e44f516307bd31f1e09f83f4bd78e5d476749f5cc` | `92b0f6cc8b0c6f1c244ab772330d75fa3857a6d38284e0bd4a5f1028cd7ab4d2` | `593a755f91915e0a54bd5398e7e29c591da5e1157b07fa06574ba0471d8f549f` | `null` |
| zingo-2.0.24 | `ab72bac51d83c3e3374982bee1f74eed9a4b10c0afc9e4d81771676f7530fbe4` | `223c8fbdce0be62203316bd5f6eda913738b7781ab76bdf661758b27143eb006` | `ec2ffa731a11d383389c55a186a0395bd6a1a1455c78b72764338e8ce89fdc0b` | `34fd5c5fea5b5e93b49ab81cba5c1273b623157dcd4116d0aa4c50760e5b8fcc` | `d92a1dde3acb7fd346fd6db1aa7602c666ba2568b933a5910f50d97a18805be7` |
| cozy-1.1.39 | `6b1519db179e517adcbf3a50dfb240bdfde58876105192809998f1084fdb93e4` | `9daa195746f37b1b59bdbbbd4ce9fa4d1405823360df14529aa37b075b2065fb` | `764c1d9f022addcbaa76ae9a406c6891f57d7f8bde0b1cdc58661cbd641ed27f` | `87cb844931d48ac57144caacfc82c802eec2b94830140e9856c80fd1eed49755` | `06a3657405a591757da841210d5f07e5fa64f46676582b3a4ab2ede6671af767` |
| enuts-0.3.0 | `93688fe42871c10f6fd7fb1e4cf4765cbf02bab02adcd7dd4d51e9ce32448bd5` | `a7bbe92e130d2f4489c55f42fa5a60346d1c658df6384427c89d5bd2ce7da0e8` | `27c2a68ac20910254829c45b74aa20ac42cb79af656cfc94283dcae8215462c1` | `079bb11efbed34619a7f2c021a84687ec536124ed87f9775c272456fc1b342a1` | `06b061eb67e99e31f091df83998160bf1bb6890546bf3b75b5e034ffe6e4fe5f` |
| keeper-2.5.9 | `061b24e2a3c05b69105aeb5ceb652d950d46f353e298e01710d3d33b95ebc1b1` | `43ae6ec766f53e30d3b970d8ef5d442cfe5c865e56a99397e03a8d84b58ab707` | `5e1bbdc11b15c056c06bce59bb01fb2ffb4069bf339c4cf45f305b513ea7a0fe` | `d254bd67221aab0be99455f647084f68cf0d280b39407639987d7c626358572a` | `3f746de57b1b29b77408e5aec3b585b3abbcfde536898509dd679e6214d14181` |
| lxmusic-1.0.2 | `76c7bb2fda6440d06d4e8d8f9cedb549b78548df7fa59796f83642dadad33562` | `6266c1a319ffa2f2004d24bf9404d9ae450160dbbabea484f37704463cf2fefb` | `fe0e1560d7b11f059499f68b42439e6a922f12dc8ce9fb7f8dd607f6db1578da` | `31f1ab8260b3e367b00f6eb393dc8b2e89ecc9353d15376afb5e474236bd83cf` | `a309254085f2fe1204e04654eb78ca96335a85ce3376ffd4234f8e47ba5d6cb5` |
| mangareader-0.7.7 | `c2713530e85dcd6e569fda4119c0b6063e76db339a695b356a11ac0196a500ab` | `745681ca6df5e187e372fa29a792719e3ab5506e7a7cc5aaf3303f1b39f95151` | `4bc8bb0a401b948e6f7b1e32a1230552686b2c8391df8462533d638e8529758c` | `85c6884d5e7dc667736d0f22a055341e91caf50dcfceb23501be8507b6d9ae37` | `0974da6541f5dcfb322629ca190f7a63b768c112cd3e8dcf6f16939a057db8d6` |
| pixelfed-2.0.0 | `3e3df025c3455996f080d8a713c64678929aeb10a185e4d339ee60ab12b00ada` | `4ecbcc2efd9c08d8a8cef2ca070c1203285188abfd6abab37d88c3849cc9f20b` | `26847477caf0597bbaaed456dd06146d70d154685586b8c2477aa2f1f425e5fd` | `792ea79bd02a4abce38e2085f977d3cf798c50242eb3d726d50b262f30f89449` | `f217d3ebcfba3fef7cba9b2eeaa7cd8f09c33a193e2e2830f9d9c3bd64fd341b` |
| pubky-ring-0.0.16 | `9220caada1722e9eff36e88a3814b48bb8790cd72ae49d9cd1a603a7083f002c` | `bbdf54e89c8561adb279e4c7d648ff1dcb60eee2e7c40bc639d9cf9fe677ca3b` | `13fd5aafc9b1bdbb263f85afcd1ac88677dac5d0548dceef5212ff72b358029d` | `de82e495e8824d1fe8e1fa2493e6040a11e2703c9f25deab13648ba5f673355a` | `a1c9f7b2ec137b2e9e10af1928661961ea9a971a69e08d55d7e27811c308c1d0` |

### Build workarounds (environment/build plumbing only; no app logic changed)
- minds, hexa: `RNBUNDLE=1` (`react-native bundle`; Expo SDK 48 CLI has no `export:embed`).
- hexa, keeper: repo setup (`rn-nodeify` crypto/stream/… `--hack`); hexa also `patch-package`.
- zeus: postinstall (`patches/index.mjs`, `rn-nodeify`) + symlink of app's `hermes-compiler` hermesc into the path `build_oss.sh` expects.
- eNuts: entry `src/App.ts` (package.json `main`).
- pixelfed: `git lfs pull` after installing git-lfs; **plus** stub `Stories.tsx` (see GT diff).
- Sources deleted after each build to save disk; commits from `git ls-remote` of the tag.

### Groups (for later scoring; not yet queried)
- covered = app RN minor has fingerprints at `HEDIS_MAX_OID=6ad0ea700000000000000000`
- zeus = always its own third group (even though RN 0.85 has no fingerprints)
- fallback = everything else
Do **not** put zeus in covered just because HBC 98 dumps succeed.
