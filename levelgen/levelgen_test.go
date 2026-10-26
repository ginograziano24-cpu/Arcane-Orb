package levelgen

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

var glyph = map[Kind]byte{Empty: '.', Basic: 'o', Steel: '#', Armored: 'A', Explosive: '*', Chain: 'z', Split: 's',
	Regen: 'r', Moving: 'm', Shield: 'S', Teleport: 't', Spawner: 'p', Multiplier: 'x', Crystal: '$', Mystery: '?', Elite: 'E'}

func ascii(lv *Level) string {
	s := ""
	for r := 0; r < lv.Rows; r++ {
		for c := 0; c < Cols; c++ {
			cl := lv.Cells[r][c]
			ch := glyph[cl.Kind]
			if cl.Kind == Basic {
				ch = byte('0' + cl.Stack)
			}
			if lv.Outside[r][c] {
				ch = ' '
			}
			s += string(ch)
		}
		s += "\n"
	}
	return s
}

func TestGen(t *testing.T) {
	forms := map[string]int{}
	for seed := int64(0); seed < 200; seed++ {
		for d := 1; d <= 30; d++ {
			p := Params{Seed: seed, Depth: d, Act: 1 + d/10, Elite: d%7 == 0}
			a, b := Generate(p), Generate(p)
			if ascii(a) != ascii(b) {
				t.Fatal("nondeterministic")
			}
			if a.Required() < 16 {
				t.Fatalf("too few: %d", a.Required())
			}
			forms[a.Formation]++
		}
	}
	fmt.Println(len(forms), "distinct formation combos")
	for _, d := range []int{1, 6, 12, 20} {
		lv := Generate(Params{Seed: 42, Depth: d, Act: 1})
		fmt.Printf("depth %d  %s  [%s / %s / %s] obj=%d\n%s\n", d, lv.Name, lv.Formation, lv.Structure, ShapeNames[lv.Shape], lv.Objective, ascii(lv))
	}
}

func TestDesigns(t *testing.T) {
	for i, d := range Designs {
		for j, row := range d.Art {
			want := 14
			if d.Half {
				want = 7
			}
			if len(row) != want {
				t.Fatalf("%s art row %d has %d chars", d.Name, j, len(row))
			}
			if d.Tall != nil && len(d.Tall[j]) != want {
				t.Fatalf("%s tall row %d has %d chars", d.Name, j, len(d.Tall[j]))
			}
		}
		lv := &Level{}
		r := rand.New(rand.NewSource(int64(i)))
		if !applyDesign(r, lv, Params{Depth: 6}, 0.3, i) {
			t.Fatalf("%s failed", d.Name)
		}
		if lv.Required() < 16 {
			t.Errorf("%s only has %d bricks", d.Name, lv.Required())
		}
		if i%6 == 0 {
			fmt.Printf("%s\n%s\n", d.Name, ascii(lv))
		}
	}
	n := 0
	for s := int64(0); s < 400; s++ {
		if strings.HasPrefix(Generate(Params{Seed: s, Depth: 5}).Formation, "design") {
			n++
		}
	}
	fmt.Printf("designed levels: %d / 400\n", n)
}
