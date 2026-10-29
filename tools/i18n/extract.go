//go:build ignore

// extract lists the English strings in the game's source that a player can
// see, for translation. Run from the repo root:
//
//	go run tools/i18n/extract.go > tools/i18n/strings.txt
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

var skipFiles = map[string]bool{"balance.go": true, "capture_on.go": true, "capture_off.go": true, "debug_on.go": true, "debug_off.go": true,
	"debug_js.go": true, "sfx.go": true, "trailer_audio.go": true, "i18n.go": true, "crash_other.go": true,
	"steam_windows.go": true, "steam_lb_windows.go": true, "steam_other.go": true, "steam_net_other.go": true, "presence.go": true,
	"orb.go": true, "post.go": true, "skyfx.go": true, "sdf.go": true, "duel_net_native.go": true, "duel_net_js.go": true, "i18n_names.go": true}

// calls whose string arguments are never shown to the player
var skipCalls = map[string]bool{"Errorf": true, "Printf": true, "Println": true, "Fprintf": true, "Fprintln": true, "panic": true, "Play": true,
	"PlayVol": true, "PlayKeyed": true, "unlock": true, "coach": true, "anim": true, "navReg": true, "Getenv": true, "ReadFile": true, "duelLog": true,
	"hudPortrait": true, "portrait": true, "cached": true, "glyphImage": true, "MustCompile": true, "Contains": true, "HasPrefix": true, "Split": true,
	"Trim": true, "TrimSpace": true, "Join": true, "Replace": true, "ReplaceAll": true, "Index": true, "Setenv": true, "hashStr": true, "Mix": true}

var ident = regexp.MustCompile(`^[a-z0-9_.\-/#:%]+$`)
var junk = regexp.MustCompile(`^[A-Z0-9]{20,}$|^Btn |^A D$|^%[-0-9.a-z]*$|^SteamAPI_|^====|^MessageBoxW$|^//kage|^json:|^[a-z]+[A-Z][A-Za-z0-9]*$|^__|^\{[a-z]+\}$`)

func visible(s string) bool {
	if len(s) < 2 || !strings.ContainsFunc(s, unicode.IsLetter) || junk.MatchString(strings.TrimSpace(s)) {
		return false
	}
	if ident.MatchString(s) {
		return false // ids, sound names, paths
	}
	for _, p := range []string{"assets/", "http", ".png", ".json", ".txt", "ACH_", "#Status", "Arcane Orb/", "aorb"} {
		if strings.Contains(s, p) {
			return false
		}
	}
	return true
}

func main() {
	fs := token.NewFileSet()
	seen := map[string]bool{}
	files, _ := filepath.Glob("*.go")
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") || skipFiles[f] {
			continue
		}
		af, err := parser.ParseFile(fs, f, nil, 0)
		if err != nil {
			panic(err)
		}
		var walk func(n ast.Node, skip bool)
		walk = func(n ast.Node, skip bool) {
			ast.Inspect(n, func(c ast.Node) bool {
				switch x := c.(type) {
				case *ast.ImportSpec:
					return false
				case *ast.CallExpr:
					name := ""
					switch fn := x.Fun.(type) {
					case *ast.Ident:
						name = fn.Name
					case *ast.SelectorExpr:
						name = fn.Sel.Name
					}
					if c != n {
						walk(x.Fun, skip)
						for _, a := range x.Args {
							walk(a, skip || skipCalls[name])
						}
						return false
					}
				case *ast.IndexExpr:
					if c != n {
						walk(x.X, skip)
						walk(x.Index, true) // map lookups by key
						return false
					}
				case *ast.KeyValueExpr:
					if c != n {
						if _, isLit := x.Key.(*ast.BasicLit); isLit {
							walk(x.Value, skip) // map keys are ids
							return false
						}
					}
				case *ast.BasicLit:
					if x.Kind != token.STRING || skip {
						return true
					}
					s, err := strconv.Unquote(x.Value)
					if err == nil && visible(s) {
						seen[s] = true
					}
				}
				return true
			})
		}
		walk(af, false)
	}
	var out []string
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	for _, s := range out {
		fmt.Println(strconv.Quote(s))
	}
	fmt.Fprintln(os.Stderr, len(out), "strings")
}
