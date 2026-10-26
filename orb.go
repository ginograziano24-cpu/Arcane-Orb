package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// The hovering title-screen orb: a glass sphere full of swirling arcane
// energy, rendered per pixel by a shader (volumetric swirl, energy
// filaments, crawling surface arcs, fresnel rim, specular glints and a
// flickering corona), wrapped in an orbiting rune ring and motes.

const orbSrc = `//kage:unit pixels
package main

var Time float
var Center vec2
var Radius float

func hash(p vec3) float {
	p = fract(p*0.3183099 + vec3(0.1, 0.2, 0.3))
	p *= 17.0
	return fract(p.x * p.y * p.z * (p.x + p.y + p.z))
}

func noise(x vec3) float {
	i := floor(x)
	f := fract(x)
	f = f * f * (3.0 - 2.0*f)
	a := mix(hash(i), hash(i+vec3(1, 0, 0)), f.x)
	b := mix(hash(i+vec3(0, 1, 0)), hash(i+vec3(1, 1, 0)), f.x)
	c := mix(hash(i+vec3(0, 0, 1)), hash(i+vec3(1, 0, 1)), f.x)
	d := mix(hash(i+vec3(0, 1, 1)), hash(i+vec3(1, 1, 1)), f.x)
	return mix(mix(a, b, f.y), mix(c, d, f.y), f.z)
}

func fbm(p vec3) float {
	v := 0.0
	a := 0.5
	for i := 0; i < 4; i++ {
		v += a * noise(p)
		p = p*2.03 + vec3(1.7, 9.2, 3.1)
		a *= 0.5
	}
	return v
}

func rotY(p vec3, a float) vec3 {
	c := cos(a)
	s := sin(a)
	return vec3(c*p.x+s*p.z, p.y, -s*p.x+c*p.z)
}

func rotX(p vec3, a float) vec3 {
	c := cos(a)
	s := sin(a)
	return vec3(p.x, c*p.y-s*p.z, s*p.y+c*p.z)
}

func palette(d float) vec3 {
	c := mix(vec3(0.16, 0.03, 0.42), vec3(0.62, 0.22, 1.0), smoothstep(0.35, 0.6, d))
	c = mix(c, vec3(1.0, 0.55, 0.95), smoothstep(0.6, 0.78, d))
	return mix(c, vec3(1.0, 0.92, 1.0), smoothstep(0.78, 0.95, d))
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	p := (dstPos.xy - Center) / Radius
	p.y = -p.y
	r := length(p)
	t := Time
	ang := atan2(p.y, p.x)

	if r > 1.0 {
		// ---- aura and corona (additive: alpha 0)
		e := r - 1.0
		glow := vec3(0.42, 0.16, 0.9)*exp(-e*3.2)*0.55 + vec3(0.85, 0.55, 1.0)*exp(-e*16.0)*0.75
		dir := vec3(cos(ang), sin(ang), 0)
		n := fbm(dir*2.2 + vec3(0, 0, e*2.6-t*0.9))
		n2 := fbm(dir*4.5 + vec3(3.1, 1.3, e*4.0-t*1.6))
		flame := smoothstep(0.42, 0.78, n*0.65+n2*0.45) * exp(-e*4.2)
		glow += vec3(0.75, 0.32, 1.0) * flame * 0.9
		// faint outer ring of light
		glow += vec3(0.6, 0.4, 1.0) * exp(-pow((r-1.42)*22.0, 2.0)) * 0.12 * (0.6 + 0.4*sin(ang*3.0+t))
		return vec4(glow, 0.0)
	}

	z := sqrt(1.0 - r*r)
	nrm := vec3(p.x, p.y, z)
	// ---- volumetric swirl through the sphere
	front := nrm
	back := vec3(p.x, p.y, -z)
	acc := vec3(0)
	trans := 1.0
	for i := 0; i < 9; i++ {
		s := (float(i) + 0.5) / 9.0
		q := mix(front, back, s)
		// swirl: rotate faster near the core
		q = rotY(q, t*0.35+(1.2-length(q))*1.6)
		q = rotX(q, 0.4*sin(t*0.21))
		w := vec3(fbm(q*1.6+vec3(0, t*0.15, 0)), fbm(q*1.6+vec3(5.2, -t*0.12, 1.3)), 0.0)
		qq := q*2.1 + w*1.4
		d := fbm(qq + vec3(0, -t*0.3, 0))
		fil := 1.0 - abs(noise(qq*1.7+vec3(t*0.2, 0, 0))*2.0-1.0)
		fil = pow(fil, 9.0)
		core := exp(-dot(q, q) * 3.5)
		dens := smoothstep(0.38, 0.75, d)*0.9 + fil*0.9 + core*0.6
		c := palette(d+core*0.35)*dens + vec3(0.9, 0.75, 1.0)*fil*1.4
		acc += c * trans * (1.5 * z / 9.0)
		trans *= 1.0 - clamp(dens*0.12*z, 0.0, 0.6)
	}
	// hot pulsing core
	pulse := 0.85 + 0.15*sin(t*2.3)
	acc += vec3(1.0, 0.8, 1.0) * exp(-r*r*9.0) * 0.55 * pulse
	acc += vec3(0.6, 0.3, 1.0) * exp(-r*r*2.5) * 0.35
	// dark glass depth toward the edge
	base := vec3(0.035, 0.01, 0.08)
	col := base + acc*mix(0.55, 1.0, z)

	// ---- crawling surface arcs
	sp := rotY(nrm, t*0.5)
	arc := 1.0 - abs(noise(sp*4.2+vec3(0, t*0.8, 0))*2.0-1.0)
	arc = pow(arc, 26.0)
	flick := step(0.35, noise(vec3(t*7.0, sp.x*3.0, 0)))
	col += vec3(0.85, 0.7, 1.0) * arc * 1.6 * flick * (0.4 + 0.6*(1.0-z))

	// ---- glass: fresnel rim, glints
	fres := pow(1.0-z, 3.0)
	col += vec3(0.7, 0.45, 1.0) * fres * 1.3
	refl := vec3(2.0*nrm.x*nrm.z, 2.0*nrm.y*nrm.z, 2.0*nrm.z*nrm.z-1.0)
	L := normalize(vec3(-0.55, 0.65, 0.55))
	spec := pow(max(dot(refl, L), 0.0), 90.0)
	col += vec3(1.0, 0.97, 1.0) * spec * 1.6
	// soft window highlight
	hp := p - vec2(-0.38, 0.42)
	col += vec3(1.0, 0.95, 1.0) * smoothstep(0.24, 0.0, length(hp*vec2(1.0, 1.8))) * 0.22
	// pink bounce light from below-right
	B := normalize(vec3(0.6, -0.7, 0.4))
	col += vec3(1.0, 0.35, 0.75) * pow(max(dot(nrm, B), 0.0), 3.0) * 0.35

	// anti-aliased edge into the aura
	edge := smoothstep(1.0, 1.0-2.0/Radius, r)
	glowEdge := vec3(0.85, 0.55, 1.0) * 0.75
	return vec4(col*edge+glowEdge*(1.0-edge), edge)
}
`

var orbShader *ebiten.Shader

type orbMote struct {
	a, r, tilt, spd, size, phase float64
	hue                          color.RGBA
}

var orbMotes []orbMote

func initOrbMotes() {
	if orbMotes != nil {
		return
	}
	for i := 0; i < 64; i++ {
		f := float64(i)
		orbMotes = append(orbMotes, orbMote{
			a:     f * 2.399,
			r:     1.22 + 0.55*math.Mod(f*0.618, 1),
			tilt:  (math.Mod(f*0.37, 1) - 0.5) * 0.9,
			spd:   0.25 + 0.5*math.Mod(f*0.77, 1),
			size:  4 + 8*math.Mod(f*0.53, 1),
			phase: f,
			hue:   []color.RGBA{{190, 140, 255, 255}, {255, 150, 230, 255}, {150, 170, 255, 255}}[i%3],
		})
	}
}

// drawTitleOrb renders the orb centred at (cx, cy) in virtual pixels.
func (g *Game) drawTitleOrb(dst *ebiten.Image, cx, cy, R, t float64) {
	if orbShader == nil {
		sh, err := ebiten.NewShader([]byte(orbSrc))
		if err != nil {
			panic(err)
		}
		orbShader = sh
	}
	initOrbMotes()
	u := g.ui
	S := u.S
	bob := math.Sin(t*1.25)*14 + math.Sin(t*0.53)*5
	cy += bob
	sx, sy, sr := u.X(cx), u.Y(cy), R*S

	// light pool on the ground below, tighter and brighter when lower
	k := 1 - (bob+19)/38*0.35
	drawSprite(dst, sprSoft, u.X(cx-R*1.9), u.Y(cy+R*1.45-bob*0.6), R*3.8*S, R*0.9*S, ca(rgba(150, 80, 255, 255), 0.45*k), true)
	drawSprite(dst, sprSoft, u.X(cx-R*1.0), u.Y(cy+R*1.62-bob*0.6), R*2.0*S, R*0.4*S, ca(rgba(230, 170, 255, 255), 0.35*k), true)
	// wide ambient bloom
	drawSprite(dst, sprSoft, sx-sr*3.2, sy-sr*3.2, sr*6.4, sr*6.4, ca(rgba(120, 50, 255, 255), 0.22), true)

	// rune ring and motes: back halves first
	g.orbRing(dst, sx, sy, sr, t, false)
	g.orbMotes(dst, sx, sy, sr, t, false)

	ext := sr * 2.3
	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]any{
		"Time":   float32(math.Mod(t, 1000)),
		"Center": []float32{float32(sx), float32(sy)},
		"Radius": float32(sr),
	}
	op.GeoM.Translate(sx-ext, sy-ext)
	dst.DrawRectShader(int(2*ext), int(2*ext), orbShader, op)

	g.orbRing(dst, sx, sy, sr, t, true)
	g.orbMotes(dst, sx, sy, sr, t, true)
}

// orbRing draws a tilted orbiting ring of light, back or front half.
func (g *Game) orbRing(dst *ebiten.Image, sx, sy, sr, t float64, front bool) {
	const n = 140
	tilt := 0.28 + 0.04*math.Sin(t*0.4)
	rot := -0.32
	for i := 0; i < n; i++ {
		a := float64(i)/n*2*math.Pi + t*0.6
		z := math.Sin(a)
		if (z > 0) != front {
			continue
		}
		x := math.Cos(a) * sr * 1.45
		y := z * sr * 1.45 * tilt
		px := sx + x*math.Cos(rot) - y*math.Sin(rot)
		py := sy + x*math.Sin(rot) + y*math.Cos(rot)
		// a bright comet head chases around the ring
		head := math.Mod(float64(i)/n-math.Mod(t*0.18, 1)+2, 1)
		b := 0.35 + 0.65*math.Pow(1-head, 6)
		if !front {
			b *= 0.45
		}
		sz := sr * (0.06 + 0.09*math.Pow(1-head, 8))
		drawSprite(dst, sprSoft, px-sz, py-sz, sz*2, sz*2, ca(rgba(190, 140, 255, 255), b), true)
		c := sz * 0.35
		drawSprite(dst, sprSoft, px-c, py-c, c*2, c*2, ca(rgba(255, 240, 255, 255), b*0.9), true)
	}
}

func (g *Game) orbMotes(dst *ebiten.Image, sx, sy, sr, t float64, front bool) {
	for _, m := range orbMotes {
		a := m.a + t*m.spd
		z := math.Sin(a)
		if (z > 0) != front {
			continue
		}
		x := math.Cos(a) * m.r * sr
		y := z*m.r*sr*0.3 + m.tilt*sr + math.Sin(t*1.7+m.phase)*sr*0.06
		tw := 0.5 + 0.5*math.Sin(t*3+m.phase*1.7)
		al := (0.35 + 0.65*tw)
		if !front {
			al *= 0.5
		}
		sz := m.size * g.ui.S * (0.7 + 0.3*z)
		drawSprite(dst, sprSoft, sx+x-sz, sy+y-sz, sz*2, sz*2, ca(m.hue, al), true)
		drawSprite(dst, sprSoft, sx+x-sz*0.3, sy+y-sz*0.3, sz*0.6, sz*0.6, ca(cWhite, al*0.8), true)
	}
}
