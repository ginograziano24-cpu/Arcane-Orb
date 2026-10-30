package levelgen

import (
	"math"
	"math/rand"
)

// World-space layout shared with the game (units).
const (
	WorldHalfW = 14.0
	WorldFarZ  = 31.5
	WorldNearZ = -3.0
	GridTopZ   = 30.3
	lCellW     = 2.0
	lCellD     = 1.1
)

func lCellX(c int) float64 { return -WorldHalfW + (float64(c)+0.5)*lCellW }
func lCellZ(r int) float64 { return GridTopZ - (float64(r)+0.5)*lCellD }

type Pt struct{ X, Z float64 }

const (
	ShapeHall = iota
	ShapeOctagon
	ShapeGable
	ShapeDome
	ShapeBastion
	ShapeNotched
	ShapeFunnel
	NumShapes
)

var ShapeNames = []string{"Hall", "Octagon", "Gable", "Dome", "Bastion", "Notched", "Funnel"}

// Arena returns the wall polyline, running from the near-left corner up and
// around the far end to the near-right corner. The near side is open.
// The region close to the paddle is always full width.
func Arena(shape int) []Pt {
	const W, F, N = WorldHalfW, WorldFarZ, WorldNearZ
	switch shape {
	case ShapeOctagon:
		return []Pt{{-W, N}, {-W, 24}, {-7.5, F}, {7.5, F}, {W, 24}, {W, N}}
	case ShapeGable:
		return []Pt{{-W, N}, {-W, 21}, {-5, F}, {5, F}, {W, 21}, {W, N}}
	case ShapeDome:
		pts := []Pt{{-W, N}, {-W, 20}}
		for i := 1; i < 12; i++ {
			a := math.Pi - float64(i)*math.Pi/12
			pts = append(pts, Pt{math.Cos(a) * W, 20 + math.Sin(a)*(F-20)})
		}
		return append(pts, Pt{W, 20}, Pt{W, N})
	case ShapeBastion:
		return []Pt{{-W, N}, {-W, 19}, {-11, 19}, {-11, 25.5}, {-7, 25.5}, {-7, F}, {7, F}, {7, 25.5}, {11, 25.5}, {11, 19}, {W, 19}, {W, N}}
	case ShapeNotched:
		return []Pt{{-W, N}, {-W, 13}, {-11.5, 16}, {-W, 19}, {-W, F}, {W, F}, {W, 19}, {11.5, 16}, {W, 13}, {W, N}}
	case ShapeFunnel:
		return []Pt{{-W, N}, {-W, 14}, {-10, F}, {10, F}, {W, 14}, {W, N}}
	}
	return []Pt{{-W, N}, {-W, F}, {W, F}, {W, N}}
}

// Inside tests whether p lies inside the arena (polyline closed along the near edge).
func Inside(poly []Pt, p Pt) bool {
	in := false
	n := len(poly)
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		a, b := poly[i], poly[j]
		if (a.Z > p.Z) != (b.Z > p.Z) {
			x := a.X + (p.Z-a.Z)/(b.Z-a.Z)*(b.X-a.X)
			if p.X < x {
				in = !in
			}
		}
	}
	return in
}

// lCellInside requires the whole cell footprint to be inside the walls.
func lCellInside(poly []Pt, r, c int) bool {
	x, z := lCellX(c), lCellZ(r)
	hw, hd := lCellW/2-0.05, lCellD/2-0.05
	for _, d := range [][2]float64{{-hw, -hd}, {hw, -hd}, {hw, hd}, {-hw, hd}} {
		if !Inside(poly, Pt{x + d[0], z + d[1]}) {
			return false
		}
	}
	return true
}

var StructureNames = []string{"flat", "pyramid", "towers", "terraces", "ramparts", "battlements", "dunes", "dome"}

// buildStacks raises columns of bricks into 3D structures.
func buildStacks(r *rand.Rand, lv *coarse, p Params, d float64, sym bool) {
	maxStack := 3 + int(0.5+d*4.5) // boards rise into structures: 3 high early, 8 late
	if p.Elite {
		maxStack++
	}
	maxStack = min(8, maxStack)
	mode := r.Intn(len(StructureNames))
	if p.Depth == 1 && mode == 0 {
		mode = 1
	}
	lv.Structure = StructureNames[mode]
	lv.LayerShift = []int{0, 1, 2, 3}[r.Intn(4)]
	// bounds of the formation for centring
	minR, maxR, minC, maxC := lRows, -1, lCols, -1
	for rr := 0; rr < lRows; rr++ {
		for c := 0; c < lCols; c++ {
			if lv.Cells[rr][c].Kind != Empty {
				minR, maxR = min(minR, rr), max(maxR, rr)
				minC, maxC = min(minC, c), max(maxC, c)
			}
		}
	}
	if maxR < 0 {
		return
	}
	cr, cc := float64(minR+maxR)/2, float64(minC+maxC)/2
	hr, hc := math.Max(1, float64(maxR-minR)/2+0.5), math.Max(1, float64(maxC-minC)/2+0.5)
	var blocks [lRows/2 + 1][lCols/2 + 1]int
	for i := range blocks {
		for j := range blocks[i] {
			blocks[i][j] = 1 + r.Intn(maxStack)
		}
	}
	if sym {
		for i := range blocks {
			for j := 0; j < len(blocks[i])/2; j++ {
				blocks[i][len(blocks[i])-1-j] = blocks[i][j]
			}
		}
	}
	freq := 0.6 + r.Float64()*0.6
	ph := r.Float64() * 6
	for rr := 0; rr < lRows; rr++ {
		for c := 0; c < lCols; c++ {
			cell := &lv.Cells[rr][c]
			if cell.Kind != Basic && cell.Kind != Steel && cell.Kind != Explosive && cell.Kind != Crystal &&
				cell.Kind != Armored && cell.Kind != Regen && cell.Kind != Chain && cell.Kind != Split {
				continue
			}
			dx, dz := (float64(c)-cc)/hc, (float64(rr)-cr)/hr
			dist := math.Min(1, math.Hypot(dx, dz)/math.Sqrt2)
			box := math.Min(1, math.Max(math.Abs(dx), math.Abs(dz)))
			h := 1
			switch mode {
			case 1: // pyramid
				h = 1 + int(math.Round((1-box)*float64(maxStack-1)))
			case 2: // towers
				h = blocks[rr/2][c/2]
			case 3: // terraces (taller toward the back)
				h = 1 + int(float64(maxR-rr)/float64(max(1, maxR-minR))*float64(maxStack-1)+0.3)
			case 4: // ramparts: tall outer edge
				if edgeCell(lv, rr, c) {
					h = maxStack
				}
			case 5: // battlements
				if (rr+c)%2 == 0 {
					h = maxStack
				}
			case 6: // dunes
				x := float64(c)
				if sym {
					x = math.Abs(float64(c) - 6.5)
				}
				h = 1 + int(math.Round((math.Sin(x*freq+float64(rr)*0.5+ph)+1)/2*float64(maxStack-1)))
			case 7: // dome
				h = 1 + int(math.Round(math.Sqrt(math.Max(0, 1-dist*dist))*float64(maxStack-1)))
			}
			h = max(1, min(maxStack, h))
			if cell.Kind == Steel {
				h = min(h, 2)
			}
			cell.Stack = h
			if cell.Kind == Basic && h >= 3 && r.Float64() < 0.12 {
				if r.Float64() < 0.6 {
					cell.Cap = Crystal
				} else {
					cell.Cap = Explosive
				}
			}
		}
	}
}

func edgeCell(lv *coarse, r, c int) bool {
	for _, d := range dirs4 {
		a, b := r+d[0], c+d[1]
		if a < 0 || a >= lRows || b < 0 || b >= lCols || lv.Cells[a][b].Kind == Empty {
			return true
		}
	}
	return false
}
