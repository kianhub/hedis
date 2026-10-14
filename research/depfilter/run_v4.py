#!/usr/bin/env python3
"""v4 one-shot harness: runs FROZEN depfilter.py (unchanged, sha 5290c1c2...) with APPDIR extended for the v4 apps.
The frozen code only knows the 8 v3 app dirs; unknown apps would fall back to apps/synth/<app> (non-existent) -> empty root set."""
import sys, runpy
import depfilter as D
D.APPDIR.update({"alby-go-1.13.1": "apps/oss/alby-go-1.13.1", "karakeep-mobile-0.25.0": "apps/oss/karakeep-mobile-0.25.0/apps/mobile",
 "trezor-mobile-25.5.3": "apps/oss/trezor-mobile-25.5.3/suite-native/app", "expensify-9.0.91": "apps/oss/expensify-9.0.91",
 "rabby-mobile-0.6.44": "apps/oss/rabby-mobile-0.6.44/apps/mobile", "edge-4.8.0": "apps/oss/edge-4.8.0",
 "eigen-8.73.0": "apps/oss/eigen-8.73.0", "chatwoot-4.3.0": "apps/oss/chatwoot-4.3.0"})
g = dict(vars(D)); g["__name__"] = "__main__"
exec(compile(open(D.__file__).read().split('if __name__ == "__main__":')[1].replace("\n    ", "\n"), "main", "exec"), g)
