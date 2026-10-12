#!/usr/bin/env python3
"""Disassemble an .hbc (hedis disassemble) and extract version-evidence features beyond NS/NC:
  str:<s>   every referenced string incl. short ones (NC drops len<=2) and string-table entries used by code
  num:<op>:<v>  numeric constant operands (LoadConstUInt8/Int/Double/...)
  fcount    function count
  ins:<h>   per-function hash of the full instruction text with operands, minus addresses/string ids/registers? (kept raw operands except string_id)
usage: hbc_features.py <in.hbc> <out.json>"""
import sys, re, subprocess, json, hashlib, collections, os, tempfile
src, out = sys.argv[1:3]
fd, tmp = tempfile.mkstemp(suffix=".txt"); os.close(fd)
subprocess.run(["/workspace/hedis-eval/bin/hedis-norm", "disassemble", "-i", src, "-o", tmp], capture_output=True)
S = collections.Counter(); N = collections.Counter(); I = collections.Counter(); fc = 0; cur = []
INS = re.compile(r"^\s+[0-9a-f]{8}: <(\w+)>: (<[^>]*>)?(.*)$")
def flush():
    if cur: I[hashlib.sha1("\n".join(cur).encode()).hexdigest()[:16]] += 1
for line in open(tmp, errors="replace"):
    m = INS.match(line)
    if not m:
        if line.lstrip().startswith("=> [Function #"): flush(); cur = []; fc += 1
        continue
    op, opr, rest = m.group(1), m.group(2) or "", m.group(3)
    sm = re.search(r'# String "(.*)" \(', rest)
    if sm: S[sm.group(1)] += 1
    if op.startswith("LoadConst") and op not in ("LoadConstString", "LoadConstStringLongIndex", "LoadConstUndefined", "LoadConstNull", "LoadConstTrue", "LoadConstFalse", "LoadConstZero", "LoadConstEmpty"):
        v = opr.split(",")[-1].strip(" <>").split(":")[-1].strip(); N[f"{op}:{v}"] += 1
    cur.append(op + re.sub(r"string_id: \d+", "s", opr) + (("#" + sm.group(1)) if sm else ""))
flush(); os.remove(tmp)
json.dump({"fcount": fc, "str": S, "num": N, "ins": I}, open(out, "w"))
