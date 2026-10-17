#!/usr/bin/env python3
"""python3 eval/test_oneclass.py: RN mapping regression test for oneclass.py (0.85 apps must map to rn085, not rn076)."""
import sys, os, unittest, importlib.util, json, tempfile
HERE = os.path.dirname(os.path.abspath(__file__))
src = open(os.path.join(HERE, "oneclass.py")).read()
ns = {}
ns.update(E_="/workspace/hedis-eval"); exec(src[src.index("def rn_key"):src.index("APPS = next(")], ns)
class T(unittest.TestCase):
    def test_rn_key(self):
        for v, k in [("0.85", "rn085"), ("0.85.3", "rn085"), ("0.74.7", "rn074"), ("0.76", "rn076"), ("0.80", "rn080"), ("rn085", "rn085"), ("0.72", "rn072")]:
            self.assertEqual(ns["rn_key"](v), k)
    def test_app_rn(self):
        self.assertEqual(ns["app_rn"]("synth-03-rn076"), "rn076")
        self.assertEqual(ns["app_rn"]("blink-beta-3.0.31", "0.85"), "rn085")
        self.assertEqual(ns["app_rn"]("valora-1.110.0", "0.76.9"), "rn076")
    def test_split_vs_apps_agree(self):
        # same app scored via split and via --apps must give the same RN
        if os.path.exists("/workspace/hedis-eval/runs/apps_rel/zeus-13.1.3.triples"):
            self.assertEqual(ns["app_rn"]("zeus-13.1.3"), ns["app_rn"]("zeus-13.1.3", "0.85"))
if __name__ == "__main__": unittest.main()
