import json,re,sys,bisect,collections,subprocess,os
B64={c:i for i,c in enumerate('ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/')}
def vlq(s):
    out=[];v=sh=0
    for ch in s:
        d=B64[ch];v+=(d&31)<<sh
        if d&32: sh+=5
        else: out.append(-(v>>1) if v&1 else v>>1);v=sh=0
    return out
def load_map(p):
    m=json.load(open(p));lines=[];src=0
    for ln in m['mappings'].split(';'):
        col=0;segs=[]
        for seg in ln.split(','):
            if not seg: continue
            f=vlq(seg);col+=f[0]
            if len(f)>1: src+=f[1];segs.append((col,src))
            # line/col of source ignored but must accumulate
        lines.append(segs)
    # need proper accumulation of fields 2,3 too -> recompute
    return m,lines
def load_map2(p):
    m=json.load(open(p));lines=[];s=sl=sc=n=0
    for ln in m['mappings'].split(';'):
        col=0;segs=[]
        for seg in ln.split(','):
            if not seg: continue
            f=vlq(seg);col+=f[0]
            if len(f)>=4:
                s+=f[1];sl+=f[2];sc+=f[3]
                if len(f)>=5:n+=f[4]
                segs.append((col,s))
        lines.append(segs)
    return m['sources'],lines
def lookup(srcs,lines,L,C):
    if L-1>=len(lines): return None
    segs=lines[L-1]; i=bisect.bisect_right([c for c,_ in segs],C-1)-1
    return srcs[segs[max(i,0)][1]] if segs else None
def pkg_of(path):
    if path is None: return '?'
    if '/node_modules/' not in path: return '<app>'
    t=path.rsplit('/node_modules/',1)[1].split('/')
    return '/'.join(t[:2]) if t[0].startswith('@') else t[0]
W=sys.argv[1]; app=os.path.basename(W)
locs=[]
for l in open(W+'/app.dis'):
    if re.match(r'^[A-Za-z]*Function<',l): locs.append(None)
    elif l.startswith('; app.js:') and locs and locs[-1] is None:
        _,L,C=l.strip().rsplit(':',2)[0],*l.strip().split(':')[-2:]; locs[-1]=(int(L),int(C))
srcs,lines=load_map2(W+'/app.map')
fn=[]
subprocess.run(['/tmp/attrgo/allh',W+'/app.hbc',W+'/app.fn.tsv'],capture_output=True)
for l in open(W+'/app.fn.tsv'): fn.append([x for x in l.rstrip('\n').split('\t')[1:6] if x])
assert len(fn)==len(locs),(len(fn),len(locs))
idx=collections.defaultdict(set)
for i,f in enumerate(fn):
    p=pkg_of(lookup(srcs,lines,*locs[i])) if locs[i] else '?'
    for a in f:
        for b in f: idx[(a,b)].add(p)
U=set(json.load(open('/workspace/hedis-eval/data/universe_targets.json')))
hits={(p['package'],p['version']):p for p in json.load(open(f'/workspace/hedis-eval/runs/m38f/{app}.hits.json'))}
res=[]
for o in json.load(open('/tmp/fm.json')):
    if o['app']!=app: continue
    H=hits[(o['pkg'],o['ver'])]['hits']
    for lab,h in (('all',H),('df<=2',[x for x in H if x['df']<=2])):
     c=collections.Counter(); un=0
     for x in h:
        a,b=x['k'].split('|'); ps=idx.get((a,b))
        if not ps: un+=1; continue
        for p in ps: c[p]+=1/len(ps)
     tot=sum(c.values())
     res.append(dict(sel=lab,app=app,pkg=o['pkg'],ver=o['ver'],bucket=o['bucket'],hits=len(h),unresolved=un,
      share_self=round(c[o['pkg']]/tot,3) if tot else None,
      share_outside30=round(sum(v for p,v in c.items() if p not in U)/tot,3) if tot else None,
      top=[(p,round(v,1)) for p,v in c.most_common(4)]))
[print(json.dumps(r)) for r in res]
