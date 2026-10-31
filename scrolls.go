package main

import (
	"image/color"
	"math"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"

	"orb/levelgen"
)

// Scrolls are one-shot spells you carry between chambers (three slots)
// and read mid-fight: keys 1-3, or Y on a pad (LB picks the slot).
// They drop from elites and bosses, now and then from cleared chambers,
// and are sold in shops and handed out by events.

type scrollDef struct {
	ID, Name, Desc, Glyph string
	Col                   color.RGBA
	Price                 int
	Boss                  bool // works on a Guardian fight
	Use                   func(L *Level) bool
}

var scrollDefs = []scrollDef{
	{"multi", "Scroll of Multitudes", "Every orb in play splits into three.", "twinorb", rgba(255, 210, 70, 255), 40, true, func(L *Level) bool {
		L.release()
		var add []*Ball
		for _, b := range L.Balls {
			for _, a := range []float64{-0.35, 0.35} {
				if len(L.Balls)+len(add) >= 18 {
					break
				}
				ca, sa := math.Cos(a), math.Sin(a)
				nb := &Ball{X: b.X, Z: b.Z, R: b.R, VX: b.VX*ca - b.VZ*sa, VZ: b.VX*sa + b.VZ*ca, PierceLeft: b.PierceLeft, Phantom: b.Phantom}
				fixAngle(nb)
				add = append(add, nb)
			}
		}
		L.Balls = append(L.Balls, add...)
		return true
	}},
	{"meteor", "Scroll of Falling Stars", "Three meteors crash into the thickest bricks.", "comet", rgba(255, 130, 50, 255), 45, true, func(L *Level) bool {
		for i := 0; i < 3; i++ {
			L.meteor()
		}
		return true
	}},
	{"frost", "Scroll of Frost Nova", "Freeze every brick for 6 seconds. Frozen bricks take +50% damage.", "snow", rgba(150, 220, 255, 255), 35, false, func(L *Level) bool {
		n := 0
		for _, b := range L.Bricks {
			if b.Alive && b.Kind != levelgen.Steel && !b.BossPart {
				b.Frozen = 6 + L.g.run.Stats.FreezeDur
				n++
			}
		}
		L.Rings = append(L.Rings, &Ring{X: 0, Z: 18, MaxR: 20, Life: 0.6, Max: 0.6, Col: color.RGBA{170, 230, 255, 255}})
		return n > 0
	}},
	{"inferno", "Scroll of Wildfire", "Set every brick ablaze.", "flame", rgba(255, 110, 40, 255), 40, true, func(L *Level) bool {
		for _, b := range L.Bricks {
			if b.Alive && b.Kind != levelgen.Steel {
				b.Burn = 5
			}
		}
		L.Rings = append(L.Rings, &Ring{X: 0, Z: 18, MaxR: 20, Life: 0.6, Max: 0.6, Col: color.RGBA{255, 140, 60, 255}})
		return true
	}},
	{"storm", "Scroll of the Tempest", "Twelve bolts of chain lightning rain down.", "storm", rgba(150, 200, 255, 255), 45, true, func(L *Level) bool {
		s := &L.g.run.Stats
		dmg := math.Max(1.5, s.Damage*s.DmgMult*1.5)
		var live []*Brick
		for _, b := range L.Bricks {
			if b.Alive && b.Kind != levelgen.Steel && (!b.BossPart || b.Core) {
				live = append(live, b)
			}
		}
		for i := 0; i < 12 && len(live) > 0; i++ {
			b := live[L.rng.Intn(len(live))]
			L.lightning(b.X(), b.Z(), 2+s.ChainCount, dmg, nil, false)
		}
		return len(live) > 0
	}},
	{"slow", "Scroll of Stillness", "Time slows to half speed for 8 seconds.", "hourglass", rgba(200, 180, 255, 255), 30, true, func(L *Level) bool {
		L.timeWarp = 8
		return true
	}},
	{"aegis", "Scroll of the Aegis", "Raise three barrier charges.", "shield", rgba(170, 150, 255, 255), 35, true, func(L *Level) bool {
		L.Barrier = min(5, L.Barrier+3)
		return true
	}},
	{"heal", "Draught of Mending", "Restore 30 health.", "heart", rgba(255, 90, 120, 255), 30, true, func(L *Level) bool {
		r := L.g.run
		if r.HP >= r.Stats.MaxHP {
			return false
		}
		r.HP = math.Min(r.Stats.MaxHP, r.HP+30)
		L.g.snd.PlayKeyed("heal")
		return true
	}},
	{"midas", "Scroll of Midas", "For 10 seconds every hit throws off a coin.", "coinstack", rgba(255, 214, 90, 255), 30, false, func(L *Level) bool {
		L.midasT = 10
		return true
	}},
	{"giant", "Scroll of the Titan", "Your orbs swell to twice their size and pierce for 10 seconds.", "orb", rgba(255, 160, 220, 255), 40, true, func(L *Level) bool {
		L.titanT = 10
		return true
	}},
	{"laser", "Scroll of Searing Light", "Paddle lasers fire rapidly for 12 seconds.", "laser", rgba(255, 90, 90, 255), 35, true, func(L *Level) bool {
		L.Timers[PLaser] = 12
		return true
	}},
	{"quake", "Scroll of the Earthshaker", "Every brick takes a heavy blow.", "shatter", rgba(210, 170, 120, 255), 45, true, func(L *Level) bool {
		s := &L.g.run.Stats
		dmg := math.Max(2, s.Damage*s.DmgMult*2)
		var list []*Brick
		for _, b := range L.Bricks {
			if b.Alive && b.Kind != levelgen.Steel && (!b.BossPart || b.Core) {
				list = append(list, b)
			}
		}
		for _, b := range list {
			if b.Alive {
				d := dmg
				if b.Core {
					d *= 2
				}
				L.damage(b, Hit{Src: "Scrolls", Dmg: d, Silent: true})
			}
		}
		L.Shake = math.Max(L.Shake, 0.9)
		return len(list) > 0
	}},
	{"void", "Scroll of Unmaking", "Erase the six toughest bricks.", "blackhole", rgba(170, 110, 255, 255), 55, false, func(L *Level) bool {
		var list []*Brick
		for _, b := range L.Bricks {
			if b.Alive && b.Kind.Required() && !b.BossPart {
				list = append(list, b)
			}
		}
		sort.Slice(list, func(i, j int) bool { return list[i].HP > list[j].HP })
		for i := 0; i < 6 && i < len(list); i++ {
			L.damage(list[i], Hit{Src: "Scrolls", Dmg: list[i].HP + 1000, Void: true, Silent: true})
		}
		return len(list) > 0
	}},
	{"magnet", "Lodestone Charm", "Pull every falling coin and power-up to your paddle.", "magnet", rgba(120, 200, 255, 255), 25, true, func(L *Level) bool {
		for _, d := range L.Drops {
			d.X, d.Z = L.PaddleX, PaddleZ+0.4
		}
		L.magnetT = 6
		return true
	}},
}

var scrollByID = func() map[string]*scrollDef {
	m := map[string]*scrollDef{}
	for i := range scrollDefs {
		m[scrollDefs[i].ID] = &scrollDefs[i]
	}
	return m
}()

// addScroll gives the player a scroll if a slot is free.
func (g *Game) addScroll(id string) bool {
	r := g.run
	sd := scrollByID[id]
	if r == nil || sd == nil || len(r.Scrolls) >= r.scrollCap() {
		return false
	}
	r.Scrolls = append(r.Scrolls, id)
	g.save.Seen["scroll:"+id] = true
	g.toastImg("Scroll found", sd.Name+" — "+sd.Desc, sd.Col, glyphImage(sd.Glyph, sd.Col))
	g.snd.Play("pick.1." + itoa(g.snd.Key))
	g.coach("scroll")
	return true
}

func randomScroll(rng *rand.Rand) string { return scrollDefs[rng.Intn(len(scrollDefs))].ID }

// useScroll reads the scroll in slot i.
func (L *Level) useScroll(i int) {
	g := L.g
	r := g.run
	if i < 0 || i >= len(r.Scrolls) || L.Cleared || L.introT > 0 {
		return
	}
	sd := scrollByID[r.Scrolls[i]]
	if sd == nil {
		return
	}
	if L.Boss != nil && !sd.Boss {
		L.text(V3{L.PaddleX, 1.6, PaddleZ + 1.5}, "The Guardian resists", cDim, false)
		g.snd.PlayVol("deny", 0.6)
		return
	}
	if !sd.Use(L) {
		g.snd.PlayVol("deny", 0.6)
		return
	}
	if L.rng.Float64() < r.Stats.ScrollKeep {
		L.text(V3{L.PaddleX, 2.8, PaddleZ + 2.4}, "ECHO — the scroll remains", rgba(200, 180, 255, 255), false)
	} else {
		r.Scrolls = append(r.Scrolls[:i], r.Scrolls[i+1:]...)
	}
	g.scrollSel = min(g.scrollSel, max(0, len(r.Scrolls)-1))
	L.text(V3{L.PaddleX, 2.2, PaddleZ + 2}, sd.Name, lighten(sd.Col, 0.3), true)
	L.Flash = math.Max(L.Flash, 0.25)
	L.abilFlash = 1
	g.snd.Play("scroll")
	g.snd.Duck(0.5, 0.8)
}

// scrollInput reads the scroll keys during a chamber.
func (g *Game) scrollInput(L *Level) {
	for i, id := range []string{"scroll1", "scroll2", "scroll3"} {
		if g.actJust(id) {
			L.useScroll(i)
		}
	}
	if padJust(ebiten.StandardGamepadButtonFrontTopLeft) && len(g.run.Scrolls) > 0 {
		g.scrollSel = (g.scrollSel + 1) % len(g.run.Scrolls)
		g.snd.Play("hover")
	}
	if g.actJust("readsel") {
		L.useScroll(g.scrollSel)
	}
}

// drawScrolls shows the three scroll slots (bottom-right, above relics).
func (g *Game) drawScrolls(screen *ebiten.Image, L *Level) {
	u := g.ui
	S := u.S
	r := g.run
	th := u.theme()
	size, gap := 58.0, 10.0
	n := r.scrollCap()
	x0 := 1898 - float64(n)*(size+gap) + gap
	y := 944.0
	for i := 0; i < n; i++ {
		x := x0 + float64(i)*(size+gap)
		box := boxPath(u.X(x), u.Y(y), size*S, size*S, 10*S, false)
		fillPath(screen, box, rgba(20, 14, 34, 200), rgba(8, 6, 16, 220), vgrad(u.Y(y), u.Y(y+size)), false)
		edge := ca(th.Edge, 0.6)
		if g.padActive && i == g.scrollSel && i < len(r.Scrolls) {
			edge = ca(cGold, 0.9+0.1*math.Sin(L.T*6))
		}
		if !u.kitFrame(screen, x+3, y+3, size-6, size-6, 0.95) || edge != ca(th.Edge, 0.6) {
			strokePath(screen, box, 1.6*S, edge, false)
		}
		if i < len(r.Scrolls) {
			sd := scrollByID[r.Scrolls[i]]
			drawSprite(screen, sprSoft, u.X(x-10), u.Y(y-10), (size+20)*S, (size+20)*S, ca(sd.Col, 0.18), true)
			u.drawEmblem(screen, glyphImage(sd.Glyph, sd.Col), x+size/2, y+size/2, 44, 1)
			if !g.padActive {
				u.keycap(screen, itoa(i+1), x+size-18, y-12, 0.9)
			}
			if u.hover(x, y, size, size) {
				g.tooltip = sd.Name + " — " + sd.Desc
			}
		}
	}
	if g.padActive && len(r.Scrolls) > 0 {
		u.text(screen, "LB pick · Y read", fBold, 14, 1898, y-22, cDim, alignRight)
	}
}
