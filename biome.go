package main

import (
	"image"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"

	"orb/levelgen"
)

// Biome describes one world's complete art direction: palette, lighting,
// materials, sky, scenery, effects and paddle trim.
type Biome struct {
	ID        int
	Name      string
	Subtitle  string
	Hues      [6]color.RGBA
	Floor     color.RGBA
	FloorAlt  color.RGBA
	Outer     color.RGBA
	Wall      color.RGBA
	Trim      color.RGBA
	Fog       color.RGBA
	SkyTop    color.RGBA
	SkyBottom color.RGBA
	Ambient   int // ambient particle style (see world IDs)
	Prop      int
	MusicRoot float64
	BossName  string

	SkyKind    int
	LightDir   V3
	LightCol   [3]float32
	AmbCol     [3]float32
	Neon       bool // emissive trims and glowing grid
	BrickMat   int
	FloorMat   int
	Taper      float64 // brick top scale (1 = box, <1 faceted)
	PaddleBody color.RGBA
	PaddleCap  color.RGBA
	FogNear    float64
	FogFar     float64
}

const (
	WNeon = iota
	WMeadow
	WCaverns
	WRuins
	WClouds
	WMachine
	NumWorlds
)

var worlds = []Biome{
	{
		ID: WNeon, Name: "Neon Grove", Subtitle: "a glowing garden of light",
		Hues:  [6]color.RGBA{{255, 70, 200, 255}, {40, 230, 255, 255}, {170, 90, 255, 255}, {255, 220, 80, 255}, {255, 130, 60, 255}, {90, 255, 170, 255}},
		Floor: color.RGBA{40, 230, 255, 255}, FloorAlt: color.RGBA{255, 70, 200, 255}, Outer: color.RGBA{14, 6, 28, 255},
		Wall: color.RGBA{58, 44, 92, 255}, Trim: color.RGBA{255, 80, 210, 255},
		Fog: color.RGBA{70, 18, 80, 255}, SkyTop: color.RGBA{18, 6, 44, 255}, SkyBottom: color.RGBA{255, 70, 150, 255},
		SkyKind: WNeon, LightDir: V3{-0.35, 0.85, -0.4}, LightCol: [3]float32{0.75, 0.7, 0.85}, AmbCol: [3]float32{0.45, 0.38, 0.62},
		Neon: true, BrickMat: WNeon, FloorMat: WNeon, Taper: 1,
		PaddleBody: color.RGBA{70, 60, 110, 255}, PaddleCap: color.RGBA{220, 180, 90, 255}, FogNear: 42, FogFar: 100,
	},
	{
		ID: WMeadow, Name: "Sunset Meadow", Subtitle: "warm hills at the end of the day",
		Hues:  [6]color.RGBA{{196, 132, 78, 255}, {232, 186, 96, 255}, {150, 190, 90, 255}, {222, 110, 90, 255}, {240, 214, 150, 255}, {120, 160, 200, 255}},
		Floor: color.RGBA{214, 170, 120, 255}, FloorAlt: color.RGBA{190, 120, 90, 255}, Outer: color.RGBA{96, 140, 70, 255},
		Wall: color.RGBA{150, 104, 70, 255}, Trim: color.RGBA{255, 200, 120, 255},
		Fog: color.RGBA{134, 170, 96, 255}, SkyTop: color.RGBA{60, 90, 160, 255}, SkyBottom: color.RGBA{255, 170, 110, 255},
		SkyKind: WMeadow, LightDir: V3{-0.7, 0.55, 0.2}, LightCol: [3]float32{0.92, 0.75, 0.52}, AmbCol: [3]float32{0.48, 0.52, 0.66},
		BrickMat: WMeadow, FloorMat: WMeadow, Taper: 1,
		PaddleBody: color.RGBA{120, 80, 50, 255}, PaddleCap: color.RGBA{210, 170, 90, 255}, FogNear: 45, FogFar: 150,
	},
	{
		ID: WCaverns, Name: "Crystal Caverns", Subtitle: "deep below, the stones are singing",
		Hues:  [6]color.RGBA{{90, 200, 255, 255}, {170, 130, 255, 255}, {120, 255, 220, 255}, {255, 140, 220, 255}, {200, 230, 255, 255}, {80, 120, 230, 255}},
		Floor: color.RGBA{70, 80, 110, 255}, FloorAlt: color.RGBA{56, 64, 92, 255}, Outer: color.RGBA{18, 20, 36, 255},
		Wall: color.RGBA{60, 64, 90, 255}, Trim: color.RGBA{120, 230, 255, 255},
		Fog: color.RGBA{22, 36, 66, 255}, SkyTop: color.RGBA{6, 8, 18, 255}, SkyBottom: color.RGBA{30, 50, 90, 255},
		SkyKind: WCaverns, LightDir: V3{0.3, 0.8, -0.5}, LightCol: [3]float32{0.5, 0.78, 0.9}, AmbCol: [3]float32{0.3, 0.32, 0.5},
		BrickMat: WCaverns, FloorMat: WCaverns, Taper: 0.72,
		PaddleBody: color.RGBA{50, 60, 90, 255}, PaddleCap: color.RGBA{140, 230, 255, 255}, FogNear: 44, FogFar: 100,
	},
	{
		ID: WRuins, Name: "Ancient Ruins", Subtitle: "something lived here long before you",
		Hues:  [6]color.RGBA{{214, 184, 130, 255}, {70, 190, 180, 255}, {200, 150, 70, 255}, {176, 120, 80, 255}, {230, 210, 160, 255}, {90, 140, 90, 255}},
		Floor: color.RGBA{214, 186, 140, 255}, FloorAlt: color.RGBA{190, 160, 116, 255}, Outer: color.RGBA{196, 160, 110, 255},
		Wall: color.RGBA{176, 150, 110, 255}, Trim: color.RGBA{230, 190, 90, 255},
		Fog: color.RGBA{230, 200, 150, 255}, SkyTop: color.RGBA{70, 150, 200, 255}, SkyBottom: color.RGBA{240, 220, 180, 255},
		SkyKind: WRuins, LightDir: V3{0.55, 0.75, -0.35}, LightCol: [3]float32{1.0, 0.9, 0.7}, AmbCol: [3]float32{0.42, 0.38, 0.4},
		BrickMat: WRuins, FloorMat: WRuins, Taper: 1,
		PaddleBody: color.RGBA{60, 120, 120, 255}, PaddleCap: color.RGBA{220, 180, 80, 255}, FogNear: 50, FogFar: 170,
	},
	{
		ID: WClouds, Name: "Cloud Kingdom", Subtitle: "castles drifting above the sky",
		Hues:  [6]color.RGBA{{150, 200, 255, 255}, {255, 180, 210, 255}, {255, 220, 120, 255}, {200, 170, 255, 255}, {250, 250, 255, 255}, {140, 230, 200, 255}},
		Floor: color.RGBA{214, 216, 232, 255}, FloorAlt: color.RGBA{196, 202, 228, 255}, Outer: color.RGBA{180, 210, 245, 255},
		Wall: color.RGBA{230, 232, 245, 255}, Trim: color.RGBA{255, 210, 110, 255},
		Fog: color.RGBA{220, 220, 238, 255}, SkyTop: color.RGBA{90, 160, 240, 255}, SkyBottom: color.RGBA{220, 235, 255, 255},
		SkyKind: WClouds, LightDir: V3{-0.2, 0.9, -0.3}, LightCol: [3]float32{0.62, 0.6, 0.55}, AmbCol: [3]float32{0.62, 0.66, 0.78},
		BrickMat: WClouds, FloorMat: WClouds, Taper: 1,
		PaddleBody: color.RGBA{240, 240, 250, 255}, PaddleCap: color.RGBA{255, 200, 100, 255}, FogNear: 40, FogFar: 130,
	},
	{
		ID: WMachine, Name: "Machine Works", Subtitle: "the whole place is one enormous engine",
		Hues:  [6]color.RGBA{{196, 112, 64, 255}, {120, 140, 170, 255}, {240, 190, 60, 255}, {90, 96, 110, 255}, {230, 120, 40, 255}, {100, 170, 210, 255}},
		Floor: color.RGBA{86, 90, 100, 255}, FloorAlt: color.RGBA{70, 74, 84, 255}, Outer: color.RGBA{30, 30, 36, 255},
		Wall: color.RGBA{80, 84, 96, 255}, Trim: color.RGBA{255, 150, 40, 255},
		Fog: color.RGBA{40, 32, 30, 255}, SkyTop: color.RGBA{18, 16, 20, 255}, SkyBottom: color.RGBA{150, 70, 30, 255},
		SkyKind: WMachine, LightDir: V3{0.4, 0.7, -0.6}, LightCol: [3]float32{0.9, 0.6, 0.35}, AmbCol: [3]float32{0.3, 0.34, 0.44},
		BrickMat: WMachine, FloorMat: WMachine, Taper: 1,
		PaddleBody: color.RGBA{70, 74, 84, 255}, PaddleCap: color.RGBA{230, 140, 40, 255}, FogNear: 44, FogFar: 110,
	},
}

type ActDef struct {
	Name, Subtitle string
	Signature      int
	BossName       string
	Pool           []int
}

var acts = []ActDef{
	{"The Outer Realms", "Act I", WNeon, "The Prism Colossus", []int{WNeon, WMeadow, WRuins, WClouds}},
	{"The Deep Realms", "Act II", WMachine, "The Crucible Golem", []int{WMachine, WRuins, WCaverns, WMeadow}},
	{"The High Realms", "Act III", WCaverns, "The Crystal Warden", []int{WCaverns, WClouds, WNeon, WMachine}},
}

// biomes keeps the act-indexed signature worlds for screens outside levels.
var biomes = []Biome{worlds[WNeon], worlds[WMachine], worlds[WCaverns]}

func init() {
	for i := range acts {
		biomes[i].BossName = acts[i].BossName
		biomes[i].Subtitle = acts[i].Subtitle
		biomes[i].Name = acts[i].Name
	}
}

// makeSky renders a synthwave sunset: gradient, stars, a striped sun and
// neon-edged mountains. The horizon sits at skyHorizon of the image height.
const skyHorizon = 0.72

func makeSynthSky(b *Biome, seed int64) *ebiten.Image {
	const w, h = 960, 540
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	r := rand.New(rand.NewSource(seed))
	style := int(uint64(seed^0x77177)>>3) % 4
	hy := float64(h) * skyHorizon
	sunR := 85.0 + r.Float64()*40
	sunX := float64(w) * (0.3 + r.Float64()*0.4)
	sunY := hy - sunR*(0.3+r.Float64()*0.3)
	sunTop := hueShift(color.RGBA{255, 232, 100, 255}, (r.Float64()-0.5)*40)
	sunBot := b.Trim
	// silhouettes: back layer (soft) and front layer (dark with neon rim)
	back := make([]float64, w)
	front := make([]float64, w)
	ph := r.Float64() * 10
	lit := make([]bool, w*h)
	for x := 0; x < w; x++ {
		fx := float64(x)
		d := math.Abs(fx-sunX) / 260
		keep := math.Min(1, 0.25+d*d)
		v, v2 := 0.0, 0.0
		for k, a := range []float64{60, 30, 14, 6} {
			v += a * math.Abs(math.Sin(fx*0.006*float64(k+1)+ph*float64(k+1)))
		}
		for k, a := range []float64{36, 18, 9} {
			v2 += a * math.Abs(math.Sin(fx*0.009*float64(k+1)+ph*2.3+float64(k)))
		}
		back[x] = v2 * math.Min(1, 0.4+d*d)
		front[x] = v * keep * 0.8
	}
	switch style {
	case 1: // neon city skyline
		for x := 0; x < w; {
			bw := 14 + r.Intn(34)
			bh := 20 + r.Float64()*110
			if math.Abs(float64(x)-sunX) < sunR*0.9 {
				bh *= 0.35
			}
			for i := x; i < x+bw && i < w; i++ {
				front[i] = bh
				back[i] = bh*0.6 + 30
			}
			// windows
			for wy := int(hy - bh + 6); wy < int(hy)-4; wy += 7 {
				for wx := x + 3; wx < x+bw-3 && wx < w; wx += 6 {
					if r.Float64() < 0.35 {
						for yy := wy; yy < wy+3; yy++ {
							for xx := wx; xx < wx+2 && xx < w; xx++ {
								if yy >= 0 {
									lit[yy*w+xx] = true
								}
							}
						}
					}
				}
			}
			x += bw + r.Intn(4)
		}
	case 2: // pyramids
		for i := range front {
			front[i], back[i] = 0, 0
		}
		for k := 0; k < 5; k++ {
			px := r.Float64() * w
			ph := 40 + r.Float64()*90
			for x := 0; x < w; x++ {
				v := ph - math.Abs(float64(x)-px)*0.9
				if k%2 == 0 {
					front[x] = math.Max(front[x], v)
				} else {
					back[x] = math.Max(back[x], v*1.2)
				}
			}
		}
	case 3: // low hills under a ringed planet
		for i := range front {
			front[i] *= 0.4
			back[i] *= 0.5
		}
	}
	plX, plY, plR := float64(w)*(0.15+r.Float64()*0.7), hy*0.35, 50+r.Float64()*30
	plCol := hueShift(b.FloorAlt, 40)
	for y := 0; y < h; y++ {
		fy := float64(y)
		f := math.Min(1, fy/hy)
		base := mixc(b.SkyTop, b.SkyBottom, math.Pow(f, 1.8))
		for x := 0; x < w; x++ {
			fx := float64(x)
			c := base
			if fy > hy {
				c = mixc(b.SkyBottom, b.Outer, math.Min(1, (fy-hy)/40))
			}
			if style == 3 {
				dx, dy := fx-plX, fy-plY
				d := math.Hypot(dx, dy)
				// ring behind/in front
				re := math.Abs(math.Hypot(dx, dy*3.2)-plR*1.75) < 5
				if d < plR {
					shade := 0.45 + 0.55*math.Max(0, -(dx+dy)/(plR*1.4)+0.3)
					c = mixc(darken(plCol, 0.5), plCol, shade)
					if math.Mod(fy*0.15+dx*0.02, 1) < 0.3 {
						c = darken(c, 0.85)
					}
					if re && dy > 0 {
						c = lighten(plCol, 0.4)
					}
				} else if re {
					c = mixc(c, lighten(plCol, 0.3), 0.8)
				} else if d < plR*1.4 {
					c = mixc(c, plCol, (1-(d-plR)/(plR*0.4))*0.25)
				}
			}
			// sun with widening horizontal gaps on its lower part
			d := math.Hypot(fx-sunX, fy-sunY)
			if d < sunR*1.9 {
				c = mixc(c, b.SkyBottom, math.Max(0, 1-d/(sunR*1.9))*0.5)
			}
			if d < sunR && fy < hy {
				t := (fy - (sunY - sunR)) / (2 * sunR)
				stripe := true
				if t > 0.45 {
					band := math.Mod((t-0.45)*30, 1)
					stripe = band > (t-0.45)*1.6
				}
				if stripe {
					c = mixc(sunTop, sunBot, t)
				}
			}
			if fy < hy {
				if fy > hy-back[x] {
					c = mixc(darken(b.SkyBottom, 0.35), b.Outer, 0.4)
					if fy < hy-back[x]+2 {
						c = lighten(b.FloorAlt, 0.2)
					}
				}
				if fy > hy-front[x] {
					c = darken(b.Outer, 1.6)
					if fy < hy-front[x]+2.5 {
						c = b.Floor
					}
					if lit[y*w+x] {
						c = mixc(b.Floor, color.RGBA{255, 240, 200, 255}, 0.4)
					}
				}
			}
			img.SetRGBA(x, y, c)
		}
	}
	for i := 0; i < 260; i++ {
		x, y := r.Intn(w), r.Intn(int(hy*0.7))
		v := uint8(140 + r.Intn(115))
		img.SetRGBA(x, y, color.RGBA{v, v, 255, 255})
	}
	return ebiten.NewImageFromImage(img)
}

func mixc(a, b color.RGBA, t float64) color.RGBA {
	t = math.Max(0, math.Min(1, t))
	return color.RGBA{
		uint8(float64(a.R)*(1-t) + float64(b.R)*t),
		uint8(float64(a.G)*(1-t) + float64(b.G)*t),
		uint8(float64(a.B)*(1-t) + float64(b.B)*t),
		uint8(float64(a.A)*(1-t) + float64(b.A)*t)}
}

func lighten(c color.RGBA, f float64) color.RGBA { return mixc(c, color.RGBA{255, 255, 255, c.A}, f) }
func darken(c color.RGBA, f float64) color.RGBA {
	return color.RGBA{uint8(float64(c.R) * f), uint8(float64(c.G) * f), uint8(float64(c.B) * f), c.A}
}
func alpha(c color.RGBA, a float64) color.RGBA {
	a = math.Max(0, math.Min(1, a))
	return color.RGBA{c.R, c.G, c.B, uint8(255 * a)}
}

// prem converts a straight-alpha colour to premultiplied for vector/text APIs.
func prem(c color.RGBA, a float64) color.RGBA {
	a = math.Max(0, math.Min(1, a)) * float64(c.A) / 255
	return color.RGBA{uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), uint8(255 * a)}
}

func kindColor(k levelgen.Kind, base color.RGBA) color.RGBA {
	switch k {
	case levelgen.Steel:
		return color.RGBA{128, 134, 146, 255}
	case levelgen.Armored:
		return color.RGBA{86, 92, 108, 255}
	case levelgen.Explosive:
		return color.RGBA{170, 40, 40, 255}
	case levelgen.Crystal:
		return color.RGBA{60, 170, 230, 255}
	case levelgen.Elite:
		return color.RGBA{60, 30, 80, 255}
	case levelgen.Mystery:
		return color.RGBA{240, 200, 80, 255}
	case levelgen.Multiplier:
		return color.RGBA{250, 120, 200, 255}
	case levelgen.Shield:
		return color.RGBA{90, 200, 230, 255}
	case levelgen.Spawner:
		return color.RGBA{110, 170, 70, 255}
	case levelgen.Teleport:
		return color.RGBA{150, 90, 240, 255}
	case levelgen.Chain:
		return color.RGBA{70, 120, 230, 255}
	}
	return base
}

var kindIcon = map[levelgen.Kind]string{
	levelgen.Explosive: "bomb", levelgen.Chain: "bolt", levelgen.Split: "split", levelgen.Regen: "plus",
	levelgen.Moving: "arrows", levelgen.Shield: "shield", levelgen.Teleport: "warp", levelgen.Spawner: "seed",
	levelgen.Multiplier: "x2", levelgen.Crystal: "gem", levelgen.Mystery: "q", levelgen.Elite: "crown",
	levelgen.Armored: "rock",
}

// Visuals are per-level style choices derived from the level seed, so every
// chamber looks different even within the same act.
type Visuals struct {
	FloorPattern int // 0 grid, 1 dotted, 2 diamond lattice, 3 rings
	PropStyle    int // 0 pillars, 1 pyramids, 2 floating cubes, 3 arches, 4 towers
	SkyStyle     int // 0 mountains, 1 city skyline, 2 pyramids, 3 ringed planet
	HueShift     float64
}

func rgb2hsv(c color.RGBA) (h, s, v float64) {
	r, g, b := float64(c.R)/255, float64(c.G)/255, float64(c.B)/255
	mx := math.Max(r, math.Max(g, b))
	mn := math.Min(r, math.Min(g, b))
	v = mx
	d := mx - mn
	if mx > 0 {
		s = d / mx
	}
	if d == 0 {
		return 0, s, v
	}
	switch mx {
	case r:
		h = math.Mod((g-b)/d, 6)
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	return
}

func hsv(h, s, v float64) color.RGBA {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return color.RGBA{uint8((r + m) * 255), uint8((g + m) * 255), uint8((b + m) * 255), 255}
}

func hueShift(c color.RGBA, deg float64) color.RGBA {
	h, s, v := rgb2hsv(c)
	o := hsv(h+deg, s, v)
	o.A = c.A
	return o
}

// levelLook returns a re-tinted copy of the act biome plus style choices.
func levelLook(base *Biome, seed int64) (*Biome, Visuals) {
	r := rand.New(rand.NewSource(seed ^ 0x77177))
	v := Visuals{
		FloorPattern: r.Intn(4),
		PropStyle:    r.Intn(5),
		SkyStyle:     r.Intn(4),
		HueShift:     (r.Float64()*2 - 1) * 38,
	}
	if !base.Neon {
		v.HueShift *= 0.25
	}
	b := *base
	for i := range b.Hues {
		b.Hues[i] = hueShift(b.Hues[i], v.HueShift)
	}
	// rotate which neon leads the floor
	if base.Neon && r.Intn(2) == 0 {
		b.Floor, b.FloorAlt = b.FloorAlt, b.Floor
	}
	b.Floor = hueShift(b.Floor, v.HueShift)
	b.FloorAlt = hueShift(b.FloorAlt, v.HueShift)
	b.Trim = hueShift(b.Trim, v.HueShift*0.7)
	b.SkyBottom = hueShift(b.SkyBottom, v.HueShift*0.6)
	b.Fog = hueShift(b.Fog, v.HueShift*0.5)
	b.Wall = hueShift(b.Wall, v.HueShift*0.5)
	return &b, v
}

// makeSky dispatches to the world's own background painter.
func makeSky(b *Biome, seed int64) *ebiten.Image {
	switch b.SkyKind {
	case WMeadow:
		return paintSky(b, seed, skyMeadow)
	case WCaverns:
		return paintSky(b, seed, skyCaverns)
	case WRuins:
		return paintSky(b, seed, skyRuins)
	case WClouds:
		return paintSky(b, seed, skyClouds)
	case WMachine:
		return paintSky(b, seed, skyMachine)
	}
	return makeSynthSky(b, seed)
}

type canvas struct {
	img  *image.RGBA
	w, h int
	hy   float64
}

func (c *canvas) blend(x, y int, col color.RGBA, a float64) {
	if x < 0 || y < 0 || x >= c.w || y >= c.h || a <= 0 {
		return
	}
	o := c.img.RGBAAt(x, y)
	c.img.SetRGBA(x, y, mixc(o, col, math.Min(1, a)))
}

func (c *canvas) disc(cx, cy, rx, ry float64, col color.RGBA, a float64, soft float64) {
	for y := int(cy - ry - 1); y <= int(cy+ry+1); y++ {
		for x := int(cx - rx - 1); x <= int(cx+rx+1); x++ {
			d := math.Hypot((float64(x)-cx)/rx, (float64(y)-cy)/ry)
			if d < 1 {
				f := 1.0
				if soft > 0 {
					f = math.Min(1, (1-d)/soft)
				}
				c.blend(x, y, col, a*f)
			}
		}
	}
}

// ridge fills below a height profile measured up from the horizon.
func (c *canvas) ridge(hgt func(x float64) float64, col color.RGBA, rim color.RGBA, rimW float64) {
	for x := 0; x < c.w; x++ {
		top := c.hy - hgt(float64(x))
		for y := int(top); y < int(c.hy)+2 && y < c.h; y++ {
			if y < 0 {
				continue
			}
			cc := col
			if float64(y) < top+rimW {
				cc = rim
			}
			c.img.SetRGBA(x, y, cc)
		}
	}
}

func paintSky(b *Biome, seed int64, f func(c *canvas, b *Biome, r *rand.Rand)) *ebiten.Image {
	const w, h = 960, 540
	c := &canvas{img: image.NewRGBA(image.Rect(0, 0, w, h)), w: w, h: h, hy: h * skyHorizon}
	for y := 0; y < h; y++ {
		t := math.Min(1, float64(y)/c.hy)
		col := mixc(b.SkyTop, b.SkyBottom, math.Pow(t, 1.6))
		if float64(y) > c.hy {
			col = mixc(b.SkyBottom, b.Outer, math.Min(1, (float64(y)-c.hy)/30))
		}
		for x := 0; x < w; x++ {
			c.img.SetRGBA(x, y, col)
		}
	}
	f(c, b, rand.New(rand.NewSource(seed)))
	return ebiten.NewImageFromImage(c.img)
}

func skyMeadow(c *canvas, b *Biome, r *rand.Rand) {
	sx := float64(c.w) * (0.25 + r.Float64()*0.5)
	c.disc(sx, c.hy-30, 220, 160, color.RGBA{255, 210, 140, 255}, 0.35, 1)
	c.disc(sx, c.hy-30, 46, 46, color.RGBA{255, 240, 200, 255}, 1, 0.08)
	// clouds lit from below
	for i := 0; i < 9; i++ {
		cx, cy := r.Float64()*float64(c.w), 60+r.Float64()*c.hy*0.5
		for k := 0; k < 6; k++ {
			ox := (r.Float64() - 0.5) * 120
			rr := 22 + r.Float64()*28
			c.disc(cx+ox, cy+(r.Float64()-0.5)*10, rr*1.6, rr*0.8, color.RGBA{255, 235, 225, 255}, 0.75, 0.6)
			c.disc(cx+ox, cy+rr*0.35, rr*1.4, rr*0.4, color.RGBA{250, 160, 140, 255}, 0.45, 0.7)
		}
	}
	ph := r.Float64() * 9
	c.ridge(func(x float64) float64 { return 60 + 30*math.Sin(x*0.006+ph) + 14*math.Sin(x*0.017) }, color.RGBA{130, 140, 170, 255}, color.RGBA{170, 170, 190, 255}, 2)
	c.ridge(func(x float64) float64 { return 38 + 22*math.Sin(x*0.009+ph*2) }, color.RGBA{110, 150, 100, 255}, color.RGBA{150, 180, 110, 255}, 2)
	c.ridge(func(x float64) float64 { return 18 + 12*math.Sin(x*0.013+ph*3) }, color.RGBA{86, 130, 70, 255}, color.RGBA{120, 160, 80, 255}, 2)
	// little trees on the middle hills
	for i := 0; i < 26; i++ {
		x := r.Float64() * float64(c.w)
		top := c.hy - (38 + 22*math.Sin(x*0.009+ph*2))
		c.disc(x, top-6, 6, 8, color.RGBA{70, 110, 60, 255}, 1, 0)
	}
	// birds
	for i := 0; i < 5; i++ {
		bx, by := r.Float64()*float64(c.w), 50+r.Float64()*120
		for d := 0; d < 6; d++ {
			c.blend(int(bx)+d, int(by)-d/2, color.RGBA{60, 50, 70, 255}, 0.8)
			c.blend(int(bx)-d, int(by)-d/2, color.RGBA{60, 50, 70, 255}, 0.8)
		}
	}
}

func skyCaverns(c *canvas, b *Biome, r *rand.Rand) {
	// distant glowing crystal formations
	for i := 0; i < 14; i++ {
		x := r.Float64() * float64(c.w)
		hh := 30 + r.Float64()*90
		ww := 8 + r.Float64()*18
		col := b.Hues[r.Intn(6)]
		c.disc(x, c.hy-hh*0.4, ww*4, hh*0.8, col, 0.12, 1)
		for y := 0.0; y < hh; y++ {
			half := ww * (1 - y/hh)
			for dx := -half; dx <= half; dx++ {
				shade := 0.5 + 0.5*(dx/math.Max(1, half))
				c.blend(int(x+dx), int(c.hy-y), mixc(darken(col, 0.4), col, shade), 0.85)
			}
		}
	}
	c.ridge(func(x float64) float64 { return 20 + 14*math.Abs(math.Sin(x*0.02)) }, color.RGBA{20, 24, 40, 255}, color.RGBA{40, 60, 100, 255}, 2)
	// stalactites
	for i := 0; i < 40; i++ {
		x := r.Float64() * float64(c.w)
		l := 30 + r.Float64()*170
		ww := 6 + r.Float64()*16
		for y := 0.0; y < l; y++ {
			half := ww * (1 - y/l)
			for dx := -half; dx <= half; dx++ {
				c.blend(int(x+dx), int(y), color.RGBA{10, 12, 22, 255}, 0.95)
			}
			c.blend(int(x-half), int(y), color.RGBA{60, 90, 140, 255}, 0.5)
		}
	}
	for i := 0; i < 160; i++ {
		x, y := r.Intn(c.w), r.Intn(int(c.hy))
		c.blend(x, y, b.Hues[r.Intn(6)], 0.6)
	}
}

func skyRuins(c *canvas, b *Biome, r *rand.Rand) {
	sx := float64(c.w) * (0.2 + r.Float64()*0.6)
	c.disc(sx, 70, 260, 200, color.RGBA{255, 250, 230, 255}, 0.35, 1)
	c.disc(sx, 70, 34, 34, color.RGBA{255, 255, 245, 255}, 1, 0.1)
	// mesas
	for i := 0; i < 6; i++ {
		x0 := r.Float64() * float64(c.w)
		w := 60 + r.Float64()*140
		h := 30 + r.Float64()*70
		haze := 0.3 + float64(i%3)*0.2
		col := mixc(color.RGBA{190, 130, 90, 255}, b.SkyBottom, haze)
		c.ridge(func(x float64) float64 {
			d := math.Abs(x-x0) - w/2
			if d < 0 {
				return h
			}
			return math.Max(0, h-d*1.6)
		}, col, lighten(col, 0.2), 2)
	}
	// a distant step pyramid
	px := float64(c.w) * (0.15 + r.Float64()*0.7)
	for step := 0; step < 6; step++ {
		y := c.hy - float64(step)*12
		half := 90 - float64(step)*14
		for yy := y - 12; yy < y; yy++ {
			for x := px - half; x < px+half; x++ {
				c.blend(int(x), int(yy), color.RGBA{200, 160, 110, 255}, 0.8)
			}
		}
	}
	c.ridge(func(x float64) float64 { return 10 + 6*math.Sin(x*0.02) }, color.RGBA{210, 176, 126, 255}, color.RGBA{230, 200, 150, 255}, 2)
}

func skyClouds(c *canvas, b *Biome, r *rand.Rand) {
	sx := float64(c.w) * (0.3 + r.Float64()*0.4)
	c.disc(sx, c.hy*0.45, 260, 220, color.RGBA{255, 250, 220, 255}, 0.4, 1)
	c.disc(sx, c.hy*0.45, 40, 40, color.RGBA{255, 255, 240, 255}, 1, 0.1)
	// distant floating islands
	for i := 0; i < 4; i++ {
		ix := r.Float64() * float64(c.w)
		iy := c.hy*0.35 + r.Float64()*c.hy*0.4
		iw := 30 + r.Float64()*50
		col := mixc(color.RGBA{120, 130, 170, 255}, b.SkyBottom, 0.5)
		for y := 0.0; y < iw; y++ {
			half := iw * (1 - y/iw)
			for dx := -half; dx <= half; dx++ {
				c.blend(int(ix+dx), int(iy+y), col, 0.9)
			}
		}
		for t := 0; t < 3; t++ {
			tx := ix + (float64(t)-1)*iw*0.4
			th := 14 + r.Float64()*26
			for y := 0.0; y < th; y++ {
				for dx := -4.0; dx <= 4; dx++ {
					c.blend(int(tx+dx), int(iy-y), col, 0.95)
				}
			}
			for y := 0.0; y < 10; y++ {
				for dx := -6 + y*0.6; dx <= 6-y*0.6; dx++ {
					c.blend(int(tx+dx), int(iy-th-y), mixc(col, color.RGBA{230, 140, 170, 255}, 0.4), 0.95)
				}
			}
		}
	}
	// cloud sea along the horizon
	for i := 0; i < 70; i++ {
		x := r.Float64() * float64(c.w)
		y := c.hy - 10 + r.Float64()*30
		rr := 20 + r.Float64()*40
		c.disc(x, y, rr*1.8, rr, color.RGBA{255, 255, 255, 255}, 0.85, 0.5)
		c.disc(x, y+rr*0.4, rr*1.6, rr*0.5, color.RGBA{230, 210, 240, 255}, 0.5, 0.6)
	}
	for i := 0; i < 8; i++ {
		x, y := r.Float64()*float64(c.w), 40+r.Float64()*c.hy*0.4
		c.disc(x, y, 70+r.Float64()*60, 18, color.RGBA{255, 255, 255, 255}, 0.5, 0.8)
	}
}

func skyMachine(c *canvas, b *Biome, r *rand.Rand) {
	c.disc(float64(c.w)/2, c.hy, float64(c.w)*0.7, 120, color.RGBA{255, 120, 40, 255}, 0.35, 1)
	dark := color.RGBA{24, 22, 26, 255}
	// factory skyline
	for x := 0; x < c.w; {
		bw := 30 + r.Intn(70)
		bh := 30 + r.Float64()*80
		for i := x; i < x+bw && i < c.w; i++ {
			for y := int(c.hy - bh); y < int(c.hy)+2; y++ {
				c.blend(i, y, dark, 1)
			}
		}
		// chimney with smoke
		if r.Intn(2) == 0 {
			cx := float64(x + bw/2)
			ch := bh + 30 + r.Float64()*60
			for y := c.hy - ch; y < c.hy-bh; y++ {
				for dx := -5.0; dx <= 5; dx++ {
					c.blend(int(cx+dx), int(y), dark, 1)
				}
			}
			for k := 0; k < 8; k++ {
				c.disc(cx+float64(k)*9, c.hy-ch-float64(k)*14, 12+float64(k)*3, 9+float64(k)*2, color.RGBA{90, 80, 80, 255}, 0.35, 0.8)
			}
			c.disc(cx, c.hy-ch, 6, 3, color.RGBA{255, 160, 60, 255}, 0.9, 0.3)
		}
		// windows
		for wy := int(c.hy - bh + 8); wy < int(c.hy)-4; wy += 9 {
			for wx := x + 4; wx < x+bw-4; wx += 8 {
				if r.Float64() < 0.3 {
					for yy := wy; yy < wy+3; yy++ {
						for xx := wx; xx < wx+3; xx++ {
							c.blend(xx, yy, color.RGBA{255, 170, 70, 255}, 0.9)
						}
					}
				}
			}
		}
		x += bw + r.Intn(10)
	}
	// big gear silhouette
	gx, gy, gr := float64(c.w)*(0.2+r.Float64()*0.6), c.hy-60, 70.0
	for y := int(gy - gr - 15); y < int(gy+gr+15); y++ {
		for x := int(gx - gr - 15); x < int(gx+gr+15); x++ {
			dx, dy := float64(x)-gx, float64(y)-gy
			d := math.Hypot(dx, dy)
			a := math.Atan2(dy, dx)
			tooth := math.Mod(a*12/(2*math.Pi)+12, 1) < 0.5
			outer := gr
			if tooth {
				outer += 12
			}
			if d < outer && d > gr*0.35 && float64(y) < c.hy {
				c.blend(x, y, color.RGBA{36, 32, 34, 255}, 1)
			}
		}
	}
	for i := 0; i < 90; i++ {
		x, y := r.Intn(c.w), r.Intn(int(c.hy))
		c.blend(x, y, color.RGBA{255, 160, 80, 255}, 0.3)
	}
}
