#!/usr/bin/env python3
"""List Chinese/Japanese characters the translation tables use that the
embedded CJK subset fonts (assets/fonts/cjk) don't contain.

When the full Noto fonts aren't at hand to re-run subset.py, pick wording
that this script passes. Run from the repo root:  python3 tools/i18n/glyphcheck.py
"""
import json
import sys

from fontTools.ttLib import TTFont

bad = 0
for lang in ("zh", "ja"):
    have = None
    for w in ("regular", "bold"):
        cm = set(TTFont(f"assets/fonts/cjk/{lang}-{w}.ttf").getBestCmap())
        have = cm if have is None else have & cm
    table = json.load(open(f"assets/i18n/{lang}.json", encoding="utf-8"))
    for en, tr in table.items():
        miss = sorted({c for c in tr if ord(c) > 0x2E80 and ord(c) not in have})
        if miss:
            bad += 1
            print(f"[{lang}] missing {''.join(miss)}  in  {tr[:50]!r}")
print(f"{bad} strings use glyphs outside the subset fonts")
sys.exit(1 if bad else 0)
