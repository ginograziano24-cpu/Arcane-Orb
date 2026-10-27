//go:build capture && js

package main

// The store trailer, shot by shot. Times are in seconds at 120 BPM, so
// every cut lands on a bar (2 s) or a beat (0.5 s) of the score in
// trailer_audio.go.

import (
	"image/color"
	"math"
	"math/rand"
	"strings"
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2"

	"orb/levelgen"
)

const trailerBPM = 120.0

func trailerLength() float64 { return 56 }

// playShot stages a slice of real gameplay.
type playShot struct {
	world, act int
	arch       string
	mods       []string
	seed       int64
	depth      int
	boss       bool
	designed   bool
	preroll    float64
	powers     []Power
	ability    []float64 // shot-local times to fire the class ability
	cam        func(c *Camera, k float64)
}

type shot struct {
	at, dur float64
	play    *playShot
	enter   func(g *Game)
	step    func(g *Game, lt float64)
	card    func(g *Game, dst *ebiten.Image, lt float64) // draws the whole frame
}

type caption struct {
	t0, t1 float64
	text   string
}

var trailerCaptions = []caption{
	{8.45, 11.55, "Bounce the orb. Break every brick."},
	{12.45, 15.55, "Chain lightning, fire and frost"},
	{16.3, 19.7, "Seven origins. Seven playstyles."},
	{20.45, 23.55, "Pick from 100+ upgrades and relics"},
	{24.25, 25.85, "Pair two items to forge a fusion"},
	{26.3, 29.7, "Stack effects into chain reactions"},
	{38.45, 41.55, "Beat a Guardian to end each act"},
}

// flashes: white pops on the big downbeats
var trailerFlashes = [][3]float64{{6.0, 0.6, 0.85}, {8.0, 0.18, 0.35}, {32.0, 0.22, 0.45}, {38.0, 0.18, 0.35}, {46.0, 0.8, 0.9}} // time, length, peak

// ------------------------------------------------------------ cameras

// camNear brings every gameplay shot closer than the playing view.
func camNear(c *Camera) {
	c.Eye = c.Eye.Add(V3{0, -2.6, 4.2})
	c.Target = c.Target.Add(V3{0, -0.5, 1})
}

func camDolly(c *Camera, k float64) {
	e := smootherstep(k)
	c.Eye = c.Eye.Add(V3{0, -2.2 * e, 3.2 * e})
}

func camOrbit(dir float64) func(c *Camera, k float64) {
	return func(c *Camera, k float64) {
		e := smootherstep(k)
		c.Eye = c.Eye.Add(V3{dir * (-5 + 10*e), -1.5, 1 + 1.5*e})
		c.Target = c.Target.Add(V3{dir * (1 - 2*e), 0, 0})
	}
}

func camLow(c *Camera, k float64) {
	e := smootherstep(k)
	c.Eye = c.Eye.Add(V3{-2 + 3*e, -4.5 + 1.5*e, 2.5 + 2*e})
	c.Target = c.Target.Add(V3{0, 1.2, 2})
}

func camCrane(c *Camera, k float64) {
	e := smootherstep(k)
	c.Eye = c.Eye.Add(V3{0, 5 - 6*e, -2 + 4*e})
}

// ------------------------------------------------------------ shots

var trailerShots = []shot{
	{at: 0, dur: 6, card: cardColdOpen},
	{at: 6, dur: 2, card: cardTitle},
	{at: 8, dur: 4, play: &playShot{world: WNeon, act: 3, arch: "pyro", seed: 4242, depth: 22, preroll: 0.9,
		mods:    []string{"twin", "kindling", "wildfire", "split", "pyro"},
		powers:  []Power{PMulti, PMulti},
		ability: []float64{2.6}, cam: camLow}},
	{at: 12, dur: 4, play: &playShot{world: WMeadow, act: 3, arch: "storm", seed: 5202, depth: 16, preroll: 1.8,
		mods:   []string{"twin", "static", "conductor", "storm", "kindling", "frost", "deepfreeze", "inferno", "staticfield", "tesla"},
		powers: []Power{PMulti, PMulti}, ability: []float64{1.8}, cam: camOrbit(1)}},
	{at: 16, dur: 4, enter: enterOrigins, step: stepOrigins},
	{at: 20, dur: 4, play: &playShot{world: WCaverns, act: 3, arch: "arcane", seed: 6303, depth: 17, preroll: 1.8,
		mods:   []string{"twin", "frost", "frozenheart", "shatter", "frostbite", "deepfreeze", "split", "mirror"},
		powers: []Power{PMulti, PMulti}, cam: camLow}},
	{at: 24, dur: 2, enter: enterFusion, step: stepFusion},
	{at: 26, dur: 2, play: &playShot{world: WMachine, act: 3, arch: "engineer", seed: 7404, depth: 18, preroll: 1.8,
		mods:   []string{"twin", "meteor", "fuse", "nitro", "nitroflask", "blast", "laser", "rapidfire"},
		powers: []Power{PLaser, PMulti}, cam: camOrbit(-1)}},
	{at: 28, dur: 2, play: &playShot{world: WClouds, act: 3, arch: "lucky", seed: 8505, depth: 18, preroll: 1.8,
		mods:   []string{"twin", "prism", "hydra", "split", "mirror", "hive", "lodestone", "voidtouch"},
		powers: []Power{PMulti}, cam: camDolly}},
	{at: 30, dur: 2, play: &playShot{world: WRuins, act: 3, arch: "berserker", seed: 9606, depth: 19, preroll: 1.8,
		mods:   []string{"twin", "blackhole", "comboeng", "momentum", "heavy", "overclock", "split", "shrapshell"},
		powers: []Power{PMulti, PFire}, cam: camCrane}},
	{at: 32, dur: 6, play: &playShot{act: 1, arch: "guardian", seed: 1111, depth: 12, boss: true,
		mods: []string{"twin", "kindling", "static", "split"}}},
	{at: 38, dur: 4, play: &playShot{act: 3, arch: "storm", seed: 3333, depth: 30, boss: true, preroll: 6.8,
		mods:   []string{"twin", "bane", "slayer", "excalibur", "static", "kindling", "meteor", "fuse", "split", "conductor"},
		powers: []Power{PMulti}, ability: []float64{1.2}, cam: camLow}},
	{at: 42, dur: 1, play: &playShot{world: WNeon, act: 3, arch: "arcane", seed: 1212, depth: 17, preroll: 1.8, designed: true,
		mods: []string{"twin", "split", "mirror", "kindling", "wildfire", "static"}, powers: []Power{PMulti}}},
	{at: 43, dur: 1, play: &playShot{world: WMachine, act: 3, arch: "engineer", seed: 1313, depth: 18, preroll: 1.8,
		mods: []string{"twin", "laser", "rapidfire", "laserfocus", "static", "discharge"}, powers: []Power{PLaser}, cam: camLow}},
	{at: 44, dur: 1, play: &playShot{world: WMeadow, act: 3, arch: "pyro", seed: 1414, depth: 18, preroll: 1.8,
		mods: []string{"twin", "kindling", "inferno", "dragonheart", "wildfire", "split"}, powers: []Power{PMulti, PFire}, ability: []float64{0.2}, cam: camOrbit(1)}},
	{at: 45, dur: 1, play: &playShot{world: WCaverns, act: 3, arch: "storm", seed: 1515, depth: 19, preroll: 1.8,
		mods: []string{"twin", "static", "conductor", "storm", "tesla", "frost", "shatter", "frozenheart"}, ability: []float64{0.15}, cam: camDolly}},
	{at: 46, dur: 10, card: cardEnd},
}

func shotAt(t float64) (int, *shot) {
	for i := len(trailerShots) - 1; i >= 0; i-- {
		if t >= trailerShots[i].at-1e-6 {
			return i, &trailerShots[i]
		}
	}
	return 0, &trailerShots[0]
}

var dir struct {
	cur  int
	ap   autopilot
	play *playShot
	lt   float64
}

func init() { dir.cur = -1 }

// directorStep runs before the game's own Update each frame.
func directorStep(g *Game, t float64) {
	i, sh := shotAt(t)
	lt := t - sh.at
	dir.lt = lt
	if i != dir.cur {
		dir.cur = i
		dir.play = sh.play
		inputHook, camHook = nil, nil
		g.fadeT = 0
		if sh.enter != nil {
			sh.enter(g)
		}
		if sh.play != nil {
			startPlay(g, sh.play)
		}
		if sh.card != nil {
			g.state = StMenu // a quiet state; the card draws everything
		}
	}
	if sh.step != nil {
		sh.step(g, lt)
	}
	if p := dir.play; p != nil && g.run != nil {
		g.run.HP = g.run.Stats.MaxHP
		k := lt / sh.dur
		if p.cam != nil {
			camHook = func(c *Camera, L *Level) {
				if L.introT <= 0 {
					camNear(c)
					p.cam(c, k)
				}
			}
		}
	}
}

func levelgenDesign(on bool) {
	if on {
		levelgen.DesignChance = 1
	} else {
		levelgen.DesignChance = 0.22
	}
}

func archByID(id string) *Archetype {
	for i := range archetypes {
		if archetypes[i].ID == id {
			return &archetypes[i]
		}
	}
	return &archetypes[0]
}

func startPlay(g *Game, p *playShot) {
	rec := capState.recording
	capState.recording = false
	defer func() { capState.recording = rec }()
	forceWorld = p.world
	g.run = NewRun("TRAILER", archByID(p.arch), &g.save, false)
	g.run.Act = p.act
	for _, id := range p.mods {
		if m := modByID[id]; m != nil {
			g.take(m)
		}
	}
	g.toasts = nil
	g.bannerT = 0
	g.run.Depth = p.depth
	g.run.HP = g.run.Stats.MaxHP
	if p.designed {
		levelgenDesign(true)
	}
	node := NCombat
	if p.boss {
		node = NBoss
	}
	capState.recording = rec && p.preroll == 0 // the boss intro sting belongs in the mix
	g.level = g.newLevel(node, p.seed)
	capState.recording = false
	if p.designed {
		levelgenDesign(false)
	}
	forceWorld = -1
	g.state, g.stateT = StLevel, 0
	L := g.level
	for _, pw := range p.powers {
		L.applyPower(pw)
	}
	g.snd.Key, g.snd.World = g.run.AI(), L.Biome.ID
	dir.ap = autopilot{rng: rand.New(rand.NewSource(p.seed))}
	inputHook = func(g *Game) Input {
		return dir.ap.input(g, g.level, 1/capState.fps, dir.play, dir.lt)
	}
	// run the chamber for a moment off camera so the shot opens mid-action
	step := 1.0 / 60
	for i := 0; i < int(p.preroll*60); i++ {
		in := dir.ap.input(g, L, step, p, -1)
		L.Update(step, in)
		g.run.HP = g.run.Stats.MaxHP
		if L.Cleared {
			break
		}
	}
	g.toasts = nil
	g.bannerT = 0
}

// ------------------------------------------------------------ autopilot

type autopilot struct {
	rng      *rand.Rand
	offset   float64
	tracking *Ball
	stuckT   float64
	fired    map[int]bool
}

// input plays like a confident human: it reads where the most urgent orb
// will land and meets it slightly off-centre to angle the shot.
func (a *autopilot) input(g *Game, L *Level, dt float64, p *playShot, lt float64) Input {
	in := Input{UseKeys: true}
	zp := PaddleZ + PaddleD/2
	var best *Ball
	bt := math.Inf(1)
	bx := L.PaddleX
	for _, b := range L.Balls {
		if b.Stuck || b.VZ >= 0 {
			continue
		}
		t := (b.Z - zp - b.R) / -b.VZ
		if b.Phantom > 0 {
			t += 2 // phantoms are expendable
		}
		if t < bt {
			bt, best = t, b
			bx = foldX(b.X+b.VX*math.Max(0, t), HalfW-b.R)
		}
	}
	if best == nil {
		n := 0.0
		sum := 0.0
		for _, b := range L.Balls {
			if !b.Stuck {
				sum += b.X
				n++
			}
		}
		if n > 0 {
			bx = sum / n * 0.6
		}
	}
	if best != a.tracking {
		a.tracking = best
		a.offset = (a.rng.Float64()*2 - 1) * L.PaddleW * 0.3
	}
	target := bx - a.offset
	d := target - L.PaddleX
	in.Fast = math.Abs(d) > 0.6
	sp := 30.0
	if in.Fast {
		sp = 46
	}
	in.KeyDir = math.Max(-1, math.Min(1, d/(sp*dt)))
	if L.anyStuck() && L.introT <= 0 {
		a.stuckT += dt
		if a.stuckT > 0.25 {
			in.Fire = true
			a.stuckT = 0
		}
	}
	if p != nil && lt >= 0 {
		if a.fired == nil {
			a.fired = map[int]bool{}
		}
		for i, at := range p.ability {
			if lt >= at && !a.fired[i] {
				a.fired[i] = true
				g.run.AbilT = 0
				in.Ability = true
			}
		}
	}
	return in
}

// foldX reflects x into [-w, w] as the arena walls would.
func foldX(x, w float64) float64 {
	p := 4 * w
	x = math.Mod(x+w, p)
	if x < 0 {
		x += p
	}
	if x > 2*w {
		x = p - x
	}
	return x - w
}

// ------------------------------------------------------------ origins

var originOrder = []string{"pyro", "storm", "berserker"}

func enterOrigins(g *Game) {
	g.state, g.stateT = StArch, 0
	g.endlessMode, g.daily = false, false
	g.archSel = indexOfArch(originOrder[0])
	g.archShown, g.archPrev, g.archFade = g.archSel, -1, 1
	g.ui.nav.mode = false
}

func stepOrigins(g *Game, lt float64) {
	i := min(len(originOrder)-1, int(lt/(4.0/3)))
	if s := indexOfArch(originOrder[i]); s != g.archSel {
		g.archSel = s
		g.snd.Play("hover")
	}
}

func indexOfArch(id string) int {
	for i := range archetypes {
		if archetypes[i].ID == id {
			return i
		}
	}
	return 0
}

// ------------------------------------------------------------ fusion

var fusionPick = "blast" // fuses with Frost Touch into Cryo Charge

func enterFusion(g *Game) {
	rec := capState.recording
	capState.recording = false
	g.run = NewRun("TRAILER", archByID("arcane"), &g.save, false)
	g.run.Act = 2
	g.take(modByID["frost"])
	g.toasts = nil
	g.bannerT = 0
	capState.recording = rec
	g.rewards = []rewardSet{{Title: "Choose an upgrade", Mods: []*Mod{modByID["excalibur"], modByID[fusionPick], modByID["hive"]}, Skip: 15}}
	g.state, g.stateT = StReward, 0
	g.snd.Play("deal")
	g.ui.nav.mode = false
}

func stepFusion(g *Game, lt float64) {
	if lt >= 0.55 && g.state == StReward {
		g.ui.nav.mode = true
		if g.ui.nav.focus != "card"+fusionPick {
			g.ui.nav.focus = "card" + fusionPick
			g.snd.Play("cardhover")
		}
	}
	if lt >= 1.0 && lt < 1.0+1/capState.fps+1e-6 && g.state == StReward {
		g.navLatch.confirm = true // the draw this frame takes the card
	}
	if g.state != StReward {
		g.ui.nav.mode = false
	}
}

// ------------------------------------------------------------ cards

var trailerLogo *ebiten.Image

func logoImage(g *Game) *ebiten.Image {
	if trailerLogo == nil {
		trailerLogo = ebiten.NewImage(1920, 260)
		g.arcaneTitle(trailerLogo, "Arcane Orb", 168, 960, 30)
	}
	return trailerLogo
}

func drawLogo(g *Game, dst *ebiten.Image, cy, scale, a float64) {
	img := logoImage(g)
	u := g.ui
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-960, -130)
	op.GeoM.Scale(scale*u.S, scale*u.S)
	op.GeoM.Translate(u.X(960), u.Y(cy))
	op.ColorScale.ScaleAlpha(float32(a))
	dst.DrawImage(img, op)
}

func blackout(g *Game, dst *ebiten.Image, a float64) {
	if a <= 0 {
		return
	}
	sw, sh := float64(dst.Bounds().Dx()), float64(dst.Bounds().Dy())
	fillRect(dst, 0, 0, sw, sh, color.RGBA{0, 0, 0, uint8(255 * clamp01(a))})
}

func spaced(s string) string { return strings.Join(strings.Split(strings.ToUpper(s), ""), " ") }

func cardColdOpen(g *Game, dst *ebiten.Image, lt float64) {
	u := g.ui
	k := lt / 6
	g.drawHallK(dst, 960, 520, smootherstep(clamp01((lt-2.6)/2.8)))
	g.drawRunes(dst, 26, 960, 1300)
	R := 50 + 80*smootherstep(clamp01(k*1.05))
	g.drawTitleOrb(dst, 960, 520, R, g.frontT)
	blackout(g, dst, 1-smootherstep(clamp01(lt/3.2))*0.9-0.1*smootherstep(clamp01((lt-3.2)/2)))
	lines := []struct {
		t0, t1 float64
		s      string
	}{{0.8, 2.85, "Walls of bricks have sealed every realm."}, {3.1, 5.75, "One orb can break them open."}}
	for _, l := range lines {
		if lt < l.t0 || lt > l.t1 {
			continue
		}
		a := math.Min(ease((lt-l.t0)/0.6), ease((l.t1-lt)/0.45))
		y := 850 - 10*ease((lt-l.t0)/1.2)
		u.textOutlined(dst, l.s, fSerif, 44, 960, y, ca(rgba(236, 226, 252, 255), a), alignCenter)
	}
}

func cardTitle(g *Game, dst *ebiten.Image, lt float64) {
	u := g.ui
	g.drawHallK(dst, 960, 610, 1)
	g.drawRunes(dst, 26, 960, 1300)
	g.drawTitleOrb(dst, 960, 610, 150+12*math.Exp(-lt*4), g.frontT)
	sl := ease(clamp01(lt / 0.32))
	drawLogo(g, dst, 300, 1.22-0.22*sl, sl)
	sub := "A  ROGUELITE  OF  ORB  AND  RUNE"
	a := ease(clamp01((lt - 0.35) / 0.5))
	u.text(dst, sub, fSerif, 26, 960, 430, ca(rgba(200, 170, 255, 255), a), alignCenter)
	blackout(g, dst, smootherstep(clamp01((lt-1.62)/0.38)))
}

func cardEnd(g *Game, dst *ebiten.Image, lt float64) {
	u := g.ui
	g.drawHallK(dst, 960, 600, 1)
	g.drawRunes(dst, 30, 960, 1300)
	g.drawTitleOrb(dst, 960, 600, 165+14*math.Exp(-lt*3), g.frontT)
	sl := ease(clamp01(lt / 0.45))
	drawLogo(g, dst, 250, 1.15-0.15*sl, sl)
	sub := "A  ROGUELITE  OF  ORB  AND  RUNE"
	a := ease(clamp01((lt - 0.6) / 0.6))
	u.text(dst, sub, fSerif, 26, 960, 380, ca(rgba(200, 170, 255, 255), a), alignCenter)
	// the call to action
	b := ease(clamp01((lt - 1.9) / 0.7))
	if b > 0 {
		y := 892 + (1-b)*14
		gold := rgba(255, 214, 130, 255)
		drawSprite(dst, sprSoft, u.X(960-520), u.Y(y-70), 1040*u.S, 220*u.S, ca(rgba(255, 170, 60, 255), 0.22*b), true)
		u.textOutlined(dst, "WISHLIST NOW ON STEAM", fRune, 58, 960, y, ca(gold, b), alignCenter)
		w := u.measure("WISHLIST NOW ON STEAM", fRune, 58)
		for _, sg := range []float64{-1, 1} {
			x0 := 960 + sg*(w/2+30)
			u.glowLine(dst, math.Min(x0, x0+sg*170), math.Max(x0, x0+sg*170), y+36, 1.5, gold, 0.85*b)
			fillPath(dst, diamondPath(u.X(x0+sg*178), u.Y(y+36), 6*u.S, 9*u.S), rgba(255, 226, 160, 255), rgba(160, 110, 50, 255), grad{}, false)
		}
		c := ease(clamp01((lt - 2.6) / 0.7))
		u.text(dst, "PC   ·   STEAM DECK", fSerif, 24, 960, y+70, ca(rgba(210, 196, 236, 255), c*b), alignCenter)
	}
	blackout(g, dst, smootherstep(clamp01((lt-8.4)/1.5)))
}

// ------------------------------------------------------------ draw hooks

// directorDraw draws card shots itself; it returns true when it did.
func directorDraw(g *Game, dst *ebiten.Image, t float64) bool {
	if ka := js.Global().Get("__keyart"); ka.Type() == js.TypeObject {
		drawKeyart(g, dst, ka)
		captureAfterDraw(g, dst)
		return true
	}
	_, sh := shotAt(t)
	if sh.card == nil {
		return false
	}
	sw, shh := float64(dst.Bounds().Dx()), float64(dst.Bounds().Dy())
	g.ui.Setup(sw, shh)
	dt := 1 / capState.fps
	g.ui.DT = dt
	g.ui.time += dt
	g.frontT += dt
	g.lastDraw = nowFn()
	dst.Clear()
	sh.card(g, dst, t-sh.at)
	captureAfterDraw(g, dst)
	return true
}

// drawKeyart renders store art: the hall and the orb at a chosen spot,
// optionally the logo alone on black (for the transparent library logo).
func drawKeyart(g *Game, dst *ebiten.Image, ka js.Value) {
	sw, shh := float64(dst.Bounds().Dx()), float64(dst.Bounds().Dy())
	g.ui.Setup(sw, shh)
	g.ui.DT = 1 / capState.fps
	g.frontT += g.ui.DT
	num := func(k string, d float64) float64 {
		if v := ka.Get(k); v.Type() == js.TypeNumber {
			return v.Float()
		}
		return d
	}
	dst.Clear()
	if num("logoOnly", 0) > 0 {
		drawLogo(g, dst, num("logoY", 540), num("logoS", 1), 1)
		return
	}
	ox, oy, R := num("orbX", 960), num("orbY", 560), num("R", 160)
	g.drawHallK(dst, ox, oy, num("circle", 1))
	g.drawRunes(dst, int(num("runes", 30)), ox, 1500)
	g.drawTitleOrb(dst, ox, oy, R, g.frontT)
	if s := num("logoS", 0); s > 0 {
		drawLogo(g, dst, num("logoY", 250), s, 1)
	}
}

// directorOverlay adds captions, flashes and the fades over a frame.
func directorOverlay(g *Game, dst *ebiten.Image, t float64) {
	if v := js.Global().Get("__noOverlay"); v.Type() == js.TypeBoolean && v.Bool() {
		return
	}
	u := g.ui
	sw, sh := float64(dst.Bounds().Dx()), float64(dst.Bounds().Dy())
	for _, c := range trailerCaptions {
		if t < c.t0 || t > c.t1 {
			continue
		}
		a := math.Min(ease((t-c.t0)/0.35), ease((c.t1-t)/0.3))
		drawCaption(g, dst, c.text, a, ease((t-c.t0)/0.9))
	}
	for _, f := range trailerFlashes {
		if d := t - f[0]; d >= 0 && d < f[1] {
			a := (1 - d/f[1])
			fillRect(dst, 0, 0, sw, sh, ca(rgba(255, 246, 255, 255), f[2]*a*a))
		}
	}
	_ = u
}

var capCache = map[string]*ebiten.Image{}

// drawCaption is a lower-third title in engraved Cinzel with a soft band.
func drawCaption(g *Game, dst *ebiten.Image, s string, a, rise float64) {
	u := g.ui
	sw, sh := float64(dst.Bounds().Dx()), float64(dst.Bounds().Dy())
	band := 340.0 * u.S
	fillPath(dst, boxPath(0, sh-band, sw, band*0.65, 0, false), rgba(0, 0, 0, 0), ca(rgba(4, 2, 10, 255), 0.88*a), vgrad(sh-band, sh-band*0.35), false)
	fillRect(dst, 0, sh-band*0.35, sw, band*0.35, ca(rgba(4, 2, 10, 255), 0.88*a))
	txt := spaced(s)
	size := 58.0
	if w := u.measure(txt, fRune, size); w > 1380 {
		size *= 1380 / w // long lines shrink to fit between the rules
	}
	img, ok := capCache[txt]
	if !ok {
		img = ebiten.NewImage(1920, 160)
		g.arcaneTitle(img, txt, size, 960, 30)
		capCache[txt] = img
	}
	y := 868 + (1-rise)*16
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-960, -60)
	op.GeoM.Scale(u.S, u.S)
	op.GeoM.Translate(u.X(960), u.Y(y))
	op.ColorScale.ScaleAlpha(float32(a))
	dst.DrawImage(img, op)
	w := u.measure(txt, fRune, size)
	gold := rgba(214, 176, 104, 255)
	for _, sg := range []float64{-1, 1} {
		x0 := 960 + sg*(w/2+28)
		x1 := x0 + sg*120*rise
		u.glowLine(dst, math.Min(x0, x1), math.Max(x0, x1), y+6, 1, gold, 0.8*a)
		fillPath(dst, diamondPath(u.X(x1+sg*6), u.Y(y+6), 4*u.S, 6*u.S), ca(rgba(240, 210, 150, 255), a), ca(rgba(150, 110, 60, 255), a), grad{}, false)
	}
}
