#!/usr/bin/env python3
"""cache_keys.py <rn> : dump nsnc keys per (pkg, ver) for one RN from hedis_rel (fp_done rows only, no DB pin) to runs/m40/keys_<rn>.pkl
{(pkg,ver): (rootkeys frozenset, subkeys frozenset)}; keys interned as ints via a shared table."""
import sys, pickle, pymongo, collections
rn=sys.argv[1]; db=pymongo.MongoClient().hedis_rel
done={d['_id'].rsplit('|',1)[0] for d in db.fp_done.find({'_id':{'$regex':r'\|'+rn.replace('.',r'\.')+'$'}},{'_id':1})}
pk={str(d['_id']):(d['package_name'],d['package_version']) for d in db.packages.find({},{'package_name':1,'package_version':1})}
ids={}; R=collections.defaultdict(set); S=collections.defaultdict(set)
for h in db.hashes.find({'react_native_version':rn},{'_id':0,'package_id':1,'ns':1,'nc':1,'sub':1}).batch_size(100000):
    p=str(h['package_id'])
    if p not in done or p not in pk: continue
    k=ids.setdefault((h.get('ns'),h.get('nc')),len(ids))
    (S if h.get('sub') else R)[pk[p]].add(k)
out={pv:(frozenset(R[pv]),frozenset(S[pv])) for pv in set(R)|set(S)}
pickle.dump({'ids':ids,'pv':out},open(f'/workspace/hedis-eval/runs/m40/keys_{rn}.pkl','wb'))
print(rn,len(out),len(ids))
