// Package loc translates the game's English strings at draw time; see
// i18n.go in the main package for how tables are loaded.
package loc

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ---------------------------------------------------------------- tables

const numTok = '\x01' // a number lifted out of a string

type trPat struct {
	re      *regexp.Regexp
	lit     string // longest literal piece, checked before the regexp
	kinds   []byte // per capture group: 'n' number, 's' text
	out     string // translation with \x02<i> placeholders for captures
	numbers int
}

// Table is one language's translations.
type Table struct {
	raw   map[string]string // fixed English text with numbers -> translation
	exact map[string]string // skeleton -> skeleton translation (\x01<i> = i-th number)
	upper map[string]string // UPPER-CASE skeleton -> translation, for strings shown in capitals
	pats  []trPat
	frags []frag
	memo  map[string]string
	// Hook translates strings no table can hold (generated names); "" = no.
	Hook func(string) string
	done map[string]bool // strings tr produced
}

type frag struct{ en, tr string }

// verbRe matches the fmt verbs the game's format strings use. The space
// flag is left out on purpose: "50% damage" is text, not a verb.
var verbRe = regexp.MustCompile(`%(\[\d+\])?[-+#0]*\d*(\.\d+)?[dfgvsqx%]`)
var digitRe = regexp.MustCompile(`\d+(?:\.\d+)?`)

// skeleton lifts the numbers out of s.
func skeleton(s string) (string, []string) {
	var nums []string
	sk := digitRe.ReplaceAllStringFunc(s, func(m string) string {
		nums = append(nums, m)
		return string(numTok)
	})
	return sk, nums
}

// arg is one fmt verb of an English key: a number (with its position
// among all the key's numbers) or a text piece (with its index among the
// key's text pieces).
type arg struct {
	text bool
	pos  int
}

// keySkeleton turns an English string into its lookup form: every number,
// from a verb or written in the text, becomes \x01 and every %s becomes
// \x00s. It also lists the verbs as arguments.
func keySkeleton(k string) (string, []arg) {
	var args []arg
	nums, texts := 0, 0
	var b strings.Builder
	last := 0
	for _, m := range verbOrDigitRe.FindAllStringIndex(k, -1) {
		b.WriteString(k[last:m[0]])
		tok := k[m[0]:m[1]]
		last = m[1]
		switch {
		case tok[0] != '%':
			b.WriteByte(numTok)
			nums++
		case tok[len(tok)-1] == '%':
			b.WriteByte('%')
		case strings.ContainsAny(tok[len(tok)-1:], "sqv"):
			b.WriteString("\x00s")
			args = append(args, arg{true, texts})
			texts++
		default:
			b.WriteByte(numTok)
			args = append(args, arg{false, nums})
			nums++
		}
	}
	b.WriteString(k[last:])
	return b.String(), args
}

var verbOrDigitRe = regexp.MustCompile(`%(\[\d+\])?[-+#0]*\d*(\.\d+)?[dfgvsqx%]|\d+(?:\.\d+)?`)

// valueTemplate rewrites a translation's verbs as placeholders that point
// at the English key's values: \x01<i> for the key's i-th number, \x02<i>
// for its i-th text piece. Numbers written out in the translation stay as
// they are (they're constants in the English too, or a translator's own).
func valueTemplate(v string, args []arg) string {
	next := 0
	return verbRe.ReplaceAllStringFunc(v, func(m string) string {
		if m[len(m)-1] == '%' {
			return "%"
		}
		idx := next
		if strings.HasPrefix(m, "%[") {
			idx, _ = strconv.Atoi(m[2:strings.IndexByte(m, ']')])
			idx--
		}
		next = idx + 1
		if idx < 0 || idx >= len(args) {
			return ""
		}
		a := args[idx]
		if a.text {
			return "\x02" + string(rune('0'+a.pos))
		}
		return "\x01" + string(rune('0'+a.pos))
	})
}

// New builds a table from English -> translation pairs.
func New(m map[string]string) *Table {
	t := &Table{raw: map[string]string{}, exact: map[string]string{}, upper: map[string]string{}, memo: map[string]string{}, done: map[string]bool{}}
	for en, v := range m {
		if v == "" || v == en {
			continue
		}
		sk, args := keySkeleton(en)
		tpl := valueTemplate(v, args)
		hasS := strings.Contains(sk, "\x00s")
		if hasS {
			t.pats = append(t.pats, makePat(sk, tpl))
			continue
		}
		if len(args) == 0 && digitRe.MatchString(en) {
			// fixed text with numbers in it ("+0.5 orb damage."): match it
			// exactly, so a different number never borrows its translation
			t.raw[en] = v
			if up := strings.ToUpper(en); up != en {
				t.raw[up] = strings.ToUpper(v)
			}
			continue
		}
		t.exact[sk] = tpl
		if up := strings.ToUpper(sk); up != sk {
			if _, ok := t.upper[up]; !ok {
				t.upper[up] = strings.ToUpper(tpl)
			}
		}
		// whole phrases that also turn up inside longer, joined strings
		if utf8.RuneCountInString(strings.TrimSpace(en)) >= 4 && ascending(tpl) {
			t.frags = append(t.frags, frag{sk, tpl})
		}
	}
	// "PASSIVE  ·  " is also "PASSIVE" when a line is split at its dots
	for sk, tpl := range t.exact {
		if st := strings.Trim(sk, edgeCut); st != sk && st != "" {
			if _, ok := t.exact[st]; !ok {
				if v := strings.Trim(tpl, edgeCut+"：，"); v != "" {
					t.exact[st] = v
					if utf8.RuneCountInString(st) >= 4 && ascending(v) {
						t.frags = append(t.frags, frag{st, v})
					}
					if up := strings.ToUpper(st); up != st {
						if _, ok := t.upper[up]; !ok {
							t.upper[up] = strings.ToUpper(v)
						}
					}
				}
			}
		}
	}
	sort.Slice(t.frags, func(i, j int) bool { return len(t.frags[i].en) > len(t.frags[j].en) })
	sort.Slice(t.pats, func(i, j int) bool { return len(t.pats[i].lit) > len(t.pats[j].lit) })
	return t
}

const edgeCut = " ·:—–-"

// ascending reports whether a template's numbers keep their order (so a
// fragment can be spliced into a longer string and renumbered).
func ascending(tpl string) bool {
	last := -1
	for i := 0; i+1 < len(tpl); i++ {
		if tpl[i] == 1 {
			n := int(tpl[i+1] - '0')
			if n != last+1 {
				return false
			}
			last = n
			i++
		}
	}
	return true
}

func makePat(sk, tpl string) trPat {
	var b strings.Builder
	b.WriteString("^")
	p := trPat{out: tpl}
	parts := strings.Split(sk, "\x00s")
	longest := ""
	for i, part := range parts {
		// numbers inside literal parts are capture groups as well
		segs := strings.Split(part, string(numTok))
		for j, seg := range segs {
			b.WriteString(regexp.QuoteMeta(seg))
			if len(strings.TrimSpace(seg)) > len(longest) {
				longest = seg
			}
			if j < len(segs)-1 {
				b.WriteString("(" + string(numTok) + ")")
				p.kinds = append(p.kinds, 'n')
				p.numbers++
			}
		}
		if i < len(parts)-1 {
			b.WriteString("(.+?)")
			p.kinds = append(p.kinds, 's')
		}
	}
	b.WriteString("$")
	p.re = regexp.MustCompile(b.String())
	p.lit = longest
	return p
}

// ---------------------------------------------------------------- lookup

// Tr translates a string about to be drawn.
func (t *Table) Tr(s string) string {
	if t == nil || s == "" {
		return s
	}
	if t.done[s] {
		return s // already translated (e.g. a wrapped line drawn again)
	}
	if v, ok := t.raw[s]; ok {
		return v
	}
	if ts := strings.TrimSpace(s); ts != s {
		if v, ok := t.raw[ts]; ok {
			i := strings.Index(s, ts)
			return s[:i] + v + s[i+len(ts):]
		}
	}
	sk, nums := skeleton(s)
	out, ok := t.memo[sk]
	if !ok {
		out = t.lookup(sk, 0)
		if len(t.memo) > 8000 {
			t.memo = map[string]string{}
		}
		t.memo[sk] = out
	}
	if out == "" {
		return s
	}
	r := fill(out, nums, nil)
	if len(t.done) > 8000 {
		t.done = map[string]bool{}
	}
	t.done[r] = true
	return r
}

// fill puts the numbers (and text pieces) back into a template.
func fill(tpl string, nums, texts []string) string {
	if !strings.ContainsAny(tpl, "\x01\x02") {
		return tpl
	}
	var b strings.Builder
	for i := 0; i < len(tpl); i++ {
		c := tpl[i]
		if (c == 1 || c == 2) && i+1 < len(tpl) {
			idx := int(tpl[i+1] - '0')
			i++
			src := nums
			if c == 2 {
				src = texts
			}
			if idx >= 0 && idx < len(src) {
				b.WriteString(src[idx])
			}
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// lookup finds a template for a skeleton; "" means leave it in English.
// Templates refer to the skeleton's own numbers by position.
func (t *Table) lookup(sk string, depth int) string {
	if v, ok := t.exact[sk]; ok {
		return v
	}
	trimmed := strings.TrimSpace(sk)
	if trimmed != sk && trimmed != "" {
		if v := t.lookup(trimmed, depth+1); v != "" {
			lead := sk[:strings.Index(sk, trimmed)]
			trail := sk[len(lead)+len(trimmed):]
			return lead + shiftNums(v, numsBefore(lead)) + trail
		}
	}
	if v, ok := t.upper[sk]; ok {
		return v
	}
	if t.Hook != nil && !strings.ContainsRune(sk, numTok) {
		if v := t.Hook(sk); v != "" {
			return v
		}
	}
	if depth > 3 {
		return ""
	}
	// formats with text pieces
	for i := range t.pats {
		p := &t.pats[i]
		if p.lit != "" && !strings.Contains(sk, p.lit) {
			continue
		}
		m := p.re.FindStringSubmatchIndex(sk)
		if m == nil {
			continue
		}
		return t.applyPat(sk, p, m, depth)
	}
	// "A  ·  B" lines
	for _, sep := range []string{"   ·   ", "  ·  ", " · ", " — "} {
		if strings.Contains(sk, sep) {
			parts := strings.Split(sk, sep)
			any := false
			n := 0
			for i, part := range parts {
				k := strings.Count(part, string(numTok))
				if v := t.lookup(part, depth+1); v != "" {
					parts[i] = shiftNums(v, n)
					any = true
				} else {
					parts[i] = shiftNums(numberPlaceholders(part), n)
				}
				n += k
			}
			if any {
				return strings.Join(parts, sep)
			}
			return ""
		}
	}
	// known phrases inside a longer joined string
	if depth <= 1 {
		return t.fragments(sk)
	}
	return ""
}

func (t *Table) applyPat(sk string, p *trPat, m []int, depth int) string {
	// numbers outside the text pieces keep their place; text pieces are
	// translated on their own and spliced in
	var texts []string
	numIdx := make([]int, 0, p.numbers) // skeleton number index of each 'n' capture
	pos := 0
	count := 0
	for g, kind := range p.kinds {
		s, e := m[2+2*g], m[3+2*g]
		count += strings.Count(sk[pos:s], string(numTok))
		piece := sk[s:e]
		if kind == 'n' {
			numIdx = append(numIdx, count)
			count++
		} else {
			v := t.lookup(piece, depth+1)
			if v == "" {
				v = numberPlaceholders(piece)
			}
			texts = append(texts, shiftNums(v, count))
			count += strings.Count(piece, string(numTok))
		}
		pos = e
	}
	// remap the pattern's own numbers to the skeleton's
	out := p.out
	var b strings.Builder
	for i := 0; i < len(out); i++ {
		if out[i] == 1 && i+1 < len(out) {
			idx := int(out[i+1] - '0')
			i++
			if idx >= 0 && idx < len(numIdx) {
				b.WriteString("\x01" + string(rune('0'+numIdx[idx])))
			}
			continue
		}
		b.WriteByte(out[i])
	}
	return fill(b.String(), placeholderList(), texts)
}

// placeholderList maps \x01<i> back to itself so fill leaves numbers for
// the final pass.
func placeholderList() []string {
	l := make([]string, 10)
	for i := range l {
		l[i] = "\x01" + string(rune('0'+i))
	}
	return l
}

// numberPlaceholders turns bare \x01 marks into indexed ones (from 0).
func numberPlaceholders(s string) string {
	n := 0
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == numTok {
			b.WriteString("\x01" + string(rune('0'+n)))
			n++
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// shiftNums offsets the number indices in a template.
func shiftNums(tpl string, by int) string {
	if by == 0 || !strings.ContainsRune(tpl, 1) {
		return tpl
	}
	b := []byte(tpl)
	for i := 0; i+1 < len(b); i++ {
		if b[i] == 1 {
			b[i+1] += byte(by)
			i++
		}
	}
	return string(b)
}

func numsBefore(s string) int { return strings.Count(s, string(numTok)) }

func isWordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'' }

// fragments swaps known phrases inside a string the code built by joining
// pieces, longest first, on word boundaries.
func (t *Table) fragments(sk string) string {
	if len(sk) > 600 {
		return ""
	}
	type span struct {
		s, e int
		v    string
	}
	var spans []span
	taken := make([]bool, len(sk))
	for _, f := range t.frags {
		from := 0
		for {
			i := strings.Index(sk[from:], f.en)
			if i < 0 {
				break
			}
			i += from
			e := i + len(f.en)
			from = e
			ok := true
			if i > 0 {
				r, _ := utf8.DecodeLastRuneInString(sk[:i])
				ok = !isWordRune(r) || !isWordRune([]rune(f.en)[0])
			}
			if ok && e < len(sk) {
				r, _ := utf8.DecodeRuneInString(sk[e:])
				fr, _ := utf8.DecodeLastRuneInString(f.en)
				ok = !isWordRune(r) || !isWordRune(fr)
			}
			for k := i; ok && k < e; k++ {
				ok = !taken[k]
			}
			if !ok {
				continue
			}
			for k := i; k < e; k++ {
				taken[k] = true
			}
			spans = append(spans, span{i, e, f.tr})
		}
	}
	if len(spans) == 0 {
		return ""
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].s < spans[j].s })
	var b strings.Builder
	pos := 0
	for _, sp := range spans {
		b.WriteString(numberPlaceholders(sk[pos:sp.s]))
		b.WriteString(sp.v)
		pos = sp.e
	}
	b.WriteString(sk[pos:])
	// fragments carry no numbers of their own, so only the gaps do; reindex
	return reindex(b.String())
}

// reindex renumbers \x01<i> placeholders left to right and turns any bare
// \x01 into the next index.
func reindex(s string) string {
	n := 0
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 1 {
			if i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '9' {
				i++
			}
			b.WriteString("\x01" + string(rune('0'+n)))
			n++
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// ---------------------------------------------------------------- line breaking

func IsCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) ||
		(r >= 0x3000 && r <= 0x303f) || (r >= 0xff00 && r <= 0xffef)
}

// noLineStart lists characters that must not begin a line in CJK text.
const noLineStart = "、。，．！？：；）」』】〕》〉ー々ゃゅょっャュョッぁぃぅぇぉァィゥェォ…・"

// breakTokens splits text into units a line may break between: words for
// spaced scripts, single characters for CJK.
func BreakTokens(para string) []string {
	var toks []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			toks = append(toks, cur.String())
			cur.Reset()
		}
	}
	for _, r := range para {
		switch {
		case r == ' ':
			flush()
			toks = append(toks, " ")
		case IsCJK(r):
			flush()
			if strings.ContainsRune(noLineStart, r) && len(toks) > 0 && toks[len(toks)-1] != " " {
				toks[len(toks)-1] += string(r)
			} else {
				toks = append(toks, string(r))
			}
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return toks
}
