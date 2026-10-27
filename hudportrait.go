package main

import (
	"bytes"
	"image"
	_ "image/png"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// The hero portrait in the top-left corner of the HUD. It idles through a
// few poses (rest, a glance aside, a determined stare) and reacts to the
// run: it flinches when hurt, cracks and greys when a life is lost, lights
// up gold for new upgrades and fusions, and grows grim when a Guardian
// rises. Pose art lives in assets/portraits/hud/<origin>_<pose>.png.

type reaction int

const (
	reactNone reaction = iota
	reactHurt
	reactLifeLost
	reactLife
	reactUpgrade
	reactFusion
	reactCombo
	reactBoss
	reactClear
)

type heroPortrait struct {
	pose, prev string
	fade       float64 // 0..1 cross-fade from prev to pose
	cycleT     float64
	cycleI     int
	react      reaction
	reactT     float64 // seconds left on the reaction
	reactMax   float64
	shake      float64
	lastCombo  int
	bossSeen   *Level
	clearSeen  *Level
}

var heroPoses = []string{"rest", "look", "rest", "fierce"}

var hudPortraitCache = map[string]*ebiten.Image{}

func hudPortrait(id, pose string) *ebiten.Image {
	key := id + "_" + pose
	if img, ok := hudPortraitCache[key]; ok {
		return img
	}
	var img *ebiten.Image
	if b, err := portraitFS.ReadFile("assets/portraits/hud/" + key + ".png"); err == nil {
		if im, _, err := image.Decode(bytes.NewReader(b)); err == nil {
			img = ebiten.NewImageFromImage(im)
		}
	}
	if img == nil && pose != "rest" {
		img = hudPortrait(id, "rest")
	}
	hudPortraitCache[key] = img
	return img
}

// react starts a portrait reaction; stronger reactions win over weaker
// ones that are still playing.
func (g *Game) react(r reaction) {
	p := &g.hero
	rank := func(x reaction) int {
		switch x {
		case reactLifeLost:
			return 5
		case reactLife, reactFusion:
			return 4
		case reactHurt, reactBoss:
			return 3
		case reactUpgrade, reactClear:
			return 2
		}
		return 1
	}
	if p.reactT > 0 && rank(r) < rank(p.react) {
		return
	}
	dur := map[reaction]float64{reactHurt: 0.9, reactLifeLost: 2.6, reactLife: 2.0, reactUpgrade: 1.6,
		reactFusion: 2.4, reactCombo: 1.2, reactBoss: 3.0, reactClear: 2.2}[r]
	p.react, p.reactT, p.reactMax = r, dur, dur
	switch r {
	case reactHurt:
		p.shake = 0.6
		p.setPose("hurt")
	case reactLifeLost:
		p.shake = 1
		p.setPose("hurt")
	case reactBoss:
		p.setPose("look")
	default:
		p.setPose("fierce")
	}
}

func (p *heroPortrait) setPose(s string) {
	if s == p.pose {
		return
	}
	p.prev, p.pose, p.fade = p.pose, s, 0
}

// updateHero advances the idle cycle and watches the level for moments
// worth reacting to.
func (g *Game) updateHero(L *Level, dt float64) {
	p := &g.hero
	if p.pose == "" {
		p.pose, p.prev, p.fade = "rest", "rest", 1
	}
	p.fade = math.Min(1, p.fade+dt*3.2)
	p.shake = math.Max(0, p.shake-dt*1.8)
	if p.reactT > 0 {
		p.reactT -= dt
		if p.reactT <= 0 {
			p.react = reactNone
			p.cycleT = 0
			p.setPose(g.restPose())
		}
	} else {
		p.cycleT += dt
		if p.cycleT > 4.5 {
			p.cycleT = 0
			p.cycleI = (p.cycleI + 1) % len(heroPoses)
			pose := heroPoses[p.cycleI]
			if pose == "rest" {
				pose = g.restPose()
			}
			p.setPose(pose)
		}
	}
	if L == nil {
		return
	}
	if L.Boss != nil && p.bossSeen != L {
		p.bossSeen = L
		g.react(reactBoss)
	}
	if L.Cleared && p.clearSeen != L {
		p.clearSeen = L
		g.react(reactClear)
	}
	for _, m := range []int{25, 50, 100} {
		if p.lastCombo < m && L.Combo >= m {
			g.react(reactCombo)
		}
	}
	p.lastCombo = L.Combo
}

// restPose is the idle face: battered when the run is on its last legs.
func (g *Game) restPose() string {
	if r := g.run; r != nil && (r.Lives <= 1 && r.HP < r.Stats.MaxHP*0.35) {
		return "hurt"
	}
	return "rest"
}

// drawHero draws the framed portrait at (x, y), w by h.
func (g *Game) drawHero(dst *ebiten.Image, L *Level, x, y, w, h float64) {
	u := g.ui
	S := u.S
	r := g.run
	p := &g.hero
	info := classInfo[r.Arch.ID]
	col := info.Col
	t := u.time
	k := 0.0
	if p.reactMax > 0 {
		k = clamp01(p.reactT / p.reactMax)
	}
	// frame: an arched window with the origin's colour behind
	u.frame(dst, x, y, w, h, 0.95)
	ix, iy, iw, ih := x+8, y+8, w-16, h-16
	fillPath(dst, boxPath(u.X(ix), u.Y(iy), iw*S, ih*S, 8*S, false), ca(darken(col, 0.55), 0.9), rgba(6, 4, 12, 240), vgrad(u.Y(iy), u.Y(iy+ih)), false)
	glow := 0.25 + 0.08*math.Sin(t*1.7)
	switch p.react {
	case reactUpgrade, reactLife, reactClear, reactCombo:
		glow += 0.5 * k
	case reactFusion:
		glow += 0.8 * k
	}
	drawSprite(dst, sprSoft, u.X(ix-iw*0.2), u.Y(iy+ih*0.05), iw*1.4*S, ih*1.1*S, ca(col, glow), true)
	// the portrait, clipped to the window
	clip := dst.SubImage(image.Rect(int(u.X(ix)), int(u.Y(iy)), int(u.X(ix+iw)), int(u.Y(iy+ih)))).(*ebiten.Image)
	breathe := 1 + 0.012*math.Sin(t*1.9)
	sx, sy := 0.0, 0.0
	if p.shake > 0 {
		sx = math.Sin(t*61) * 6 * p.shake
		sy = math.Cos(t*47) * 3 * p.shake
	}
	bob := math.Sin(t*1.9) * 1.5
	drawPose := func(pose string, a float64) {
		img := hudPortrait(r.Arch.ID, pose)
		if img == nil || a <= 0 {
			return
		}
		bw, bh := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
		sc := iw / bw * breathe
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Translate(-bw/2, -bh)
		op.GeoM.Scale(sc*S, sc*S)
		op.GeoM.Translate(u.X(ix+iw/2+sx), u.Y(iy+ih+sy+bob+4))
		switch {
		case p.react == reactLifeLost:
			// drained of colour and blood-red
			op.ColorScale.Scale(float32(1-0.2*k), float32(1-0.55*k), float32(1-0.55*k), 1)
		case p.react == reactHurt:
			op.ColorScale.Scale(1, float32(1-0.45*k), float32(1-0.45*k), 1)
		case p.react == reactBoss:
			op.ColorScale.Scale(float32(1-0.25*k), float32(1-0.25*k), float32(1-0.15*k), 1)
		}
		op.ColorScale.ScaleAlpha(float32(a))
		clip.DrawImage(img, op)
	}
	if p.fade < 1 {
		drawPose(p.prev, 1-p.fade)
	}
	drawPose(p.pose, p.fade)
	// reaction overlays
	switch p.react {
	case reactLifeLost:
		fillRectA(clip, u.X(ix), u.Y(iy), iw*S, ih*S, rgba(255, 30, 50, 255), 0.35*k)
		g.drawCracks(clip, ix, iy, iw, ih, k)
	case reactHurt:
		fillRectA(clip, u.X(ix), u.Y(iy), iw*S, ih*S, rgba(255, 40, 60, 255), 0.3*k)
	case reactUpgrade, reactLife, reactFusion, reactClear:
		c := cGold
		if p.react == reactLife {
			c = rgba(255, 140, 180, 255)
		}
		if p.react == reactFusion {
			c = lighten(col, 0.3)
		}
		// a sweep of light across the glass, and rising motes
		sw := ix - iw*0.4 + (1-k)*iw*1.8
		drawSprite(clip, sprSoft, u.X(sw-30), u.Y(iy), 60*S, ih*S, ca(c, 0.45*k), true)
		for i := 0; i < 8; i++ {
			f := float64(i)
			ph := math.Mod((1-k)*1.4+f*0.13, 1)
			mx := ix + iw*math.Mod(f*0.37+0.1, 1)
			my := iy + ih*(1-ph)
			drawSprite(clip, sprSoft, u.X(mx-6), u.Y(my-6), 12*S, 12*S, ca(c, k*math.Sin(ph*math.Pi)), true)
		}
	}
	// low health: a slow red pulse around the edge
	if r.HP < r.Stats.MaxHP*0.3 || r.Lives <= 1 {
		a := 0.18 + 0.14*math.Max(0, math.Sin(t*5))
		strokePath(dst, boxPath(u.X(ix), u.Y(iy), iw*S, ih*S, 8*S, false), 3*S, ca(rgba(255, 50, 70, 255), a), false)
	}
	strokePath(dst, boxPath(u.X(ix), u.Y(iy), iw*S, ih*S, 8*S, false), 1.5*S, ca(lighten(col, 0.2), 0.7), false)
	// name plate
	u.textOutlined(dst, strings.ToUpper(r.Arch.Name), fSerif, 16, x+w/2, y+h-30, ca(lighten(col, 0.35), 0.95), alignCenter)
}

// drawCracks scratches a few fracture lines over the portrait glass.
func (g *Game) drawCracks(dst *ebiten.Image, x, y, w, h, a float64) {
	u := g.ui
	S := u.S
	cx, cy := x+w*0.62, y+h*0.38
	for i := 0; i < 6; i++ {
		an := float64(i)*1.05 + 0.3
		px, py := cx, cy
		for s := 0; s < 4; s++ {
			ln := 14 + 10*math.Mod(float64(i*7+s*3), 5)
			an += (math.Mod(float64(i*13+s*5), 7) - 3) * 0.12
			nx, ny := px+math.Cos(an)*ln, py+math.Sin(an)*ln
			strokePath(dst, linePath(u.X(px), u.Y(py), u.X(nx), u.Y(ny)), 1.6*S, ca(rgba(255, 235, 240, 255), 0.75*a), false)
			px, py = nx, ny
		}
	}
}
