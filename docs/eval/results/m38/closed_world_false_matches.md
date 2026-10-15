# Closed-world exact false matches on dev, m38f (2026-10-15, CT)

Run settings: effort high, model 'Grok Bot, default model'. Source: data/predictions/m38f_*, universe = 30 curated names,
--subsha release identity (same as oneclass.py). List: fm_attr/false_matches_m38f.json (eval/m38_fm_list.py).

| partition | false matches | (a) pkg absent from GT | (b) pkg in GT at another version |
|---|---|---|---|
| real_dev | 20 | 5 | 15 |
| synthetic (dev) | 4 | 0 | 4 |

(a) real_dev: rainbow @nozbe/watermelondb 0.27.1; bluesky @gorhom/bottom-sheet 4.6.4; valora react-native-keychain 9.2.3,
react-native-webview 13.15.0; leather react-native-url-polyfill 2.0.0.
(b) is mostly adjacent/older versions: react-native-screens 3.37.0 (x4 real, x2 synth), react-native-svg 13.14.1 (GT 15.11.x, x2),
@react-navigation/native 6.1.18 (GT 7.x, x2), plus semver, zod, fast-image, webview, immer patch neighbours.

## Source-module attribution
- **real_dev: no source maps and no .hbc on the box** (the .hbc files were lost in the restore; only the dumped triples
  remain in runs/apps_rel, which carry no source positions). Anything said about real_dev is **suggestive only**: by
  analogy with synth, the (b) cases look like version resolution within the right package, not cross-package leakage.
- **synthetic: rebuilt cheaply** (npm ci of apps/synth/<app>, Metro release bundle with source map, hermesc -O of the same RN;
  rebuilt triples agree with the originals to within 0.3-2% of rows). Each matched key was mapped to HBC function ->
  `hermesc -dump-source-location` -> Metro map -> node_modules package (eval/m38_fm_attr.py; Go hash dumper fm_attr/allh_main.go.txt).

| app | false match | all hit keys: share outside the 30 | discriminative keys (df<=2): share self / outside 30 |
|---|---|---|---|
| synth-01-rn076 | react-native-screens 3.37.0 (GT 3.35.0) | 0.915 (mostly react-native core) | 0.962 / 0.000 (rest react-freeze) |
| synth-10-rn074 | react-native-screens 3.37.0 (GT 3.33.0) | 0.950 | 0.960 / 0.000 (rest react-freeze) |
| synth-11-rn076 | react-native-fast-image 9.0.2 (GT 9.0.1) | 0.959 | 1.000 / 0.000 |
| synth-19-rn076 | immer 11.1.3 (GT 11.1.2) | 0.512 (@babel/runtime, sentry...) | 1.000 / 0.000 |

Reading: the bulk of raw key hits are shared RN/runtime code that every package bundle contains, but the keys that
discriminate the package all come from the right package's own modules. Synthetic false matches are wrong-version picks
inside the correct package, not functions borrowed from packages outside the 30. (react-freeze is outside the 30 and
is a react-native-screens dependency.)
