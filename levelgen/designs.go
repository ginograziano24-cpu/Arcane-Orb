package levelgen

import (
	"math"
	"math/rand"
	"strings"
)

// Hand-drawn showcase layouts. Each design is drawn either as the left half
// (7 columns, mirrored) or full width (14 columns).
//
// Art legend:
//   a-f  brick in palette colour 0-5 (1 high)     A-F  same colour, raised (2 high)
//   #    steel        *  explosive   $  crystal   ?  mystery    @  elite
//   %    armoured     z  chain       s  split     r  regen      h  shield
//   x    multiplier   .  empty
// Tall (optional) gives an exact stack height 1-4 per cell, overriding case.

type Design struct {
	Name, Cat string
	Half      bool
	Art       []string
	Tall      []string
}

var Designs = []Design{
	// ------------------------------------------------ castles & buildings
	{Name: "Castle Keep", Cat: "castle", Half: true,
		Art: []string{
			"a.a.a.b",
			"aaaaa.b",
			"aaaaabb",
			"a$aaabb",
			"ccccccc",
			"cccc*c.",
			"cccccc.",
		},
		Tall: []string{
			"4.4.4.3",
			"44444.3",
			"4444433",
			"4444433",
			"2222222",
			"222222.",
			"222222.",
		}},
	{Name: "Cathedral", Cat: "castle", Half: true,
		Art: []string{
			"......a",
			".....ab",
			"a....ab",
			"a...abb",
			"aa..bc$",
			"aa.bcc$",
			"aaabccc",
			"aaabccc",
			"ddddddd",
			"ddddd..",
		},
		Tall: []string{
			"......4",
			".....44",
			"3....44",
			"3...444",
			"33..333",
			"33.3333",
			"3333333",
			"3333333",
			"2222222",
			"22222..",
		}},
	{Name: "Sun Ziggurat", Cat: "castle", Half: true,
		Art: []string{
			"......$",
			".....aa",
			"....bbb",
			"...cccc",
			"..ddddd",
			".eeeeee",
			"fffffff",
		},
		Tall: []string{
			"......4",
			".....44",
			"....444",
			"...3333",
			"..33333",
			".222222",
			"1111111",
		}},
	{Name: "Old Stone Bridge", Cat: "castle", Half: true,
		Art: []string{
			"aaaaaaa",
			"aaaaaaa",
			"bb...bb",
			"bb...bb",
			"bb...bb",
			"zcccczc",
		},
		Tall: []string{
			"3333333",
			"3333333",
			"22...22",
			"22...22",
			"22...22",
			"1111111",
		}},
	{Name: "Pagoda", Cat: "castle", Half: true,
		Art: []string{
			"......a",
			"....aaa",
			".bbbbbb",
			"...cccc",
			".dddddd",
			"...eeee",
			"fffffff",
			"...$fff",
		},
		Tall: []string{
			"......4",
			"....444",
			".333333",
			"...3333",
			".222222",
			"...2222",
			"1111111",
			"...1111",
		}},
	{Name: "Twin Watchtowers", Cat: "castle", Half: true,
		Art: []string{
			"a.a....",
			"aaa....",
			"a$a....",
			"aaa.b..",
			"aaabbbb",
			"aaab*bb",
			"aaabbbb",
			"cc.....",
		},
		Tall: []string{
			"4.4....",
			"444....",
			"444....",
			"444.2..",
			"4442222",
			"4442222",
			"4442222",
			"11.....",
		}},
	// ------------------------------------------------ animals
	{Name: "Cat", Cat: "animal", Half: true,
		Art: []string{
			"..A....",
			".AAA...",
			".aaaaaa",
			"zaaaaaa",
			"aaCCaaa",
			"aaCCaaa",
			"zaaaaaD",
			"aaaaeaa",
			".aaaaee",
			"..aaaaa",
		}},
	{Name: "Owl", Cat: "animal", Half: true,
		Art: []string{
			"A......",
			"AA.....",
			"aaaaaaa",
			"abbbaaa",
			"abCbaaa",
			"abbbaaD",
			"aaaaaaD",
			"aeaeaea",
			"aaeaeae",
			".aaaaaa",
			"...f...",
		}},
	{Name: "Fish", Cat: "animal",
		Art: []string{
			"$...aaaa......",
			"..aaaaaaaa...b",
			".aCaaaaaaaa.bb",
			"aaaacaacaaabbb",
			".aaaaaaaaaa.bb",
			"..aaaaaaaa...b",
			"....aaaa......",
		}},
	{Name: "Butterfly", Cat: "animal", Half: true,
		Art: []string{
			"....z..",
			"Aa.....",
			"aaa....",
			"abaa..C",
			"abbaa.C",
			"aabaa.C",
			".aaa..C",
			"..dd..C",
			".dddd.C",
			".deed.C",
			"..dd...",
		}},
	{Name: "Crab", Cat: "animal", Half: true,
		Art: []string{
			"AA.A...",
			"AAAA...",
			".AA..C.",
			"..aaaaa",
			".aaaaaa",
			"zaaaaaa",
			"a.aaaaa",
			"a.a.a..",
		}},
	{Name: "Turtle", Cat: "animal", Half: true,
		Art: []string{
			"......a",
			".a..bbb",
			"a..BCBC",
			"..BCBCB",
			"..CBCBC",
			"..BCBCB",
			"a..BCBC",
			"aa..bbb",
			"......b",
		},
		Tall: []string{
			"......1",
			".1..111",
			"1..2223",
			"..22333",
			"..23333",
			"..22333",
			"1..2223",
			"11..111",
			"......1",
		}},
	{Name: "Bat", Cat: "animal", Half: true,
		Art: []string{
			"a....A.",
			"aa...aa",
			"aaa.aaa",
			"aaaaaaa",
			"aaaaaCa",
			"aaaaaaa",
			".a.aaaa",
			"...a.aa",
			"......a",
		}},
	{Name: "Snail", Cat: "animal",
		Art: []string{
			"........a..a..",
			".......Aa.Aa..",
			"..bbbb...aa...",
			".bccccb..aa...",
			"bcbbbbcb.aa...",
			"bcbccbcb.aa...",
			"bcbbcbcbaaa...",
			".bccccbaaa....",
			"aaaaaaaaaa....",
		},
		Tall: []string{
			"........1..1..",
			".......21.21..",
			"..2222...11...",
			".233332..11...",
			"23444432.11...",
			"23433432.11...",
			"23443432111...",
			".2333321111...",
			"1111111111....",
		}},
	// ------------------------------------------------ faces
	{Name: "Grin", Cat: "face", Half: true,
		Art: []string{
			"...aaaa",
			".aaaaaa",
			"aaaaaaa",
			"aaBBaaa",
			"aaBBaaa",
			"aaaaaaa",
			"aCaaaaa",
			"aaCaaaa",
			"aaaCCCC",
			".aaaaaa",
			"...aaaa",
		}},
	{Name: "Grinning Skull", Cat: "face", Half: true,
		Art: []string{
			"...aaaa",
			".aaaaaa",
			"aaaaaaa",
			"a..aaaa",
			"a...aaa",
			"a...aaa",
			"aaaaaa.",
			".aaaaaa",
			"..a.a.a",
			"..aaaaa",
		}},
	{Name: "Robot", Cat: "face", Half: true,
		Art: []string{
			"......z",
			"...aaaa",
			"..aaaaa",
			"..aBBaa",
			"..aBBaa",
			"..aaaaa",
			"..acdcd",
			"..aaaaa",
			"....aaa",
			".eeeeee",
			"eeeeeee",
			"e.eeee*",
		}},
	{Name: "Visitor", Cat: "face", Half: true,
		Art: []string{
			"...aaaa",
			"..aaaaa",
			".aaaaaa",
			"aBBBaaa",
			"aaBBBaa",
			".aaaaaa",
			"..aaaaa",
			"...aaaa",
			"....aaa",
			"......a",
		}},
	{Name: "Lantern Gourd", Cat: "face", Half: true,
		Art: []string{
			"......B",
			"...aaaa",
			".aaaaaa",
			"aa.aaaa",
			"a...aaa",
			"aaaaa.a",
			"aaaaaaa",
			"a.a.a.a",
			"aa.a.a.",
			".aaaaaa",
			"...aaaa",
		}},
	{Name: "Stone Mask", Cat: "face", Half: true,
		Art: []string{
			"..aaaaa",
			".aaaaaa",
			"aAAAAaa",
			"a.%..aa",
			"aAAAAaa",
			"aaaaaBa",
			"aaaaaBa",
			".aaaaaa",
			".aCCCCC",
			"..aaaaa",
		},
		Tall: []string{
			"..22222",
			".222222",
			"2333322",
			"2.3..22",
			"2333322",
			"2222232",
			"2222232",
			".222222",
			".233333",
			"..22222",
		}},
	// ------------------------------------------------ objects & patterns
	{Name: "Heart", Cat: "object", Half: true,
		Art: []string{
			".aaa...",
			"aa$aa..",
			"aaAaaaa",
			"aAAAaaa",
			"aaAaaaa",
			".aaaaaa",
			"..aaaaa",
			"...aaaa",
			"....aaa",
			".....aa",
			"......a",
		}},
	{Name: "Star", Cat: "object", Half: true,
		Art: []string{
			"......A",
			".....AA",
			".....aa",
			"aaaaaaa",
			".aaaaaa",
			"..aaaaa",
			"...aaaa",
			"..aaa.a",
			".aa....",
			"aa.....",
		}},
	{Name: "Sword", Cat: "object", Half: true,
		Art: []string{
			"......$",
			"......A",
			"......A",
			"......A",
			"......A",
			"......A",
			"......A",
			"..bBbbb",
			"......c",
			"......c",
			".....dd",
		}},
	{Name: "Rocket", Cat: "object", Half: true,
		Art: []string{
			"......a",
			".....aa",
			"....aaa",
			"....aaB",
			"....aaa",
			"....aaa",
			"...cccc",
			"..d.aaa",
			".dd.aaa",
			"dd...**",
		}},
	{Name: "Crown", Cat: "object", Half: true,
		Art: []string{
			"a..a..a",
			"a..a..a",
			"aa.aa.a",
			"aaaaaaa",
			"a$aa$aa",
			"aaaaaaa",
			"bbbbbbb",
		},
		Tall: []string{
			"3..3..3",
			"3..3..3",
			"22.22.2",
			"2222222",
			"2222222",
			"2222222",
			"1111111",
		}},
	{Name: "Ringed World", Cat: "object", Half: true,
		Art: []string{
			"....aaa",
			"...aaaa",
			"bb.aaaa",
			"bbbbbbb",
			".bbaaaa",
			"...aaaa",
			"....aaa",
		},
		Tall: []string{
			"....223",
			"...2333",
			"11.2333",
			"1111111",
			".112333",
			"...2333",
			"....223",
		}},
	{Name: "Chessboard", Cat: "pattern", Half: true,
		Art: []string{
			"aBaBaBa",
			"BaBaBaB",
			"aBaBaBa",
			"BaBaBaB",
			"aBaBaBa",
			"BaBaBaB",
			"aBaBaBa",
			"BaBaBaB",
		}},
	{Name: "Target", Cat: "pattern", Half: true,
		Art: []string{
			"aaaaaaa",
			"abbbbbb",
			"abccccc",
			"abcdddd",
			"abcdeee",
			"abcdeE$",
			"abcdeee",
			"abcdddd",
			"abccccc",
			"abbbbbb",
			"aaaaaaa",
		},
		Tall: []string{
			"1111111",
			"1222222",
			"1233333",
			"1234444",
			"1234444",
			"1234444",
			"1234444",
			"1234444",
			"1233333",
			"1222222",
			"1111111",
		}},
}

var designKinds = map[byte]Kind{'#': Steel, '*': Explosive, '$': Crystal, '?': Mystery, '@': Elite, '%': Armored,
	'z': Chain, 's': Split, 'r': Regen, 'h': Shield, 'x': Multiplier}

func designRow(d *Design, rows []string, i int) string {
	row := rows[i]
	if !d.Half {
		return row
	}
	b := []byte(row)
	out := make([]byte, 0, 14)
	out = append(out, b...)
	for j := len(b) - 1; j >= 0; j-- {
		out = append(out, b[j])
	}
	return string(out)
}

// applyDesign fills the level from a showcase design; returns false if it
// cannot be placed.
func applyDesign(r *rand.Rand, lv *Level, p Params, d float64, idx int) bool {
	ds := &Designs[idx]
	flip := !ds.Half && r.Intn(2) == 0
	rows := len(ds.Art)
	top := max(0, min(MaxRows-rows, (12-rows)/2+r.Intn(2)))
	colShift := r.Intn(6)
	maxHP := 1 + int(math.Round(d*2))
	extra := 0
	if d > 0.5 {
		extra = 1
	}
	for i := 0; i < rows; i++ {
		art := designRow(ds, ds.Art, i)
		tall := ""
		if ds.Tall != nil {
			tall = designRow(ds, ds.Tall, i)
		}
		if len(art) != Cols {
			return false
		}
		for c := 0; c < Cols; c++ {
			cc := c
			if flip {
				cc = Cols - 1 - c
			}
			ch := art[c]
			if ch == '.' || ch == ' ' {
				continue
			}
			cell := Cell{Kind: Basic, HP: 1, Stack: 1}
			switch {
			case ch >= 'a' && ch <= 'f':
				cell.Color = (int(ch-'a') + colShift) % 6
			case ch >= 'A' && ch <= 'F':
				cell.Color = (int(ch-'A') + colShift) % 6
				cell.Stack = 2
			default:
				k, ok := designKinds[ch]
				if !ok {
					continue
				}
				cell.Kind = k
				cell.Color = colShift
			}
			if tall != "" && tall[c] >= '1' && tall[c] <= '4' {
				cell.Stack = int(tall[c] - '0')
			}
			if cell.Kind == Basic {
				cell.Stack = min(4, cell.Stack+extra)
				if r.Float64() < 0.1+0.35*d {
					cell.HP = min(maxHP, 2)
				}
				if d > 0.4 && r.Float64() < 0.15 {
					cell.HP = maxHP
				}
			}
			switch cell.Kind {
			case Armored:
				cell.HP = 2
			case Elite:
				cell.HP = 5 + p.Depth/3
				cell.Stack = 1
			case Steel:
				cell.Stack = min(cell.Stack, 2)
			}
			lv.Cells[top+i][cc] = cell
		}
	}
	lv.Rows = top + rows
	lv.LayerShift = 0
	if r.Intn(3) == 0 {
		lv.LayerShift = 1
	}
	lv.Name = "The " + ds.Name
	lv.Formation = "design: " + strings.ToLower(ds.Name)
	lv.Structure = ds.Cat
	return true
}

// designShape picks an arena outline that doesn't cut into the design.
func designShape(r *rand.Rand, lv *Level) {
	order := r.Perm(NumShapes)
	for _, s := range order {
		poly := Arena(s)
		ok := true
		for rr := 0; rr < MaxRows && ok; rr++ {
			for c := 0; c < Cols; c++ {
				if lv.Cells[rr][c].Kind != Empty && !CellInside(poly, rr, c) {
					ok = false
					break
				}
			}
		}
		if ok {
			lv.Shape = s
			for rr := 0; rr < MaxRows; rr++ {
				for c := 0; c < Cols; c++ {
					lv.Outside[rr][c] = !CellInside(poly, rr, c)
				}
			}
			return
		}
	}
	lv.Shape = ShapeHall
}
