#!/usr/bin/env python3
"""luaurun: a small Luau interpreter for testing the Roblox port headless.

There is no Luau runtime in the cloud workspace, so this runs the subset of
Luau the port's simulation uses, with just enough of the Roblox API stubbed
(Random, Color3, script/require, game:GetService for ReplicatedStorage) to
play whole runs of Sim/Level.luau and Sim/Run.luau with an autopilot. It
catches what the syntax checker can't: nil fields, typos in field names,
arithmetic on nil, wrong argument order.

    python3 luaurun.py ../../roblox/src tests/sim_test.luau
"""
import math
import os
import random
import sys

sys.setrecursionlimit(20000)
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import luaucheck as lc  # noqa: E402


class LuaError(Exception):
    def __init__(self, msg, line=None):
        super().__init__(msg)
        self.line = line


class Table:
    __slots__ = ("h", "meta")

    def __init__(self, h=None):
        self.h = h if h is not None else {}
        self.meta = None

    def get(self, k):
        if isinstance(k, float) and k.is_integer():
            k = int(k)
        v = self.h.get(k)
        if v is None and self.meta is not None:
            idx = self.meta.h.get("__index")
            if idx is not None:
                if isinstance(idx, Table):
                    return idx.get(k)
                return call(idx, [self, k])[0] if call(idx, [self, k]) else None
        return v

    def set(self, k, v):
        if k is None:
            raise LuaError("table index is nil")
        if isinstance(k, float) and k.is_integer():
            k = int(k)
        if v is None:
            self.h.pop(k, None)
        else:
            self.h[k] = v

    def length(self):
        n = 0
        while (n + 1) in self.h:
            n += 1
        return n


class Function:
    __slots__ = ("params", "vararg", "body", "env", "name")

    def __init__(self, params, vararg, body, env, name="?"):
        self.params, self.vararg, self.body, self.env, self.name = params, vararg, body, env, name


class Builtin:
    __slots__ = ("fn", "name")

    def __init__(self, fn, name):
        self.fn, self.name = fn, name


class Return(Exception):
    def __init__(self, vals):
        self.vals = vals


class Break(Exception):
    pass


class Continue(Exception):
    pass


# ---------------------------------------------------------------- AST builder

class Builder(lc.Parser):
    """Re-parses with the checker's grammar, producing tuples."""

    def chunk(self):
        b = self.block()
        if self.peek()[0] != "eof":
            tok = self.peek()
            raise lc.Err("line %d: unexpected %r" % (tok[2], tok[1]))
        return b

    def block(self):
        out = []
        while not self.block_end():
            if self.at("return"):
                line = self.next()[2]
                vals = []
                if not self.block_end() and not self.at(";"):
                    vals = self.exprlist()
                if self.at(";"):
                    self.next()
                out.append(("return", line, vals))
                return out
            st = self.statement()
            if st is not None:
                out.append(st)
            if self.at(";"):
                self.next()
        return out

    def scoped_block(self):
        return self.block()

    def statement(self):
        tok = self.peek()
        k, v, line = tok
        if k == "kw":
            if v == "local":
                self.next()
                if self.at("function"):
                    self.next()
                    n = self.name()
                    f = self.funcbody(n)
                    return ("localfunc", line, n, f)
                names = [self.typed_name()]
                while self.at(","):
                    self.next()
                    names.append(self.typed_name())
                vals = []
                if self.at("="):
                    self.next()
                    vals = self.exprlist()
                return ("local", line, names, vals)
            if v == "function":
                self.next()
                n = self.name()
                target = ("name", line, n)
                method = False
                full = n
                while self.at(".") or self.at(":"):
                    sep = self.next()[1]
                    f = self.name()
                    full += sep + f
                    if sep == ":":
                        method = True
                    target = ("index", line, target, ("const", line, f))
                fn = self.funcbody(full, method)
                return ("assign", line, [target], [fn])
            if v == "if":
                self.next()
                arms = []
                c = self.expr()
                self.expect("then")
                arms.append((c, self.block()))
                els = None
                while self.at("elseif"):
                    self.next()
                    c = self.expr()
                    self.expect("then")
                    arms.append((c, self.block()))
                if self.at("else"):
                    self.next()
                    els = self.block()
                self.expect("end")
                return ("if", line, arms, els)
            if v == "while":
                self.next()
                c = self.expr()
                self.expect("do")
                b = self.block()
                self.expect("end")
                return ("while", line, c, b)
            if v == "do":
                self.next()
                b = self.block()
                self.expect("end")
                return ("do", line, b)
            if v == "for":
                self.next()
                n1 = self.typed_name()
                if self.at("="):
                    self.next()
                    a = self.expr()
                    self.expect(",")
                    b = self.expr()
                    c = None
                    if self.at(","):
                        self.next()
                        c = self.expr()
                    self.expect("do")
                    body = self.block()
                    self.expect("end")
                    return ("fornum", line, n1, a, b, c, body)
                names = [n1]
                while self.at(","):
                    self.next()
                    names.append(self.typed_name())
                self.expect("in")
                exprs = self.exprlist()
                self.expect("do")
                body = self.block()
                self.expect("end")
                return ("forin", line, names, exprs, body)
            if v == "repeat":
                self.next()
                b = self.block()
                self.expect("until")
                c = self.expr()
                return ("repeat", line, b, c)
            if v == "break":
                self.next()
                return ("break", line)
            raise lc.Err("line %d: unexpected %r" % (line, v))
        if k == "name" and v == "continue" and not self.at("(", 1) and not self.at("=", 1) and not self.at(".", 1):
            self.next()
            return ("continue", line)
        if k == "name" and v in ("type", "export") and self.peek(1)[0] == "name":
            if v == "export":
                self.next()
            self.expect("type")
            self.name()
            if self.at("<"):
                self.generic_params()
            self.expect("=")
            self.type_()
            return None
        e = self.suffixed()
        if self.at("=") or self.at(","):
            targets = [e]
            while self.at(","):
                self.next()
                targets.append(self.suffixed())
            self.expect("=")
            vals = self.exprlist()
            return ("assign", line, targets, vals)
        op = self.peek()[1]
        if op in ("+=", "-=", "*=", "/=", "//=", "%=", "^=", "..="):
            self.next()
            val = self.expr()
            return ("compound", line, e, op[:-1], val)
        return ("exprstat", line, e)

    def funcbody(self, name="?", method=False):
        line = self.peek()[2]
        if self.at("<"):
            self.generic_params()
        self.expect("(")
        params = ["self"] if method else []
        vararg = False
        if not self.at(")"):
            while True:
                if self.at("..."):
                    self.next()
                    vararg = True
                    if self.at(":"):
                        self.next()
                        self.type_()
                    break
                params.append(self.typed_name())
                if not self.at(","):
                    break
                self.next()
        self.expect(")")
        if self.at(":"):
            self.next()
            self.return_type()
        body = self.block()
        self.expect("end")
        return ("function", line, params, vararg, body, "%s:%s" % (getattr(self, "file", "?"), name))

    def exprlist(self):
        out = [self.expr()]
        while self.at(","):
            self.next()
            out.append(self.expr())
        return out

    def expr(self, limit=0):
        tok = self.peek()
        if tok[1] in ("not", "-", "#") and tok[0] in ("kw", "op"):
            self.next()
            operand = self.expr(lc.UNPRI)
            left = ("unop", tok[2], tok[1], operand)
        else:
            left = self.simple()
        while True:
            op = self.peek()
            if op[0] in ("op", "kw") and op[1] in lc.BINPRI and lc.BINPRI[op[1]][0] > limit:
                self.next()
                right = self.expr(lc.BINPRI[op[1]][1])
                left = ("binop", op[2], op[1], left, right)
            else:
                break
        if self.at("::"):
            self.next()
            self.type_()
        return left

    def simple(self):
        tok = self.peek()
        k, v, line = tok
        if k == "num":
            self.next()
            t = v.replace("_", "")
            if t.lower().startswith("0x"):
                return ("const", line, float(int(t, 16)))
            if t.lower().startswith("0b"):
                return ("const", line, float(int(t[2:], 2)))
            return ("const", line, float(t))
        if k == "str":
            self.next()
            return self.string_lit(v, line)
        if k == "kw" and v in ("nil", "true", "false"):
            self.next()
            return ("const", line, {"nil": None, "true": True, "false": False}[v])
        if v == "...":
            self.next()
            return ("vararg", line)
        if k == "kw" and v == "function":
            self.next()
            return self.funcbody()
        if k == "kw" and v == "if":
            self.next()
            arms = []
            c = self.expr()
            self.expect("then")
            arms.append((c, self.expr()))
            while self.at("elseif"):
                self.next()
                c = self.expr()
                self.expect("then")
                arms.append((c, self.expr()))
            self.expect("else")
            return ("ifexpr", line, arms, self.expr())
        if v == "{":
            return self.table()
        return self.suffixed()

    def string_lit(self, v, line):
        if v.startswith("["):
            eq = v.index("[", 1)
            body = v[eq + 1: len(v) - eq - 1]
            if body.startswith("\n"):
                body = body[1:]
            return ("const", line, body)
        q = v[0]
        body = v[1:-1]
        out, i = [], 0
        while i < len(body):
            ch = body[i]
            if ch == "\\":
                i += 1
                e = body[i]
                out.append({"n": "\n", "t": "\t", "\\": "\\", '"': '"', "'": "'", "`": "`", "r": "\r", "0": "\0"}.get(e, e))
            else:
                out.append(ch)
            i += 1
        s = "".join(out)
        if q == "`":
            return ("interp", line, s)
        return ("const", line, s)

    def table(self):
        line = self.expect("{")[2]
        items = []
        while not self.at("}"):
            if self.at("["):
                self.next()
                kexp = self.expr()
                self.expect("]")
                self.expect("=")
                items.append(("kv", kexp, self.expr()))
            elif self.peek()[0] == "name" and self.at("=", 1):
                n = self.next()[1]
                self.next()
                items.append(("kv", ("const", line, n), self.expr()))
            else:
                items.append(("pos", self.expr()))
            if self.at(",") or self.at(";"):
                self.next()
            else:
                break
        self.expect("}")
        return ("table", line, items)

    def primary(self):
        tok = self.next()
        if tok[0] == "name":
            return ("name", tok[2], tok[1])
        if tok[1] == "(":
            e = self.expr()
            self.expect(")")
            return ("paren", tok[2], e)
        raise lc.Err("line %d: unexpected %r" % (tok[2], tok[1]))

    def suffixed(self):
        e = self.primary()
        while True:
            line = self.peek()[2]
            if self.at("."):
                self.next()
                n = self.next()[1]
                e = ("index", line, e, ("const", line, n))
            elif self.at("["):
                self.next()
                k = self.expr()
                self.expect("]")
                e = ("index", line, e, k)
            elif self.at(":") and self.peek(1)[0] == "name" and (self.at("(", 2) or self.peek(2)[0] == "str" or self.at("{", 2)):
                self.next()
                n = self.next()[1]
                e = ("method", line, e, n, self.args())
            elif self.at("(") or self.peek()[0] == "str" or self.at("{"):
                e = ("call", line, e, self.args())
            else:
                return e

    def args(self):
        if self.peek()[0] == "str":
            tok = self.next()
            return [self.string_lit(tok[1], tok[2])]
        if self.at("{"):
            return [self.table()]
        self.expect("(")
        a = [] if self.at(")") else self.exprlist()
        self.expect(")")
        return a


# ---------------------------------------------------------------- values

def tostr(v):
    if v is None:
        return "nil"
    if v is True:
        return "true"
    if v is False:
        return "false"
    if isinstance(v, float):
        if v.is_integer() and abs(v) < 1e15:
            return str(int(v))
        if v != v:
            return "nan"
        if v in (math.inf, -math.inf):
            return "inf" if v > 0 else "-inf"
        return "%.14g" % v
    if isinstance(v, int):
        return str(v)
    if isinstance(v, str):
        return v
    if isinstance(v, Table):
        return "table: 0x%x" % id(v)
    if isinstance(v, (Function, Builtin)):
        return "function"
    return str(v)


def truthy(v):
    return v is not None and v is not False


def num(v, what="arithmetic"):
    if isinstance(v, bool) or v is None:
        raise LuaError("attempt to perform %s on %s" % (what, "nil" if v is None else "boolean"))
    if isinstance(v, (int, float)):
        return float(v)
    if isinstance(v, str):
        try:
            return float(v)
        except ValueError:
            pass
    raise LuaError("attempt to perform %s on %s" % (what, type_of(v)))


def type_of(v):
    if v is None:
        return "nil"
    if isinstance(v, bool):
        return "boolean"
    if isinstance(v, (int, float)):
        return "number"
    if isinstance(v, str):
        return "string"
    if isinstance(v, Table):
        return "table"
    if isinstance(v, (Function, Builtin)):
        return "function"
    return "userdata"


def index(obj, key, line=None):
    if isinstance(obj, Table):
        return obj.get(key)
    if isinstance(obj, str):
        return STRING_LIB.get(key)
    if hasattr(obj, "lua_index"):
        return obj.lua_index(key)
    raise LuaError("attempt to index %s with %r" % (type_of(obj), key), line)


def setindex(obj, key, val, line=None):
    if isinstance(obj, Table):
        obj.set(key, val)
        return
    if hasattr(obj, "lua_setindex"):
        obj.lua_setindex(key, val)
        return
    raise LuaError("attempt to index %s with %r" % (type_of(obj), key), line)


def call(f, args):
    if isinstance(f, Builtin):
        r = f.fn(*args)
        if r is None:
            return []
        if isinstance(r, tuple):
            return list(r)
        return [r]
    if isinstance(f, Function):
        env = Env(f.env)
        for i, p in enumerate(f.params):
            env.declare(p, args[i] if i < len(args) else None)
        if f.vararg:
            env.varargs = list(args[len(f.params):])
        try:
            exec_block(f.body, env)
        except Return as r:
            return r.vals
        return []
    if isinstance(f, Table) and f.meta is not None and "__call" in f.meta.h:
        return call(f.meta.h["__call"], [f] + list(args))
    raise LuaError("attempt to call a %s value" % type_of(f))


class Env:
    __slots__ = ("vars", "parent", "varargs")

    def __init__(self, parent=None):
        self.vars = {}
        self.parent = parent
        self.varargs = parent.varargs if parent is not None else []

    def declare(self, n, v):
        self.vars[n] = [v]

    def find(self, n):
        e = self
        while e is not None:
            c = e.vars.get(n)
            if c is not None:
                return c
            e = e.parent
        return None


GLOBALS = Env()


def lookup(env, n, line):
    c = env.find(n)
    if c is None:
        raise LuaError("unknown global %r" % n, line)
    return c[0]


def arith(op, a, b, line):
    if op == "..":
        if isinstance(a, (str, int, float)) and not isinstance(a, bool) and isinstance(b, (str, int, float)) and not isinstance(b, bool):
            return tostr(a) + tostr(b)
        raise LuaError("attempt to concatenate %s with %s" % (type_of(a), type_of(b)), line)
    if op in ("==", "~="):
        eq = a == b if not (isinstance(a, bool) ^ isinstance(b, bool)) else False
        if type_of(a) != type_of(b):
            eq = False
        return eq if op == "==" else not eq
    if op in ("<", ">", "<=", ">="):
        if isinstance(a, str) and isinstance(b, str):
            x, y = a, b
        else:
            if type_of(a) != "number" or type_of(b) != "number":
                raise LuaError("attempt to compare %s %s %s" % (type_of(a), op, type_of(b)), line)
            x, y = a, b
        return {"<": x < y, ">": x > y, "<=": x <= y, ">=": x >= y}[op]
    if hasattr(a, "lua_arith"):
        return a.lua_arith(op, b)
    if hasattr(b, "lua_rarith"):
        return b.lua_rarith(op, a)
    x, y = num(a), num(b)
    try:
        if op == "+":
            return x + y
        if op == "-":
            return x - y
        if op == "*":
            return x * y
        if op == "/":
            if y == 0:
                return math.inf if x > 0 else (-math.inf if x < 0 else math.nan)
            return x / y
        if op == "//":
            if y == 0:
                return math.inf if x > 0 else (-math.inf if x < 0 else math.nan)
            return float(math.floor(x / y))
        if op == "%":
            if y == 0:
                return math.nan
            return x - math.floor(x / y) * y
        if op == "^":
            return float(x ** y)
    except OverflowError:
        return math.inf
    raise LuaError("bad operator " + op, line)


def eval_multi(e, env):
    """Evaluates an expression that may produce several values."""
    t = e[0]
    if t == "call":
        f = eval_expr(e[2], env)
        args = eval_args(e[3], env)
        try:
            return call(f, args)
        except LuaError as err:
            if err.line is None:
                err.line = e[1]
            if isinstance(f, (Function, Builtin)) and not getattr(err, "traced", False):
                err.args = (str(err.args[0]) + "\n    in " + f.name + " (called at line %d)" % e[1],)
            raise
    if t == "method":
        obj = eval_expr(e[2], env)
        f = index(obj, e[3], e[1])
        if f is None:
            raise LuaError("attempt to call missing method %r on %s" % (e[3], type_of(obj)), e[1])
        args = [obj] + eval_args(e[4], env)
        try:
            return call(f, args)
        except LuaError as err:
            if err.line is None:
                err.line = e[1]
            err.args = (str(err.args[0]) + "\n    in :" + e[3] + " (called at line %d)" % e[1],)
            raise
    if t == "vararg":
        return list(env.varargs)
    return [eval_expr(e, env)]


def eval_args(exprs, env):
    out = []
    for i, a in enumerate(exprs):
        if i == len(exprs) - 1:
            out.extend(eval_multi(a, env))
        else:
            out.append(eval_expr(a, env))
    return out


def eval_expr(e, env):
    t = e[0]
    if t == "const":
        return e[2]
    if t == "name":
        return lookup(env, e[2], e[1])
    if t == "index":
        obj = eval_expr(e[2], env)
        key = eval_expr(e[3], env)
        if obj is None:
            raise LuaError("attempt to index nil with %r" % (key,), e[1])
        return index(obj, key, e[1])
    if t == "call" or t == "method" or t == "vararg":
        r = eval_multi(e, env)
        return r[0] if r else None
    if t == "paren":
        return eval_expr(e[2], env)
    if t == "binop":
        op = e[2]
        if op == "and":
            a = eval_expr(e[3], env)
            return eval_expr(e[4], env) if truthy(a) else a
        if op == "or":
            a = eval_expr(e[3], env)
            return a if truthy(a) else eval_expr(e[4], env)
        return arith(op, eval_expr(e[3], env), eval_expr(e[4], env), e[1])
    if t == "unop":
        v = eval_expr(e[3], env)
        if e[2] == "not":
            return not truthy(v)
        if e[2] == "-":
            if hasattr(v, "lua_neg"):
                return v.lua_neg()
            return -num(v)
        if isinstance(v, str):
            return float(len(v))
        if isinstance(v, Table):
            return float(v.length())
        raise LuaError("attempt to get length of " + type_of(v), e[1])
    if t == "table":
        tb = Table()
        n = 1
        items = e[2]
        for i, it in enumerate(items):
            if it[0] == "kv":
                tb.set(eval_expr(it[1], env), eval_expr(it[2], env))
            elif i == len(items) - 1:
                for v in eval_multi(it[1], env):
                    tb.set(n, v)
                    n += 1
            else:
                tb.set(n, eval_expr(it[1], env))
                n += 1
        return tb
    if t == "function":
        return Function(e[2], e[3], e[4], env, e[5])
    if t == "ifexpr":
        for c, v in e[2]:
            if truthy(eval_expr(c, env)):
                return eval_expr(v, env)
        return eval_expr(e[3], env)
    if t == "interp":
        return e[2]
    raise LuaError("cannot evaluate " + t, e[1])


def assign(target, val, env):
    t = target[0]
    if t == "name":
        c = env.find(target[2])
        if c is None:
            GLOBALS.declare(target[2], val)
        else:
            c[0] = val
    elif t == "index":
        obj = eval_expr(target[2], env)
        key = eval_expr(target[3], env)
        if obj is None:
            raise LuaError("attempt to index nil with %r" % (key,), target[1])
        setindex(obj, key, val, target[1])
    else:
        raise LuaError("cannot assign to " + t, target[1])


def exec_block(stmts, env):
    for st in stmts:
        exec_stmt(st, env)


def exec_stmt(st, env):
    t = st[0]
    try:
        if t == "local":
            vals = eval_args(st[3], env) if st[3] else []
            for i, n in enumerate(st[2]):
                env.declare(n, vals[i] if i < len(vals) else None)
        elif t == "assign":
            vals = eval_args(st[3], env)
            for i, tg in enumerate(st[2]):
                assign(tg, vals[i] if i < len(vals) else None, env)
        elif t == "compound":
            cur = eval_expr(st[2], env)
            assign(st[2], arith(st[3], cur, eval_expr(st[4], env), st[1]), env)
        elif t == "exprstat":
            eval_multi(st[2], env)
        elif t == "localfunc":
            env.declare(st[2], None)
            env.find(st[2])[0] = Function(st[3][2], st[3][3], st[3][4], env, st[2])
        elif t == "if":
            for c, b in st[2]:
                if truthy(eval_expr(c, env)):
                    exec_block(b, Env(env))
                    return
            if st[3] is not None:
                exec_block(st[3], Env(env))
        elif t == "while":
            while truthy(eval_expr(st[2], env)):
                try:
                    exec_block(st[3], Env(env))
                except Break:
                    break
                except Continue:
                    pass
        elif t == "repeat":
            while True:
                inner = Env(env)
                try:
                    exec_block(st[2], inner)
                except Break:
                    break
                except Continue:
                    pass
                if truthy(eval_expr(st[3], inner)):
                    break
        elif t == "fornum":
            a = num(eval_expr(st[3], env), "for start")
            b = num(eval_expr(st[4], env), "for limit")
            c = num(eval_expr(st[5], env), "for step") if st[5] else 1.0
            i = a
            while (c > 0 and i <= b) or (c < 0 and i >= b):
                inner = Env(env)
                inner.declare(st[2], i)
                try:
                    exec_block(st[6], inner)
                except Break:
                    break
                except Continue:
                    pass
                i += c
        elif t == "forin":
            vals = eval_args(st[3], env)
            f = vals[0] if vals else None
            if isinstance(f, Table) and not (f.meta is not None and "__call" in f.meta.h):
                # generalized iteration: arrays in order, then the rest
                items = generic_items(f)
                for k, v in items:
                    inner = Env(env)
                    inner.declare(st[2][0], k)
                    if len(st[2]) > 1:
                        inner.declare(st[2][1], v)
                    try:
                        exec_block(st[4], inner)
                    except Break:
                        break
                    except Continue:
                        pass
                return
            s = vals[1] if len(vals) > 1 else None
            ctl = vals[2] if len(vals) > 2 else None
            while True:
                r = call(f, [s, ctl])
                if not r or r[0] is None:
                    break
                ctl = r[0]
                inner = Env(env)
                for i, n in enumerate(st[2]):
                    inner.declare(n, r[i] if i < len(r) else None)
                try:
                    exec_block(st[4], inner)
                except Break:
                    break
                except Continue:
                    pass
        elif t == "do":
            exec_block(st[2], Env(env))
        elif t == "return":
            raise Return(eval_args(st[2], env))
        elif t == "break":
            raise Break()
        elif t == "continue":
            raise Continue()
        else:
            raise LuaError("cannot execute " + t, st[1])
    except LuaError as err:
        if err.line is None:
            err.line = st[1]
        raise


def generic_items(tb):
    n = tb.length()
    out = [(float(i), tb.h[i]) for i in range(1, n + 1)]
    for k, v in list(tb.h.items()):
        if isinstance(k, int) and 1 <= k <= n:
            continue
        out.append((float(k) if isinstance(k, int) else k, v))
    return out


# ---------------------------------------------------------------- library

def B(name):
    def deco(fn):
        return Builtin(fn, name)
    return deco


def lua_format(fmt, *args):
    out, i, ai = [], 0, 0
    while i < len(fmt):
        ch = fmt[i]
        if ch != "%":
            out.append(ch)
            i += 1
            continue
        j = i + 1
        if fmt[j] == "%":
            out.append("%")
            i = j + 1
            continue
        while fmt[j] in "-+ #0123456789.":
            j += 1
        spec = fmt[i:j + 1]
        conv = fmt[j]
        a = args[ai] if ai < len(args) else None
        ai += 1
        if conv in "di":
            if a is None or isinstance(a, bool):
                raise LuaError("bad argument #%d to 'format' (number expected, got %s)" % (ai + 1, type_of(a)))
            out.append((spec[:-1] + "d") % int(num(a)))
        elif conv in "fFgGeE":
            out.append(spec % num(a))
        elif conv == "x" or conv == "X":
            out.append(spec % int(num(a)))
        elif conv == "s":
            out.append((spec[:-1] + "s") % tostr(a))
        elif conv == "q":
            out.append('"%s"' % tostr(a))
        else:
            raise LuaError("bad format " + spec)
        i = j + 1
    return "".join(out)


def setup_globals(root):
    G = GLOBALS

    def reg(name, fn):
        G.declare(name, Builtin(fn, name))

    reg("print", lambda *a: print("\t".join(tostr(x) for x in a)))
    reg("warn", lambda *a: print("WARN", *[tostr(x) for x in a]))
    reg("tostring", lambda v=None: tostr(v))

    def tonumber(v=None, base=None):
        try:
            return float(v) if base is None else float(int(v, int(base)))
        except (TypeError, ValueError):
            return None
    reg("tonumber", tonumber)
    reg("type", lambda v=None: type_of(v))
    reg("typeof", lambda v=None: getattr(v, "lua_type", None) or type_of(v))

    def lua_error(msg=None, level=None):
        raise LuaError("error: " + tostr(msg))
    reg("error", lua_error)

    def lua_assert(v=None, msg="assertion failed!", *rest):
        if not truthy(v):
            raise LuaError(tostr(msg))
        return (v, msg) + rest
    reg("assert", lua_assert)

    def ipairs_iter(t, i):
        i = int(i) + 1
        v = t.get(i)
        if v is None:
            return None
        return (float(i), v)
    reg("ipairs", lambda t: (Builtin(ipairs_iter, "ipairs_iter"), t, 0.0))

    def lua_next(t, k=None):
        items = generic_items(t)
        if k is None:
            return items[0] if items else None
        for i, (kk, _) in enumerate(items):
            if kk == k:
                return items[i + 1] if i + 1 < len(items) else None
        return None
    reg("next", lua_next)
    reg("pairs", lambda t: (Builtin(lua_next, "next"), t, None))
    reg("select", lambda n, *a: float(len(a)) if n == "#" else tuple(a[int(n) - 1:]))
    reg("rawget", lambda t, k: t.h.get(int(k) if isinstance(k, float) and k.is_integer() else k))
    reg("rawequal", lambda a, b: a is b or a == b)

    def setmeta(t, m):
        t.meta = m
        return t
    reg("setmetatable", setmeta)
    reg("getmetatable", lambda t: t.meta if isinstance(t, Table) else None)

    def pcall(f, *a):
        try:
            return tuple([True] + call(f, list(a)))
        except LuaError as e:
            return (False, str(e))
    reg("pcall", pcall)

    m = Table()
    for k, fn in {
        "floor": lambda x: float(math.floor(num(x, "math.floor"))),
        "ceil": lambda x: float(math.ceil(num(x, "math.ceil"))),
        "abs": lambda x: abs(num(x, "math.abs")),
        "sqrt": lambda x: math.sqrt(num(x, "math.sqrt")) if num(x) >= 0 else math.nan,
        "sin": lambda x: math.sin(num(x)), "cos": lambda x: math.cos(num(x)), "tan": lambda x: math.tan(num(x)),
        "atan2": lambda y, x: math.atan2(num(y), num(x)), "atan": lambda y, x=None: math.atan(num(y)) if x is None else math.atan2(num(y), num(x)),
        "exp": lambda x: math.exp(num(x)), "log": lambda x, b=None: math.log(num(x)) if b is None else math.log(num(x), num(b)),
        "pow": lambda x, y: num(x) ** num(y),
        "max": lambda *a: max(num(x, "math.max") for x in a),
        "min": lambda *a: min(num(x, "math.min") for x in a),
        "clamp": lambda x, lo, hi: max(num(lo), min(num(hi), num(x, "math.clamp"))),
        "round": lambda x: float(math.floor(num(x, "math.round") + 0.5)),
        "random": lambda a=None, b=None: random.random() if a is None else float(random.randint(1, int(a))) if b is None else float(random.randint(int(a), int(b))),
        "sign": lambda x: float((num(x) > 0) - (num(x) < 0)),
        "fmod": lambda a, b: math.fmod(num(a), num(b)),
    }.items():
        m.set(k, Builtin(fn, "math." + k))
    m.set("pi", math.pi)
    m.set("huge", math.inf)
    G.declare("math", m)

    t = Table()

    def tinsert(tb, a, b=None):
        if b is None:
            tb.set(tb.length() + 1, a)
        else:
            pos = int(a)
            n = tb.length()
            for i in range(n, pos - 1, -1):
                tb.h[i + 1] = tb.h[i]
            tb.h[pos] = b

    def tremove(tb, pos=None):
        n = tb.length()
        if n == 0:
            return None
        pos = n if pos is None else int(pos)
        v = tb.h.get(pos)
        for i in range(pos, n):
            tb.h[i] = tb.h[i + 1]
        tb.h.pop(n, None)
        return v

    def tsort(tb, cmp=None):
        n = tb.length()
        items = [tb.h[i] for i in range(1, n + 1)]
        import functools
        if cmp is None:
            items.sort(key=lambda v: v)
        else:
            def c(a, b):
                if truthy(call(cmp, [a, b])[0] if call(cmp, [a, b]) else None):
                    return -1
                if truthy(call(cmp, [b, a])[0] if call(cmp, [b, a]) else None):
                    return 1
                return 0
            items.sort(key=functools.cmp_to_key(c))
        for i, v in enumerate(items):
            tb.h[i + 1] = v

    def tconcat(tb, sep="", i=1, j=None):
        j = tb.length() if j is None else int(j)
        return sep.join(tostr(tb.h[k]) for k in range(int(i), j + 1))

    def tclear(tb):
        tb.h.clear()

    def tfind(tb, v, init=1):
        for i in range(int(init), tb.length() + 1):
            if tb.h[i] == v:
                return float(i)
        return None
    for k, fn in {"insert": tinsert, "remove": tremove, "sort": tsort, "concat": tconcat, "clear": tclear, "find": tfind,
                  "unpack": lambda tb, i=1, j=None: tuple(tb.h.get(k) for k in range(int(i), (tb.length() if j is None else int(j)) + 1))}.items():
        t.set(k, Builtin(fn, "table." + k))
    G.declare("table", t)

    G.declare("string", STRING_TABLE)
    G.declare("Random", RANDOM)
    G.declare("Color3", COLOR3)
    G.declare("Vector3", Table())
    G.declare("task", Table())
    G.declare("game", Game(root))


# strings
def s_sub(s, i=1, j=-1):
    n = len(s)
    i, j = int(i), int(j)
    if i < 0:
        i = n + i + 1
    if j < 0:
        j = n + j + 1
    i = max(i, 1)
    j = min(j, n)
    return s[i - 1:j] if i <= j else ""


def s_byte(s, i=1, j=None):
    i = int(i)
    j = i if j is None else int(j)
    return tuple(float(ord(c)) for c in s_sub(s, i, j))


STRING_LIB = {
    "format": Builtin(lua_format, "string.format"),
    "sub": Builtin(s_sub, "string.sub"),
    "byte": Builtin(s_byte, "string.byte"),
    "char": Builtin(lambda *a: "".join(chr(int(x)) for x in a), "string.char"),
    "upper": Builtin(lambda s: s.upper(), "string.upper"),
    "lower": Builtin(lambda s: s.lower(), "string.lower"),
    "len": Builtin(lambda s: float(len(s)), "string.len"),
    "rep": Builtin(lambda s, n, sep="": sep.join([s] * int(n)), "string.rep"),
}
STRING_TABLE = Table(dict(STRING_LIB))


# Roblox stubs
class RandomObj:
    lua_type = "Random"

    def __init__(self, seed):
        self.r = random.Random(int(num(seed)) if seed is not None else None)

    def lua_index(self, k):
        if k == "NextNumber":
            return Builtin(lambda self_, a=None, b=None: self.r.random() if a is None else num(a) + self.r.random() * (num(b) - num(a)), "NextNumber")
        if k == "NextInteger":
            def ni(self_, a, b):
                a, b = int(num(a)), int(num(b))
                if a > b:
                    raise LuaError("NextInteger: min > max")
                return float(self.r.randint(a, b))
            return Builtin(ni, "NextInteger")
        raise LuaError("Random has no member %r" % k)


RANDOM = Table({"new": Builtin(lambda seed=None: RandomObj(seed), "Random.new")})


class Color3Obj:
    lua_type = "Color3"
    __slots__ = ("r", "g", "b")

    def __init__(self, r, g, b):
        self.r, self.g, self.b = r, g, b

    def lua_index(self, k):
        if k == "R":
            return self.r
        if k == "G":
            return self.g
        if k == "B":
            return self.b
        if k == "Lerp":
            return Builtin(lambda s, o, a: Color3Obj(s.r + (o.r - s.r) * num(a), s.g + (o.g - s.g) * num(a), s.b + (o.b - s.b) * num(a)), "Lerp")
        raise LuaError("Color3 has no member %r" % k)


COLOR3 = Table({
    "fromRGB": Builtin(lambda r=0, g=0, b=0: Color3Obj(num(r) / 255, num(g) / 255, num(b) / 255), "Color3.fromRGB"),
    "new": Builtin(lambda r=0, g=0, b=0: Color3Obj(num(r), num(g), num(b)), "Color3.new"),
})


class Inst:
    """A folder or ModuleScript from the Rojo tree on disk."""
    lua_type = "Instance"

    def __init__(self, path, parent=None, name=None):
        self.path = path
        self.parent = parent
        self.name = name or os.path.splitext(os.path.basename(path))[0]
        self.module = None

    def child(self, name):
        if os.path.isdir(self.path):
            d = os.path.join(self.path, name)
            if os.path.isdir(d):
                return Inst(d, self, name)
            for ext in (".luau", ".lua"):
                f = os.path.join(self.path, name + ext)
                if os.path.isfile(f):
                    return Inst(f, self, name)
        return None

    def lua_index(self, k):
        if k == "Parent":
            return self.parent
        if k == "Name":
            return self.name
        if k in ("FindFirstChild", "WaitForChild"):
            return Builtin(lambda s, n, *a: self.child(n), k)
        c = self.child(k)
        if c is None:
            raise LuaError("%s is not a valid member of %s" % (k, self.name))
        return c


class Game:
    lua_type = "Instance"

    def __init__(self, root):
        self.rs = Inst(root, None, "ReplicatedStorage")
        self.rs_children = {"ArcaneOrb": Inst(os.path.join(root, "shared"), self.rs, "ArcaneOrb")}

    def lua_index(self, k):
        if k == "GetService":
            def gs(s, name):
                if name == "ReplicatedStorage":
                    return RSProxy(self)
                raise LuaError("service %s is not stubbed" % name)
            return Builtin(gs, "GetService")
        raise LuaError("game.%s is not stubbed" % k)


class RSProxy:
    lua_type = "Instance"

    def __init__(self, g):
        self.g = g

    def lua_index(self, k):
        if k in ("WaitForChild", "FindFirstChild"):
            return Builtin(lambda s, n, *a: self.g.rs_children.get(n), k)
        return self.g.rs_children.get(k)


MODULE_CACHE = {}


def require(inst):
    if not isinstance(inst, Inst) or not os.path.isfile(inst.path):
        raise LuaError("require: not a ModuleScript (%r)" % getattr(inst, "path", inst))
    if inst.path in MODULE_CACHE:
        return MODULE_CACHE[inst.path]
    v = run_file(inst.path, inst)
    MODULE_CACHE[inst.path] = v
    return v


def run_file(path, script_inst):
    src = open(path, encoding="utf-8").read()
    try:
        bld = Builder(lc.tokenize(src))
        bld.file = os.path.basename(path)
        ast = bld.chunk()
    except lc.Err as e:
        raise LuaError("%s: %s" % (path, e))
    env = Env(GLOBALS)
    env.declare("script", script_inst)
    try:
        exec_block(ast, env)
    except Return as r:
        return r.vals[0] if r.vals else None
    except LuaError as e:
        if not getattr(e, "traced", False):
            e.args = ("%s:%s: %s" % (os.path.relpath(path), e.line, e.args[0]),)
            e.traced = True
        raise
    return None


def main():
    root, test = sys.argv[1], sys.argv[2]
    setup_globals(root)
    GLOBALS.declare("require", Builtin(require, "require"))
    random.seed(1)
    try:
        run_file(test, Inst(test))
    except LuaError as e:
        print("LUA ERROR:", e)
        sys.exit(1)


if __name__ == "__main__":
    main()
