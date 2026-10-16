#!/usr/bin/env python3
"""fix_empty.py <prefix>...: prediction files left empty/invalid by to_predictions.py when the matcher wrote results=null
(no package predicted) become {"predictions": []}."""
import sys, glob, json
for P in sys.argv[1:]:
    for f in glob.glob(f"/workspace/hedis-eval/data/predictions/{P}_*.json"):
        try: json.load(open(f))
        except Exception:
            a=f.rsplit('/',1)[1][len(P)+1:-5]; json.dump({"app":a,"method":P,"status":"no_predictions","predictions":[]},open(f,'w')); print("fixed",f)
