package main

import (
	"image/color"
	"math"
	"math/rand"
)

// Each world frames the arena with one hero set piece, a few supporting
// pieces and restrained detail, all built from stylised low-poly meshes.
// Everything stays outside the play area so the ball, paddle and bricks
// always read first.

type part = struct {
	M   *Mesh
	Pos V3
	S   V3
	R   Mat3
}

func (g *Game) drawScenery(B *Batch, L *Level) {
	bio := L.Biome
	t := L.T + g.stateT
	r := rand.New(rand.NewSource(L.propSeed))
	side := float64(r.Intn(2)*2 - 1) // which side the hero leans to
	ground := func(col color.RGBA) {
		// tessellated so distance fog is evaluated accurately across it
		xs := []float64{-300, -200, -140, -100, -75, -55, -40, -30, -22, -16, 0, 16, 22, 30, 40, 55, 75, 100, 140, 200, 300}
		zs := []float64{-20, -5, 5, 15, 25, 35, 50, 70, 100, 140, 200, 280, 400}
		for i := 0; i+1 < len(xs); i++ {
			for j := 0; j+1 < len(zs); j++ {
				B.Quad([4]V3{{xs[i], -0.6, zs[j+1]}, {xs[i+1], -0.6, zs[j+1]}, {xs[i+1], -0.6, zs[j]}, {xs[i], -0.6, zs[j]}}, sprWhite, col, [4]float32{1, 1, 1, 1}, false)
				B.quads[len(B.quads)-1].depth = 1e9
			}
		}
	}
	switch bio.ID {
	case WNeon:
		ground(bio.Outer)
		scroll := math.Mod(L.T*1.5, 6)
		for x := -96.0; x <= 96; x += 6 {
			if math.Abs(x) < HalfW+2 {
				continue
			}
			B.Quad([4]V3{{x - 0.06, -0.55, 400}, {x + 0.06, -0.55, 400}, {x + 0.06, -0.55, -20}, {x - 0.06, -0.55, -20}}, sprWhite, alpha(bio.Floor, 0.55), [4]float32{1, 1, 1, 1}, true)
		}
		for z := -18.0 + scroll; z < 300; z += 6 {
			B.Quad([4]V3{{-300, -0.55, z + 0.08}, {300, -0.55, z + 0.08}, {300, -0.55, z - 0.08}, {-300, -0.55, z - 0.08}}, sprWhite, alpha(bio.FloorAlt, 0.4), [4]float32{1, 1, 1, 1}, true)
		}
		lumenTree(B, side*13, 46, 1.6, bio, t)
		lumenTree(B, -side*30, 32, 0.85, bio, t+2)
		for i, p := range [][3]float64{{-side * 25, 7, 42}, {side * 28, 9, 30}} {
			y := p[1] + math.Sin(t*0.7+float64(i))*0.6
			floatingRock(B, p[0], y, p[2], 2.6, color.RGBA{58, 40, 92, 255}, color.RGBA{24, 14, 44, 255}, int64(i))
			for k := 0; k < 3; k++ {
				col := bio.Hues[(k+i*2)%6]
				cx := p[0] + float64(k-1)*1.1
				B.Draw(cached("crystal"+colKey(col), func() *Mesh { return Crystal(col) }), V3{cx, y + 0.4, p[2] + float64(k%2)*0.6}, V3{0.6, 1.6 + float64(k%2), 0.6}, rotXYZ(0, float64(k), float64(k-1)*0.25), color.RGBA{}, 0.3)
				B.Billboard(V3{cx, y + 1.4, p[2]}, 2.6, sprSoft, alpha(col, 0.3), true)
			}
		}
		// energy conduits feeding the arena from pylons
		for _, sx := range []float64{-1, 1} {
			for k := 0.0; k < 30; k += 2 {
				pulse := 0.4 + 0.6*math.Max(0, math.Sin(k*0.4-t*4))
				B.Decal(sx*(HalfW+3+k), FarZ-2, -0.5, 1.6, 0.25, sprWhite, alpha(bio.Floor, 0.6*pulse), true)
			}
			px := sx * (HalfW + 33)
			B.Draw(cached("pylon", func() *Mesh {
				return Lathe([][2]float64{{1.2, 0}, {1, 0.5}, {0.5, 1}, {0.35, 6}, {0.7, 6.3}, {0.7, 6.6}, {0.2, 7.4}}, 8,
					[]color.RGBA{{40, 28, 70, 255}, {50, 34, 86, 255}, {60, 40, 100, 255}, {60, 40, 100, 255}, {255, 90, 220, 255}, {90, 240, 255, 255}}, true)
			}), V3{px, -0.6, FarZ - 2}, v3(1), identity, color.RGBA{}, 0)
			B.Billboard(V3{px, 6.4, FarZ - 2}, 3.2, sprSoft, alpha(bio.Trim, 0.5+0.3*math.Sin(t*3)), true)
		}
	case WMeadow:
		ground(bio.Outer)
		windmill(B, side*32, 50, L.env.mill/0.8, bio)
		for i, p := range [][3]float64{{-side * 24, 30, 1.2}, {-side * 27, 22, 0.9}, {side * 22, 16, 1}} {
			oakTree(B, p[0], p[1], p[2], int64(i))
		}
		// split-rail fence with capped posts
		for _, sx := range []float64{-1, 1} {
			for z := -2.0; z < 34; z += 3 {
				x := sx * (HalfW + 3.2)
				B.Draw(cached("fencepost", func() *Mesh {
					return Lathe([][2]float64{{0.16, 0}, {0.14, 1.4}, {0.18, 1.45}, {0, 1.6}}, 6, []color.RGBA{{150, 108, 70, 255}, {176, 130, 86, 255}, {176, 130, 86, 255}}, false)
				}), V3{x, -0.6, z}, v3(1), identity, color.RGBA{}, 0)
				B.Box(x-0.05, 0.25, z, x+0.05, 0.4, z+3, color.RGBA{170, 125, 80, 255}, sprWhite, 0)
				B.Box(x-0.05, 0.75, z, x+0.05, 0.9, z+3, color.RGBA{170, 125, 80, 255}, sprWhite, 0)
			}
		}
		// round hay bales resting on their sides
		for i := 0; i < 2; i++ {
			x := float64(i*2-1) * (HalfW + 8 + r.Float64()*4)
			z := 10 + r.Float64()*16
			B.Draw(cached("hay", func() *Mesh {
				return Lathe([][2]float64{{0, -0.7}, {0.9, -0.7}, {1.05, -0.5}, {1.05, 0.5}, {0.9, 0.7}, {0, 0.7}}, 12,
					[]color.RGBA{{214, 170, 90, 255}, {226, 186, 104, 255}, {236, 198, 116, 255}, {226, 186, 104, 255}, {214, 170, 90, 255}}, false)
			}), V3{x, 0.45, z}, v3(1), rotXYZ(0, r.Float64(), math.Pi/2), color.RGBA{}, 0)
		}
		// flowering shrubs
		for i := 0; i < 4; i++ {
			x := float64(i%2*2-1) * (HalfW + 3 + r.Float64()*7)
			z := 2 + r.Float64()*28
			shrub(B, x, z, []color.RGBA{{255, 130, 150, 255}, {255, 226, 120, 255}, {200, 150, 255, 255}, {255, 255, 255, 255}}[i], int64(i))
		}
	case WCaverns:
		ground(bio.Outer)
		px := -side * 24
		B.Decal(px, 26, -0.55, 18, 14, sprSoft, alpha(color.RGBA{60, 150, 220, 255}, 0.55), true)
		B.Decal(px, 26, -0.54, 12, 9, sprSoft, alpha(color.RGBA{150, 230, 255, 255}, 0.3+0.1*math.Sin(t*2)), true)
		for k := 0; k < 9; k++ {
			a := float64(k) / 9 * 2 * math.Pi
			boulder(B, px+math.Cos(a)*8, 26+math.Sin(a)*6, 0.8+0.5*math.Abs(math.Sin(a*3)), color.RGBA{60, 66, 92, 255}, int64(k))
		}
		crystalCluster(B, side*14, 48, 2.2, bio, t)
		crystalCluster(B, -side*30, 38, 1.0, bio, t+1)
		crystalCluster(B, side*29, 18, 0.7, bio, t+2)
		wx := -side * 34
		for k := 0; k < 12; k++ {
			y := 22 - float64(k)*2
			B.Quad([4]V3{{wx - 1.5, y, 44}, {wx + 1.5, y, 44}, {wx + 1.5, y - 2, 44}, {wx - 1.5, y - 2, 44}}, sprWhite,
				alpha(color.RGBA{160, 220, 255, 255}, 0.18+0.1*math.Sin(t*6+float64(k))), [4]float32{1, 1, 1, 1}, true)
		}
		B.Billboard(V3{wx, 0, 44}, 9, sprSoft, alpha(color.RGBA{180, 230, 255, 255}, 0.45), true)
		for i := 0; i < 6; i++ {
			x := float64(i%2*2-1) * (HalfW + 5 + r.Float64()*14)
			z := 6 + r.Float64()*40
			B.Draw(cached("stalag"+itoa(i%3), func() *Mesh {
				return Icosphere(1, 0.18, int64(i), color.RGBA{70, 76, 104, 255}, color.RGBA{36, 40, 60, 255})
			}), V3{x, -0.6, z}, V3{1.1, 4 + r.Float64()*4, 1.1}, rotXYZ(0, float64(i), 0), color.RGBA{}, 0)
		}
		for i := 0; i < 3; i++ {
			x := float64(i%2*2-1) * (HalfW + 3.5 + r.Float64()*4)
			z := 4 + r.Float64()*26
			mushrooms(B, x, z, bio.Hues[(i*2)%6], int64(i), t)
		}
	case WRuins:
		ground(bio.Outer)
		for _, sx := range []float64{-1, 1} {
			B.Decal(sx*(HalfW+4), 16, -0.55, 2.4, 50, sprWhite, color.RGBA{60, 160, 170, 255}, false)
			B.Decal(sx*(HalfW+4), 16, -0.54, 1.6, 50, sprSoft, alpha(color.RGBA{150, 230, 220, 255}, 0.3), true)
		}
		stoneHead(B, side*16, 52, bio)
		for i := 0; i < 5; i++ {
			for _, sx := range []float64{-1, 1} {
				x := sx * (HalfW + 8)
				z := 6 + float64(i)*8
				broken := (i+int(sx+1))%3 == 1
				column(B, x, z, bio, broken, int64(i*2+int(sx+1)))
			}
		}
		for i := 0; i < 2; i++ {
			palm(B, -side*(HalfW+16+float64(i)*6), 20+float64(i)*12, t, int64(i))
		}
		ax := -side * 26
		B.Box(ax-2, -0.6, 30, ax+2, 1.2, 33, bio.Wall, matTop[WRuins], 0)
		ry := 3.6 + math.Sin(t*1.5)*0.4
		B.Draw(cached("relic", func() *Mesh {
			return Lathe([][2]float64{{0, -1}, {0.9, 0}, {0, 1}}, 4, []color.RGBA{{230, 180, 60, 255}, {255, 220, 110, 255}}, false)
		}), V3{ax, ry, 31.5}, v3(1), rotXYZ(0, t, 0), color.RGBA{}, 0.35)
		for k := 0; k < 10; k++ {
			a := t*0.8 + float64(k)/10*2*math.Pi
			dir := V3{math.Cos(a), 0, math.Sin(a)}
			B.OBox(V3{ax, ry, 31.5}.Add(dir.Mul(1.8)), dir.Mul(0.06), V3{0, 0.06, 0}, V3{-dir.Z, 0, dir.X}.Mul(0.55), color.RGBA{255, 210, 100, 255}, 0.4)
		}
		B.Billboard(V3{ax, ry, 31.5}, 5, sprSoft, alpha(color.RGBA{255, 220, 120, 255}, 0.45), true)
	case WClouds:
		// a sea of faceted cumulus clusters drifting below the arena
		nc := len(B.quads)
		for i := 0; i < 26; i++ {
			x := (r.Float64() - 0.5) * 200
			z := -6 + r.Float64()*170
			if math.Abs(x) < HalfW+8 && z < FarZ+6 {
				continue
			}
			y := -9 - r.Float64()*5 + math.Sin(t*0.3+float64(i))*0.3
			s := 5 + r.Float64()*5
			drift := math.Sin(t*0.05+float64(i)) * 2
			B.Draw(cloudCluster(i%4), V3{x + drift, y, z}, V3{s, s * 0.62, s}, rotXYZ(0, float64(i)*1.7, 0), color.RGBA{}, 0)
		}
		for i := nc; i < len(B.quads); i++ {
			B.quads[i].depth += 5000
		}
		n0 := len(B.quads)
		B.Draw(cached("arenaRock", func() *Mesh {
			return Lathe([][2]float64{{0, -1}, {0.35, -0.7}, {0.7, -0.35}, {0.95, -0.08}, {1, 0}}, 10, []color.RGBA{{120, 104, 100, 255}, {140, 122, 114, 255}, {156, 136, 124, 255}, {166, 146, 132, 255}}, false)
		}), V3{0, -0.6, FarZ / 2}, V3{HalfW + 3, 8, FarZ/2 + 5}, identity, color.RGBA{}, 0)
		for i := n0; i < len(B.quads); i++ {
			B.quads[i].depth += 4000
		}
		skyCastle(B, side*16, 78, 1.0, t, bio)
		skyCastle(B, -side*40, 44, 0.55, t+3, bio)
		airship(B, math.Mod(t*1.2, 140)-70, 14, 52, t)
		for i := 0; i < 3; i++ {
			x := float64(i%2*2-1) * (HalfW + 10 + r.Float64()*10)
			z := 8 + r.Float64()*30
			y := -2 + math.Sin(t*0.5+float64(i))*0.5
			floatingRock(B, x, y, z, 2.6, color.RGBA{130, 200, 110, 255}, color.RGBA{140, 122, 114, 255}, int64(i+7))
			oakTree(B, x, z, 0.6, int64(i+5)).y(B, y)
		}
	case WMachine:
		ground(bio.Outer)
		gear(B, side*8, 9, 52, 9, L.env.gearA*side, color.RGBA{110, 100, 92, 255}, bio)
		gear(B, side*8-side*14, 6, 50, 5, -L.env.gearA*1.8*side, color.RGBA{150, 100, 60, 255}, bio)
		B.Draw(cached("axle", func() *Mesh {
			return Lathe([][2]float64{{0, 0}, {1.6, 0}, {1.6, 1.2}, {1, 1.4}, {0, 1.4}}, 10, []color.RGBA{{60, 62, 70, 255}, {80, 82, 92, 255}, {100, 102, 112, 255}}, false)
		}), V3{side * 8, 9, 53}, v3(1), rotXYZ(-math.Pi/2, 0, 0), color.RGBA{}, 0)
		pipeMesh := cached("pipe", func() *Mesh {
			prof := [][2]float64{{0, 0}}
			for k := 0.0; k <= 48; k += 6 {
				prof = append(prof, [2]float64{0.42, k}, [2]float64{0.55, k + 0.1}, [2]float64{0.55, k + 0.45}, [2]float64{0.42, k + 0.55})
			}
			prof = append(prof, [2]float64{0, 48})
			return Lathe(prof, 8, []color.RGBA{{150, 92, 52, 255}}, false)
		})
		for _, sx := range []float64{-1, 1} {
			x := sx * (HalfW + 4)
			for k := 0; k < 3; k++ {
				tint := []color.RGBA{{255, 255, 255, 255}, {170, 190, 210, 255}, {230, 210, 190, 255}}[k]
				B.Draw(pipeMesh, V3{x + float64(k)*sx*1.2, 0.6 + float64(k)*1.3, -4}, v3(1), rotXYZ(math.Pi/2, 0, 0), tint, 0)
			}
			for z := 6.0; z < 40; z += 13 {
				controlCabinet(B, sx*(HalfW+9), z, t, sx)
			}
		}
		for i, p := range [][2]float64{{-side * 26, 34}, {-side * 32, 22}, {side * 30, 26}} {
			boiler(B, p[0], p[1], t+float64(i))
		}
	}
	g.drawDressing(B, L, t, side)
}

func colKey(c color.RGBA) string { return itoa(int(c.R)) + "-" + itoa(int(c.G)) + "-" + itoa(int(c.B)) }

// cloudCluster is a low-poly cumulus: several faceted puffs merged, bright
// on top and lavender underneath.
func cloudCluster(v int) *Mesh {
	return cached("cloud"+itoa(v), func() *Mesh {
		r := rand.New(rand.NewSource(int64(v*31 + 7)))
		type part = struct {
			M   *Mesh
			Pos V3
			S   V3
			R   Mat3
		}
		var parts []part
		n := 4 + r.Intn(3)
		for k := 0; k < n; k++ {
			a := float64(k) / float64(n) * 2 * math.Pi
			rad := 0.55 + r.Float64()*0.35
			if k == 0 {
				rad = 1.0
			}
			off := 0.0
			if k > 0 {
				off = 0.7 + r.Float64()*0.3
			}
			pos := V3{math.Cos(a) * off, (rad - 1) * 0.4, math.Sin(a) * off * 0.8}
			parts = append(parts, part{Icosphere(1, 0.1, int64(v*10+k), color.RGBA{252, 252, 255, 255}, color.RGBA{190, 194, 230, 255}), pos, v3(rad), identity})
		}
		return Merge(parts...)
	})
}

// controlCabinet is a machine-room console: plinth, riveted body, cap,
// a glowing readout facing the arena, gauges, hazard band and a beacon.
func controlCabinet(B *Batch, x, z, t, sx float64) {
	steel := color.RGBA{88, 92, 102, 255}
	B.Box(x-1.95, 0, z-0.95, x+1.95, 0.35, z+0.95, darken(steel, 0.65), sprWhite, 0)
	B.Box(x-1.7, 0.35, z-0.72, x+1.7, 3.3, z+0.72, steel, matTop[WMachine], 0)
	B.Box(x-1.72, 0.55, z-0.74, x+1.72, 0.85, z+0.74, color.RGBA{232, 168, 40, 255}, sprWhite, 0.1)
	B.Box(x-1.88, 3.3, z-0.88, x+1.88, 3.55, z+0.88, lighten(steel, 0.18), sprWhite, 0)
	// arena-facing readout and gauges
	fx := x - sx*1.7
	flick := 0.75 + 0.25*math.Sin(t*7+z*3)
	B.Box(fx-0.05, 1.55, z-0.52, fx+0.05, 2.85, z+0.52, color.RGBA{20, 26, 30, 255}, sprWhite, 0)
	B.Box(fx-sx*0.06-0.03, 1.65, z-0.44, fx-sx*0.06+0.03, 2.75, z+0.44, color.RGBA{40, 210, 170, 255}, sprWhite, float32(0.9*flick))
	for k, gz := range []float64{-0.35, 0.35} {
		B.Billboard(V3{fx - sx*0.12, 1.15, z + gz}, 0.34, sprSphere, color.RGBA{230, 220, 200, 255}, false)
		B.Billboard(V3{fx - sx*0.14, 1.15, z + gz}, 0.6, sprSoft, alpha(color.RGBA{255, 170, 60, 255}, 0.35+0.2*math.Sin(t*2+float64(k))), true)
	}
	beacon := cached("beacon", func() *Mesh {
		return Lathe([][2]float64{{0.32, 0}, {0.32, 0.12}, {0.24, 0.16}, {0.26, 0.42}, {0.18, 0.56}, {0, 0.62}}, 10,
			[]color.RGBA{{60, 62, 70, 255}, {60, 62, 70, 255}, {255, 150, 40, 255}, {255, 170, 60, 255}, {255, 190, 90, 255}}, false)
	})
	blink := math.Max(0, math.Sin(t*4+z))
	B.Draw(beacon, V3{x, 3.55, z}, v3(1), identity, color.RGBA{}, float32(0.6*blink))
	B.Billboard(V3{x, 3.85, z}, 1.8, sprSoft, alpha(color.RGBA{255, 150, 40, 255}, 0.2+0.6*blink), true)
}

// ---------------------------------------------------------------- set pieces

func lumenTree(B *Batch, x, z, s float64, bio *Biome, t float64) {
	B.Draw(cached("lumenTrunk", func() *Mesh {
		return Lathe([][2]float64{{0.9, 0}, {0.55, 1}, {0.45, 3}, {0.5, 4.6}, {0.75, 5.6}, {0, 6}}, 7,
			[]color.RGBA{{40, 24, 66, 255}, {52, 30, 84, 255}, {60, 36, 96, 255}, {70, 42, 110, 255}}, false)
	}), V3{x, -0.6, z}, v3(s), identity, color.RGBA{}, 0)
	for i := 0; i < 5; i++ {
		a := float64(i) * 1.3
		col := bio.Hues[i%6]
		p := V3{x + math.Cos(a)*2.2*s, (6.2 + float64(i%3)*1.1) * s, z + math.Sin(a)*1.5*s}
		m := cached("lumenLeaf"+itoa(i), func() *Mesh {
			return Icosphere(1, 0.22, int64(i+40), color.RGBA{120, 70, 200, 255}, color.RGBA{50, 24, 96, 255})
		})
		B.Draw(m, p, V3{2.1 * s, 1.5 * s, 2.1 * s}, rotXYZ(0, a, 0), color.RGBA{}, 0.15)
		B.Billboard(p, 5.5*s, sprSoft, alpha(col, 0.35+0.12*math.Sin(t*1.5+a)), true)
		// hanging light
		hx, hz := p.X+math.Cos(a+1)*1.2*s, p.Z+math.Sin(a+1)*0.8*s
		hy := p.Y - (2.2+math.Sin(t+a)*0.15)*s
		B.Box(hx-0.02, hy, hz-0.02, hx+0.02, p.Y-0.8*s, hz+0.02, color.RGBA{160, 140, 220, 255}, sprWhite, 0.3)
		B.Billboard(V3{hx, hy, hz}, 0.5*s, sprSphere, lighten(col, 0.5), false)
		B.Billboard(V3{hx, hy, hz}, 1.6*s, sprSoft, alpha(col, 0.7), true)
	}
}

func floatingRock(B *Batch, x, y, z, s float64, top, under color.RGBA, seed int64) {
	B.Draw(cached("frock"+itoa(int(seed)), func() *Mesh {
		return Lathe([][2]float64{{0, -1.6}, {0.45, -1.1}, {0.8, -0.55}, {1, -0.12}, {1, 0}, {0, 0.05}}, 7,
			[]color.RGBA{darken(under, 0.8), under, lighten(under, 0.08), lighten(under, 0.12), top}, false)
	}), V3{x, y, z}, V3{s, s, s * 0.8}, rotXYZ(0, float64(seed), 0), color.RGBA{}, 0)
}

type treeRef struct{ x, z, s float64 }

func (t treeRef) y(B *Batch, y float64) {} // placement helper (trees sit on the ground plane)

func oakTree(B *Batch, x, z, s float64, seed int64) treeRef {
	B.Draw(cached("oakTrunk", func() *Mesh {
		return Lathe([][2]float64{{0.55, 0}, {0.36, 0.6}, {0.3, 2.6}, {0.2, 3.4}}, 6, []color.RGBA{{110, 76, 50, 255}, {124, 86, 56, 255}}, false)
	}), V3{x, -0.6, z}, v3(s), identity, color.RGBA{}, 0)
	for k := 0; k < 3; k++ {
		m := cached("oakLeaf"+itoa(k), func() *Mesh {
			return Icosphere(1, 0.16, int64(k+10), color.RGBA{132, 190, 92, 255}, color.RGBA{66, 112, 58, 255})
		})
		a := float64(seed) + float64(k)*2.1
		p := V3{x + math.Cos(a)*0.9*s, (3.6 + float64(k%2)*0.8) * s, z + math.Sin(a)*0.7*s}
		B.Draw(m, p, V3{1.9 * s, 1.6 * s, 1.9 * s}, rotXYZ(0, a, 0), color.RGBA{}, 0)
	}
	return treeRef{x, z, s}
}

func shrub(B *Batch, x, z float64, bloom color.RGBA, seed int64) {
	B.Draw(cached("shrub", func() *Mesh {
		return Icosphere(1, 0.2, 77, color.RGBA{110, 168, 80, 255}, color.RGBA{60, 100, 52, 255})
	}), V3{x, 0, z}, V3{1.3, 0.9, 1.3}, rotXYZ(0, float64(seed), 0), color.RGBA{}, 0)
	for k := 0; k < 6; k++ {
		a := float64(k)*1.05 + float64(seed)
		p := V3{x + math.Cos(a)*0.9, 0.55 + float64(k%3)*0.25, z + math.Sin(a)*0.9}
		B.Draw(cached("bloom"+colKey(bloom), func() *Mesh {
			return Icosphere(0, 0.1, 5, bloom, darken(bloom, 0.8))
		}), p, v3(0.22), identity, color.RGBA{}, 0.1)
	}
}

func windmill(B *Batch, x, z, t float64, bio *Biome) {
	B.Draw(cached("millHill", func() *Mesh {
		return Icosphere(2, 0.05, 3, color.RGBA{128, 176, 90, 255}, color.RGBA{86, 128, 66, 255})
	}), V3{x, -2.2, z}, V3{7, 4.4, 6}, identity, color.RGBA{}, 0)
	B.Draw(cached("millTower", func() *Mesh {
		return Lathe([][2]float64{{2.3, 0}, {2.2, 1.2}, {1.95, 1.3}, {1.6, 9.6}, {1.75, 9.8}, {1.75, 10.2}, {1.5, 10.3}}, 8,
			[]color.RGBA{{150, 140, 128, 255}, {170, 160, 146, 255}, {238, 226, 206, 255}, {120, 90, 70, 255}, {120, 90, 70, 255}, {120, 90, 70, 255}}, true)
	}), V3{x, 1.6, z}, v3(1), identity, color.RGBA{}, 0)
	B.Draw(cached("millCap", func() *Mesh {
		return Lathe([][2]float64{{1.7, 0}, {1.6, 0.8}, {1.2, 1.6}, {0.5, 2.2}, {0, 2.4}}, 8,
			[]color.RGBA{{178, 76, 60, 255}, {196, 88, 66, 255}, {206, 98, 72, 255}, {214, 108, 80, 255}}, false)
	}), V3{x, 11.9, z}, v3(1), identity, color.RGBA{}, 0)
	// door and windows
	B.Box(x-0.5, 1.6, z-2.05, x+0.5, 3.4, z-1.9, color.RGBA{90, 62, 44, 255}, sprWhite, 0)
	B.Box(x-0.35, 6.5, z-1.8, x+0.35, 7.3, z-1.65, color.RGBA{70, 80, 110, 255}, sprWhite, 0.15)
	hub := V3{x, 12.6, z - 1.9}
	B.Billboard(hub, 0.9, sprSphere, color.RGBA{90, 70, 60, 255}, false)
	for k := 0; k < 4; k++ {
		a := t*0.8 + float64(k)*math.Pi/2
		dir := V3{math.Cos(a), math.Sin(a), 0}
		perp := V3{-math.Sin(a), math.Cos(a), 0}
		B.OBox(hub.Add(dir.Mul(4.4)), dir.Mul(4.4), perp.Mul(0.09), V3{0, 0, 0.12}, color.RGBA{120, 90, 70, 255}, 0)
		sail := hub.Add(dir.Mul(5)).Add(perp.Mul(0.55)).Add(V3{0, 0, -0.08})
		B.OBox(sail, dir.Mul(3.4), perp.Mul(0.5), V3{0, 0, 0.04}, color.RGBA{244, 238, 226, 255}, 0)
		for j := 0; j < 4; j++ {
			c := hub.Add(dir.Mul(2.2 + float64(j)*1.6)).Add(perp.Mul(0.55)).Add(V3{0, 0, -0.14})
			B.OBox(c, dir.Mul(0.05), perp.Mul(0.55), V3{0, 0, 0.04}, color.RGBA{140, 104, 78, 255}, 0)
		}
	}
}

func boulder(B *Batch, x, z, s float64, col color.RGBA, seed int64) {
	B.Draw(cached("boulder"+itoa(int(seed%4)), func() *Mesh {
		return Icosphere(1, 0.22, seed%4+20, lighten(col, 0.1), darken(col, 0.7))
	}), V3{x, -0.4, z}, V3{s, s * 0.7, s}, rotXYZ(0, float64(seed), 0), color.RGBA{}, 0)
}

func crystalCluster(B *Batch, x, z, s float64, bio *Biome, t float64) {
	B.Draw(cached("clusterBase", func() *Mesh {
		return Icosphere(1, 0.2, 31, color.RGBA{70, 74, 104, 255}, color.RGBA{30, 32, 50, 255})
	}), V3{x, -0.6, z}, V3{3.4 * s, 1.2 * s, 2.8 * s}, identity, color.RGBA{}, 0)
	for i := 0; i < 8; i++ {
		a := float64(i) * 0.82
		col := bio.Hues[i%6]
		rad := 0.6 * float64(i%3)
		cx, cz := x+math.Cos(a)*rad*2*s, z+math.Sin(a)*rad*1.4*s
		h := (5 + float64((i*5)%7)) * s
		tilt := rad * 0.35
		B.Draw(cached("crystal"+colKey(col), func() *Mesh { return Crystal(col) }), V3{cx, -0.2, cz},
			V3{(1.1 + float64(i%3)*0.3) * s, h, (1.1 + float64(i%3)*0.3) * s}, rotXYZ(math.Sin(a)*tilt, a, -math.Cos(a)*tilt), color.RGBA{}, 0.2)
		B.Billboard(V3{cx, h * 0.55, cz}, 5*s, sprSoft, alpha(col, 0.2+0.08*math.Sin(t*2+a)), true)
	}
	B.Billboard(V3{x, 3 * s, z}, 14*s, sprSoft, alpha(bio.Trim, 0.16), true)
}

func mushrooms(B *Batch, x, z float64, col color.RGBA, seed int64, t float64) {
	stem := cached("mstem", func() *Mesh {
		return Lathe([][2]float64{{0.22, 0}, {0.16, 0.6}, {0.14, 1.2}, {0.18, 1.4}}, 6, []color.RGBA{{210, 220, 236, 255}}, false)
	})
	cap := cached("mcap"+colKey(col), func() *Mesh {
		m := Lathe([][2]float64{{0.2, -0.05}, {0.9, -0.05}, {0.85, 0.2}, {0.6, 0.45}, {0, 0.55}}, 9,
			[]color.RGBA{lighten(col, 0.4), col, col, lighten(col, 0.2)}, false)
		for i := range m.Tris {
			m.Tris[i].Emi = 0.35
		}
		return m
	})
	r := rand.New(rand.NewSource(seed))
	for k := 0; k < 3; k++ {
		s := 0.6 + r.Float64()*0.8
		px, pz := x+(r.Float64()-0.5)*2.2, z+(r.Float64()-0.5)*2.2
		B.Draw(stem, V3{px, -0.6, pz}, v3(s), identity, color.RGBA{}, 0.1)
		B.Draw(cap, V3{px, -0.6 + 1.4*s, pz}, v3(s), rotXYZ(0.1, float64(k), 0), color.RGBA{}, 0)
		B.Billboard(V3{px, -0.6 + 1.2*s, pz}, 2.4*s, sprSoft, alpha(col, 0.35+0.1*math.Sin(t*2+float64(k))), true)
	}
}

func column(B *Batch, x, z float64, bio *Biome, broken bool, seed int64) {
	stone := lighten(bio.Wall, 0.05)
	hgt := 8.0
	if broken {
		hgt = 3 + float64(seed%3)
	}
	key := "column"
	if broken {
		key += itoa(int(hgt))
	}
	B.Draw(cached(key, func() *Mesh {
		prof := [][2]float64{{1.25, 0}, {1.25, 0.4}, {1.05, 0.5}, {1.1, 0.7}, {0.85, 0.85}, {0.78, hgt - 0.9}}
		cols := []color.RGBA{darken(stone, 0.9), stone, stone, stone, stone}
		if !broken {
			prof = append(prof, [2]float64{1, hgt - 0.6}, [2]float64{1.3, hgt - 0.3}, [2]float64{1.3, hgt})
			cols = append(cols, lighten(stone, 0.06), lighten(stone, 0.08), lighten(stone, 0.1))
		}
		return Lathe(prof, 12, cols, true)
	}), V3{x, -0.6, z}, v3(1), rotXYZ(0, float64(seed), 0), color.RGBA{}, 0)
	if broken {
		boulder(B, x+1.6, z+1.2, 0.8, stone, seed)
		boulder(B, x-1.2, z-1.6, 0.5, stone, seed+1)
	} else {
		B.Box(x-1.5, hgt-0.6, z-1.5, x+1.5, hgt-0.2, z+1.5, lighten(stone, 0.06), matTop[WRuins], 0)
	}
}

func palm(B *Batch, x, z, t float64, seed int64) {
	lean := 0.25 + float64(seed)*0.1
	B.Draw(cached("palmTrunk", func() *Mesh {
		prof := [][2]float64{}
		for k := 0; k <= 9; k++ {
			r := 0.42 - float64(k)*0.022
			prof = append(prof, [2]float64{r * 1.12, float64(k) * 0.8}, [2]float64{r, float64(k)*0.8 + 0.6})
		}
		return Lathe(prof, 6, []color.RGBA{{150, 116, 80, 255}, {130, 98, 66, 255}}, true)
	}), V3{x, -0.6, z}, v3(1), rotXYZ(lean, float64(seed), 0), color.RGBA{}, 0)
	top := V3{x, -0.6, z}.Add(rotXYZ(lean, float64(seed), 0).Mul(V3{0, 7.8, 0}))
	for k := 0; k < 7; k++ {
		a := float64(k)/7*2*math.Pi + math.Sin(t*0.6)*0.05
		dir := V3{math.Cos(a), -0.35, math.Sin(a)}.Norm()
		perp := V3{-math.Sin(a), 0, math.Cos(a)}
		B.OBox(top.Add(dir.Mul(1.8)), dir.Mul(1.9), V3{0, 0.04, 0}, perp.Mul(0.45), color.RGBA{86, 150, 70, 255}, 0)
		B.OBox(top.Add(dir.Mul(3.5)).Add(V3{0, -0.5, 0}), V3{dir.X, -0.8, dir.Z}.Norm().Mul(0.7), V3{0, 0.04, 0}, perp.Mul(0.3), color.RGBA{72, 130, 60, 255}, 0)
	}
}

func stoneHead(B *Batch, x, z float64, bio *Biome) {
	stone := lighten(bio.Wall, 0.05)
	dark := darken(bio.Wall, 0.5)
	B.Box(x-5, -0.6, z-3, x+5, 11, z+3, stone, matTop[WRuins], 0)
	B.BoxTex(x-5.6, 9, z-3.4, x+5.6, 13, z+3, darken(stone, 0.92), matTop[WRuins], matSide[WRuins], 0)
	for k := -2; k <= 2; k++ {
		B.Box(x+float64(k)*2-0.6, 13, z-2.8, x+float64(k)*2+0.6, 15.5-math.Abs(float64(k))*0.6, z+2.4, stone, matTop[WRuins], 0)
	}
	B.Box(x-4.2, 7, z-3.6, x+4.2, 7.8, z-3, darken(stone, 0.85), sprWhite, 0)
	B.Box(x-3.5, 5.6, z-3.15, x-1.2, 6.8, z-3, dark, sprWhite, 0)
	B.Box(x+1.2, 5.6, z-3.15, x+3.5, 6.8, z-3, dark, sprWhite, 0)
	B.Billboard(V3{x - 2.35, 6.2, z - 3.3}, 0.5, sprSphere, color.RGBA{90, 220, 210, 255}, false)
	B.Billboard(V3{x + 2.35, 6.2, z - 3.3}, 0.5, sprSphere, color.RGBA{90, 220, 210, 255}, false)
	B.Box(x-0.8, 3.2, z-4.2, x+0.8, 6.8, z-3, stone, sprWhite, 0)
	B.Box(x-2.4, 1.4, z-3.4, x+2.4, 2.4, z-3, darken(stone, 0.8), sprWhite, 0)
	boulder(B, x+7, z-5, 1.5, stone, 3)
	boulder(B, x-7.5, z-4.2, 1.1, stone, 4)
	boulder(B, x-6, z-6.4, 0.7, stone, 5)
	for k := 0; k < 5; k++ {
		B.Draw(cached("moss", func() *Mesh {
			return Icosphere(1, 0.25, 61, color.RGBA{110, 150, 80, 255}, color.RGBA{70, 104, 60, 255})
		}), V3{x - 4.6 + float64(k)*2.3, 12.9 + float64(k%2)*0.2, z - 3.2}, V3{0.8, 0.35, 0.6}, identity, color.RGBA{}, 0)
	}
}

func skyCastle(B *Batch, x, z, s, t float64, bio *Biome) {
	y := 1*s + math.Sin(t*0.4)*0.6
	floatingRock(B, x, y, z, 7*s, color.RGBA{130, 200, 110, 255}, color.RGBA{150, 130, 120, 255}, 11)
	tower := cached("skyTower", func() *Mesh {
		return Lathe([][2]float64{{1, 0}, {1, 5}, {1.3, 5.1}, {1.3, 5.7}, {1.05, 5.75}}, 10,
			[]color.RGBA{{245, 240, 250, 255}, {232, 228, 242, 255}, {245, 240, 250, 255}, {220, 216, 232, 255}}, true)
	})
	for i, p := range [][3]float64{{-3.2, 0, 1.1}, {3.2, 0, 1.1}, {0, 1.6, 1.6}, {-1.9, -2.6, 0.9}, {2.3, -2.4, 0.95}} {
		tx, tz, hs := x+p[0]*s, z+p[1]*s, p[2]*s
		B.Draw(tower, V3{tx, y, tz}, V3{s, hs, s}, identity, color.RGBA{}, 0)
		roof := []color.RGBA{{120, 140, 230, 255}, {240, 130, 170, 255}, {255, 200, 90, 255}}[i%3]
		B.Draw(cached("roof"+colKey(roof), func() *Mesh {
			return Lathe([][2]float64{{1.45, 0}, {1.1, 0.8}, {0.5, 2}, {0, 2.9}}, 10, []color.RGBA{darken(roof, 0.85), roof, lighten(roof, 0.15)}, false)
		}), V3{tx, y + 5.7*hs, tz}, V3{s, s, s}, identity, color.RGBA{}, 0.05)
		fy := y + 5.7*hs + 2.9*s
		B.Box(tx-0.03, fy, tz-0.03, tx+0.03, fy+1.2*s, tz+0.03, color.RGBA{120, 100, 80, 255}, sprWhite, 0)
		B.OBox(V3{tx + 0.5*s, fy + 0.9*s, tz}, V3{0.5 * s, 0, 0}, V3{0, 0.25 * s, math.Sin(t*3+float64(i)) * 0.1}, V3{0, 0, 0.02}, roof, 0.1)
	}
	B.BoxTex(x-3.2*s, y, z-0.6*s, x+3.2*s, y+3.4*s, z+0.6*s, color.RGBA{245, 240, 250, 255}, matTop[WClouds], matSide[WClouds], 0)
	B.Billboard(V3{x, y + 8*s, z}, 3*s, sprSoft, alpha(color.RGBA{255, 230, 150, 255}, 0.4), true)
}

func airship(B *Batch, x, y, z, t float64) {
	y += math.Sin(t*0.7) * 0.6
	B.Draw(cached("balloon", func() *Mesh {
		prof := [][2]float64{}
		for k := 0; k <= 10; k++ {
			a := float64(k) / 10 * math.Pi
			prof = append(prof, [2]float64{math.Sin(a) * 1.0, -math.Cos(a) * 1.0})
		}
		cols := []color.RGBA{}
		for k := 0; k < 10; k++ {
			if k%2 == 0 {
				cols = append(cols, color.RGBA{240, 150, 170, 255})
			} else {
				cols = append(cols, color.RGBA{250, 236, 220, 255})
			}
		}
		return Lathe(prof, 12, cols, false)
	}), V3{x, y + 2.6, z}, V3{2.2, 4.6, 2.2}, rotXYZ(0, 0, math.Pi/2), color.RGBA{}, 0)
	B.BoxTex(x-2.4, y-1, z-0.8, x+2.4, y, z+0.8, color.RGBA{150, 100, 70, 255}, matTop[WMeadow], matSide[WMeadow], 0)
	for _, dx := range []float64{-1.8, 1.8} {
		B.Box(x+dx-0.03, y, z-0.03, x+dx+0.03, y+1.2, z+0.03, color.RGBA{90, 70, 60, 255}, sprWhite, 0)
	}
	B.OBox(V3{x - 3, y - 0.5, z}, V3{0.1, 0, 0}, V3{0, math.Cos(t*20) * 0.9, 0}, V3{0, 0, math.Sin(t*20)*0.9 + 0.05}, color.RGBA{90, 80, 70, 255}, 0)
}

func gear(B *Batch, x, y, z, rad, a float64, col color.RGBA, bio *Biome) {
	teeth := int(rad * 2.2)
	for i := 0; i < teeth; i++ {
		an := a + float64(i)/float64(teeth)*2*math.Pi
		dir := V3{math.Cos(an), math.Sin(an), 0}
		perp := V3{-math.Sin(an), math.Cos(an), 0}
		B.OBox(V3{x, y, z}.Add(dir.Mul(rad*0.9)), dir.Mul(rad*0.12), perp.Mul(rad*math.Pi/float64(teeth)*1.05), V3{0, 0, 0.6}, col, 0)
		B.OBox(V3{x, y, z}.Add(dir.Mul(rad*1.08)), dir.Mul(rad*0.08), perp.Mul(rad*math.Pi/float64(teeth)*0.5), V3{0, 0, 0.5}, darken(col, 0.9), 0)
	}
	for k := 0; k < 4; k++ {
		an := a + float64(k)*math.Pi/4
		dir := V3{math.Cos(an), math.Sin(an), 0}
		perp := V3{-math.Sin(an), math.Cos(an), 0}
		B.OBox(V3{x, y, z}, dir.Mul(rad*0.85), perp.Mul(rad*0.07), V3{0, 0, 0.4}, darken(col, 0.85), 0)
	}
	B.Billboard(V3{x, y, z - 0.7}, rad*0.45, sprSphere, lighten(col, 0.2), false)
	B.Billboard(V3{x, y, z - 0.8}, rad*0.2, sprSoft, alpha(bio.Trim, 0.6), true)
}

func boiler(B *Batch, x, z, t float64) {
	B.Draw(cached("boiler", func() *Mesh {
		return Lathe([][2]float64{{2.4, 0}, {2.4, 0.4}, {2.2, 0.5}, {2.2, 2.4}, {2.35, 2.5}, {2.35, 2.7}, {2.2, 2.8}, {2.2, 5.6}, {2.35, 5.7}, {2.35, 5.9}, {2.1, 6}, {1.6, 7}, {0.8, 7.5}, {0.6, 7.6}, {0.6, 10}, {0.75, 10.1}, {0, 10.2}},
			10, []color.RGBA{{90, 92, 100, 255}, {80, 82, 90, 255}, {124, 92, 70, 255}, {110, 112, 120, 255}, {110, 112, 120, 255}, {80, 82, 90, 255}, {132, 98, 74, 255},
				{110, 112, 120, 255}, {110, 112, 120, 255}, {80, 82, 90, 255}, {140, 104, 78, 255}, {150, 112, 82, 255}, {90, 92, 100, 255}, {70, 72, 80, 255}, {90, 92, 100, 255}}, false)
	}), V3{x, -0.6, z}, v3(1), identity, color.RGBA{}, 0)
	B.Billboard(V3{x, 3, z - 2.3}, 1.6, sprSoft, alpha(color.RGBA{255, 120, 40, 255}, 0.7+0.3*math.Sin(t*5)), true)
	B.Billboard(V3{x, 3, z - 2.25}, 0.5, sprSphere, color.RGBA{255, 200, 120, 255}, false)
	for k := 0; k < 5; k++ { // steam
		ph := math.Mod(t*0.5+float64(k)*0.2, 1)
		B.Billboard(V3{x + ph*2, 10 + ph*6, z}, 2+ph*4, sprSoft, color.RGBA{200, 190, 190, uint8(120 * (1 - ph))}, false)
	}
}

// drawPost draws the world-styled corner post at an arena vertex.
func (g *Game) drawPost(B *Batch, L *Level, x, z, wallH float64) {
	bio := L.Biome
	t := L.T
	pulse := 0.6 + 0.4*math.Sin(t*2+z)
	switch bio.ID {
	case WNeon:
		B.Draw(cached("neonPost", func() *Mesh {
			return Lathe([][2]float64{{0.85, 0}, {0.85, 1.6}, {0.6, 1.8}, {0.6, 2.2}, {0.3, 2.3}}, 8,
				[]color.RGBA{{70, 56, 110, 255}, {90, 70, 140, 255}, {255, 90, 220, 255}, {90, 70, 140, 255}}, true)
		}), V3{x, 0, z}, v3(1), identity, color.RGBA{}, 0.1)
		B.Billboard(V3{x, wallH + 1.4, z}, 1.5, sprSoft, alpha(bio.Trim, 0.6*pulse), true)
		B.Billboard(V3{x, wallH + 1.4, z}, 0.5, sprSphere, lighten(bio.Trim, 0.4), false)
	case WMeadow:
		B.Draw(cached("meadowPost", func() *Mesh {
			return Lathe([][2]float64{{0.55, 0}, {0.5, 2.8}, {0.6, 2.9}, {0.6, 3.05}, {0.3, 3.1}}, 6,
				[]color.RGBA{{140, 100, 64, 255}, {120, 86, 56, 255}, {110, 80, 52, 255}}, true)
		}), V3{x, 0, z}, v3(1), identity, color.RGBA{}, 0)
		B.Box(x-0.3, 3.1, z-0.3, x+0.3, 3.7, z+0.3, color.RGBA{60, 50, 40, 255}, sprWhite, 0)
		B.Billboard(V3{x, 3.4, z}, 0.4, sprSphere, color.RGBA{255, 220, 140, 255}, false)
		B.Billboard(V3{x, 3.4, z}, 1.8, sprSoft, alpha(color.RGBA{255, 200, 110, 255}, 0.55*pulse), true)
	case WCaverns:
		B.Draw(cached("cavePost", func() *Mesh {
			return Icosphere(1, 0.2, 9, color.RGBA{80, 86, 116, 255}, color.RGBA{40, 44, 64, 255})
		}), V3{x, 0.3, z}, V3{0.9, 1.4, 0.9}, identity, color.RGBA{}, 0)
		col := bio.Hues[int(math.Abs(x+z))%6]
		B.Draw(cached("crystal"+colKey(col), func() *Mesh { return Crystal(col) }), V3{x, 1.4, z}, V3{0.5, 1.8, 0.5}, rotXYZ(0.15, x, 0.1), color.RGBA{}, 0.3)
		B.Billboard(V3{x, wallH + 1.2, z}, 2.2, sprSoft, alpha(col, 0.35*pulse), true)
	case WRuins:
		B.Draw(cached("ruinPost", func() *Mesh {
			return Lathe([][2]float64{{0.9, 0}, {0.9, 0.3}, {0.7, 0.4}, {0.65, 2.4}, {0.9, 2.6}, {0.9, 2.9}, {0, 2.95}}, 10,
				[]color.RGBA{{176, 150, 110, 255}, {190, 162, 120, 255}, {196, 168, 124, 255}, {220, 180, 80, 255}, {230, 190, 90, 255}, {230, 190, 90, 255}}, false)
		}), V3{x, 0, z}, v3(1), identity, color.RGBA{}, 0)
	case WClouds:
		B.Draw(cached("cloudPost", func() *Mesh {
			return Lathe([][2]float64{{0.8, 0}, {0.8, 2.4}, {1, 2.5}, {1, 2.8}, {0.95, 2.85}}, 10,
				[]color.RGBA{{238, 236, 248, 255}, {220, 220, 236, 255}, {238, 236, 248, 255}}, true)
		}), V3{x, 0, z}, v3(1), identity, color.RGBA{}, 0)
		B.Draw(cached("cloudPostRoof", func() *Mesh {
			return Lathe([][2]float64{{1.1, 0}, {0.6, 0.9}, {0, 1.6}}, 10, []color.RGBA{{110, 140, 230, 255}, {130, 160, 240, 255}}, false)
		}), V3{x, 2.85, z}, v3(1), identity, color.RGBA{}, 0.05)
	case WMachine:
		B.Draw(cached("machPost", func() *Mesh {
			return Lathe([][2]float64{{0.9, 0}, {0.9, 0.3}, {0.7, 0.35}, {0.7, 1.8}, {0.9, 1.85}, {0.9, 2.1}, {0.4, 2.2}}, 8,
				[]color.RGBA{{70, 72, 80, 255}, {60, 62, 70, 255}, {90, 92, 100, 255}, {60, 62, 70, 255}, {200, 120, 40, 255}, {80, 82, 90, 255}}, true)
		}), V3{x, 0, z}, v3(1), identity, color.RGBA{}, 0)
		blink := 0.3 + 0.7*math.Max(0, math.Sin(t*4+z))
		B.Billboard(V3{x, wallH + 1.1, z}, 1.3, sprSoft, alpha(color.RGBA{255, 140, 30, 255}, blink), true)
		B.Billboard(V3{x, wallH + 1.1, z}, 0.38, sprSphere, color.RGBA{255, 190, 90, 255}, false)
	}
}
