#!/usr/bin/env python3
"""Build the small CJK fonts the game embeds (assets/fonts/cjk/*.ttf).

The full Noto Sans SC / JP fonts are 10-18 MB each, so the game carries
only the glyphs its Chinese and Japanese tables use. Re-run this whenever
those tables change:

    python3 tools/i18n/subset.py /path/to/NotoSansSC[wght].ttf /path/to/NotoSansJP[wght].ttf

The variable fonts come from github.com/google/fonts (ofl/notosanssc and
ofl/notosansjp), SIL Open Font License 1.1. Needs fonttools.
"""
import json
import sys

from fontTools import subset
from fontTools.ttLib import TTFont
from fontTools.varLib import instancer

# weights to match the UI fonts: Fira Sans SemiBold for body text, Bold for headings
WEIGHTS = {"regular": 600, "bold": 800}
LANG_NAMES = "简体中文日本語"  # drawn on the language picker in every language


def chars_for(lang):
    table = json.load(open(f"assets/i18n/{lang}.json", encoding="utf-8"))
    text = "".join(table.values()) + LANG_NAMES
    # generated chamber names are composed in code (i18n_names.go)
    text += "".join(c for c in open("i18n_names.go", encoding="utf-8").read() if ord(c) > 0x2E80)
    # ASCII and common CJK punctuation so mixed strings never miss a glyph
    text += "".join(chr(c) for c in range(0x20, 0x7F))
    text += "　、。〈〉《》「」『』【】〔〕・ー～！？：；（），．…—–·×✓✦"
    return sorted(set(text))


def build(src, lang):
    chars = chars_for(lang)
    for name, wght in WEIGHTS.items():
        font = TTFont(src)
        if "fvar" in font:
            font = instancer.instantiateVariableFont(font, {"wght": wght})
        opts = subset.Options()
        opts.layout_features = ["*"]
        opts.name_IDs = ["*"]
        opts.notdef_outline = True
        sub = subset.Subsetter(opts)
        sub.populate(unicodes=[ord(c) for c in chars])
        sub.subset(font)
        out = f"assets/fonts/cjk/{lang}-{name}.ttf"
        font.save(out)
        print(out, len(chars), "characters")


if __name__ == "__main__":
    if len(sys.argv) != 3:
        sys.exit(__doc__)
    build(sys.argv[1], "zh")
    build(sys.argv[2], "ja")
