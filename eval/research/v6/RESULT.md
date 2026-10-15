# v6 RESULT (scored once, 2026-10-15 ~11:10–11:20 CT)

## Pre-score commit
`research/v6-freeze` @ `91a50696c844cf197ad92864d2b901096be88554` (groups + zeus dumper exception after lxmusic cross-check). Author Kian Pasani <kian.pasani@gmail.com>.

## Verification
- `data/split.v6.json` sha256 `16f22fea07227bd50b1a712ccf0842904fe1aaee62000e40b776e8e60dea2bb8`
- `bin/hedis-m37` `647bfeadac67671d94c9fe2aa1a3bdcea57f1c0263a8d963e426af9ec61c29d7` @ 359ec3f
- `bin/hedis-norm-hits` `a29d0be7…` (11 apps); zeus triples `e9230d4e…` via hedis-m37 (HBC 98)
- Cross-check: lxmusic-1.0.2 triples byte-identical a29d0be7 vs 647bfead (`a3092540…6cb5`)
- thresholds/deps match prior freezes

## Groups (before scoring; pin 6ad0ea70…)
Covered RN minors with fingerprints: 0.72, 0.74, 0.75, 0.76, 0.78, 0.79, 0.80.
| group | apps |
|---|---|
| covered (5) | alephium, enuts, mangareader, pixelfed, pubky-ring |
| fallback (6) | minds, hexa, zingo, cozy, keeper, lxmusic |
| zeus (1) | zeus-13.1.3 |

## Matcher runs
- **M37:** `HEDIS_RN_FALLBACK=nearest HEDIS_MAX_OID=6ad0ea700000000000000000` + MANIFEST flags via `run_m37.sh` prefix `v6m37`. Zeus predicted 0 packages (empty results → empty predictions). All 12 have prediction files.
- **t35:** `bin/hedis-vset` + `thesis_vset.py` prefix `v6t35`, same pin + `HEDIS_RN_FALLBACK=nearest`. **Deviation:** hedis-vset lacks M37 RN fallback / HBC 98 — crashed `$in needs an array` on hexa, lxmusic, minds (0.71); HBC 98 EOF on zeus. Those 4 scored as empty predictions. cozy/keeper/zingo (0.77/0.81) still produced outputs (vset auto RN path differs).

## Overall (12 apps, primary)
| method | exact P/R/F1 (oneclass --subsha) | tp/fp/fn | m35 strict F1 | set-hit (version-set recall) | mean set | adv P/R/F1 (set) | adv F1 strict | app acc set / strict |
|---|---|---|---|---|---|---|---|---|
| **M37** | **0.486/0.458/0.472** | **71/75/84** | 0.458 (69/77/86) | 0.480 | 5.53 | 0.840/0.929/0.883 | 0.902 | 11/12 / 11/12 |
| t35 | 0.206/0.303/0.245 | 47/181/108 | 0.219 | 0.461 | 16.07 | 0.101/0.400/0.161 | 0.175 | 5/12 / 5/12 |

Vulnerable apps in truth: 5. Primary exact-version = oneclass --subsha.

## Per group (M37 primary / t35)
| group | M37 exact F1 (tp/fp/fn) | M37 set-hit | M37 adv F1 (set) | M37 app acc set/strict | t35 exact F1 | t35 adv F1 |
|---|---|---|---|---|---|---|
| covered | 0.500 (32/33/31) | 0.524 | 0.667 | 4/5 / 5/5 | 0.233 | 0.019 |
| fallback | 0.484 (39/42/41) | 0.519 | 0.890 | 6/6 / 5/6 | 0.279 | 0.302 |
| zeus | 0.000 (0/0/12) | 0.0 | 0 | 1/1 / 1/1 | 0.000 | 0 |

## Version sets (M37, universe predictions)
full sets=146 median=3 max=42 mean=5.53 span_vuln_and_fixed=6/146=0.041 sets_with_any_vuln=11

## Error breakdown (M37 oneclass --diag, 12 apps)
`{'FN version not in DB': 62, 'FN extra nested version': 3, 'FP package not bundled': 9, 'FN package missed': 12, 'FN wrong version': 7}`
FP total 75 = 9 not bundled + wrong-version predictions counted as FP when the true version is also FN.

## Per app (M37 exact, oneclass --subsha logic; tp/fp/fn)
| app | group | F1 | tp/fp/fn |
|---|---|---|---|
| pubky-ring-0.0.16 | covered | .897 | 13/2/1 |
| zingo-2.0.24 | fallback | .857 | 9/2/1 |
| keeper-2.5.9 | fallback | .811 | 15/4/3 |
| pixelfed-2.0.0 | covered | .720 | 9/4/3 |
| cozy-1.1.39 | fallback | .706 | 12/5/5 |
| enuts-0.3.0 | covered | .348 | 4/7/8 |
| mangareader-0.7.7 | covered | .258 | 4/12/11 |
| alephium-1.0.2 | covered | .200 | 2/8/8 |
| hexa-2.4.2 | fallback | .133 | 2/13/13 |
| minds-4.37.0 | fallback | .062 | 1/15/15 |
| lxmusic-1.0.2 | fallback | .000 | 0/3/1 |
| zeus-13.1.3 | zeus | .000 | 0/0/12 |

## Output hashes
    3a17eab9d023bc3956ef2e8d0cf1483f935e55aecd42b91aaddbe32d990fea57  research/v6/result/oneclass.full.txt
    cbb29bc423b96f1949803f1c36af21d74423da7b09e03f8935caa92870435a6e  research/v6/result/m35.full.txt
    3d85220e4a52db5e23e657196c84cffdda22b0bb3932c15008649b71d30c6c06  research/v6/result/t35.oneclass.txt
    781c5fc0bcd4cb776edea2c377b3795867944fd7084da81aa02ad36b2b4a9f11  research/v6/result/t35.m35.txt
    16f22fea07227bd50b1a712ccf0842904fe1aaee62000e40b776e8e60dea2bb8  data/split.v6.json
    316a3784af6c2e5d2ce0359e5770b442ed0332fce209f58bb619d5ce6b3c135c  research/v6/MANIFEST.md
    27eca51cbe3369314ede8038836ea72706a4ea428cd07b5120f449fd4b7b84bc  research/v6/GROUPS.md

## Fair subset: 8 apps where t35 ran

Scoring on **existing** prediction files only (no new matcher run). Excludes the 4 apps where t35 crashed with empty predictions: hexa-2.4.2, lxmusic-1.0.2, minds-4.37.0, zeus-13.1.3.

**8 apps:** alephium-1.0.2, enuts-0.3.0, mangareader-0.7.7, pixelfed-2.0.0, pubky-ring-0.0.16, zingo-2.0.24, cozy-1.1.39, keeper-2.5.9.

Primary: `oneclass.py --subsha`. Secondary: `m35_eval.py` (vuln-level / set metrics). Split: `research/v6/split.v6.t35fair8.json`.

### Overall (8 apps)

| method | exact P/R/F1 (oneclass --subsha) | tp/fp/fn | set-hit | adv F1 (set) | app acc set / strict |
|---|---|---|---|---|---|
| **M37** | **0.607/0.618/0.613** | **68/44/42** | 0.648 | 0.919 | 7/8 / 7/8 |
| **t35** | **0.206/0.427/0.278** | **47/181/63** | 0.648 | 0.182 | 3/8 / 3/8 |

Vulnerable apps in truth among the 8: 3.

### Per group among the 8 (zeus excluded)

| group | apps | M37 exact F1 (tp/fp/fn) | M37 set-hit | M37 adv F1 | M37 app acc set/strict | t35 exact F1 (tp/fp/fn) | t35 set-hit | t35 adv F1 | t35 app acc set/strict |
|---|---|---|---|---|---|---|---|---|---|
| covered (5) | alephium, enuts, mangareader, pixelfed, pubky-ring | 0.500 (32/33/31) | 0.524 | 0.667 | 4/5 / 5/5 | 0.233 (24/119/39) | 0.571 | 0.019 | 1/5 / 1/5 |
| fallback (3) | zingo, cozy, keeper | 0.766 (36/11/11) | 0.822 | 0.941 | 3/3 / 2/3 | 0.348 (23/62/24) | 0.756 | 0.393 | 2/3 / 2/3 |

On this fair subset M37 exact F1 is 0.613 vs t35 0.278 (same set-hit 0.648; M37 adv F1 0.919 vs t35 0.182).

### Fair-subset output hashes
    6d81383b3a83d3752b82571c13c2dda6da73715df2622aeb58a3b4b8e284422d  research/v6/split.v6.t35fair8.json
    3fb6cbfa9e63093cdeebbf270c354252d65e9927a8008ae51562c6a27eed64a9  research/v6/result/v6m37.t35fair8.oneclass.txt
    0d4604f73c6c5db590a36d17fc725396e743f7649258470095b249767567ef1d  research/v6/result/v6m37.t35fair8.m35.txt
    f417eff3e02924a47c8d934ab20d6e19079072eee3e87e11f7a29736b92887ed  research/v6/result/v6t35.t35fair8.oneclass.txt
    5ecdbd04baa0872b36fd77ac2543d0e1e81f9b13a0afaf786abda334f24abee1  research/v6/result/v6t35.t35fair8.m35.txt
    9201c80248e6fce23acbf60bcb902ca7432e847992263069393109b2ed663ecd  research/v6/RESULT.md
