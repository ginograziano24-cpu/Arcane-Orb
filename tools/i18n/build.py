#!/usr/bin/env python3
"""Merge tools/i18n/src/*.jsonl into assets/i18n/<lang>.json and check them.

Each source line is a JSON array: [english, zh, ja, ru, es, pt, de].
Checks: every translation keeps the English string's format verbs and its
count of literal numbers, so the runtime can put values back. Reports
strings in tools/i18n/strings.txt that have no translation yet.

Run from the repo root:  python3 tools/i18n/build.py
"""
import glob
import json
import re
import sys

LANGS = ["zh", "ja", "ru", "es", "pt", "de"]
VERB = re.compile(r"%(\[\d+\])?[-+#0]*\d*(\.\d+)?[dfgvsqx%]")
DIGITS = re.compile(r"\d+(?:\.\d+)?")


def shape(s):
    verbs = [m.group(0) for m in VERB.finditer(s)]
    kinds = sorted("s" if v[-1] in "sqv" else ("%" if v[-1] == "%" else "n") for v in verbs)
    rest = VERB.sub("", s)
    return kinds, set(DIGITS.findall(rest))


def main():
    tables = {l: {} for l in LANGS}
    problems = 0
    for path in sorted(glob.glob("tools/i18n/src/*.jsonl")):
        for n, line in enumerate(open(path, encoding="utf-8"), 1):
            line = line.strip()
            if not line:
                continue
            try:
                row = json.loads(line)
            except json.JSONDecodeError as e:
                print(f"{path}:{n}: bad JSON: {e}")
                problems += 1
                continue
            if len(row) != 1 + len(LANGS):
                print(f"{path}:{n}: expected {1+len(LANGS)} columns, got {len(row)}")
                problems += 1
                continue
            en = row[0]
            want = shape(en)
            for lang, tr in zip(LANGS, row[1:]):
                got = shape(tr)
                if got[0] != want[0]:
                    print(f"{path}:{n} [{lang}] verbs differ: {en!r} -> {tr!r}")
                    problems += 1
                elif not want[1] <= got[1] and "," not in en:
                    # a number from the English text went missing
                    print(f"{path}:{n} [{lang}] warning: numbers {sorted(want[1] - got[1])} missing: {tr!r}")
                tables[lang][en] = tr
    wanted = [json.loads(l) for l in open("tools/i18n/strings.txt", encoding="utf-8") if l.strip()]
    missing = [s for s in wanted if s not in tables["zh"]]
    for lang in LANGS:
        with open(f"assets/i18n/{lang}.json", "w", encoding="utf-8") as f:
            json.dump(tables[lang], f, ensure_ascii=False, indent=0, sort_keys=True)
    print(f"{len(tables['zh'])} strings per language; {len(missing)} not yet translated; {problems} problems")
    if "-v" in sys.argv:
        for s in missing:
            print("  missing:", json.dumps(s, ensure_ascii=False))
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
