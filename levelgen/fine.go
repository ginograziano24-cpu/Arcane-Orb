package levelgen

import "math/rand"

// The game plays on a fine grid: every cell of the coarse layout the
// generator draws (formations, picture boards) becomes a 2x2 block of
// smaller bricks, so boards carry four times as many bricks over the same
// floor and picture boards keep their exact shapes.
const (
	Cols    = lCols * 2
	MaxRows = lRows * 2
	CellW   = lCellW / 2
	CellD   = lCellD / 2
)

func CellX(c int) float64 { return -WorldHalfW + (float64(c)+0.5)*CellW }
func CellZ(r int) float64 { return GridTopZ - (float64(r)+0.5)*CellD }

// CellInside requires the whole fine cell to be inside the walls.
func CellInside(poly []Pt, r, c int) bool {
	x, z := CellX(c), CellZ(r)
	hw, hd := CellW/2-0.03, CellD/2-0.03
	for _, d := range [][2]float64{{-hw, -hd}, {hw, -hd}, {hw, hd}, {-hw, hd}} {
		if !Inside(poly, Pt{x + d[0], z + d[1]}) {
			return false
		}
	}
	return true
}

type Level struct {
	Name       string
	Formation  string
	Structure  string
	Rows       int
	Cells      [MaxRows][Cols]Cell
	Outside    [MaxRows][Cols]bool // cells clipped away by the arena walls
	Objective  Objective
	Seed       int64
	Shape      int
	LayerShift int // colour step between stacked layers
}

func (l *Level) Count(pred func(Cell) bool) int {
	n := 0
	for r := 0; r < MaxRows; r++ {
		for c := 0; c < Cols; c++ {
			if pred(l.Cells[r][c]) {
				n++
			}
		}
	}
	return n
}

func (l *Level) Required() int {
	n := 0
	for r := 0; r < MaxRows; r++ {
		for c := 0; c < Cols; c++ {
			if cl := l.Cells[r][c]; cl.Kind.Required() {
				n += max(1, cl.Stack)
			}
		}
	}
	return n
}

// Generate deterministically builds a level from its parameters.
func Generate(p Params) *Level {
	cl := generateCoarse(p)
	r := rand.New(rand.NewSource(Mix(p.Seed, int64(p.Depth), 977)))
	return refine(cl, r)
}

// refine turns a coarse layout into the fine one. A special brick keeps
// one of its four places (on the side facing the centre and the paddle)
// and the rest become plain bricks of its colour. Generated formations
// get rounded corners and, sometimes, a mosaic of shades; picture boards
// stay exact.
func refine(cl *coarse, r *rand.Rand) *Level {
	lv := &Level{Name: cl.Name, Formation: cl.Formation, Structure: cl.Structure, Rows: cl.Rows * 2,
		Objective: cl.Objective, Seed: cl.Seed, Shape: cl.Shape, LayerShift: cl.LayerShift}
	designed := len(cl.Formation) >= 6 && cl.Formation[:6] == "design"
	round := !designed && r.Float64() < 0.7
	mosaic := 0
	if !designed && r.Float64() < 0.45 {
		mosaic = 1 + r.Intn(3)
	}
	empty := func(rr, c int) bool {
		return rr < 0 || rr >= lRows || c < 0 || c >= lCols || cl.Cells[rr][c].Kind == Empty
	}
	for rr := 0; rr < lRows; rr++ {
		for c := 0; c < lCols; c++ {
			cell := cl.Cells[rr][c]
			if cell.Kind == Empty {
				continue
			}
			sc := 0
			if c < lCols/2 {
				sc = 1
			}
			special := cell.Kind != Basic && cell.Kind != Steel
			for dr := 0; dr < 2; dr++ {
				for dc := 0; dc < 2; dc++ {
					fc := cell
					if special && !(dr == 1 && dc == sc) {
						fc = Cell{Kind: Basic, HP: min(max(1, cell.HP), 2), Color: cell.Color, Stack: cell.Stack}
						if cell.Kind == Elite {
							fc.Stack = 1
							fc.HP = 2
						}
					}
					if fc.Kind == Basic && !special {
						fc.Cap = cell.Cap
						if cell.Cap != Empty && !(dr == 1 && dc == sc) {
							fc.Cap = Empty
						}
					}
					if fc.Kind == Basic && mosaic > 0 {
						switch mosaic {
						case 1: // diagonal shades
							if (dr+dc)%2 == 1 {
								fc.Color = (fc.Color + 1) % 6
							}
						case 2: // stripes across
							if dr == 1 {
								fc.Color = (fc.Color + 1) % 6
							}
						default: // stripes along
							if dc == 1 {
								fc.Color = (fc.Color + 5) % 6
							}
						}
					}
					if round && fc.Kind == Basic && fc.Cap == Empty {
						// trim a corner the formation turns around
						vr, vc := rr-1+2*dr, c-1+2*dc
						or, oc := rr+1-2*dr, c+1-2*dc
						if empty(vr, c) && empty(rr, vc) && empty(vr, vc) && !empty(or, c) && !empty(rr, oc) {
							continue
						}
					}
					lv.Cells[2*rr+dr][2*c+dc] = fc
				}
			}
		}
	}
	poly := Arena(lv.Shape)
	for rr := 0; rr < MaxRows; rr++ {
		for c := 0; c < Cols; c++ {
			lv.Outside[rr][c] = !CellInside(poly, rr, c)
			if lv.Outside[rr][c] {
				lv.Cells[rr][c] = Cell{}
			}
		}
	}
	return lv
}
