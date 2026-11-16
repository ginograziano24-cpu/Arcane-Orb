package main

import (
	"image/color"
	"math"
)

// Sunking's Road: a stamped cobblestone road between flowering topiary
// balls, running up a green hill to a blue fairy-tale castle; a glossy
// violet lake of candy-striped spires on one side; a galaxy overhead and a
// grinning sun watching it all (drawDreamSky).

var dreamWhite = color.RGBA{255, 255, 255, 255}

func (g *Game) drawDreamMeadow(B *Batch, L *Level, t, side float64) {
	tx := dreamTextures()
	low := g.lowFX()

	// the road carries on past the far wall up to the castle gate
	if tx.cobble != nil {
		for z := 33.0; z < 78; z += 9 {
			z1 := z + 9
			w0, w1 := 7.0-(z-33)*0.03, 7.0-(z1-33)*0.03
			v0, v1 := float32((z-33)/36*float64(tx.cobble.h)), float32((z1-33)/36*float64(tx.cobble.h))
			B.QuadUV([4]V3{{-w1, -0.58, z1}, {w1, -0.58, z1}, {w0, -0.58, z}, {-w0, -0.58, z}}, tx.cobble.id,
				[4][2]float32{{tx.cobble.w * 0.25, -v1}, {tx.cobble.w * 0.75, -v1}, {tx.cobble.w * 0.75, -v0}, {tx.cobble.w * 0.25, -v0}},
				B.lit(dreamWhite, V3{0, 1, 0}), [4]float32{1, 1, 1, 1}, false)
		}
	}

	// the violet lake, glossy, with a winding shore of pebbles and glowing
	// reeds; it runs on into the fog so it never shows a far edge
	lx := -side
	shore := func(z float64) float64 { // inner edge, distance from the arena side
		return HalfW + 9 + 2.6*math.Sin(z*0.19) + 1.6*math.Sin(z*0.067+1.1)
	}
	x0, x1 := lx*(HalfW+9), lx*(HalfW+150)
	lake := color.RGBA{86, 26, 190, 255}
	for z := -40.0; z < 150; z += 5 {
		z1 := z + 5
		e0, e1 := shore(z), shore(z1)
		// bright violet by the shore deepening to indigo, in three bands
		bands := [][4]float64{{e0, e1, HalfW + 24, HalfW + 24}, {HalfW + 24, HalfW + 24, HalfW + 50, HalfW + 50}, {HalfW + 50, HalfW + 50, HalfW + 150, HalfW + 150}}
		for j, bd := range bands {
			k := float32(1.5 - 0.22*float64(j))
			p := [4]V3{{lx * bd[1], -0.56, z1}, {lx * bd[3], -0.56, z1}, {lx * bd[2], -0.56, z}, {lx * bd[0], -0.56, z}}
			if tx.sky != nil {
				// the water mirrors the galaxy: far water shows the horizon,
				// near water the high sky, upside down
				uv := func(q V3) [2]float32 {
					u := 0.08 + 0.5*math.Min(1, math.Abs(q.X-x0)/math.Abs(x1-x0)*2.4)
					v := 0.06 + 0.62*math.Min(1, math.Max(0, (q.Z+8)/92))
					return [2]float32{float32(u) * tx.sky.w, float32(v) * tx.sky.h}
				}
				B.QuadUV(p, tx.sky.id, [4][2]float32{uv(p[0]), uv(p[1]), uv(p[2]), uv(p[3])}, color.RGBA{190, 150, 255, 255}, [4]float32{k * 0.8, k * 0.8, k * 0.8, k * 0.8}, false)
				continue
			}
			B.Quad(p, sprWhite, lake, [4]float32{k, k, k, k}, false)
		}
		// the bank: a pink shimmer line, pebbles and a few glowing reeds
		B.Decal(lx*(e0+0.2), z+2.5, -0.53, 0.9, 5.2, sprSoft, alpha(color.RGBA{255, 160, 255, 255}, 0.5), true)
		if z > -16 && z < 110 {
			peb := cached("dreamPebble3", func() *Mesh { return growRock(77, false) })
			for k := 0; k < 3; k++ {
				fk := float64(k)
				pz := z + 0.8 + fk*1.6
				ps := 0.35 + 0.25*math.Abs(math.Sin(pz*1.7))
				B.Draw(peb, V3{lx * (shore(pz) - 0.3 + 0.5*math.Sin(pz*3.1)), -0.62 + ps*0.2, pz}, V3{ps * 0.75, ps * 0.5, ps * 0.6}, rotXYZ(0, pz, 0), color.RGBA{150, 128, 200, 255}, 0)
			}
			if !low && int(z)%10 == 0 {
				for k := 0; k < 4; k++ {
					fk := float64(k)
					rx := lx * (shore(z) + 0.6 + fk*0.35)
					rz := z + fk*0.7
					h := 1.2 + 0.5*math.Sin(fk*2+z)
					sway := 0.12 * math.Sin(t*1.4+fk+z)
					B.Draw(cached("dreamReed"+itoa(k%2), func() *Mesh { return growReed(int64(k%2) + 3) }), V3{rx, -0.6, rz}, V3{1, h, 1}, rotXYZ(sway*0.3, fk*1.9+z, 0), color.RGBA{}, 0.2)
					B.Billboard(V3{rx + sway, -0.6 + h, rz}, 0.28, sprSoft, alpha(color.RGBA{255, 230, 140, 255}, 0.85), true)
				}
			}
		}
	}
	for i := 0; i < 7; i++ {
		z := -4 + float64(i)*16
		sh := 0.25 + 0.15*math.Sin(t*1.3+float64(i))
		B.Decal(lx*(HalfW+30), z, -0.54, 40, 6, sprSoft, alpha(color.RGBA{255, 140, 255, 255}, sh*0.6), true)
	}
	// glossy water: drifting bands of reflected galaxy light and glinting stars
	for i := 0; i < 10; i++ {
		fi := float64(i)
		z := math.Mod(fi*11.3+t*1.2, 112) - 6
		w := 14 + math.Mod(fi*7.1, 18)
		col := []color.RGBA{{90, 160, 255, 255}, {255, 90, 220, 255}, {170, 110, 255, 255}}[i%3]
		B.Decal(lx*(HalfW+14+math.Mod(fi*9.7, 40)), z, -0.545, w, 1.6, sprSoft, alpha(col, 0.35), true)
	}
	if !low {
		for i := 0; i < 22; i++ {
			fi := float64(i)
			tw := math.Max(0, math.Sin(t*2.7+fi*2.1))
			B.Billboard(V3{lx * (HalfW + 11 + math.Mod(fi*17.3, 60)), -0.3, math.Mod(fi*23.7, 100)}, 0.9+tw*1.4, sprSoft, alpha(color.RGBA{255, 240, 255, 255}, 0.8*tw), true)
		}
	}

	// candy-striped spires rising out of the lake
	for i, s := range [][4]float64{{13, 6, 1.6, 10}, {19, 22, 2.1, 13}, {12, 34, 1.3, 7}, {26, 44, 2.4, 15}, {16, 58, 1.7, 10}, {31, 70, 2.2, 13}, {21, 84, 1.9, 11}, {38, 30, 1.8, 11}, {44, 56, 2.6, 16}} {
		x := lx * (HalfW + s[0])
		bob := 0.1 * math.Sin(t*0.8+float64(i)*1.7)
		B.Draw(candyMesh(true), V3{x, -0.6 + bob, s[1]}, V3{s[2], s[3], s[2]}, rotXYZ(0, float64(i)*0.9, 0), color.RGBA{}, 0.15)
		if !low {
			B.Decal(x, s[1], -0.53, s[2]*3.2, s[2]*3.2, sprSoft, alpha(color.RGBA{255, 90, 150, 255}, 0.25), true)
		}
	}

	// topiary flower balls lining the near side and both banks
	balls := [][3]float64{}
	for z := 2.0; z < 62; z += 9.5 {
		balls = append(balls, [3]float64{side * (HalfW + 5.2), z, 1.7})
		balls = append(balls, [3]float64{-side * (HalfW + 5.6), z + 3, 1.5})
	}
	for z := 8.0; z < 70; z += 14 {
		balls = append(balls, [3]float64{side * (HalfW + 12), z, 2.3})
	}
	for z := 40.0; z < 80; z += 10 {
		for _, sx := range []float64{-1, 1} {
			balls = append(balls, [3]float64{sx * (8.4 - (z-33)*0.03), z, 1.4})
		}
	}
	// three topiary shapes and three blossom tints, so the banks read as a
	// tended garden rather than a row of identical forms. Every form is
	// grown from overlapping leaves and five-petal blossoms (flora.go).
	tints := []color.RGBA{{255, 170, 214, 255}, {214, 176, 255, 255}, {255, 196, 170, 255}}
	topi := func(shape, v int, bloom color.RGBA, far bool) *floraSet {
		key := "topi" + itoa(shape) + "-" + itoa(v) + colKey(bloom)
		if far {
			key += "L"
		}
		return floraGet(key, func() *floraSet { return growTopiary(int64(31+v*7+shape), shape, palDream, bloom, far) })
	}
	// one form: its stem and leaf shells, at full or light detail
	form := func(add func(s *floraSet, pos, scale V3, ry float64), i int, b [3]float64, far bool) {
		r := b[2]
		x, z := b[0], b[1]
		bloom := tints[(i*7/3)%3]
		ry := float64(i) * 1.3
		switch i % 5 {
		case 1, 3: // three clipped tiers on a barked stem
			add(floraGet("topiStem", func() *floraSet { return growStem(3, 3.0) }), V3{x, -0.6, z}, v3(r*0.9), ry)
			for k, f := range []float64{0.62, 0.48, 0.34} {
				y := -0.6 + r*(0.62+float64(k)*1.0)
				add(topi(0, k, bloom, far), V3{x, y, z}, V3{r * f, r * f * 0.92, r * f}, ry+float64(k))
			}
		case 4: // a clipped spire
			add(topi(1, i%2, bloom, far), V3{x, -0.6, z}, v3(r*0.8), ry)
		default:
			add(topi(0, i%3, bloom, far), V3{x, -0.6 + r*0.92, z}, V3{r, r * 0.95, r}, ry)
		}
	}
	// the forms beyond the arena never move and sit wholly in the cached
	// background: merged into one mesh per chamber
	farZ := FarZ + 6.0
	for band := 0; band < 5; band++ {
		z0, z1 := farZ+float64(band)*12, farZ+float64(band+1)*12
		// in depth bands, so each group is small enough for the cache to skip
		mergedStatic(B, L, "dreamTopiFar"+itoa(int(side))+"-"+itoa(band), func(add func(s *floraSet, pos, scale V3, ry float64)) {
			for i, b := range balls {
				if b[1] >= z0 && b[1] < z1 && b[0]*side > 0 {
					form(add, i, b, true)
				}
			}
		})
		mergedStatic(B, L, "dreamTopiFarX"+itoa(int(side))+"-"+itoa(band), func(add func(s *floraSet, pos, scale V3, ry float64)) {
			for i, b := range balls {
				if b[1] >= z0 && b[1] < z1 && b[0]*side <= 0 {
					form(add, i, b, true)
				}
			}
		})
	}
	for i, b := range balls {
		r := b[2]
		x, z := b[0], b[1]
		if z >= farZ {
			continue
		}
		if !B.objBegin(x, r*1.5, z, r*2+0.5) {
			B.objEnd()
			continue
		}
		form(func(s *floraSet, pos, scale V3, ry float64) {
			drawFlora(B, s, pos, scale, ry, color.RGBA{}, t, 0)
		}, i, b, floraFar(B, V3{x, 0, z}))
		B.objEnd()
	}

	// blossom trees fill the far meadow: barked trunks, spreading limbs and
	// pink canopies of petals and flowers
	pink := color.RGBA{255, 236, 246, 255}
	for i, p := range [][3]float64{{22, 36, 1.1}, {30, 50, 1.4}, {40, 40, 1.2}, {26, 66, 1.5}, {44, 60, 1.3}, {36, 80, 1.6}, {52, 74, 1.4}, {18, 86, 1.2}, {58, 48, 1.5}} {
		x, z, sc := side*p[0], p[1], p[2]
		if !B.objBegin(x, 4*sc, z, 4*sc+0.5) {
			B.objEnd()
			continue
		}
		set := treeVariantLOD("blossom", int64(i), floraLOD(B, V3{x, 0, z}), func(sd int64, lite bool) *floraSet {
			return growTree(sd+40, barkDream, palBlossom, lite, &pink)
		})
		drawFlora(B, set, V3{x, -0.55, z}, v3(sc*0.85), float64(i)*2.1, color.RGBA{}, t+float64(i), 0.012)
		B.objEnd()
	}
	// glowing flower patches and drifting fairy lights over the grass
	for i := 0; i < 26; i++ {
		fi := float64(i)
		x := side * (HalfW + 6 + math.Mod(fi*7.7, 44))
		z := math.Mod(fi*13.1, 88) - 2
		col := []color.RGBA{{255, 80, 200, 255}, {120, 200, 255, 255}, {255, 210, 90, 255}}[i%3]
		B.Decal(x, z, -0.57, 3.5+math.Mod(fi*3.3, 4), 2.6+math.Mod(fi*2.1, 3), sprSoft, alpha(col, 0.22), true)
		if !low {
			for k := 0; k < 4; k++ {
				fk := float64(k)
				B.Billboard(V3{x + math.Sin(fi+fk*1.7)*1.2, -0.35 + 0.08*fk, z + math.Cos(fi*1.3+fk)*0.9}, 0.32, sprSphere, col, false)
			}
		}
	}
	if !low {
		for i := 0; i < 16; i++ {
			fi := float64(i)
			x := side*(HalfW+4+math.Mod(fi*9.1, 40)) + math.Sin(t*0.4+fi)*1.5
			z := math.Mod(fi*17.9, 80) + math.Cos(t*0.3+fi)*1.2
			y := 1.2 + math.Mod(fi*1.7, 3) + math.Sin(t*1.1+fi*2)*0.4
			tw := 0.6 + 0.4*math.Sin(t*4+fi*3)
			B.Billboard(V3{x, y, z}, 0.9*tw+0.3, sprSoft, alpha(color.RGBA{255, 240, 140, 255}, 0.8*tw), true)
		}
	}
	// light sources and their bloom: floating paper lanterns in candy
	// colours drifting over both banks, glowing stars crowning the candy
	// spires, and a soft pink glow inside every blossom canopy
	lantern := dreamLanternMesh()
	hues := []color.RGBA{{255, 120, 200, 255}, {120, 220, 255, 255}, {255, 210, 110, 255}, {190, 140, 255, 255}, {120, 255, 190, 255}}
	for i := 0; i < 14; i++ {
		fi := float64(i)
		sx := side
		if i%2 == 1 {
			sx = -side
		}
		x := sx*(HalfW+5+math.Mod(fi*6.7, 22)) + math.Sin(t*0.3+fi)*0.8
		z := 4 + math.Mod(fi*9.3, 70)
		y := 3.2 + math.Mod(fi*1.9, 3) + math.Sin(t*0.7+fi*1.3)*0.35
		c := hues[i%len(hues)]
		fl := 0.85 + 0.15*math.Sin(t*3.1+fi*2.3)
		B.Draw(lantern, V3{x, y, z}, v3(1.1), rotXYZ(0, t*0.2+fi, math.Sin(t*0.8+fi)*0.08), c, float32(0.35*fl))
		B.Billboard(V3{x, y, z}, 2.6, sprSoft, alpha(c, 0.55*fl), true)
		if !low {
			B.Billboard(V3{x, y, z}, 6.5, sprSoft, alpha(c, 0.16*fl), true)
			B.Decal(x, z, -0.56, 4.5, 4.5, sprSoft, alpha(c, 0.12*fl), true)
		}
	}
	for i, s := range [][4]float64{{13, 6, 1.6, 10}, {19, 22, 2.1, 13}, {12, 34, 1.3, 7}, {26, 44, 2.4, 15}, {16, 58, 1.7, 10}, {31, 70, 2.2, 13}, {21, 84, 1.9, 11}, {38, 30, 1.8, 11}, {44, 56, 2.6, 16}} {
		x := lx * (HalfW + s[0])
		tip := V3{x, -0.6 + s[3] + 0.4, s[1]}
		c := hues[(i+2)%len(hues)]
		tw := 0.7 + 0.3*math.Sin(t*2.2+float64(i)*1.9)
		oc := tip.Add(V3{0, 0.7, 0})
		drawDreamOrb(B, oc, 1.1, c, t, float64(i)*2.1, false, !low)
		B.Billboard(oc, 3.4*tw, sprSoft, alpha(c, 0.6), true)
		if !low {
			B.Billboard(oc, 8*tw, sprSoft, alpha(c, 0.14), true)
		}
	}
	if !low {
		for i, p := range [][3]float64{{22, 36, 1.1}, {30, 50, 1.4}, {40, 40, 1.2}, {26, 66, 1.5}, {44, 60, 1.3}, {36, 80, 1.6}, {52, 74, 1.4}, {18, 86, 1.2}, {58, 48, 1.5}} {
			B.Billboard(V3{side * p[0], 4.6 * p[2], p[1]}, 7*p[2], sprSoft, alpha(color.RGBA{255, 130, 200, 255}, 0.13+0.04*math.Sin(t+float64(i))), true)
		}
	}
	// (the castle itself is painted into the galaxy backdrop at the end of the road)

	// twinkling star sparkles drifting over the scene
	if !low {
		for i := 0; i < 18; i++ {
			fi := float64(i)
			x := math.Sin(fi*2.3+t*0.05)*42 + side*fi*0.3
			z := 10 + math.Mod(fi*13.7, 80)
			y := 4 + math.Mod(fi*5.3, 12) + math.Sin(t*0.7+fi)*0.6
			tw := 0.5 + 0.5*math.Sin(t*3.1+fi*1.9)
			col := []color.RGBA{{255, 220, 120, 255}, {255, 140, 230, 255}, {160, 220, 255, 255}}[i%3]
			B.Billboard(V3{x, y, z}, 1.4+tw, sprSoft, alpha(col, 0.55*tw), true)
		}
	}
}

// dreamPine is a stack of three dark faceted cones on a stub of trunk.
func dreamPine(B *Batch, x, z, s float64, seed int) {
	B.Draw(cached("dreamTrunk", func() *Mesh {
		return Lathe([][2]float64{{0.45, 0}, {0.35, 2}}, 6, []color.RGBA{{90, 50, 40, 255}}, false)
	}), V3{x, -0.6, z}, v3(s), identity, color.RGBA{}, 0)
	for k := 0; k < 3; k++ {
		fk := float64(k)
		B.Draw(cached("dreamPineTier", func() *Mesh {
			return Lathe([][2]float64{{1, 0}, {0, 1}}, 7, []color.RGBA{{24, 92, 60, 255}}, false)
		}), V3{x, -0.6 + (1.6+fk*2.1)*s, z}, V3{(3.4 - fk*0.8) * s, 4.2 * s, (3.4 - fk*0.8) * s}, rotXYZ(0, float64(seed)+fk, 0), color.RGBA{}, 0)
	}
}

// dreamCastle is the blue castle with red cone roofs and lit windows.
func dreamCastle(B *Batch, L *Level, tx dreamTexSet, cx, cz, t float64, low bool) {
	// the hill
	B.DrawTex(cached("dreamHill", func() *Mesh {
		return Icosphere(2, 0.05, 23, dreamWhite, color.RGBA{200, 230, 200, 255})
	}), V3{cx, -9, cz + 6}, V3{48, 14, 30}, identity, tx.grass, 6, uvBox, 0)
	base := 4.6
	box := func(x, y, z, sx, sy, sz float64) {
		B.DrawTex(unitBox(), V3{x, y, z}, V3{sx, sy, sz}, identity, tx.brick, 3, uvBox, 0)
	}
	// curtain wall and keep
	box(cx, base+3, cz, 12, 3, 4)
	box(cx, base+8, cz+3, 7, 8, 4)
	// towers with cone roofs
	tower := func(x, z, r, h float64) {
		B.DrawTex(dreamCylinder(10), V3{x, base, z}, V3{r, h, r}, identity, tx.brick, 3, uvCyl, 0)
		B.DrawTex(dreamCone(10), V3{x, base + h, z}, V3{r * 1.35, r * 2.6, r * 1.35}, identity, tx.roof, 2, uvCyl, 0)
		B.Billboard(V3{x, base + h + r*2.6 + 0.4, z}, 1.2, sprSoft, alpha(color.RGBA{255, 220, 120, 255}, 0.5), true)
	}
	tower(cx-12.5, cz-3, 2.4, 11)
	tower(cx+12.5, cz-3, 2.4, 11)
	tower(cx-7, cz+5, 2.1, 17)
	tower(cx+7, cz+5, 2.1, 17)
	tower(cx, cz+4, 3, 24)
	// glowing windows and the gate
	win := color.RGBA{255, 214, 90, 255}
	pulse := 0.85 + 0.15*math.Sin(t*2.3)
	for _, w := range [][3]float64{{-8, 6, -4.05}, {-4, 6, -4.05}, {4, 6, -4.05}, {8, 6, -4.05}, {-3, 12, -1.05}, {3, 12, -1.05}, {0, 16, -1.05}, {-12.5, 10, -5.45}, {12.5, 10, -5.45}, {0, 22, 0.95}} {
		x, y, z := cx+w[0], base+w[1], cz+w[2]
		B.Box(x-0.6, y-0.9, z-0.05, x+0.6, y+0.9, z+0.05, win, sprWhite, float32(1.2*pulse))
		if !low {
			B.Billboard(V3{x, y, z - 0.3}, 2.4, sprSoft, alpha(win, 0.35*pulse), true)
		}
	}
	B.Box(cx-1.8, base, cz-4.1, cx+1.8, base+3.6, cz-4.0, color.RGBA{255, 170, 60, 255}, sprWhite, 1.4)
	B.Billboard(V3{cx, base + 1.8, cz - 4.6}, 5, sprSoft, alpha(color.RGBA{255, 180, 80, 255}, 0.4*pulse), true)
}

// dreamHedge draws one wall section as part of a continuous enclosure: a
// blue castle-stone curb with a glowing gold trim along the arena edge, and
// on it a rounded hedge of pink flowers. Where the orb last struck, the
// hedge squashes back and wobbles (dreamWallHit spawns the petals).
func dreamHedge(B *Batch, tx dreamTexSet, p0, p1 [2]float64, nx, nz, h float64, e *envState) {
	dx, dz := p1[0]-p0[0], p1[1]-p0[1]
	l := math.Hypot(dx, dz)
	ang := math.Atan2(dx, dz)
	mx, mz := (p0[0]+p1[0])/2, (p0[1]+p1[1])/2
	rot := rotXYZ(0, ang, 0)
	// how hard this section is still ringing from the last hit
	wob := 0.0
	if e != nil && e.hitT < 1.2 {
		d := math.Hypot(e.hitX-mx, e.hitZ-mz)
		if near := 1 - d/3.2; near > 0 {
			wob = near * math.Exp(-e.hitT*5) * math.Cos(e.hitT*26)
		}
	}
	back := 0.9 + 0.22*math.Max(0, wob) // pushed away from the arena
	cx, cz := mx-nx*back, mz-nz*back
	// curb
	curbH := 0.42
	B.Draw(bevelBox(0.12), V3{mx - nx*0.8, curbH / 2, mz - nz*0.8}, V3{0.85, curbH / 2, l/2 + 0.02}, rot, color.RGBA{150, 118, 170, 255}, 0.15)
	// glowing gold trim on the curb's inner edge, and its light on the floor
	B.Draw(unitBox(), V3{mx - nx*0.02, curbH + 0.03, mz - nz*0.02}, V3{0.07, 0.04, l/2 + 0.02}, rot, color.RGBA{255, 210, 90, 255}, 1.1)
	B.Quad([4]V3{{p0[0], -0.585, p0[1]}, {p1[0], -0.585, p1[1]}, {p1[0] + nx*0.9, -0.585, p1[1] + nz*0.9}, {p0[0] + nx*0.9, -0.585, p0[1] + nz*0.9}},
		sprSoft, color.RGBA{255, 160, 220, 255}, [4]float32{0.35, 0.35, 0.35, 0.35}, true)
	// hedge: tiles of clipped leaves with blossoms along the top roll,
	// squashed back where the orb struck
	sq := 1 - 0.14*wob
	top := (h + 0.55) * sq
	tiles := max(1, int(math.Round(l/2.0)))
	tl := l / float64(tiles)
	ux, uz := dx/l, dz/l
	for k := 0; k < tiles; k++ {
		off := -l/2 + tl*(float64(k)+0.5)
		set := floraGet("hedgeTile"+itoa(k%3), func() *floraSet {
			return growTopiary(int64(61+k%3), 2, palDream, color.RGBA{255, 170, 214, 255}, false)
		})
		drawFlora(B, set, V3{cx + ux*off, curbH - 0.02, cz + uz*off}, V3{1 + 0.12*wob, top - curbH + 0.02, tl * 1.08}, ang, color.RGBA{}, 0, 0)
	}
}

// dreamWallHit makes the hedge react where the orb struck it: a squash
// and wobble (drawn by dreamHedge), a burst of petals and a pink flash.
func (L *Level) dreamWallHit(x, z float64, side int) {
	e := &L.env
	e.hitX, e.hitZ, e.hitT = x, z, 0
	n := 10
	if L.g.settings.Particles == 0 {
		n = 4
	}
	n = L.burstBudget(n)
	out := V3{-1, 0, 0}
	switch side {
	case 1:
		out = V3{1, 0, 0}
	case 2:
		out = V3{0, 0, 1}
	}
	pal := []color.RGBA{{255, 90, 200, 255}, {255, 160, 230, 255}, {255, 240, 250, 255}, {255, 210, 80, 255}}
	for i := 0; i < n; i++ {
		a := L.rng.Float64() * 2 * math.Pi
		s := 2 + L.rng.Float64()*3.5
		v := V3{math.Cos(a)*s*0.6 - out.X*3, 2.5 + L.rng.Float64()*3, math.Sin(a)*s*0.6 - out.Z*3}
		L.Parts = append(L.Parts, Particle{P: V3{x + out.X*0.4, 0.9, z + out.Z*0.4}, V: v,
			Life: 0.7 + L.rng.Float64()*0.5, Max: 1.2, Size: 0.22 + L.rng.Float64()*0.2, Col: pal[i%len(pal)], Kind: pSpark, Grav: 3})
	}
	L.Rings = append(L.Rings, &Ring{X: x, Z: z, MaxR: 1.8, Life: 0.35, Max: 0.35, Col: color.RGBA{255, 120, 220, 255}})
}

// dreamPost is a candy-cane post crowned with a crystal orrery.
func dreamPost(B *Batch, tx dreamTexSet, x, z, t float64) {
	B.Draw(candyMesh(false), V3{x, 0, z}, V3{0.5, 3.4, 0.5}, identity, color.RGBA{}, 0.1)
	pulse := 0.7 + 0.3*math.Sin(t*2+z)
	drawDreamOrb(B, V3{x, 4.25, z}, 0.75, color.RGBA{255, 150, 240, 255}, t, x*0.7+z, true, true)
	B.Billboard(V3{x, 4.25, z}, 2.4, sprSoft, alpha(color.RGBA{255, 110, 230, 255}, 0.6*pulse), true)
}

// dreamGround is the textured meadow around the arena, stopping short of
// the horizon so the painted hillside under the castle shows beyond it
// rather than a band of distant flat ground.
func dreamGround(B *Batch, t *bakedTex, tint color.RGBA, y float64) {
	drawBakedGround(B, t, tint, y)
}
