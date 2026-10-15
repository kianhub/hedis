import json,hashlib,os,subprocess,glob,struct
F='research/v7/frozen/';T='research/v7/triples/'
def sha(p): return hashlib.sha256(open(p,'rb').read()).hexdigest() if os.path.exists(p) else None
q={}
for f in sorted(glob.glob('research/v7/queue*.txt')):
    for l in open(f):
        x=l.rstrip('\n').split('|')
        if len(x)>3: q[x[0]]=x
G={'jitsi-9.1.0':'fill','xaman-2.8.0':'fill','daimo-audit-1':'fill','gsy-4.0.2':'fill','safe-1.54.0':'control','jfaudio-2.4.4':'control','vega-3.1.1':'control','blink-beta-3.0.31':'unsupported_hbc98','breathly-ios-2.2-18':'unsupported_hbc98','bbplayer-2.5.2':'unsupported_hbc98','yoroi-4.12.0':'fallback','readany-1.3.4':'fallback','ogam-0.0.49':'fallback'}
apps={}
for a,g in G.items():
    hv=struct.unpack('<QI',open(F+a+'.hbc','rb').read(12))[1]
    rv=open(F+a+'.rnversion').read().strip()
    rn=rv.split('"')[3] if rv else None
    r=q[a]; ref=r[2]
    c=subprocess.run(['timeout','30','git','ls-remote','https://github.com/'+r[1],'refs/tags/'+ref,'refs/tags/'+ref+'^{}'],capture_output=True,text=True).stdout.split()
    apps[a]=dict(repo=r[1],tag=ref,commit=c[-2] if c else None,subdir=r[3] or '.',entry=r[4] or None,pre=r[5] or None,env=(r[7] if len(r)>7 else '') or None,rn=rn,
      rn_minor='.'.join(rn.split('.')[:2]) if rn else None,package_json_rn_label='0.85.3' if a=='bbplayer-2.5.2' else None,hbc_version=hv,group=g,
      dump_binary=None if not rn else ('hedis-m37' if hv>=98 else 'hedis-norm-hits'),dumper_failure=(rn is None),
      hbc_sha256=sha(F+a+'.hbc'),js_sha256=sha(F+a+'.js'),map_sha256=sha(F+a+'.js.map'),truth_sha256=sha(F+a+'.gt.json'),
      rnversion_sha256=sha(F+a+'.rnversion'),triples_sha256=sha(T+a+'.triples'))
B={'hedis-m37':sha('bin/hedis-m37'),'hedis-norm-hits':sha('bin/hedis-norm-hits'),'hedis-m38':sha('bin/hedis-m38'),'hedis-vset':sha('bin/hedis-vset')}
cov=['0.72','0.74','0.75','0.76','0.78','0.79','0.80']
d=dict(frozen=subprocess.run(['date','-Iseconds'],capture_output=True,text=True).stdout.strip(),
 rule="v7 real_test: stratified held-out (fill/control/unsupported_hbc98/fallback), disjoint from burned sets (v3 real/real_dev, v4, v5, v6 real_test) and Notesnook/Uniswap; built with eval/build_oss.sh; truth=bundled_truth.py. Never matched or scored before this freeze.",
 primary_scorer="oneclass.py --subsha",secondary_scorer="m35_eval.py (vulnerability-level only)",binaries=B,
 covered_rn_minors={'6ad0ea700000000000000000':cov,'6ad13ed00000000000000000':cov},
 rn085_rule="unsupported_bytecode/unsupported flag = correct abstention; own table; excluded from exact F1",
 bbplayer_note="dumper failure (empty detected rnversion); no valid triples; package.json 0.85.3 = group label only, never passed to dump or matcher; quarantined .INVALID triples excluded",
 real_test=apps)
json.dump(d,open('data/split.v7.json','w'),indent=1)
json.dump({a:{k:v for k,v in x.items() if not k.endswith('sha256')} for a,x in apps.items()},open('research/v7/apps.json','w'),indent=1)
for a,x in apps.items(): print(a,x['group'],x['rn'],x['hbc_version'],x['dump_binary'],(x['commit'] or 'NOCOMMIT')[:8],(x['triples_sha256'] or '-')[:8])
