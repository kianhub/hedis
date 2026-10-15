import json,sys
sys.argv=['x','m38f','real_dev','--subsha']
src=open('/workspace/hedis-wt-v2/eval/oneclass.py').read().split('indb =')[0]
exec(src)
split=json.load(open('data/split.v3dev.json'))
out=[]
for part in ('real_dev','dev'):
  for a in [x for x in split[part] if x.startswith('synth') or part.startswith('real')]:
    rn=("rn074" if split[part][a]["rn"].startswith("0.74") else "rn076") if part.startswith("real") else ("rn074" if "rn074" in a else "rn076")
    T={n:vs for n,vs in json.load(open(f"data/groundtruth/{a}.json"))["bundled"].items() if n in UNI}
    for x in json.load(open(f"data/predictions/m38f_{a}.json"))["predictions"]:
      n,v=x["package"],x["version"]
      if n not in UNI: continue
      if n in T and any(v==w or (sha.get((n,v,rn)) and sha.get((n,v,rn))==sha.get((n,w,rn))) for w in T[n]): continue
      out.append(dict(part=part,app=a,pkg=n,ver=v,bucket='a_absent' if n not in T else 'b_other_version',truth=T.get(n),score=x.get('score'),matched=x.get('matched_fos'),total=x.get('total_fos')))
json.dump(out,open('/tmp/fm.json','w'),indent=1)
import collections
print(collections.Counter((o['part'],o['bucket']) for o in out))
for o in out: print(o)
