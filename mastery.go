package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// Origin Mastery: every origin levels from 1 to 10 with the XP its runs
// earn. Levels unlock options rather than raw power: a look for the
// paddle and orb, a signature item that joins that origin's pool, an
// alternate passive (Aspect), a starting gift, and only at the very end a
// small stat bonus. Duels ignore mastery entirely.

const masteryMax = 10

// masteryXP is the total XP an origin needs to reach each level (index 0 =
// level 1).
var masteryXP = []int{0, 300, 800, 1500, 2400, 3500, 4800, 6300, 8000, 10000}

func masteryLevel(xp int) int {
	lv := 1
	for lv < masteryMax && xp >= masteryXP[lv] {
		lv++
	}
	return lv
}

// option bits stored per origin
const (
	mOptLivery = 1 << iota
	mOptTrail
	mOptAspect
	mOptGift
)

type masteryStep struct {
	Lv         int
	Name, Desc string
	Opt        int // the toggle this level unlocks (0 = always on)
}

func masterySteps(id string) []masteryStep {
	o := originMastery[id]
	sig := modByID[o.Sig]
	sigName := "Signature item"
	if sig != nil {
		sigName = sig.Name
	}
	return []masteryStep{
		{2, "Livery", "Your paddle wears the " + o.Livery + " colours.", mOptLivery},
		{3, sigName, "A signature item joins this origin's item pool.", 0},
		{4, "Orb Trail", "Your orb burns in this origin's colour.", mOptTrail},
		{5, "Aspect of " + o.Aspect, o.AspectDesc, mOptAspect},
		{6, "Starting Gift", "Begin every world already holding " + sigName + ".", mOptGift},
		{7, "Seasoned", "+1 free shop reroll.", 0},
		{8, "Silver Frame", "A silver frame for your portrait.", 0},
		{9, "Honed", "+5% damage.", 0},
		{10, "Master", "A gold frame and the title of Master.", 0},
	}
}

type originM struct {
	Sig                string
	Livery             string
	Body, Cap, Trail   color.RGBA
	Aspect, AspectDesc string
}

var originMastery = map[string]originM{
	"guardian": {"sig_halo", "Warden's", rgba(236, 228, 210, 255), rgba(255, 200, 90, 255), rgba(255, 215, 130, 255),
		"Aegis", "Parry catches on a far wider part of the paddle, but needs 1.4 seconds between waves and strikes 3 bricks."},
	"berserker": {"sig_bloodprice", "Forge-pit", rgba(70, 24, 26, 255), rgba(220, 50, 40, 255), rgba(255, 80, 60, 255),
		"Bloodlust", "Rage builds faster, +8% per break up to 12 stacks (+96%), but fades after 3 seconds."},
	"arcane": {"sig_lens", "Weaver's", rgba(60, 40, 110, 255), rgba(120, 230, 255, 255), rgba(170, 120, 255, 255),
		"Flux", "Your element changes every 4 seconds instead of 8, but is unleashed on 35% of hits."},
	"engineer": {"sig_coolant", "Crucible", rgba(70, 74, 84, 255), rgba(255, 150, 50, 255), rgba(255, 170, 80, 255),
		"Railgun", "Lasers fire every 3.2 seconds, but every volley pierces a brick and hits 40% harder."},
	"lucky": {"sig_dice", "Lucky Green", rgba(24, 80, 56, 255), rgba(255, 214, 90, 255), rgba(90, 230, 160, 255),
		"Jackpot", "High Roller ramps faster, +1% damage for every 4 gold, but stops at +40%."},
	"pyro": {"sig_ember", "Ember", rgba(60, 26, 18, 255), rgba(255, 120, 30, 255), rgba(255, 130, 40, 255),
		"Wildfire", "Every 3rd hit ignites, but stoking a burning brick adds +25% damage instead of +50%."},
	"storm": {"sig_conduit", "Thunderhead", rgba(36, 50, 86, 255), rgba(200, 230, 255, 255), rgba(120, 200, 255, 255),
		"Overcharge", "Static builds 35% slower, but a supercharge lasts 5 seconds."},
}

// signature items: only offered to their origin from Mastery 3
var sigMods = []*Mod{
	{ID: "sig_halo", Origin: "guardian", Icon: "halo", Name: "Halo of Mercy", Desc: "Guardian signature: Parry waves strike 2 more bricks, and every Parry restores 1 health.", Rarity: Uncommon, Tags: []string{"guard"}, Max: 1,
		Apply: func(s *Stats) { s.SigHalo = true }},
	{ID: "sig_bloodprice", Origin: "berserker", Icon: "fang", Name: "Blood Price", Desc: "Berserker signature: Rage survives dropping the orb and lasts 3 seconds longer.", Rarity: Uncommon, Tags: []string{"power"}, Max: 1,
		Apply: func(s *Stats) { s.SigBloodPrice = true }},
	{ID: "sig_lens", Origin: "arcane", Icon: "lens", Name: "Resonant Lens", Desc: "Arcanist signature: Attunement triggers 15% more often and its elements hit 50% harder.", Rarity: Uncommon, Tags: []string{"void"}, Max: 1,
		Apply: func(s *Stats) { s.SigLens = true }},
	{ID: "sig_coolant", Origin: "engineer", Icon: "battery", Name: "Coolant Loop", Desc: "Engineer signature: Overclock charges in 0.8 seconds, and overclocked beams pierce 2 more bricks.", Rarity: Uncommon, Tags: []string{"power"}, Max: 1,
		Apply: func(s *Stats) { s.SigCoolant = true }},
	{ID: "sig_dice", Origin: "lucky", Icon: "clover", Name: "Loaded Dice", Desc: "Gambler signature: High Roller's limit rises by 30%, and critical hits have a 20% chance to drop a coin.", Rarity: Uncommon, Tags: []string{"wealth"}, Max: 1,
		Apply: func(s *Stats) { s.SigDice = true; s.CritCoin += 0.2 }},
	{ID: "sig_ember", Origin: "pyro", Icon: "volcano", Name: "Ember Heart", Desc: "Pyromancer signature: a burning brick that breaks sets every neighbour alight.", Rarity: Uncommon, Tags: []string{"fire"}, Max: 1,
		Apply: func(s *Stats) { s.SigEmber = true }},
	{ID: "sig_conduit", Origin: "storm", Icon: "storm", Name: "Lightning Rod", Desc: "Stormcaller signature: supercharges last 2 seconds longer and their arcs jump to one more brick.", Rarity: Uncommon, Tags: []string{"storm"}, Max: 1,
		Apply: func(s *Stats) { s.SigConduit = true }},
}

func init() {
	for _, m := range sigMods {
		modList = append(modList, m)
		modByID[m.ID] = m
	}
}

func (sd *SaveData) masteryOf(id string) (lv, xp int) {
	xp = sd.Mastery[id]
	return masteryLevel(xp), xp
}

func (sd *SaveData) masteryOpt(id string, bit int) bool { return sd.MOpts[id]&bit != 0 }

// applyMastery adds an origin's mastery effects; it runs straight after the
// origin's own stats so items still scale on top.
func applyMastery(s *Stats, meta *SaveData, id string) {
	lv, _ := meta.masteryOf(id)
	if lv >= 5 && meta.masteryOpt(id, mOptAspect) {
		s.Aspect = originMastery[id].Aspect
		switch s.Aspect {
		case "Railgun":
			s.LaserInterval = 3.2
		case "Wildfire":
			s.ClassIgnite = 3
		}
	}
	if lv >= 7 {
		s.FreeRerolls++
	}
	if lv >= 9 {
		s.DmgMult *= 1.05
	}
}

// ---------------------------------------------------------------- passive tuning

func (s *Stats) rageMax() int {
	if s.Aspect == "Bloodlust" {
		return 12
	}
	return 20
}
func (s *Stats) ragePer() float64 {
	if s.Aspect == "Bloodlust" {
		return 0.08
	}
	return 0.05
}
func (s *Stats) rageTime() float64 {
	t := 5.0
	if s.Aspect == "Bloodlust" {
		t = 3
	}
	if s.SigBloodPrice {
		t += 3
	}
	return t
}
func (s *Stats) parryWindow() float64 {
	if s.Aspect == "Aegis" {
		return 0.36
	}
	return 0.22
}
func (s *Stats) parryCooldown() float64 {
	if s.Aspect == "Aegis" {
		return 1.4
	}
	return 0.8
}
func (s *Stats) parryHits() int {
	n := 4
	if s.Aspect == "Aegis" {
		n = 3
	}
	if s.SigHalo {
		n += 2
	}
	return n
}
func (s *Stats) attunePeriod() float64 {
	if s.Aspect == "Flux" {
		return 4
	}
	return 8
}
func (s *Stats) attuneChance() float64 {
	c := 0.5
	if s.Aspect == "Flux" {
		c = 0.35
	}
	if s.SigLens {
		c += 0.15
	}
	return c
}
func (s *Stats) overclockTime() float64 {
	if s.SigCoolant {
		return 0.8
	}
	return 1.5
}
func (s *Stats) highRoller(gold int) float64 {
	per, cap := 6.0, 0.6
	if s.Aspect == "Jackpot" {
		per, cap = 4, 0.4
	}
	if s.SigDice {
		cap += 0.3
	}
	return math.Min(cap, float64(gold)/per/100)
}
func (s *Stats) highRollerCap() float64 { return s.highRoller(1 << 20) }
func (s *Stats) stokeMult() float64 {
	if s.Aspect == "Wildfire" {
		return 1.25
	}
	return 1.5
}
func (s *Stats) staticRate() float64 {
	if s.Aspect == "Overcharge" {
		return 0.65
	}
	return 1
}
func (s *Stats) stormTime() float64 {
	t := 3.0
	if s.Aspect == "Overcharge" {
		t = 5
	}
	if s.SigConduit {
		t += 2
	}
	return t
}

// ---------------------------------------------------------------- cosmetics

// paddleLivery returns the paddle colours for the run's origin, if its
// livery is on.
func (r *Run) paddleLivery() (body, cap color.RGBA, ok bool) {
	if r == nil || !r.Livery || r.Arch == nil {
		return
	}
	o, found := originMastery[r.Arch.ID]
	return o.Body, o.Cap, found
}

func (r *Run) orbTrail() (color.RGBA, bool) {
	if r == nil || !r.TrailCol || r.Arch == nil {
		return color.RGBA{}, false
	}
	o, found := originMastery[r.Arch.ID]
	return o.Trail, found
}

// frameCol is the portrait frame mastery earns (silver at 8, gold at 10).
func frameCol(lv int) (color.RGBA, bool) {
	switch {
	case lv >= 10:
		return rgba(255, 210, 100, 255), true
	case lv >= 8:
		return rgba(210, 220, 236, 255), true
	}
	return color.RGBA{}, false
}

// ---------------------------------------------------------------- UI

// drawMasteryPanel shows an origin's level, XP and unlock track, with
// toggles for the options it has earned. It sits under the origin list.
func (g *Game) drawMasteryPanel(s *ebiten.Image, a *Archetype, x, y, w float64) {
	u := g.ui
	S := u.S
	col := classInfo[a.ID].Col
	lv, xp := g.save.masteryOf(a.ID)
	u.frame(s, x, y, w, 196, 0.86)
	cx := x + 24
	title := fmt.Sprintf("MASTERY  %d", lv)
	if lv >= masteryMax {
		title = "MASTER " + strings.ToUpper(a.Name)
	}
	u.text(s, title, fBold, 19, cx, y+14, lighten(col, 0.35), alignLeft)
	frac := 1.0
	prog := "max level"
	if lv < masteryMax {
		lo, hi := masteryXP[lv-1], masteryXP[lv]
		frac = float64(xp-lo) / float64(hi-lo)
		prog = fmt.Sprintf("%d / %d XP", xp-lo, hi-lo)
	}
	u.text(s, prog, fReg, 16, x+w-24, y+16, cDim, alignRight)
	u.meter(s, cx, y+48, w-48, frac, col)
	// the track: one pip per level
	steps := masterySteps(a.ID)
	pw := (w - 48) / 9
	for i, st := range steps {
		px := cx + pw*float64(i) + pw/2
		py := y + 82.0
		got := lv >= st.Lv
		c := ca(cDim, 0.45)
		if got {
			c = col
		}
		fillPath(s, circlePath(u.X(px), u.Y(py), 12*S), ca(darken(c, 0.4), 0.9), ca(darken(c, 0.7), 0.9), grad{}, false)
		strokePath(s, circlePath(u.X(px), u.Y(py), 12*S), 1.5*S, c, false)
		tc := cWhite
		if !got {
			tc = ca(cDim, 0.8)
		}
		u.text(s, itoa(st.Lv), fBold, 14, px, py-9, tc, alignCenter)
		if u.hover(px-pw/2, py-16, pw, 32) {
			state := "Unlocked"
			if !got {
				state = fmt.Sprintf("Reach Mastery %d", st.Lv)
			}
			g.tooltip = fmt.Sprintf("%s — %s (%s)", st.Name, st.Desc, state)
		}
	}
	// toggles for earned options
	type tg struct {
		label string
		bit   int
		lv    int
	}
	tgs := []tg{{"Livery", mOptLivery, 2}, {"Orb Trail", mOptTrail, 4}, {"Aspect", mOptAspect, 5}, {"Gift", mOptGift, 6}}
	bw := (w - 48 - 3*8) / 4
	for i, t := range tgs {
		bx := cx + float64(i)*(bw+8)
		by := y + 110.0
		ok := lv >= t.lv
		on := ok && g.save.masteryOpt(a.ID, t.bit)
		label := t.label
		if !ok {
			label = fmt.Sprintf("Lv %d", t.lv)
		}
		acc := cAcc
		if on {
			acc = col
		}
		if u.button(s, "mopt"+a.ID+itoa(t.bit), label, bx, by, bw, 40, acc, ok) {
			if g.save.MOpts == nil {
				g.save.MOpts = map[string]int{}
			}
			g.save.MOpts[a.ID] ^= t.bit
			writeSave(&g.save)
			g.snd.Play("click")
		}
		if on {
			// a lit pip marks the options that are switched on
			fillPath(s, circlePath(u.X(bx+bw-9), u.Y(by+9), 4*S), lighten(col, 0.5), col, grad{}, false)
			drawSprite(s, sprSoft, u.X(bx+bw-19), u.Y(by-1), 20*S, 20*S, ca(col, 0.7), true)
		} else if ok {
			strokePath(s, circlePath(u.X(bx+bw-9), u.Y(by+9), 4*S), 1.2*S, ca(cDim, 0.7), false)
		}
	}
	// what comes next
	next := ""
	for _, st := range steps {
		if st.Lv > lv {
			next = fmt.Sprintf("Next · Mastery %d: %s", st.Lv, st.Name)
			break
		}
	}
	if next == "" {
		next = "Every mastery reward earned"
	}
	u.text(s, next, fReg, 16, cx, y+162, ca(cDim, 0.9), alignLeft)
}

// masteryAward adds a finished run's XP to its origin and reports the level
// before and after.
func (g *Game) masteryAward(r *Run, xp int) {
	if r == nil || r.Duel || r.Arch == nil || xp <= 0 {
		g.mastBefore, g.mastAfter = 0, 0
		return
	}
	if g.save.Mastery == nil {
		g.save.Mastery = map[string]int{}
	}
	g.mastBefore, _ = g.save.masteryOf(r.Arch.ID)
	g.save.Mastery[r.Arch.ID] += xp
	g.mastAfter, _ = g.save.masteryOf(r.Arch.ID)
}
