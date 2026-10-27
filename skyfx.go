package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Living skies: a per-world shader layered additively over the painted
// backdrop, behind the arena. Neon gets aurora curtains, the meadow and the
// clouds get drifting sun shafts and pollen, the caverns a crystal shimmer
// with spores, the ruins blowing dust and heat haze, the machine world
// smoke and rising embers. Every world gets faint twinkling magic motes.

const skyfxSrc = `//kage:unit pixels
package main

var Time float
var Mode float
var Res vec2
var Horizon float
var Tint vec3
var Tint2 vec3

func hash2(p vec2) float {
	return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453)
}

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

func motes(uv vec2, t float, dens float, col vec3) vec3 {
	cell := floor(uv * vec2(90.0, 50.0) + vec2(0.0, t*0.6))
	h := hash2(cell)
	if h > 1.0-dens {
		f := fract(uv*vec2(90.0, 50.0) + vec2(0.0, t*0.6)) - 0.5
		d := length(f)
		tw := 0.5 + 0.5*sin(t*(1.0+h*4.0)+h*60.0)
		return col * smoothstep(0.18, 0.0, d) * tw
	}
	return vec3(0)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	uv := dstPos.xy / Res
	t := Time
	hz := Horizon / Res.y
	sky := smoothstep(hz+0.02, hz-0.25, uv.y) // 1 high in the sky, 0 at the horizon
	col := vec3(0)
	m := int(Mode + 0.5)
	if m == 0 || m == 2 {
		// aurora curtains
		x := uv.x*3.0 + t*0.03
		band := fbm(vec3(x, uv.y*1.5, t*0.05))
		curtain := smoothstep(0.45, 0.75, band) * smoothstep(hz-0.05, hz-0.45, uv.y) * smoothstep(0.0, 0.25, uv.y)
		streak := 0.6 + 0.4*sin(uv.x*90.0+band*20.0)
		ac := mix(Tint, Tint2, smoothstep(0.3, 0.7, fbm(vec3(x*0.5, 3.0, t*0.03))))
		k := 0.35
		if m == 2 {
			k = 0.22
		}
		col += ac * curtain * streak * k
	}
	if m == 1 || m == 4 {
		// sun shafts fanning from the upper right
		sp := uv - vec2(0.82, -0.15)
		a := atan2(sp.y, sp.x)
		r := fbm(vec3(a*7.0, t*0.08, 1.0))
		shaft := smoothstep(0.5, 0.85, r) * smoothstep(1.4, 0.2, length(sp))
		col += Tint * shaft * sky * 0.22
		// slow cloud shadows / brightening
		cl := fbm(vec3(uv.x*2.0-t*0.02, uv.y*3.0, 4.0))
		col += Tint2 * smoothstep(0.55, 0.8, cl) * sky * 0.08
	}
	if m == 3 {
		// blowing dust and heat haze near the horizon
		d := fbm(vec3(uv.x*4.0-t*0.25, uv.y*10.0, t*0.1))
		haze := smoothstep(hz-0.35, hz, uv.y) * smoothstep(hz+0.15, hz, uv.y)
		col += Tint * smoothstep(0.45, 0.8, d) * haze * 0.3
	}
	if m == 5 {
		// smoke columns and drifting soot
		d := fbm(vec3(uv.x*3.0, uv.y*2.0+t*0.15, t*0.05))
		col += Tint2 * smoothstep(0.5, 0.85, d) * sky * 0.12
		col += motes(uv+vec2(sin(uv.y*20.0+t)*0.003, 0.0), -t*1.5, 0.008, Tint) * 1.2 * smoothstep(0.0, hz, uv.y)
	}
	// twinkling magic motes in every sky
	col += motes(uv, t*0.2, 0.006, mix(Tint, vec3(1.0), 0.5)) * sky * 0.8
	return vec4(col, 0.0)
}
`

var skyfxShader *ebiten.Shader

var skyTints = [NumWorlds][2][3]float32{
	WNeon:    {{0.4, 1.0, 0.9}, {1.0, 0.3, 0.85}},
	WMeadow:  {{1.0, 0.85, 0.55}, {1.0, 0.95, 0.85}},
	WCaverns: {{0.5, 0.75, 1.0}, {0.75, 0.45, 1.0}},
	WRuins:   {{1.0, 0.8, 0.55}, {1.0, 0.7, 0.45}},
	WClouds:  {{1.0, 0.95, 0.8}, {1.0, 1.0, 1.0}},
	WMachine: {{1.0, 0.55, 0.2}, {0.35, 0.33, 0.32}},
}

// drawSkyFX layers the world's sky effect over the backdrop.
func (g *Game) drawSkyFX(dst *ebiten.Image, world int, horizonY, t float64) {
	if skyfxShader == nil {
		sh, err := ebiten.NewShader([]byte(skyfxSrc))
		if err != nil {
			panic(err)
		}
		skyfxShader = sh
	}
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	tc := skyTints[world]
	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]any{
		"Time":    float32(math.Mod(t, 2000)),
		"Mode":    float32(world),
		"Res":     []float32{float32(w), float32(h)},
		"Horizon": float32(horizonY),
		"Tint":    tc[0][:],
		"Tint2":   tc[1][:],
	}
	dst.DrawRectShader(w, h, skyfxShader, op)
}
