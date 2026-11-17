// Command luauport turns Arcane Orb's data tables (items, curses, familiars,
// signature items, fusions, synergies, origins, keepsakes) and the Stats
// struct into Luau modules for the Roblox port.
//
// The Go code stays the single source of truth: run this again after a
// balance change and the Roblox tables follow. Closures such as
// Apply: func(s *Stats) { s.Damage += 0.5 } are translated statement by
// statement; anything it can't translate is reported and stops the run, so
// a table is never silently wrong.
//
//	go run . -src ../.. -out ../../roblox/src/shared/Data
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var (
	fset    = token.NewFileSet()
	structs = map[string][]field{} // type name -> fields in order
	consts  = map[string]string{}  // const name -> Luau literal
	vars    = map[string]*ast.CompositeLit{}
	funcs   = map[string]*ast.FuncDecl{}
	errs    []string
	helpers []string // package functions the current module calls, in first-use order
	// keepFields, when set, limits records to these fields (big structs
	// whose rendering fields mean nothing outside the Go renderer)
	keepFields map[string]bool
)

type field struct {
	name string
	typ  ast.Expr
}

func main() {
	src := flag.String("src", "../..", "game source folder (package main)")
	out := flag.String("out", "../../roblox/src/shared/Data", "output folder for the Luau modules")
	flag.Parse()
	load(*src)
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fail(err)
	}
	write(*out, "Stats", statsModule())
	write(*out, "Mods", listModule("Mods", "Every item, curse, familiar and signature item, in offer order.",
		[]string{"modList", "moreMods", "familiarMods", "sigMods", "curseList"}, "Mod"))
	write(*out, "Fusions", listModule("Fusions", "Two owned items forge a fusion.", []string{"fusions"}, "Fusion"))
	write(*out, "Synergies", listModule("Synergies", "Three items sharing a tag switch on a synergy.", []string{"synergies"}, "Synergy"))
	write(*out, "Origins", listModule("Origins", "The playable origins (heroes) and their passives.", []string{"archetypes"}, "Archetype"))
	write(*out, "Keepsakes", listModule("Keepsakes", "Keepsake upgrade tree: one charm carried into a run.", []string{"keepsakes"}, "keepsake"))
	keepFields = map[string]bool{"ID": true, "Name": true, "Subtitle": true, "Hues": true, "Floor": true, "FloorAlt": true, "Outer": true,
		"Wall": true, "Trim": true, "Fog": true, "SkyTop": true, "SkyBottom": true, "Neon": true, "PaddleBody": true, "PaddleCap": true}
	write(*out, "Worlds", listModule("Worlds", "The worlds' names and colours (the 3D scenery is rebuilt in Roblox).", []string{"worlds"}, "Biome"))
	keepFields = nil
	if len(errs) > 0 {
		fmt.Fprintln(os.Stderr, "could not translate:")
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, "  "+e)
		}
		os.Exit(1)
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }

func write(dir, name, body string) {
	p := filepath.Join(dir, name+".luau")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		fail(err)
	}
	fmt.Println("wrote", p)
}

// ------------------------------------------------------------------ loading

func load(dir string) {
	pk, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		n := fi.Name()
		return !strings.HasSuffix(n, "_test.go")
	}, 0)
	if err != nil {
		fail(err)
	}
	p := pk["main"]
	if p == nil {
		fail(fmt.Errorf("no package main in %s", dir))
	}
	names := make([]string, 0, len(p.Files))
	for n := range p.Files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		for _, d := range p.Files[n].Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil {
					funcs[d.Name.Name] = d
				}
			case *ast.GenDecl:
				switch d.Tok {
				case token.TYPE:
					for _, s := range d.Specs {
						ts := s.(*ast.TypeSpec)
						if st, ok := ts.Type.(*ast.StructType); ok {
							var fs []field
							for _, f := range st.Fields.List {
								for _, nm := range f.Names {
									fs = append(fs, field{nm.Name, f.Type})
								}
							}
							structs[ts.Name.Name] = fs
						}
					}
				case token.CONST:
					loadConsts(d)
				case token.VAR:
					for _, s := range d.Specs {
						vs := s.(*ast.ValueSpec)
						for i, nm := range vs.Names {
							if i < len(vs.Values) {
								if cl, ok := vs.Values[i].(*ast.CompositeLit); ok {
									vars[nm.Name] = cl
								}
							}
						}
					}
				}
			}
		}
	}
}

// loadConsts evaluates int/float/string constants, iota included.
func loadConsts(d *ast.GenDecl) {
	var last []ast.Expr
	for iota, s := range d.Specs {
		vs := s.(*ast.ValueSpec)
		vals := vs.Values
		if len(vals) == 0 {
			vals = last
		} else {
			last = vals
		}
		for i, nm := range vs.Names {
			if i >= len(vals) {
				continue
			}
			if v, ok := evalConst(vals[i], iota); ok {
				consts[nm.Name] = v.ExactString()
				if v.Kind() == constant.String {
					consts[nm.Name] = strconv.Quote(constant.StringVal(v))
				}
			}
		}
	}
}

func evalConst(e ast.Expr, iota int) (constant.Value, bool) {
	switch e := e.(type) {
	case *ast.BasicLit:
		return constant.MakeFromLiteral(e.Value, e.Kind, 0), true
	case *ast.Ident:
		if e.Name == "iota" {
			return constant.MakeInt64(int64(iota)), true
		}
		if s, ok := consts[e.Name]; ok {
			if v, err := strconv.ParseFloat(s, 64); err == nil {
				if !strings.ContainsAny(s, ".e") {
					return constant.MakeInt64(int64(v)), true
				}
				return constant.MakeFloat64(v), true
			}
		}
	case *ast.ParenExpr:
		return evalConst(e.X, iota)
	case *ast.UnaryExpr:
		if x, ok := evalConst(e.X, iota); ok {
			return constant.UnaryOp(e.Op, x, 0), true
		}
	case *ast.BinaryExpr:
		x, ok1 := evalConst(e.X, iota)
		y, ok2 := evalConst(e.Y, iota)
		if ok1 && ok2 {
			if e.Op == token.SHL || e.Op == token.SHR {
				n, _ := constant.Uint64Val(y)
				return constant.Shift(x, e.Op, uint(n)), true
			}
			return constant.BinaryOp(x, e.Op, y), true
		}
	case *ast.CallExpr: // typed constant: NodeType(3)
		if len(e.Args) == 1 {
			return evalConst(e.Args[0], iota)
		}
	}
	return nil, false
}

// ------------------------------------------------------------------ modules

func header(name, doc string) string {
	return "--!strict\n-- " + name + ".luau: " + doc + "\n-- GENERATED by tools/luauport from the Go source. Do not edit by hand:\n-- change the Go table and run `go run .` in tools/luauport again.\n\n"
}

func statsModule() string {
	fs := structs["Stats"]
	if fs == nil {
		fail(fmt.Errorf("type Stats not found"))
	}
	var b strings.Builder
	b.WriteString(header("Stats", "every number a run's build can change, and the base values."))
	b.WriteString("export type Stats = {\n")
	for _, f := range fs {
		fmt.Fprintf(&b, "\t%s: %s,\n", f.name, luauType(f.typ))
	}
	b.WriteString("}\n\nlocal Stats = {}\n\n")
	// base values from baseStats()
	base := map[string]string{}
	if fd := funcs["baseStats"]; fd != nil {
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			if cl, ok := n.(*ast.CompositeLit); ok {
				for _, el := range cl.Elts {
					if kv, ok := el.(*ast.KeyValueExpr); ok {
						base[kv.Key.(*ast.Ident).Name] = expr(kv.Value, nil)
					}
				}
				return false
			}
			return true
		})
	} else {
		errs = append(errs, "baseStats() not found")
	}
	b.WriteString("-- the starting build before origin, items and blessings (run.go baseStats)\nfunction Stats.base(): Stats\n\treturn {\n")
	for _, f := range fs {
		v, ok := base[f.name]
		if !ok {
			v = zero(f.typ)
		}
		fmt.Fprintf(&b, "\t\t%s = %s,\n", f.name, v)
	}
	b.WriteString("\t}\nend\n\nreturn Stats\n")
	return b.String()
}

func luauType(t ast.Expr) string {
	switch t := t.(type) {
	case *ast.Ident:
		switch t.Name {
		case "float64", "float32", "int", "int64", "int32", "uint8":
			return "number"
		case "bool":
			return "boolean"
		case "string":
			return "string"
		}
	case *ast.ArrayType:
		return "{" + luauType(t.Elt) + "}"
	case *ast.FuncType:
		return "any"
	}
	return "any"
}

func zero(t ast.Expr) string {
	switch luauType(t) {
	case "number":
		return "0"
	case "boolean":
		return "false"
	case "string":
		return `""`
	}
	return "nil"
}

func listModule(name, doc string, lists []string, typ string) string {
	var b strings.Builder
	b.WriteString(header(name, doc))
	b.WriteString("local Stats = require(script.Parent.Stats)\ntype Stats = Stats.Stats\n\n")
	helpers = nil
	var body strings.Builder
	for _, ln := range lists {
		cl := vars[ln]
		if cl == nil {
			errs = append(errs, "table "+ln+" not found")
			continue
		}
		fmt.Fprintf(&body, "\t-- %s\n", ln)
		for _, el := range cl.Elts {
			body.WriteString("\t" + record(el, typ, 1) + ",\n")
		}
	}
	// helpers the closures call, translated too (they may call more helpers)
	var hs strings.Builder
	for i := 0; i < len(helpers); i++ {
		fd := funcs[helpers[i]]
		var ps []string
		for _, f := range fd.Type.Params.List {
			for _, n := range f.Names {
				ps = append(ps, n.Name+": "+paramType(f.Type))
			}
		}
		ret := ""
		if fd.Type.Results != nil && len(fd.Type.Results.List) > 0 {
			ret = ": " + luauType(fd.Type.Results.List[0].Type)
		}
		var one strings.Builder
		fmt.Fprintf(&one, "local function %s(%s)%s\n", fd.Name.Name, strings.Join(ps, ", "), ret)
		block(&one, fd.Body.List, 1)
		one.WriteString("end\n\n")
		hs.WriteString(one.String())
	}
	b.WriteString(hs.String())
	b.WriteString("local list = {\n")
	b.WriteString(body.String())
	b.WriteString("}\n\nlocal byId = {}\nfor _, rec in list do\n\tif rec.ID then\n\t\tbyId[rec.ID] = rec\n\tend\nend\n\nreturn {\n\tlist = list,\n\tbyId = byId,\n}\n")
	_ = Stats
	return b.String()
}

var Stats = 0

// record writes one struct literal as a Luau table.
func record(e ast.Expr, typ string, depth int) string {
	if u, ok := e.(*ast.UnaryExpr); ok && u.Op == token.AND {
		e = u.X
	}
	cl, ok := e.(*ast.CompositeLit)
	if !ok {
		return expr(e, nil)
	}
	fs := structs[typ]
	ind := strings.Repeat("\t", depth+1)
	var parts []string
	for i, el := range cl.Elts {
		var key string
		var val ast.Expr
		if kv, ok := el.(*ast.KeyValueExpr); ok {
			key, val = kv.Key.(*ast.Ident).Name, kv.Value
		} else {
			if i >= len(fs) {
				errs = append(errs, pos(el)+": positional field past the end of "+typ)
				continue
			}
			key, val = fs[i].name, el
		}
		if key != "" && key[0] >= 'a' && key[0] <= 'z' {
			continue // unexported helpers (legacy unlock checks) stay in Go
		}
		if keepFields != nil && !keepFields[key] {
			continue
		}
		parts = append(parts, ind+key+" = "+valueIndent(val, depth+1))
	}
	if len(parts) == 0 {
		return "{}"
	}
	return "{\n" + strings.Join(parts, ",\n") + ",\n" + strings.Repeat("\t", depth) + "}"
}

func valueIndent(e ast.Expr, depth int) string {
	if fl, ok := e.(*ast.FuncLit); ok {
		return funcLit(fl, depth)
	}
	return expr(e, nil)
}

func pos(n ast.Node) string {
	p := fset.Position(n.Pos())
	return fmt.Sprintf("%s:%d", filepath.Base(p.Filename), p.Line)
}

// ------------------------------------------------------------------ code

func funcLit(fl *ast.FuncLit, depth int) string {
	var ps []string
	for _, f := range fl.Type.Params.List {
		for _, n := range f.Names {
			ps = append(ps, n.Name+": "+paramType(f.Type))
		}
	}
	ret := ""
	if fl.Type.Results != nil && len(fl.Type.Results.List) > 0 {
		ret = ": " + luauType(fl.Type.Results.List[0].Type)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "function(%s)%s\n", strings.Join(ps, ", "), ret)
	block(&b, fl.Body.List, depth+1)
	b.WriteString(strings.Repeat("\t", depth) + "end")
	return b.String()
}

func paramType(t ast.Expr) string {
	if s, ok := t.(*ast.StarExpr); ok {
		if id, ok := s.X.(*ast.Ident); ok && id.Name == "Stats" {
			return "Stats"
		}
		return "any"
	}
	return luauType(t)
}

func block(b *strings.Builder, list []ast.Stmt, depth int) {
	for _, s := range list {
		stmt(b, s, depth)
	}
}

func stmt(b *strings.Builder, s ast.Stmt, depth int) {
	ind := strings.Repeat("\t", depth)
	switch s := s.(type) {
	case *ast.AssignStmt:
		if len(s.Lhs) != len(s.Rhs) {
			errs = append(errs, pos(s)+": multi-value assignment")
			return
		}
		for i := range s.Lhs {
			l, r := expr(s.Lhs[i], nil), expr(s.Rhs[i], nil)
			switch s.Tok {
			case token.DEFINE:
				fmt.Fprintf(b, "%slocal %s = %s\n", ind, l, r)
			case token.ASSIGN:
				fmt.Fprintf(b, "%s%s = %s\n", ind, l, r)
			case token.ADD_ASSIGN, token.SUB_ASSIGN, token.MUL_ASSIGN, token.QUO_ASSIGN:
				fmt.Fprintf(b, "%s%s %s %s\n", ind, l, s.Tok.String(), r)
			default:
				errs = append(errs, pos(s)+": assignment "+s.Tok.String())
			}
		}
	case *ast.IncDecStmt:
		op := "+="
		if s.Tok == token.DEC {
			op = "-="
		}
		fmt.Fprintf(b, "%s%s %s 1\n", ind, expr(s.X, nil), op)
	case *ast.ReturnStmt:
		if len(s.Results) == 0 {
			b.WriteString(ind + "return\n")
		} else {
			fmt.Fprintf(b, "%sreturn %s\n", ind, expr(s.Results[0], nil))
		}
	case *ast.IfStmt:
		if s.Init != nil {
			stmt(b, s.Init, depth)
		}
		fmt.Fprintf(b, "%sif %s then\n", ind, expr(s.Cond, nil))
		block(b, s.Body.List, depth+1)
		for s.Else != nil {
			if ei, ok := s.Else.(*ast.IfStmt); ok {
				fmt.Fprintf(b, "%selseif %s then\n", ind, expr(ei.Cond, nil))
				block(b, ei.Body.List, depth+1)
				s = ei
				continue
			}
			b.WriteString(ind + "else\n")
			block(b, s.Else.(*ast.BlockStmt).List, depth+1)
			break
		}
		b.WriteString(ind + "end\n")
	case *ast.SwitchStmt:
		tag := expr(s.Tag, nil)
		first := true
		var def *ast.CaseClause
		for _, c := range s.Body.List {
			cc := c.(*ast.CaseClause)
			if cc.List == nil {
				def = cc
				continue
			}
			var conds []string
			for _, v := range cc.List {
				conds = append(conds, tag+" == "+expr(v, nil))
			}
			kw := "elseif"
			if first {
				kw, first = "if", false
			}
			fmt.Fprintf(b, "%s%s %s then\n", ind, kw, strings.Join(conds, " or "))
			block(b, cc.Body, depth+1)
		}
		if def != nil {
			if first {
				block(b, def.Body, depth)
			} else {
				b.WriteString(ind + "else\n")
				block(b, def.Body, depth+1)
			}
		}
		if !first {
			b.WriteString(ind + "end\n")
		}
	case *ast.ExprStmt:
		fmt.Fprintf(b, "%s%s\n", ind, expr(s.X, nil))
	case *ast.BlockStmt:
		b.WriteString(ind + "do\n")
		block(b, s.List, depth+1)
		b.WriteString(ind + "end\n")
	default:
		errs = append(errs, fmt.Sprintf("%s: statement %T", pos(s), s))
	}
}

var binOps = map[token.Token]string{
	token.ADD: "+", token.SUB: "-", token.MUL: "*", token.QUO: "/", token.REM: "%",
	token.LAND: "and", token.LOR: "or", token.EQL: "==", token.NEQ: "~=",
	token.LSS: "<", token.GTR: ">", token.LEQ: "<=", token.GEQ: ">=",
}

var mathFns = map[string]string{
	"Max": "math.max", "Min": "math.min", "Abs": "math.abs", "Floor": "math.floor", "Ceil": "math.ceil",
	"Sqrt": "math.sqrt", "Pow": "math.pow", "Sin": "math.sin", "Cos": "math.cos", "Round": "math.round",
}

func expr(e ast.Expr, _ any) string {
	switch e := e.(type) {
	case nil:
		return "nil"
	case *ast.BasicLit:
		switch e.Kind {
		case token.STRING:
			s, err := strconv.Unquote(e.Value)
			if err != nil {
				errs = append(errs, pos(e)+": string")
			}
			return luaString(s)
		case token.CHAR:
			return e.Value
		}
		return strings.TrimSuffix(e.Value, ".")
	case *ast.Ident:
		switch e.Name {
		case "true", "false":
			return e.Name
		case "nil":
			return "nil"
		}
		if v, ok := consts[e.Name]; ok {
			return v
		}
		return e.Name
	case *ast.ParenExpr:
		return "(" + expr(e.X, nil) + ")"
	case *ast.SelectorExpr:
		if id, ok := e.X.(*ast.Ident); ok && id.Name == "math" {
			switch e.Sel.Name {
			case "Pi":
				return "math.pi"
			case "MaxInt", "MaxInt64":
				return "math.huge"
			}
		}
		return expr(e.X, nil) + "." + e.Sel.Name
	case *ast.UnaryExpr:
		switch e.Op {
		case token.SUB:
			return "-" + expr(e.X, nil)
		case token.NOT:
			return "not " + expr(e.X, nil)
		case token.AND:
			return expr(e.X, nil)
		}
	case *ast.BinaryExpr:
		if op, ok := binOps[e.Op]; ok {
			return expr(e.X, nil) + " " + op + " " + expr(e.Y, nil)
		}
	case *ast.CallExpr:
		return call(e)
	case *ast.CompositeLit:
		if sel, ok := e.Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "RGBA" {
			return color(e.Elts)
		}
		if at, ok := e.Type.(*ast.ArrayType); ok {
			var xs []string
			for _, el := range e.Elts {
				// elided element types: [6]color.RGBA{{255, 90, 200, 255}, ...}
				if cl, ok := el.(*ast.CompositeLit); ok && cl.Type == nil {
					if sel, ok := at.Elt.(*ast.SelectorExpr); ok && sel.Sel.Name == "RGBA" {
						xs = append(xs, color(cl.Elts))
						continue
					}
					if id, ok := at.Elt.(*ast.Ident); ok {
						xs = append(xs, record(cl, id.Name, 2))
						continue
					}
				}
				xs = append(xs, expr(el, nil))
			}
			return "{" + strings.Join(xs, ", ") + "}"
		}
		if id, ok := e.Type.(*ast.Ident); ok {
			return record(e, id.Name, 2)
		}
	case *ast.FuncLit:
		return funcLit(e, 2)
	case *ast.IndexExpr:
		// Go slices count from 0, Luau tables from 1
		if cl, ok := e.X.(*ast.CompositeLit); ok {
			if _, ok := cl.Type.(*ast.ArrayType); ok {
				return "(" + expr(e.X, nil) + ")[" + expr(e.Index, nil) + " + 1]"
			}
		}
		if bl, ok := e.Index.(*ast.BasicLit); ok && bl.Kind == token.STRING {
			return expr(e.X, nil) + "[" + expr(e.Index, nil) + "]"
		}
		errs = append(errs, pos(e)+": index into a slice of unknown kind (port by hand)")
	}
	errs = append(errs, fmt.Sprintf("%s: expression %T", pos(e), e))
	return "nil --[[?]]"
}

func call(e *ast.CallExpr) string {
	args := func() []string {
		var xs []string
		for _, a := range e.Args {
			xs = append(xs, expr(a, nil))
		}
		return xs
	}
	switch f := e.Fun.(type) {
	case *ast.Ident:
		switch f.Name {
		case "float64", "float32":
			return expr(e.Args[0], nil)
		case "int", "int64":
			return "math.floor(" + expr(e.Args[0], nil) + ")"
		case "min":
			return "math.min(" + strings.Join(args(), ", ") + ")"
		case "max":
			return "math.max(" + strings.Join(args(), ", ") + ")"
		case "tr":
			return expr(e.Args[0], nil) // translation happens on the Roblox side
		case "rgba":
			return color(e.Args)
		}
		if fd := funcs[f.Name]; fd != nil {
			found := false
			for _, h := range helpers {
				found = found || h == f.Name
			}
			if !found {
				helpers = append(helpers, f.Name)
			}
			return f.Name + "(" + strings.Join(args(), ", ") + ")"
		}
	case *ast.SelectorExpr:
		if id, ok := f.X.(*ast.Ident); ok {
			switch id.Name {
			case "math":
				if fn, ok := mathFns[f.Sel.Name]; ok {
					return fn + "(" + strings.Join(args(), ", ") + ")"
				}
			case "fmt":
				if f.Sel.Name == "Sprintf" {
					return "string.format(" + strings.Join(args(), ", ") + ")"
				}
			}
		}
		// a method call on a value: s.familiarKinds() -> s:familiarKinds()
		return expr(f.X, nil) + ":" + f.Sel.Name + "(" + strings.Join(args(), ", ") + ")"
	}
	errs = append(errs, pos(e)+": call "+fmt.Sprintf("%T", e.Fun))
	return "nil --[[?]]"
}

func color(elts []ast.Expr) string {
	if len(elts) < 3 {
		errs = append(errs, "colour with fewer than 3 parts")
		return "Color3.new()"
	}
	return fmt.Sprintf("Color3.fromRGB(%s, %s, %s)", expr(elts[0], nil), expr(elts[1], nil), expr(elts[2], nil))
}

func luaString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
