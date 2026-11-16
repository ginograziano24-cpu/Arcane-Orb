package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"math"
	"math/rand"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"orb/levelgen"
)

// Letters between acts (ROADMAP-50 #65). After each Guardian falls, and at
// each endless loop, a letter arrives on the map: from Archmage Fumbleton
// (rambling, apologetic, very proud of you) or from Mirabel (warm, teasing
// him, signs with a lipstick kiss on the paper). Either one comes with a
// picture of the writer: a taped-on photo, or the writer drawn faintly
// behind the writing. The P.S. picks up something you did this run.
//
// An envelope slides up, its flap opens, the letter rises out and unfolds,
// and the lines appear one by one; Mirabel's kiss is pressed on last. A
// click shows the whole letter at once, the next one puts it away.

type letterDef struct {
	ID    string
	From  string   // "fumbleton" or "mirabel"
	Tier  int      // 1 after the first Guardian, 2 after the second, 3 endless, 0 any time
	Hello string   // greeting, %s = the hero's name
	Body  []string // the lines
	Sign  string
}

var letterDefs = []letterDef{
	// ---- Archmage Fumbleton
	{"f_glitter", "fumbleton", 1, "Dear %s,", []string{
		"Word reached my tower that the Prism Colossus is now a pile of very expensive glitter. Splendid work!",
		"I swept some of it up. It is in a jar labelled DO NOT SNEEZE.",
	}, "Onward and upward, Archmage Fumbleton"},
	{"f_sorry", "fumbleton", 0, "Dear %s,", []string{
		"A quick note to say the bricks are NOT my fault. Mostly not. A tiny bit.",
		"Mirabel says I should apologise properly, so: sorry. Very sorry. I have enclosed a drawing of me being sorry.",
	}, "Your humble wizard, Fumbleton"},
	{"f_tea", "fumbleton", 2, "Dear %s,", []string{
		"The Crucible Golem is down! The whole tower shook and I spilled tea on three spellbooks.",
		"Two of them are better now.",
	}, "Proudly, Fumbleton"},
	{"f_scrapbook", "fumbleton", 3, "Dear %s,", []string{
		"You beat all three Guardians and KEPT GOING? I have started a scrapbook about you.",
		"It is mostly doodles of you hitting things. Very accurate doodles.",
	}, "Your biggest fan (don't tell Mirabel), Fumbleton"},
	{"f_tip", "fumbleton", 0, "Dear %s,", []string{
		"A tip from an expert: if a brick won't break, hit it harder.",
		"That is the whole tip. I normally charge three gold for it.",
	}, "Wisely, Archmage Fumbleton"},
	// ---- Mirabel
	{"m_hello", "mirabel", 1, "Hello, %s!", []string{
		"Mirabel here. Fumbleton told me everything, then told me again with sound effects.",
		"Thank you for cleaning up his mess. I'd hug you, but you're covered in glitter.",
	}, "Kisses, Mirabel"},
	{"m_scarf", "mirabel", 0, "Dear %s,", []string{
		"Fumbleton is knitting you a scarf. It is eleven metres long and still growing.",
		"I've told him it's \"a lovely length\". Please wear it at least once.",
	}, "Love, Mirabel"},
	{"m_party", "mirabel", 2, "Darling %s,", []string{
		"One more Guardian to go! I've already picked my outfit for the victory party.",
		"It's purple. Everything I own is purple. Don't ask.",
	}, "Mwah, Mirabel"},
	{"m_surprise", "mirabel", 3, "Sweet %s,", []string{
		"Still going? You're amazing. Fumbleton made you a scrapbook. Please act surprised.",
		"The doodles are adorable. He drew your orb with little eyes.",
	}, "Love and lipstick, Mirabel"},
	{"m_frog", "mirabel", 0, "Hi %s!", []string{
		"Fumbleton tried to send you a lucky spell. It turned my hat into a frog for an hour.",
		"The frog sends its regards.",
	}, "Kisses, Mirabel"},
}

// letterPS is the P.S.: something the player did this run.
func letterPS(r *Run) string {
	switch top, share := topSource(r); {
	case r.MaxCombo >= 30:
		return fmt.Sprintf("P.S. A %d-hit combo! I had to sit down.", r.MaxCombo)
	case top != "" && top != "Orb" && share > 0.25:
		return fmt.Sprintf("P.S. I hear %s did most of the damage. Give it a pat from me.", top)
	case r.Gold >= 150:
		return fmt.Sprintf("P.S. %d gold?! Treat yourself to something shiny.", r.Gold)
	}
	return "P.S. Remember to eat something. Smashing walls is hungry work."
}

// pickLetter chooses this act's letter: either writer, about half the
// time each, preferring letters the player hasn't read yet.
func pickLetter(r *Run, seen map[string]bool) *letterDef {
	tier := min(r.Act-1, 3)
	rng := rand.New(rand.NewSource(levelgen.Mix(r.SeedN, int64(r.Act), 515)))
	from := "fumbleton"
	if rng.Intn(2) == 1 {
		from = "mirabel"
	}
	var fresh, any []*letterDef
	for i := range letterDefs {
		l := &letterDefs[i]
		if l.From != from || (l.Tier != 0 && l.Tier != tier) {
			continue
		}
		any = append(any, l)
		if !seen[l.ID] {
			fresh = append(fresh, l)
		}
	}
	pool := fresh
	if len(pool) == 0 {
		pool = any
	}
	if len(pool) == 0 {
		return nil
	}
	// a letter written for this moment beats a general one
	for _, l := range pool {
		if l.Tier == tier {
			return l
		}
	}
	return pool[rng.Intn(len(pool))]
}

type letterAnim struct {
	def   *letterDef
	hero  string
	ps    string
	t     float64
	out   float64
	full  bool // every line shown (after a skip)
	photo bool // a taped photo (else the writer drawn faintly behind)
	art   string
	kissd bool
	lines int // lines whose sound has played
}

const (
	letterOpen = 1.15 // the letter is out and open
	letterLine = 0.32 // between lines
	letterFade = 0.3
)

// queueLetter is called as a new act begins (beginWorld); the letter
// arrives on the map.
func (g *Game) queueLetter() {
	r := g.run
	if r == nil || r.Act < 2 || !g.cinematic() {
		return
	}
	d := pickLetter(r, g.save.Letters)
	if d == nil {
		return
	}
	hero := r.Arch.Name
	if h, ok := heroLore[r.Arch.ID]; ok {
		hero = h.Name
	}
	rng := rand.New(rand.NewSource(levelgen.Mix(r.SeedN, int64(r.Act), 516)))
	art := []string{"fumble-2", "fumble-0", "fumble-1"}[rng.Intn(3)]
	if d.From == "mirabel" {
		art = []string{"mirabel-0", "mirabel-1"}[rng.Intn(2)]
	}
	g.pendingLetter = &letterAnim{def: d, hero: hero, ps: letterPS(r), photo: rng.Intn(2) == 0, art: art}
}

// lineCount is how many lines the letter writes out (greeting, body, P.S.,
// signature).
func (la *letterAnim) lineCount() int { return 1 + len(la.def.Body) + 2 }

func (la *letterAnim) doneAt() float64 {
	return letterOpen + float64(la.lineCount())*letterLine + 0.35
}

// updateLetter holds the game while a letter is open. A pending letter is
// opened once the map is on screen.
func (g *Game) updateLetter(dt float64) bool {
	if g.letter == nil {
		if g.pendingLetter != nil && g.state == StMap && len(g.forges) == 0 && g.stateT > 0.35 {
			g.letter, g.pendingLetter = g.pendingLetter, nil
			if g.save.Letters == nil {
				g.save.Letters = map[string]bool{}
			}
			g.save.Letters[g.letter.def.ID] = true
			g.snd.Play("letter")
		}
		if g.letter == nil {
			return false
		}
	}
	la := g.letter
	speed := 1.0
	if g.settings.FastAnim {
		speed = 1.7
	}
	press := g.ui.Click || g.keyHit(ebiten.KeySpace) || g.keyHit(ebiten.KeyEnter) || g.keyHit(ebiten.KeyEscape) || g.ui.nav.confirm
	switch {
	case la.out > 0:
		la.out += dt * speed
		if la.out >= letterFade {
			g.letter = nil
		}
	case press && la.t > 0.4 && la.t < la.doneAt():
		la.t = la.doneAt()
		la.full = true
	case press && la.t > 0.4:
		la.out = 1e-6
		g.snd.PlayVol("letter", 0.5)
	default:
		la.t += dt * speed
	}
	if !la.full {
		for la.lines < la.lineCount() && la.t >= letterOpen+float64(la.lines)*letterLine {
			g.snd.PlayVol("scribble", 0.55)
			la.lines++
		}
	}
	if la.def.From == "mirabel" && !la.kissd && la.t >= la.doneAt()-0.2 {
		la.kissd = true
		g.snd.Play("kiss")
	}
	g.consumeInput()
	return true
}

// ---------------------------------------------------------------- drawing

var letterArtCache = map[string]*ebiten.Image{}

// letterArt is a character picture from the opening's art.
func letterArt(name string) *ebiten.Image {
	if im, ok := letterArtCache[name]; ok {
		return im
	}
	var img *ebiten.Image
	if b, err := openingFS.ReadFile("assets/opening/" + name + ".png"); err == nil {
		if im, _, err := image.Decode(bytes.NewReader(b)); err == nil {
			img = ebiten.NewImageFromImage(im)
		}
	}
	letterArtCache[name] = img
	return img
}

// polaroid builds (once) a white-framed photo of the top of a picture.
func polaroid(name string) *ebiten.Image {
	key := "polaroid:" + name
	if im, ok := letterArtCache[key]; ok {
		return im
	}
	src := letterArt(name)
	if src == nil {
		letterArtCache[key] = nil
		return nil
	}
	const fw, fh, pad, bottom = 300, 352, 18, 64
	img := ebiten.NewImage(fw, fh)
	img.Fill(color.RGBA{250, 246, 236, 255})
	pw, ph := fw-2*pad, fh-pad-bottom
	inner := ebiten.NewImage(pw, ph)
	inner.Fill(color.RGBA{214, 200, 228, 255}) // a soft lilac backdrop
	sb := src.Bounds()
	// head and shoulders: centre the face, show about two thirds of the width
	fx, fy, span := 0.48, 0.25, 0.75 // Fumbleton
	if strings.HasPrefix(name, "mirabel") {
		fx, fy, span = 0.58, 0.17, 0.58
	}
	sc := float64(pw) / (float64(sb.Dx()) * span)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(sc, sc)
	op.GeoM.Translate(float64(pw)/2-fx*float64(sb.Dx())*sc, float64(ph)/2-fy*float64(sb.Dy())*sc)
	inner.DrawImage(src, op)
	op2 := &ebiten.DrawImageOptions{}
	op2.GeoM.Translate(pad, pad)
	img.DrawImage(inner, op2)
	inner.Deallocate()
	letterArtCache[key] = img
	return img
}

// drawImageRot draws img centred at (cx, cy) (virtual coordinates),
// w wide, turned by rot radians, with alpha a.
func (g *Game) drawImageRot(s, img *ebiten.Image, cx, cy, w, rot, a float64) {
	if img == nil || a <= 0 {
		return
	}
	u := g.ui
	b := img.Bounds()
	sc := w * u.S / float64(b.Dx())
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-float64(b.Dx())/2, -float64(b.Dy())/2)
	op.GeoM.Scale(sc, sc)
	op.GeoM.Rotate(rot)
	op.GeoM.Translate(u.X(cx), u.Y(cy))
	op.ColorScale.ScaleAlpha(float32(a))
	s.DrawImage(img, op)
}

// lipsPath is a lipstick print: an upper lip with a Cupid's bow and a fuller
// lower lip, w wide, centred at (cx, cy) in screen pixels, turned by rot.
func lipsPath(cx, cy, w, rot float64, lower bool) *vector.Path {
	h := w * 0.6
	pt := func(x, y float64) (float32, float32) {
		x, y = x*w/2, y*h/2
		c, s := math.Cos(rot), math.Sin(rot)
		return float32(cx + x*c - y*s), float32(cy + x*s + y*c)
	}
	p := &vector.Path{}
	m := func(x, y float64) { a, b := pt(x, y); p.MoveTo(a, b) }
	cu := func(x1, y1, x2, y2, x3, y3 float64) {
		a, b := pt(x1, y1)
		c, d := pt(x2, y2)
		e, f := pt(x3, y3)
		p.CubicTo(a, b, c, d, e, f)
	}
	if !lower {
		m(-1, 0.08)
		cu(-0.75, -0.55, -0.38, -1.0, -0.1, -0.7) // left peak
		cu(-0.04, -0.58, 0.04, -0.58, 0.1, -0.7)  // the bow's dip
		cu(0.38, -1.0, 0.75, -0.55, 1, 0.08)      // right peak
		cu(0.55, 0.0, 0.2, 0.12, 0, 0.06)         // the parting line
		cu(-0.2, 0.12, -0.55, 0.0, -1, 0.08)
	} else {
		m(-0.96, 0.2)
		cu(-0.55, 0.16, -0.2, 0.26, 0, 0.22)
		cu(0.2, 0.26, 0.55, 0.16, 0.96, 0.2)
		cu(0.72, 0.85, 0.3, 1.05, 0, 1.05)
		cu(-0.3, 1.05, -0.72, 0.85, -0.96, 0.2)
	}
	p.Close()
	return p
}

// drawKiss presses Mirabel's lipstick onto the paper.
func (g *Game) drawKiss(s *ebiten.Image, x, y, w, rot, k float64) {
	if k <= 0 {
		return
	}
	u := g.ui
	cx, cy, sw := u.X(x), u.Y(y), w*u.S
	// pressed on: it lands a little big and settles
	pop := 1 + 0.35*(1-ease(k))
	sw *= pop
	a := math.Min(1, k*1.6)
	ink := rgba(196, 22, 74, 255)
	for _, lower := range []bool{false, true} {
		vFill(s, lipsPath(cx, cy, sw, rot, lower), ca(ink, 0.82*a), ca(darken(ink, 0.25), 0.82*a), vgrad(cy-sw*0.25, cy+sw*0.25))
	}
	// the little creases a real print leaves
	c, sn := math.Cos(rot), math.Sin(rot)
	for i := -4; i <= 4; i++ {
		fx := float64(i) / 5
		for _, seg := range [][2]float64{{-0.42, -0.12}, {0.22, 0.62}} {
			x0, y0 := fx*sw/2, seg[0]*sw/4
			x1, y1 := fx*sw/2*1.04, seg[1]*sw/4
			strokeLine(s, cx+x0*c-y0*sn, cy+x0*sn+y0*c, cx+x1*c-y1*sn, cy+x1*sn+y1*c, math.Max(1, sw*0.012), prem(rgba(255, 190, 205, 255), 0.35*a))
		}
	}
}

// drawLetter draws the letter over everything.
func (g *Game) drawLetter(s *ebiten.Image) {
	la := g.letter
	if la == nil {
		return
	}
	u := g.ui
	S := u.S
	t := la.t
	fade := 1.0
	if la.out > 0 {
		fade = 1 - smootherstep(la.out/letterFade)
	}
	mir := la.def.From == "mirabel"
	veil := smootherstep(t/0.3) * fade
	fillRectA(s, 0, 0, float64(s.Bounds().Dx()), float64(s.Bounds().Dy()), rgba(6, 4, 12, 255), 0.62*veil)

	// the envelope: slides up, opens, and drops away as the letter rises
	if t < letterOpen+0.25 {
		e := smootherstep(t / 0.45)
		drop := smootherstep((t - 0.85) / 0.4)
		ex, ey := 960.0, 1180-640*e+520*drop
		ew, eh := 560.0, 340.0
		ec := rgba(232, 214, 184, 255)
		if mir {
			ec = rgba(244, 214, 226, 255) // pink stationery
		}
		a := fade * (1 - drop)
		fillPath(s, boxPath(u.X(ex-ew/2+8), u.Y(ey-eh/2+12), ew*S, eh*S, 10*S, false), ca(cBlack, 0.35*a), ca(cBlack, 0.35*a), grad{}, false)
		fillPath(s, boxPath(u.X(ex-ew/2), u.Y(ey-eh/2), ew*S, eh*S, 10*S, false), ca(ec, a), ca(darken(ec, 0.12), a), vgrad(u.Y(ey-eh/2), u.Y(ey+eh/2)), false)
		// the flap: a triangle that folds open
		open := smootherstep((t - 0.45) / 0.25)
		tip := ey - eh/2 + eh*0.55*(1-2*open)
		fc := darken(ec, 0.08)
		vFill(s, vPoly([][2]float64{{u.X(ex - ew/2), u.Y(ey - eh/2)}, {u.X(ex + ew/2), u.Y(ey - eh/2)}, {u.X(ex), u.Y(tip)}}), ca(fc, a), ca(darken(fc, 0.1), a), grad{})
		// the seal: a heart for Mirabel, a star for Fumbleton
		if open < 0.5 {
			sc := rgba(150, 60, 190, 255)
			gl := "star"
			if mir {
				sc, gl = rgba(206, 40, 90, 255), "heart"
			}
			fillPath(s, circlePath(u.X(ex), u.Y(tip-10), 30*S), ca(sc, a), ca(darken(sc, 0.3), a), vgrad(u.Y(tip-40), u.Y(tip+20)), false)
			u.drawEmblem(s, glyphImage(gl, lighten(sc, 0.5)), ex, tip-10, 34, a*(1-2*open))
		}
	}

	// the letter: rises out of the envelope and unfolds
	if t < 0.75 {
		return
	}
	k := smootherstep((t - 0.75) / 0.4)
	pw, ph := 820.0, 640.0
	px, py := 960-pw/2, 190+40*(1-k)
	unfold := 0.25 + 0.75*k
	sy := py + ph*(1-unfold)/2
	sh := ph * unfold
	paper := rgba(248, 240, 222, 255)
	if mir {
		paper = rgba(252, 240, 244, 255)
	}
	a := fade * math.Min(1, (t-0.75)/0.15)
	fillPath(s, boxPath(u.X(px+10), u.Y(sy+14), pw*S, sh*S, 6*S, false), ca(cBlack, 0.4*a), ca(cBlack, 0.4*a), grad{}, false)
	fillPath(s, boxPath(u.X(px), u.Y(sy), pw*S, sh*S, 6*S, false), ca(paper, a), ca(darken(paper, 0.08), a), vgrad(u.Y(sy), u.Y(sy+sh)), false)
	strokePath(s, boxPath(u.X(px+14), u.Y(sy+14), (pw-28)*S, (sh-28)*S, 4*S, false), 1.2*S, ca(darken(paper, 0.22), 0.6*a), false)
	// the fold, still pressed into the paper
	fillRectA(s, u.X(px+10), u.Y(sy+sh/2), (pw-20)*S, 2*S, darken(paper, 0.18), 0.5*a)
	if k < 1 {
		return
	}

	// the writer, drawn faintly behind the writing
	if !la.photo {
		if img := letterArt(la.art); img != nil {
			b := img.Bounds()
			w := 420.0 * float64(b.Dx()) / float64(b.Dy()) * 1.25
			g.drawImageRot(s, img, px+pw-w/2-20, py+ph/2+40, w, 0, 0.16*a)
		}
	}

	ink := rgba(58, 36, 30, 255)
	if mir {
		ink = rgba(92, 24, 70, 255) // purple ink, of course
	}
	lx, tw := px+64, pw-128
	if la.photo {
		tw = pw - 128 - 150 // leave room for the photo
	}
	ly := py + 70
	line := 0
	vis := func() float64 {
		if la.full {
			return a
		}
		return a * clamp01((t-letterOpen-float64(line)*letterLine)/0.25)
	}
	write := func(txt string, f fontID, size float64, width float64) {
		al := vis()
		if al > 0 {
			raw := u.rawText
			lines := u.wrap(txt, f, size, width) // already translated
			u.rawText = true
			for _, l := range lines {
				u.textAt(s, l, f, size, u.X(lx), u.Y(ly), ca(ink, al), alignLeft)
				ly += size * 1.42
			}
			u.rawText = raw
		} else {
			ly += size * 1.42 * float64(len(u.wrap(txt, f, size, width)))
		}
		line++
	}
	write(fmt.Sprintf(tr(la.def.Hello), la.hero), fSerif, 30, tw)
	ly += 14
	for _, b := range la.def.Body {
		write(b, fReg, 24, tw)
		ly += 10
	}
	ly += 8
	write(la.ps, fReg, 21, pw-128)
	ly += 18
	sign := la.def.Sign
	signY := ly
	write(sign, fSerif, 26, pw-128)

	// the photo, taped to the top corner
	if la.photo {
		ph := polaroid(la.art)
		rot := -0.09
		if mir {
			rot = 0.08
		}
		g.drawImageRot(s, ph, px+pw-120, py+150, 210, rot, a)
		// a strip of tape
		fillPath(s, boxPath(u.X(px+pw-160), u.Y(py+28), 80*S, 26*S, 3*S, false), ca(rgba(250, 245, 220, 255), 0.55*a), ca(rgba(230, 225, 200, 255), 0.55*a), grad{}, false)
	}

	// the kiss for Mirabel, a little star doodle for Fumbleton
	end := la.doneAt() - 0.2
	if la.full {
		end = 0
	}
	if mir {
		g.drawKiss(s, px+pw-170, math.Min(py+ph-80, signY+20), 130, -0.21, clamp01((t-end)/0.3))
	} else if t >= end {
		k := ease(clamp01((t - end) / 0.3))
		u.drawEmblem(s, glyphImage("star", rgba(110, 70, 170, 255)), px+pw-110, math.Min(py+ph-80, signY+16), 46*k, a)
	}
	if t > letterOpen+0.5 && la.out == 0 {
		hint := "Click to skip"
		if t >= la.doneAt() {
			hint = "Click to continue"
		}
		u.text(s, hint, fReg, 17, 960, 1010, ca(cDim, (0.45+0.2*math.Sin(t*3))*fade), alignCenter)
	}
}
