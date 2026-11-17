#!/usr/bin/env python3
"""luaucheck: a small Luau syntax checker for the Roblox port.

No Luau toolchain is available in the cloud workspace, so this parses the
subset of Luau the port uses (typed locals and functions, type aliases,
if-expressions, compound assignment, continue, generic for, table
constructors, method calls, string interpolation) and reports the first
syntax error per file. It also warns about globals that are read but never
defined, which in Luau is almost always a typo.

    python3 luaucheck.py ../../roblox/src
"""
import os
import re
import sys

KEYWORDS = {
    "and", "break", "do", "else", "elseif", "end", "false", "for", "function", "if", "in",
    "local", "nil", "not", "or", "repeat", "return", "then", "true", "until", "while",
}
SOFT = {"continue", "type", "export", "typeof"}
GLOBALS = {
    "game", "workspace", "script", "require", "print", "warn", "error", "assert", "pairs", "ipairs",
    "next", "select", "type", "typeof", "tostring", "tonumber", "math", "string", "table", "task",
    "os", "coroutine", "utf8", "bit32", "buffer", "setmetatable", "getmetatable", "rawget", "rawset",
    "rawequal", "rawlen", "pcall", "xpcall", "unpack", "Vector3", "Vector2", "CFrame", "Color3",
    "UDim", "UDim2", "Instance", "Enum", "BrickColor", "TweenInfo", "Random", "NumberSequence",
    "ColorSequence", "NumberSequenceKeypoint", "ColorSequenceKeypoint", "NumberRange", "Ray",
    "RaycastParams", "OverlapParams", "Rect", "Region3", "tick", "time", "elapsedTime", "delay",
    "spawn", "wait", "debug", "newproxy", "shared", "_G", "DateTime", "PhysicalProperties", "Font",
}

TOKEN_RE = re.compile(r"""
    (?P<ws>[ \t\r]+) |
    (?P<nl>\n) |
    (?P<comment>--\[(?P<ceq>=*)\[.*?\](?P=ceq)\]|--[^\n]*) |
    (?P<lstr>\[(?P<seq>=*)\[.*?\](?P=seq)\]) |
    (?P<num>0[xX][0-9a-fA-F_]+|0[bB][01_]+|(?:\d[\d_]*\.?[\d_]*|\.\d[\d_]*)(?:[eE][+-]?\d+)?) |
    (?P<str>"(?:\\.|[^"\\\n])*"|'(?:\\.|[^'\\\n])*') |
    (?P<istr>`(?:\\.|[^`\\])*`) |
    (?P<name>[A-Za-z_][A-Za-z0-9_]*) |
    (?P<op>\.\.\.|\.\.=|\.\.|==|~=|<=|>=|\+=|-=|\*=|/=|//=|//|%=|\^=|::|->|[-+*/%^#<>=(){}\[\];:,.?|&])
""", re.S | re.X)


class Err(Exception):
    pass


def tokenize(src):
    toks, line, i = [], 1, 0
    while i < len(src):
        m = TOKEN_RE.match(src, i)
        if not m:
            raise Err("line %d: unexpected character %r" % (line, src[i]))
        kind = m.lastgroup
        text = m.group(0)
        if kind in ("ceq", "seq"):
            kind = "comment" if text.startswith("--") else "lstr"
        if kind == "nl":
            line += 1
        elif kind == "comment" or kind == "lstr":
            if kind == "lstr":
                toks.append(("str", text, line))
            line += text.count("\n")
        elif kind != "ws":
            if kind == "name" and text in KEYWORDS:
                kind = "kw"
            if kind == "istr":
                kind = "str"
                line += text.count("\n")
            toks.append((kind, text, line))
        i = m.end()
    toks.append(("eof", "", line))
    return toks


BINPRI = {
    "or": (1, 1), "and": (2, 2),
    "<": (3, 3), ">": (3, 3), "<=": (3, 3), ">=": (3, 3), "~=": (3, 3), "==": (3, 3),
    "..": (9, 8), "+": (10, 10), "-": (10, 10),
    "*": (11, 11), "/": (11, 11), "//": (11, 11), "%": (11, 11),
    "^": (14, 13),
}
UNPRI = 12


class Parser:
    def __init__(self, toks):
        self.t = toks
        self.i = 0
        self.scopes = [set()]
        self.undefined = {}

    # -- helpers
    def peek(self, k=0):
        return self.t[self.i + k]

    def at(self, text, k=0):
        tok = self.t[self.i + k]
        return tok[1] == text and tok[0] in ("kw", "op", "name")

    def next(self):
        tok = self.t[self.i]
        self.i += 1
        return tok

    def expect(self, text):
        tok = self.next()
        if tok[1] != text or tok[0] == "str":
            raise Err("line %d: expected %r near %r" % (tok[2], text, tok[1]))
        return tok

    def name(self):
        tok = self.next()
        if tok[0] != "name":
            raise Err("line %d: expected a name near %r" % (tok[2], tok[1]))
        return tok[1]

    def declare(self, n):
        self.scopes[-1].add(n)

    def use(self, n, line):
        for s in self.scopes:
            if n in s:
                return
        if n not in GLOBALS:
            self.undefined.setdefault(n, line)

    def push(self):
        self.scopes.append(set())

    def pop(self):
        self.scopes.pop()

    # -- blocks
    def chunk(self):
        self.block()
        if self.peek()[0] != "eof":
            tok = self.peek()
            raise Err("line %d: unexpected %r" % (tok[2], tok[1]))

    def block_end(self):
        tok = self.peek()
        return tok[0] == "eof" or (tok[0] == "kw" and tok[1] in ("end", "else", "elseif", "until"))

    def block(self):
        while not self.block_end():
            if self.at("return"):
                self.next()
                if not self.block_end() and not self.at(";"):
                    self.exprlist()
                if self.at(";"):
                    self.next()
                if not self.block_end():
                    tok = self.peek()
                    raise Err("line %d: code after return near %r" % (tok[2], tok[1]))
                return
            self.statement()
            if self.at(";"):
                self.next()

    def statement(self):
        tok = self.peek()
        k, v = tok[0], tok[1]
        if k == "kw":
            if v == "local":
                self.next()
                if self.at("function"):
                    self.next()
                    n = self.name()
                    self.declare(n)
                    self.funcbody()
                    return
                names = [self.typed_name()]
                while self.at(","):
                    self.next()
                    names.append(self.typed_name())
                if self.at("="):
                    self.next()
                    self.exprlist()
                for n in names:
                    self.declare(n)
                return
            if v == "function":
                self.next()
                n = self.name()
                self.use(n, tok[2])
                while self.at(".") or self.at(":"):
                    self.next()
                    self.name()
                self.funcbody()
                return
            if v == "if":
                self.next()
                self.expr()
                self.expect("then")
                self.scoped_block()
                while self.at("elseif"):
                    self.next()
                    self.expr()
                    self.expect("then")
                    self.scoped_block()
                if self.at("else"):
                    self.next()
                    self.scoped_block()
                self.expect("end")
                return
            if v == "while":
                self.next()
                self.expr()
                self.expect("do")
                self.scoped_block()
                self.expect("end")
                return
            if v == "do":
                self.next()
                self.scoped_block()
                self.expect("end")
                return
            if v == "for":
                self.next()
                self.push()
                n1 = self.typed_name()
                self.declare(n1)
                if self.at("="):
                    self.next()
                    self.expr()
                    self.expect(",")
                    self.expr()
                    if self.at(","):
                        self.next()
                        self.expr()
                else:
                    while self.at(","):
                        self.next()
                        self.declare(self.typed_name())
                    self.expect("in")
                    self.exprlist()
                self.expect("do")
                self.block()
                self.pop()
                self.expect("end")
                return
            if v == "repeat":
                self.next()
                self.push()
                self.block()
                self.expect("until")
                self.expr()
                self.pop()
                return
            if v == "break":
                self.next()
                return
            raise Err("line %d: unexpected %r" % (tok[2], v))
        if k == "name" and v == "continue" and not self.at("(", 1) and not self.at("=", 1) and not self.at(".", 1):
            self.next()
            return
        if k == "name" and v in ("type", "export") and self.peek(1)[0] == "name":
            if v == "export":
                self.next()
            self.expect("type")
            self.name()
            if self.at("<"):
                self.generic_params()
            self.expect("=")
            self.type_()
            return
        # expression statement or assignment
        self.suffixed()
        if self.at("=") or self.at(","):
            while self.at(","):
                self.next()
                self.suffixed()
            self.expect("=")
            self.exprlist()
        elif self.peek()[1] in ("+=", "-=", "*=", "/=", "//=", "%=", "^=", "..="):
            self.next()
            self.expr()

    def scoped_block(self):
        self.push()
        self.block()
        self.pop()

    def typed_name(self):
        n = self.name()
        if self.at(":"):
            self.next()
            self.type_()
        return n

    def funcbody(self):
        self.push()
        if self.at("<"):
            self.generic_params()
        self.expect("(")
        if not self.at(")"):
            while True:
                if self.at("..."):
                    self.next()
                    if self.at(":"):
                        self.next()
                        self.type_()
                    break
                self.declare(self.typed_name())
                if not self.at(","):
                    break
                self.next()
        self.expect(")")
        if self.at(":"):
            self.next()
            self.return_type()
        self.declare("self")
        self.block()
        self.expect("end")
        self.pop()

    # -- types (parsed loosely: enough to skip them correctly)
    def generic_params(self):
        self.expect("<")
        depth = 1
        while depth:
            tok = self.next()
            if tok[1] == "<":
                depth += 1
            elif tok[1] == ">":
                depth -= 1
            elif tok[0] == "eof":
                raise Err("unterminated generic list")

    def return_type(self):
        if self.at("("):
            self.type_()
        else:
            self.type_()

    def type_(self):
        self.simple_type()
        while self.at("|") or self.at("&") or self.at("?"):
            if self.at("?"):
                self.next()
                continue
            self.next()
            self.simple_type()

    def simple_type(self):
        tok = self.peek()
        if self.at("{"):
            self.next()
            while not self.at("}"):
                if self.at("["):
                    self.next()
                    self.type_()
                    self.expect("]")
                    self.expect(":")
                    self.type_()
                elif self.peek()[0] in ("name", "kw") and self.at(":", 1):
                    self.next()
                    self.next()
                    self.type_()
                else:
                    self.type_()
                if self.at(",") or self.at(";"):
                    self.next()
                else:
                    break
            self.expect("}")
        elif self.at("("):
            self.next()
            while not self.at(")"):
                if self.peek()[0] == "name" and self.at(":", 1):
                    self.next()
                    self.next()
                if self.at("..."):
                    self.next()
                self.type_()
                if self.at(","):
                    self.next()
                else:
                    break
            self.expect(")")
            if self.at("->"):
                self.next()
                self.type_()
        elif tok[0] == "name" and tok[1] == "typeof":
            self.next()
            self.expect("(")
            self.expr()
            self.expect(")")
        elif tok[0] in ("name", "kw", "str"):
            self.next()
            while self.at("."):
                self.next()
                self.name()
            if self.at("<"):
                self.generic_params()
        elif self.at("..."):
            self.next()
            self.type_()
        else:
            raise Err("line %d: bad type near %r" % (tok[2], tok[1]))
        if self.at("?"):
            self.next()

    # -- expressions
    def exprlist(self):
        self.expr()
        while self.at(","):
            self.next()
            self.expr()

    def expr(self, limit=0):
        tok = self.peek()
        if tok[1] in ("not", "-", "#") and tok[0] in ("kw", "op"):
            self.next()
            self.expr(UNPRI)
        else:
            self.simple()
        while True:
            op = self.peek()
            if op[0] in ("op", "kw") and op[1] in BINPRI and BINPRI[op[1]][0] > limit:
                self.next()
                self.expr(BINPRI[op[1]][1])
            else:
                break
        if self.at("::"):
            self.next()
            self.type_()

    def simple(self):
        tok = self.peek()
        k, v = tok[0], tok[1]
        if k in ("num", "str") or (k == "kw" and v in ("nil", "true", "false")) or v == "...":
            self.next()
            return
        if k == "kw" and v == "function":
            self.next()
            self.funcbody()
            return
        if k == "kw" and v == "if":
            self.next()
            self.expr()
            self.expect("then")
            self.expr()
            while self.at("elseif"):
                self.next()
                self.expr()
                self.expect("then")
                self.expr()
            self.expect("else")
            self.expr()
            return
        if v == "{":
            self.table()
            return
        self.suffixed()

    def table(self):
        self.expect("{")
        while not self.at("}"):
            if self.at("["):
                self.next()
                self.expr()
                self.expect("]")
                self.expect("=")
                self.expr()
            elif self.peek()[0] == "name" and self.at("=", 1):
                self.next()
                self.next()
                self.expr()
            else:
                self.expr()
            if self.at(",") or self.at(";"):
                self.next()
            else:
                break
        self.expect("}")

    def primary(self):
        tok = self.next()
        if tok[0] == "name":
            self.use(tok[1], tok[2])
            return
        if tok[1] == "(":
            self.expr()
            self.expect(")")
            return
        raise Err("line %d: unexpected %r" % (tok[2], tok[1]))

    def suffixed(self):
        self.primary()
        while True:
            if self.at("."):
                self.next()
                tok = self.next()
                if tok[0] not in ("name", "kw"):
                    raise Err("line %d: expected a field name near %r" % (tok[2], tok[1]))
            elif self.at("["):
                self.next()
                self.expr()
                self.expect("]")
            elif self.at(":") and self.peek(1)[0] == "name" and (self.at("(", 2) or self.peek(2)[0] == "str" or self.at("{", 2)):
                self.next()
                self.next()
                self.args()
            elif self.at("(") or self.peek()[0] == "str" or self.at("{"):
                self.args()
            else:
                return

    def args(self):
        if self.peek()[0] == "str":
            self.next()
            return
        if self.at("{"):
            self.table()
            return
        self.expect("(")
        if not self.at(")"):
            self.exprlist()
        self.expect(")")


def check(path):
    src = open(path, encoding="utf-8").read()
    try:
        p = Parser(tokenize(src))
        p.chunk()
    except Err as e:
        return "%s: %s" % (path, e), []
    return None, ["%s:%d: global %r is never defined" % (path, ln, n) for n, ln in sorted(p.undefined.items(), key=lambda x: x[1])]


def main():
    roots = sys.argv[1:] or ["."]
    bad = 0
    files = 0
    for root in roots:
        paths = [root] if os.path.isfile(root) else [
            os.path.join(d, f) for d, _, fs in os.walk(root) for f in fs if f.endswith((".luau", ".lua"))]
        for path in sorted(paths):
            files += 1
            err, warns = check(path)
            if err:
                print("ERROR " + err)
                bad += 1
            for w in warns:
                print("warn  " + w)
    print("%d files checked, %d with syntax errors" % (files, bad))
    sys.exit(1 if bad else 0)


if __name__ == "__main__":
    main()
