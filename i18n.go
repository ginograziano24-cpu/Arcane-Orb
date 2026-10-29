package main

import (
	"embed"
	"encoding/json"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"orb/loc"
)

// Localisation. Every piece of text the game draws passes through tr()
// on its way to the screen (ui.go), so the English source strings stay
// where they are in the code and each language is a table from English to
// the translation (assets/i18n/<code>.json, built from the strings that
// tools/i18n/extract.go finds).
//
// Strings built at run time are handled without touching their call sites:
//   - numbers are lifted out first, so "WAVE 2 / 3" matches "WAVE %d / %d"
//   - formats with %s become patterns whose pieces are translated in turn
//   - "Rampage  ·  Firestorm" style lines translate part by part
//   - leftover sentences made by joining strings swap in the known parts
// Translations may reorder numbers and pieces with %[n]d / %[n]s.

//go:embed assets/i18n/*.json
var i18nFS embed.FS

// subset CJK fonts (tools/i18n/subset.py), only the glyphs the tables use
//
//go:embed assets/fonts/cjk
var cjkFS embed.FS

type language struct {
	Code, Name string // Name is written in the language itself
}

var languages = []language{
	{"en", "English"},
	{"zh", "简体中文"},
	{"ja", "日本語"},
	{"ru", "Русский"},
	{"es", "Español"},
	{"pt", "Português (Brasil)"},
	{"de", "Deutsch"},
}

// curLang is the active language code ("en" or "" for English).
var curLang = "en"
var curTable *loc.Table

func langIndex(code string) int {
	for i, l := range languages {
		if l.Code == code {
			return i
		}
	}
	return 0
}

// setLanguage switches the text of the whole game.
func setLanguage(code string) {
	if code == "" || langIndex(code) == 0 && code != "en" {
		code = "en"
	}
	curLang = code
	curTable = nil
	if code == "en" {
		return
	}
	data, err := i18nFS.ReadFile("assets/i18n/" + code + ".json")
	if err != nil {
		return
	}
	m := map[string]string{}
	if json.Unmarshal(data, &m) != nil {
		return
	}
	curTable = loc.New(m)
	curTable.Hook = chamberName
}

// cjk reports whether the active language is written without spaces.
func cjk() bool { return curLang == "zh" || curLang == "ja" }

// setLang switches language and the fonts it needs.
func (g *Game) setLang(code string) {
	setLanguage(code)
	g.settings.Lang = curLang
	g.ui.setScripts(curLang)
	g.titleCache = map[string]*ebiten.Image{}
}

// detectLanguage picks the first-run language: the one chosen for the game
// in Steam, else the system's, else English.
func detectLanguage() string {
	switch steamLanguage() {
	case "schinese":
		return "zh"
	case "japanese":
		return "ja"
	case "russian":
		return "ru"
	case "spanish", "latam":
		return "es"
	case "brazilian", "portuguese":
		return "pt"
	case "german":
		return "de"
	case "english":
		return "en"
	}
	loc := strings.ToLower(osLanguage())
	for _, l := range languages {
		if strings.HasPrefix(loc, l.Code) {
			return l.Code
		}
	}
	return "en"
}

// tr translates a string about to be drawn.
func tr(s string) string { return curTable.Tr(s) }

func breakTokens(para string) []string { return loc.BreakTokens(para) }
