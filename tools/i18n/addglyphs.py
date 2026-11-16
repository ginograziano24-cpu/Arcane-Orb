#!/usr/bin/env python3
"""Add the CJK glyphs the translation tables need to the subset fonts,
without the variable Noto fonts subset.py wants.

The game's subset fonts are Noto Sans SC / JP instanced at weight 600
(regular) and 800 (bold). Most Linux systems ship the static Noto Sans CJK
fonts, drawn from the same design. Measured on glyphs the subsets already
have, static Bold carries within 3% of the ink of the 600 instance and
static Black within 3% of the 800 one, so a missing glyph is copied from
those (cubic curves converted to TrueType) and appended to the subset.
Run from the repo root after build.py:

    python3 tools/i18n/addglyphs.py [/usr/share/fonts/opentype/noto]

Then glyphcheck.py should report 0 missing. Needs fonttools.
"""
import json
import sys

from fontTools.pens.cu2quPen import Cu2QuPen
from fontTools.pens.ttGlyphPen import TTGlyphPen
from fontTools.ttLib import TTCollection, TTFont

NOTO = sys.argv[1] if len(sys.argv) > 1 else "/usr/share/fonts/opentype/noto"
WEIGHTS = {"regular": "Bold", "bold": "Black"}  # subset weight -> static source
FACE = {"zh": "Noto Sans CJK SC", "ja": "Noto Sans CJK JP"}

_cache = {}


def static(weight, lang):
    key = (weight, lang)
    if key not in _cache:
        for f in TTCollection(f"{NOTO}/NotoSansCJK-{weight}.ttc").fonts:
            if f["name"].getDebugName(1).startswith(FACE[lang]):
                _cache[key] = f
                break
    return _cache[key]


def copied(lang, ch, weight):
    font = static(weight, lang)
    name = font.getBestCmap().get(ord(ch))
    if name is None:
        return None, 0
    g = font.getGlyphSet()[name]
    pen = TTGlyphPen(None)
    g.draw(Cu2QuPen(pen, max_err=1.0, reverse_direction=True))
    return pen.glyph(), g.width


def main():
    added = 0
    for lang in ("zh", "ja"):
        table = json.load(open(f"assets/i18n/{lang}.json", encoding="utf-8"))
        want = sorted({c for v in table.values() for c in v if ord(c) > 0x2E80})
        for wname, src in WEIGHTS.items():
            path = f"assets/fonts/cjk/{lang}-{wname}.ttf"
            font = TTFont(path)
            cmap = font.getBestCmap()
            missing = [c for c in want if ord(c) not in cmap]
            if not missing:
                continue
            order = font.getGlyphOrder()
            glyf, hmtx = font["glyf"], font["hmtx"]
            vmtx = font["vmtx"] if "vmtx" in font else None
            for c in missing:
                g, adv = copied(lang, c, src)
                if g is None:
                    print(f"[{lang}-{wname}] could not build {c!r}")
                    continue
                name = f"uni{ord(c):04X}"
                while name in glyf.glyphs:
                    name += ".add"
                order.append(name)
                glyf.glyphOrder = order
                glyf[name] = g
                g.recalcBounds(glyf)
                hmtx[name] = (adv, g.xMin if hasattr(g, "xMin") else 0)
                if vmtx is not None:
                    vmtx[name] = (1000, 880 - (g.yMax if hasattr(g, "yMax") else 880))
                for st in font["cmap"].tables:
                    if st.isUnicode():
                        st.cmap[ord(c)] = name
                added += 1
            font.setGlyphOrder(order)
            font["maxp"].numGlyphs = len(order)
            font.save(path)
            print(path, "+", len(missing))
    print(added, "glyphs added")


if __name__ == "__main__":
    main()
