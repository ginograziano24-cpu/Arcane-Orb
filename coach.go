package main

import (
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ---------------------------------------------------------------- keyword glossary

// glossTerm explains a word that appears in item text. Hovering a card
// lists every term its description uses.
type glossTerm struct {
	Name  string
	Match []string // lower-case fragments that mean this term
	Text  string
	Col   color.RGBA
}

var glossary = []glossTerm{
	{"Burn", []string{"burn", "ignite", "fire"}, "Burning bricks take damage every 0.7 s for a few seconds.", rgba(255, 140, 60, 255)},
	{"Frozen", []string{"freez", "frozen", "frost"}, "Frozen bricks take +50% damage from every hit while the ice lasts.", rgba(150, 220, 255, 255)},
	{"Crit", []string{"crit"}, "A critical hit deals double damage (more with crit items) and can damage armoured bricks.", rgba(255, 100, 100, 255)},
	{"Void", []string{"void"}, "Void hits ignore armour and shields.", rgba(190, 120, 255, 255)},
	{"Combo", []string{"combo"}, "Every brick the orb hits adds 1. It fades after 3 s without a hit and resets when an orb is lost.", rgba(255, 220, 110, 255)},
	{"Lightning", []string{"lightning", "arcs", "chain"}, "Arcs leap from brick to brick, striking several in a row.", rgba(170, 220, 255, 255)},
	{"Pierce", []string{"pierc", "passes through"}, "The orb carries on through bricks instead of bouncing off them.", rgba(220, 220, 240, 255)},
	{"Explosion", []string{"explo", "blast"}, "Damages every brick within the blast radius. Blasts also hurt armoured bricks.", rgba(255, 170, 80, 255)},
	{"Barrier", []string{"barrier"}, "A glowing wall below the paddle. Each charge bounces one orb that would have been lost.", rgba(120, 220, 255, 255)},
	{"Armour", []string{"armour", "armor"}, "Armoured bricks shrug off plain hits. Crits, void, blasts and elements get through, and six plain hits crack the armour.", rgba(200, 200, 215, 255)},
	{"Shield", []string{"shield"}, "Shield bricks protect every brick around them until they break. Void hits ignore them.", rgba(120, 200, 255, 255)},
	{"Echo", []string{"echo"}, "An echoing hit lands twice.", rgba(200, 200, 255, 255)},
	{"Execute", []string{"execut"}, "Bricks pushed below the threshold break outright.", rgba(255, 90, 90, 255)},
	{"Scroll", []string{"scroll"}, "Single-use spells. Read one during a chamber with 1, 2 or 3.", rgba(240, 220, 160, 255)},
	{"Curse", []string{"curse", "cursed"}, "A lingering penalty that stays with you until something lifts it.", rgba(200, 60, 120, 255)},
	{"Lives", []string{"a life", "lives", "extra life"}, "Dropping every orb costs a life. Lives refill to two each act, and barriers behind the paddle can save an orb.", rgba(255, 130, 170, 255)},
}

func glossFor(text string) []*glossTerm {
	lt := strings.ToLower(text)
	var out []*glossTerm
	for i := range glossary {
		for _, m := range glossary[i].Match {
			if strings.Contains(lt, m) {
				out = append(out, &glossary[i])
				break
			}
		}
	}
	return out
}

// glossReq is a hovered card waiting for its glossary panel (drawn last,
// above everything).
type glossReq struct {
	m          *Mod
	x, y, w, h float64
}

func (g *Game) wantGloss(m *Mod, x, y, w, h float64) {
	g.gloss = &glossReq{m, x, y, w, h}
}

func (g *Game) drawGloss(s *ebiten.Image) {
	req := g.gloss
	g.gloss = nil
	if req == nil {
		return
	}
	u := g.ui
	type entry struct {
		name, text string
		col        color.RGBA
	}
	var es []entry
	if req.m.Relic {
		es = append(es, entry{"Relic", "A rare, powerful item. Each relic can only be held once.", rgba(255, 200, 110, 255)})
	}
	for _, t := range glossFor(req.m.Desc) {
		es = append(es, entry{t.Name, t.Text, t.Col})
	}
	if len(fusionPartners(req.m.ID)) > 0 {
		es = append(es, entry{"Fusion", "Own both halves of a fusion and they forge into a stronger item. The Codex lists every fusion you've found.", rgba(255, 170, 230, 255)})
	}
	if len(es) == 0 {
		return
	}
	w := math.Max(360, req.w)
	ts := 17 * g.textScale()
	h := 20.0
	wrapped := make([][]string, len(es))
	for i, e := range es {
		wrapped[i] = u.wrap(e.text, fReg, ts, w-32)
		h += 30 + float64(len(wrapped[i]))*ts*1.35 + 10
	}
	// below the card if it fits, else above, else beside it
	x := math.Max(20, math.Min(1900-w, req.x+req.w/2-w/2))
	y := req.y + req.h + 16
	switch {
	case y+h <= 1070:
	case req.y-h-16 >= 10:
		y = req.y - h - 16
	default:
		x, y = req.x+req.w+18, math.Max(20, math.Min(req.y, 1060-h))
		if x+w > 1900 {
			x = req.x - w - 18
		}
	}
	fillPath(s, boxPath(u.X(x+4), u.Y(y+4), (w-8)*u.S, (h-8)*u.S, 10*u.S, false), rgba(14, 9, 26, 255), rgba(8, 5, 16, 255), vgrad(u.Y(y), u.Y(y+h)), false)
	u.panel(s, x, y, w, h, 0.97)
	yy := y + 16
	for i, e := range es {
		u.text(s, strings.ToUpper(e.name), fBold, 17, x+18, yy, e.col, alignLeft)
		yy += 28
		for _, l := range wrapped[i] {
			u.text(s, l, fReg, ts, x+18, yy, rgba(225, 222, 238, 255), alignLeft)
			yy += ts * 1.35
		}
		yy += 10
	}
}

// ---------------------------------------------------------------- coach tips

// A coach tip explains something the first time it happens. The game
// waits while it's on screen; a click or any confirm key moves on.
type coachTip struct {
	ID, Title, Body string
	Glyph           string
	Basic           bool // skipped for players who already finished a run
}

var coachTips = map[string]coachTip{
	"level":   {"level", "Break every brick", "Steer with {move} and launch the Orb with {launch}. Break every brick and the chamber is yours.\n\nIf every orb slips past your paddle, you lose a life. You have two, they refill each act, and now and then a broken brick drops a heart to patch you up.\n\nWhen your hero's ability is charged, unleash it with {ability}.", "orb", true},
	"map":     {"map", "Choose your path", "Each row is one step up the act; the lines show where you can go next.\n\nChambers hand out upgrades, Elites hand out better ones, Shops want your gold, Events are a coin toss with extra steps, and Rest sites give back a life. The act's Guardian waits at the very top, tapping its foot.", "eye", true},
	"reward":  {"reward", "Take one", "Pick one card, or skip it for a little gold. Hover a card to decode its keywords.\n\nSome items fuse: own both halves and they forge into something much nastier. Cards that complete a fusion will tell you so.", "star", true},
	"shop":    {"shop", "The shop", "Gold buys items, relics, scrolls and spare lives. Whatever you leave on the shelf is gone for good, so no window shopping.", "coin", true},
	"scroll":  {"scroll", "A scroll", "Scrolls are one-shot spells. Read one mid-chamber with {scrolls}. You can carry a few, so keep one for a rainy day.", "book", false},
	"life":    {"life", "A life lost", "Don't panic: the paddle pulls itself back together. Lives return at the next act, or sooner if a broken brick drops a heart. Catch it!", "heart", false},
	"curse":   {"curse", "Cursed", "A curse nags at you until something lifts it. Keep an eye out for events and items that wash curses away.", "mask", false},
	"waves":   {"waves", "Waves and heat", "This chamber comes in waves: when one is nearly gone the rest crumble and the next drops in.\n\nThe formation creeps closer when the ADVANCE gauge under the title fills. Bricks that reach the red line near your paddle smash: the paddle reels and you spill gold.\n\nHEAT is how strong your build has grown this world. Each point toughens the bricks and hurries the advance.", "burst", false},
	"critter": {"critter", "Critters", "Little critters wander the open floor. Bonk one with the Orb and it pops into coins, sometimes a power-up. The golden ones are rare, quick and worth a fortune.", "coin", false},
	"thief":   {"thief", "Stop, thief!", "A Pilferer has its eye on that power-up! If it grabs one it runs for the wall. Bonk it with the Orb and it drops everything it stole.", "coin", false},
	"mason":   {"mason", "The Mason", "This little builder lays fresh bricks in front of the formation. Rude! Hit it with the Orb to send it packing, with a few coins for your trouble.", "anvil", false},
	"bomber":  {"bomber", "Bombs away", "A bird with a bomb is flying over the bricks. Hit it while it's above a big crowd of bricks and the bomb drops right there. Boom.", "bomb", false},
	"cache":   {"cache", "Rune cache", "Bricks sometimes drop a rune cache. Catch it with the paddle and the fight pauses while you pick an item, so your build grows mid-chamber too.", "gem", false},
	"boss":    {"boss", "The Guardian", "Its armour laughs at the Orb. Strike the glowing core instead; it hops away after a few hits, so keep chasing it.", "crown", false},
}

func (g *Game) coach(id string) {
	t, ok := coachTips[id]
	if !ok || g.settings.NoTips || g.save.Hints[id] || inputHook != nil {
		return
	}
	if r := g.run; r == nil || r.Duel || r.SeedCode == "MENU" || r.SeedCode == "SIM" || r.SeedCode == "TRAILER" || r.SeedCode == "LIFE" || r.SeedCode == "SAVEME" {
		return
	}
	if t.Basic && g.save.Runs > 1 { // Runs counts the run in progress
		return
	}
	if g.save.Hints == nil {
		g.save.Hints = map[string]bool{}
	}
	g.save.Hints[id] = true
	for _, q := range g.coachQ {
		if q.ID == id {
			return
		}
	}
	g.coachQ = append(g.coachQ, t)
	g.coachT = 0
}

// updateCoach runs while a tip is up; it reports whether the game should
// hold still.
func (g *Game) updateCoach(dt float64) bool {
	if len(g.coachQ) == 0 {
		return false
	}
	g.coachT += dt
	if g.coachT > 0.45 && (g.ui.Click || inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeyEscape) || padJust(ebiten.StandardGamepadButtonRightBottom) || padJust(ebiten.StandardGamepadButtonRightRight)) {
		g.coachQ = g.coachQ[1:]
		g.coachT = 0
		g.snd.Play("click")
		writeSave(&g.save)
	}
	g.consumeInput()
	return true
}

// coachText fills in the player's current bindings.
func (g *Game) coachText(s string) string {
	return strings.NewReplacer(
		"{move}", tr(g.bindText("move")),
		"{launch}", tr(g.bindText("launch")),
		"{ability}", tr(g.bindText("ability")),
		"{scrolls}", tr(g.bindText("scrolls")),
	).Replace(s)
}

func (g *Game) drawCoach(s *ebiten.Image) {
	if len(g.coachQ) == 0 {
		return
	}
	u := g.ui
	t := g.coachQ[0]
	k := ease(math.Min(1, g.coachT*4))
	fillRect(s, 0, 0, u.SW, u.SH, color.RGBA{4, 2, 10, uint8(150 * k)})
	ts := 22 * g.textScale()
	const w = 860.0
	var lines []string
	body := tr(t.Body) // whole, before it's split or filled in
	if r := g.run; r == nil || r.Arch.Ability == "" {
		// no active ability for this class: drop the paragraph about it
		var keep []string
		for _, p := range strings.Split(body, "\n\n") {
			if !strings.Contains(p, "{ability}") {
				keep = append(keep, p)
			}
		}
		body = strings.Join(keep, "\n\n")
	}
	raw := u.rawText
	u.rawText = true
	for _, para := range strings.Split(g.coachText(body), "\n") {
		if para == "" {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, u.wrap(para, fReg, ts, w-120)...)
	}
	u.rawText = raw
	h := 190 + float64(len(lines))*ts*1.4
	x, y := 960-w/2, 540-h/2+(1-k)*30
	u.frame(s, x, y, w, h, 0.97*k)
	u.drawEmblem(s, glyphImage(t.Glyph, cGold), x+70, y+66, 72, k)
	u.text(s, "TIP", fBold, 16, x+122, y+30, ca(cDim, k), alignLeft)
	u.text(s, t.Title, fRune, 38, x+122, y+48, ca(cGold, k), alignLeft)
	yy := y + 122
	u.rawText = true // lines are translated already
	for _, l := range lines {
		u.text(s, l, fReg, ts, x+60, yy, ca(rgba(232, 228, 245, 255), k), alignLeft)
		yy += ts * 1.4
	}
	u.rawText = raw
	if g.coachT > 0.45 {
		hint := "Click or press Space to continue"
		if g.padActive {
			hint = "Press A to continue"
		}
		u.text(s, hint, fReg, 18, 960, y+h-46, ca(cDim, k*(0.65+0.35*math.Sin(u.time*4))), alignCenter)
	}
	u.text(s, "Tips can be turned off in Settings", fReg, 14, x+w-24, y+h-26, ca(rgba(130, 120, 150, 255), k), alignRight)
}
