package main

import (
	"image/color"
	"math"

	"orb/levelgen"
)

// Special bricks wear a small 3D object on top instead of a floating badge,
// so what a brick does reads from its shape: a fused bomb, a gem, a sprout,
// a gift box, a coin stack, tesla prongs, a nest, a crown...

func disc(r, h float64, n int, top, side color.RGBA) *Mesh {
	return Lathe([][2]float64{{r, 0}, {r, h}, {0, h}}, n, []color.RGBA{side, top}, false)
}

// drawTopper draws a special brick's topper; false if the kind has none.
func (g *Game) drawTopper(B *Batch, b *Brick, x, top, z, t float64) bool {
	if b.Up != nil || b.BossPart || b.Core {
		return false
	}
	ap := clamp01(b.Appear)
	if ap <= 0 {
		return true
	}
	s := ap
	ph := t*2 + x*0.7 + z*0.3
	at := func(dx, dy, dz float64) V3 { return V3{x + dx, top + dy, z + dz} }
	switch b.Kind {
	case levelgen.Explosive:
		B.Draw(cached("tp_bomb", func() *Mesh {
			return Icosphere(1, 0.02, 3, color.RGBA{40, 38, 46, 255}, color.RGBA{24, 22, 28, 255})
		}), at(0, 0.17*s, 0), v3(0.17*s), identity, color.RGBA{}, 0)
		B.Draw(cached("tp_bombcap", func() *Mesh {
			return disc(0.4, 0.4, 8, color.RGBA{150, 140, 120, 255}, color.RGBA{110, 100, 90, 255})
		}), at(0.05, 0.31*s, 0), v3(0.1*s), rotXYZ(0, 0, -0.4), color.RGBA{}, 0)
		B.Box(x+0.07, top+0.33*s, z-0.015, x+0.1, top+0.43*s, z+0.015, color.RGBA{200, 180, 140, 255}, sprWhite, 0)
		fl := 0.6 + 0.4*math.Abs(math.Sin(t*17+x))
		B.Billboard(at(0.09, 0.46*s, 0), 0.32*fl, sprSoft, color.RGBA{255, 200, 80, 255}, true)
		B.Billboard(at(0.09, 0.46*s, 0), 0.1, sprSphere, color.RGBA{255, 250, 210, 255}, false)
	case levelgen.Crystal:
		pulse := 0.25 + 0.2*math.Sin(t*3+x)
		B.Draw(cached("tp_gem", func() *Mesh { return Crystal(color.RGBA{120, 240, 255, 255}) }),
			at(0, 0.02, 0), V3{0.16 * s, 0.42 * s, 0.16 * s}, rotXYZ(0, t*0.8+x, 0), color.RGBA{}, float32(pulse))
		B.Billboard(at(0, 0.3*s, 0), 0.7, sprSoft, alpha(color.RGBA{120, 240, 255, 255}, 0.35), true)
	case levelgen.Regen:
		sway := 0.12 * math.Sin(ph)
		B.Box(x-0.015, top, z-0.015, x+0.015, top+0.22*s, z+0.015, color.RGBA{90, 160, 60, 255}, sprWhite, 0)
		leaf := cached("tp_leaf", func() *Mesh {
			return Icosphere(1, 0.0, 5, color.RGBA{120, 220, 90, 255}, color.RGBA{70, 160, 60, 255})
		})
		B.Draw(leaf, at(-0.08, 0.2*s, 0), V3{0.1 * s, 0.03 * s, 0.05 * s}, rotXYZ(0, 0.3, 0.5+sway), color.RGBA{}, 0.1)
		B.Draw(leaf, at(0.08, 0.24*s, 0), V3{0.1 * s, 0.03 * s, 0.05 * s}, rotXYZ(0, -0.3, -0.5+sway), color.RGBA{}, 0.1)
		if b.HP < b.MaxHP {
			B.Billboard(at(0, 0.3, 0), 0.6, sprSoft, alpha(color.RGBA{140, 255, 120, 255}, 0.35+0.2*math.Sin(t*4)), true)
		}
	case levelgen.Shield:
		pulse := 0.5 + 0.3*math.Sin(t*2.5+x)
		B.Draw(cached("tp_dome", func() *Mesh {
			return Lathe([][2]float64{{1, 0}, {0.92, 0.38}, {0.7, 0.72}, {0.38, 0.93}, {0, 1}}, 10,
				[]color.RGBA{{120, 180, 255, 255}, {150, 200, 255, 255}, {180, 220, 255, 255}, {210, 235, 255, 255}}, false)
		}), at(0, 0, 0), V3{0.3 * s, 0.22 * s, 0.18 * s}, identity, color.RGBA{}, float32(0.35*pulse))
		B.Billboard(at(0, 0.15, 0), 0.9, sprSoft, alpha(color.RGBA{120, 180, 255, 255}, 0.3*pulse), true)
	case levelgen.Teleport:
		for k := 0; k < 2; k++ {
			a := t*3 + float64(k)*math.Pi
			B.Draw(cached("tp_ring", func() *Mesh {
				return Lathe([][2]float64{{1, 0}, {1, 0.15}, {0.78, 0.15}, {0.78, 0}}, 14, []color.RGBA{{200, 120, 255, 255}, {240, 190, 255, 255}, {150, 80, 220, 255}}, false)
			}), at(0, 0.18*s, 0), V3{0.2 * s, 0.2 * s, 0.2 * s}, rotXYZ(math.Pi/2, a, 0), color.RGBA{}, 0.5)
		}
		B.Billboard(at(0, 0.18*s, 0), 0.6, sprSoft, alpha(color.RGBA{210, 120, 255, 255}, 0.55), true)
	case levelgen.Mystery:
		gift := color.RGBA{200, 70, 200, 255}
		rib := color.RGBA{255, 220, 90, 255}
		hw, hd, h := 0.17*s, 0.13*s, 0.2*s
		B.Box(x-hw, top, z-hd, x+hw, top+h, z+hd, gift, sprWhite, 0.05)
		B.Box(x-0.025, top, z-hd-0.005, x+0.025, top+h+0.005, z+hd+0.005, rib, sprWhite, 0.15)
		B.Box(x-hw-0.005, top, z-0.025, x+hw+0.005, top+h+0.005, z+0.025, rib, sprWhite, 0.15)
		bow := cached("tp_bow", func() *Mesh {
			return Icosphere(1, 0, 7, color.RGBA{255, 220, 90, 255}, color.RGBA{220, 170, 60, 255})
		})
		B.Draw(bow, at(-0.05, h+0.04, 0), V3{0.06 * s, 0.04 * s, 0.03 * s}, rotXYZ(0, 0, 0.5), color.RGBA{}, 0.15)
		B.Draw(bow, at(0.05, h+0.04, 0), V3{0.06 * s, 0.04 * s, 0.03 * s}, rotXYZ(0, 0, -0.5), color.RGBA{}, 0.15)
	case levelgen.Multiplier:
		coin := cached("tp_coin", func() *Mesh {
			return disc(1, 0.25, 12, color.RGBA{255, 222, 110, 255}, color.RGBA{220, 170, 60, 255})
		})
		for k := 0; k < 3; k++ {
			B.Draw(coin, at(float64(k-1)*0.01, float64(k)*0.055*s, float64(k%2)*0.01), V3{0.13 * s, 0.22 * s, 0.13 * s}, identity, color.RGBA{}, 0.2)
		}
		if math.Mod(t*0.7+x, 2) < 0.15 {
			B.Billboard(at(0.06, 0.2, 0), 0.35, sprSpark, color.RGBA{255, 250, 210, 255}, true)
		}
	case levelgen.Chain:
		for _, dx := range []float64{-0.12, 0.12} {
			B.Box(x+dx-0.02, top, z-0.02, x+dx+0.02, top+0.28*s, z+0.02, color.RGBA{150, 160, 180, 255}, sprWhite, 0.05)
			B.Billboard(at(dx, 0.3*s, 0), 0.08, sprSphere, color.RGBA{180, 230, 255, 255}, false)
		}
		if math.Mod(t*1.3+x*0.37, 1) < 0.25 {
			B.Billboard(at(0, 0.3*s, 0), 0.45, sprSoft, color.RGBA{140, 210, 255, 255}, true)
		}
	case levelgen.Spawner:
		B.Draw(cached("tp_nest", func() *Mesh {
			return Lathe([][2]float64{{0.7, 0}, {1, 0.3}, {0.95, 0.45}, {0.6, 0.3}, {0, 0.25}}, 10,
				[]color.RGBA{{120, 86, 50, 255}, {150, 110, 64, 255}, {170, 128, 76, 255}, {90, 64, 40, 255}}, false)
		}), at(0, 0, 0), V3{0.2 * s, 0.2 * s, 0.14 * s}, identity, color.RGBA{}, 0)
		egg := cached("tp_egg", func() *Mesh {
			return Icosphere(1, 0, 9, color.RGBA{240, 240, 225, 255}, color.RGBA{210, 205, 190, 255})
		})
		B.Draw(egg, at(-0.05, 0.1*s, 0), V3{0.055 * s, 0.075 * s, 0.055 * s}, identity, color.RGBA{}, 0)
		B.Draw(egg, at(0.06, 0.1*s, 0.02), V3{0.05 * s, 0.07 * s, 0.05 * s}, identity, color.RGBA{}, 0)
	case levelgen.Elite:
		B.Draw(cached("tp_crown", func() *Mesh {
			var m Mesh
			band := Lathe([][2]float64{{1, 0}, {1, 0.35}, {0.85, 0.35}, {0.85, 0}}, 12,
				[]color.RGBA{{255, 210, 90, 255}, {255, 230, 140, 255}, {220, 170, 60, 255}}, false)
			m.Tris = append(m.Tris, band.Tris...)
			for k := 0; k < 5; k++ {
				a := float64(k) / 5 * 2 * math.Pi
				cx, cz := math.Cos(a)*0.92, math.Sin(a)*0.92
				p := [3]V3{{cx - 0.25*math.Sin(a), 0.35, cz + 0.25*math.Cos(a)}, {cx + 0.25*math.Sin(a), 0.35, cz - 0.25*math.Cos(a)}, {cx, 0.95, cz}}
				m.Tris = append(m.Tris, Tri{P: p, Col: color.RGBA{255, 220, 110, 255}},
					Tri{P: [3]V3{p[1], p[0], p[2]}, Col: color.RGBA{230, 180, 70, 255}})
			}
			return &m
		}), at(0, 0, 0), V3{0.24 * s, 0.32 * s, 0.17 * s}, rotXYZ(0, t*0.6, 0), color.RGBA{}, 0.2)
		B.Billboard(at(0, 0.25, 0), 0.9, sprSoft, alpha(color.RGBA{255, 210, 110, 255}, 0.3), true)
	default:
		return false
	}
	return true
}
