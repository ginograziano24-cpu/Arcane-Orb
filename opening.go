package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

//go:embed assets/opening/*
var openingFS embed.FS

const openingDuration = 81.5

type openingLine struct {
	Time, End float64
	Text      string
}
type openingHandAnchor struct{ X, Y float64 }
type openingLoaded struct {
	hands  map[string]openingHandAnchor
	images map[string]image.Image
	stems  [3][]byte
	lines  []openingLine
	err    error
}
type openingFilm struct {
	hands                            map[string]openingHandAnchor
	loading                          chan openingLoaded
	pending                          *openingLoaded
	uploading                        map[string]*ebiten.Image
	images                           map[string]*ebiten.Image
	lines                            []openingLine
	mix                              *openingMixer
	player                           *audio.Player
	t                                float64
	paused, captions, silent, review bool
	err                              string
}

func loadOpening() openingLoaded {
	r := openingLoaded{images: map[string]image.Image{}}
	files, err := openingFS.ReadDir("assets/opening")
	if err != nil {
		r.err = err
		return r
	}
	for _, f := range files {
		name := f.Name()
		if !strings.HasSuffix(name, ".png") && !strings.HasSuffix(name, ".jpg") {
			continue
		}
		b, e := openingFS.ReadFile("assets/opening/" + name)
		if e != nil {
			r.err = e
			return r
		}
		im, _, e := image.Decode(bytes.NewReader(b))
		if e != nil {
			r.err = e
			return r
		}
		r.images[strings.TrimSuffix(strings.TrimSuffix(name, ".png"), ".jpg")] = im
	}
	for i, name := range []string{"music", "effects", "narration"} {
		b, e := openingFS.ReadFile("assets/opening/" + name + ".wav")
		if e != nil {
			r.err = e
			return r
		}
		stream, e := wav.DecodeWithSampleRate(sampleRate, bytes.NewReader(b))
		if e != nil {
			r.err = e
			return r
		}
		r.stems[i], e = io.ReadAll(stream)
		if e != nil {
			r.err = e
			return r
		}
	}
	b, e := openingFS.ReadFile("assets/opening/subtitles.json")
	if e == nil {
		e = json.Unmarshal(b, &r.lines)
	}
	r.err = e
	if e == nil {
		b, e = openingFS.ReadFile("assets/opening/handanchors.json")
		if e == nil {
			e = json.Unmarshal(b, &r.hands)
		}
		r.err = e
	}
	return r
}

func (g *Game) startOpening() {
	if g.opening != nil {
		return
	}
	o := &openingFilm{captions: true, loading: make(chan openingLoaded, 1)}
	g.opening = o
	g.consumeInput()
	for _, p := range []*audio.Player{g.snd.player, g.snd.old} {
		if p != nil {
			p.Pause()
		}
	}
	go func() { o.loading <- loadOpening() }()
}
func (g *Game) closeOpening() {
	if g.opening == nil {
		return
	}
	if g.opening.player != nil {
		_ = g.opening.player.Close()
	}
	for _, im := range g.opening.images {
		im.Deallocate()
	}
	for _, im := range g.opening.uploading {
		im.Deallocate()
	}
	g.opening = nil
	g.consumeInput()
	g.last = nowFn()
	for _, p := range []*audio.Player{g.snd.player, g.snd.old} {
		if p != nil {
			p.Play()
		}
	}
}
func (o *openingFilm) seek(t float64) {
	o.t = math.Max(0, math.Min(openingDuration, t))
	if o.player != nil {
		if e := o.player.SetPosition(time.Duration(o.t * float64(time.Second))); e != nil {
			o.err = e.Error()
		}
	}
}
func (o *openingFilm) pause() {
	o.paused = !o.paused
	if o.player != nil {
		if o.paused {
			o.player.Pause()
		} else {
			o.player.Play()
		}
	}
}
func (g *Game) updateOpening(dt float64) {
	o := g.opening
	if backPressed() {
		g.closeOpening()
		return
	}
	if o.images == nil && o.err == "" {
		if o.pending == nil {
			select {
			case loaded := <-o.loading:
				if loaded.err != nil {
					o.err = loaded.err.Error()
					return
				}
				o.pending = &loaded
				o.uploading = make(map[string]*ebiten.Image, len(loaded.images))
			default:
				return
			}
		}
		// Spread texture creation across frames. Keep the audio clock
		// stopped until every shot is ready for playback and seeking.
		for name, im := range o.pending.images {
			o.uploading[name] = ebiten.NewImageFromImage(im)
			delete(o.pending.images, name)
			if len(o.uploading)%2 == 0 {
				break
			}
		}
		if len(o.pending.images) != 0 {
			return
		}
		loaded := o.pending
		o.images, o.uploading, o.pending = o.uploading, nil, nil
		o.lines, o.hands = loaded.lines, loaded.hands
		o.mix = newOpeningMixer(loaded.stems)
		o.mix.setGains(g.snd.Music, g.snd.SFX, g.snd.SFX)
		if !o.silent {
			p, err := g.snd.ctx.NewPlayer(o.mix)
			if err != nil {
				o.err = err.Error()
				return
			}
			o.player = p
			p.SetBufferSize(80 * time.Millisecond)
			p.Play()
		}
		return
	}
	if o.images == nil || o.err != "" || o.review {
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		o.pause()
		g.consumeInput()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		o.seek(o.t - 5)
		g.consumeInput()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		o.seek(o.t + 5)
		g.consumeInput()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		o.captions = !o.captions
		g.consumeInput()
	}
	if !o.paused {
		if o.player != nil {
			o.t = o.player.Position().Seconds()
		} else {
			o.t += dt
		}
	}
	if o.t >= openingDuration-.02 {
		g.closeOpening()
	}
}

func openingShot(t float64) (string, float64) {
	starts := []float64{0, 2.5, 11, 17.5, 26.5, 33, 38, 43, 47.5, 55.5, 69.5, 76}
	names := []string{"tower", "room", "scroll", "outside", "bricks", "arrival", "conjure", "send", "orb", "hero", "gather", "title"}
	i := 0
	for i+1 < len(starts) && t >= starts[i+1] {
		i++
	}
	return names[i], t - starts[i]
}
func openingEase(t float64) float64 { t = clamp01(t); return t * t * (3 - 2*t) }

func (g *Game) openingSprite(s *ebiten.Image, name string, x, foot, h, angle, a float64) {
	im := g.opening.images[name]
	if im == nil || a <= 0 {
		return
	}
	u := g.ui
	w, ih := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
	sc := h / ih * u.S
	fillPath(s, boxPath(u.X(x-h*.16), u.Y(foot-5), h*.32*u.S, 14*u.S, 7*u.S, false), ca(rgba(12, 6, 24, 255), a*.25), ca(rgba(12, 6, 24, 255), a*.25), grad{}, false)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-w/2, -ih)
	op.GeoM.Scale(sc, sc)
	op.GeoM.Rotate(angle)
	op.GeoM.Translate(u.X(x), u.Y(foot))
	op.ColorScale.ScaleAlpha(float32(clamp01(a)))
	s.DrawImage(im, op)
}
func (g *Game) openingHand(id string, x, foot, h, angle float64) (float64, float64) {
	a := g.opening.hands[id]
	im := g.opening.images[id]
	w := h * float64(im.Bounds().Dx()) / float64(im.Bounds().Dy())
	dx := (a.X - .5) * w
	dy := (a.Y-1)*h - 30
	return x + dx*math.Cos(angle) - dy*math.Sin(angle), foot + dx*math.Sin(angle) + dy*math.Cos(angle)
}
func (g *Game) openingBackground(s *ebiten.Image, name string, t float64) {
	im := g.opening.images[name]
	if im == nil {
		return
	}
	u := g.ui
	w, h := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
	sc := math.Max(1920/w, 1080/h) * (1.025 + .012*math.Sin(t*.3))
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(sc*u.S, sc*u.S)
	op.GeoM.Translate(u.X((1920-w*sc)/2+math.Sin(t*.2)*10), u.Y((1080-h*sc)/2))
	s.DrawImage(im, op)
}
func (g *Game) openingOrb(s *ebiten.Image, x, y, r, a float64, c color.RGBA) {
	u := g.ui
	if r <= 0 || a <= 0 {
		return
	}
	glowPath(s, circlePath(u.X(x), u.Y(y), r*u.S), r*1.6*u.S, ca(c, .42*a))
	drawSprite(s, sprSoft, u.X(x-r*3), u.Y(y-r*3), r*6*u.S, r*6*u.S, ca(c, .42*a), true)
	fillPath(s, circlePath(u.X(x), u.Y(y), r*u.S), ca(c, a), ca(cWhite, a), vgrad(u.Y(y-r), u.Y(y+r)), false)
	strokePath(s, circlePath(u.X(x), u.Y(y), r*1.15*u.S), 1.5*u.S, ca(cWhite, .7*a), true)
	fillPath(s, circlePath(u.X(x-r*.28), u.Y(y-r*.3), r*.25*u.S), ca(cWhite, .8*a), ca(cWhite, .8*a), grad{}, true)
}

func (g *Game) openingTransmutation(s *ebiten.Image, t float64) {
	u := g.ui
	gather := openingEase((t - 1.2) / 2.4)
	stamp := openingEase((t - 4.6) / .6)
	x := 500 + (1200-500)*gather
	y := 810 + (640-810)*gather
	k := 1 - gather*.96
	if gather < .995 {
		fillPath(s, boxPath(u.X(x-10*k), u.Y(y-120*k), 20*k*u.S, 120*k*u.S, 2*u.S, false), rgba(120, 78, 65, 255), rgba(58, 36, 48, 255), vgrad(u.Y(y-120*k), u.Y(y)), false)
		for i := 0; i < 3; i++ {
			p := ngonPath(u.X(x), u.Y(y-(105+float64(i)*32)*k), (72-float64(i)*12)*k*u.S, 3, -math.Pi/2)
			fillPath(s, p, rgba(117, 153, 109, 255), rgba(44, 84, 71, 255), vgrad(u.Y(y-200*k), u.Y(y-50*k)), false)
			strokePath(s, p, 3*u.S, rgba(42, 28, 48, 255), false)
		}
		if t > 1.2 {
			strokePath(s, boxPath(u.X(x-80*k), u.Y(y-220*k), 160*k*u.S, 230*k*u.S, 4*u.S, false), 2*u.S, ca(cAcc, .9), true)
		}
	}
	if gather > .02 && stamp < 1 {
		strokeLine(s, u.X(x), u.Y(y-70*k), u.X(1200), u.Y(640), 2*u.S, ca(cAcc, .65))
	}
	if gather > .55 && t < 5.5 {
		bx := 1200 + 350*stamp
		by := 640 + 165*stamp
		a := 1 - openingEase((t-5.2)/.3)
		fillPath(s, boxPath(u.X(bx-52), u.Y(by-29), 104*u.S, 58*u.S, 5*u.S, false), ca(rgba(217, 176, 161, 255), a), ca(rgba(101, 72, 112, 255), a), vgrad(u.Y(by-29), u.Y(by+29)), false)
		strokePath(s, boxPath(u.X(bx-56), u.Y(by-33), 112*u.S, 66*u.S, 4*u.S, false), 2*u.S, ca(cGold, a), true)
	}
	if t > 5.2 && t < 5.85 {
		phase := (t - 5.2) / .65
		strokePath(s, circlePath(u.X(1550), u.Y(805), (35+phase*150)*u.S), 3*u.S, ca(cAcc, 1-phase), true)
	}
}
func (g *Game) openingBricks(s *ebiten.Image, t float64, rows int) {
	u := g.ui
	shot, local := openingShot(g.opening.t)
	broken := shot == "orb" && math.Mod(local, 2)/2 > .7
	for row := 0; row < rows; row++ {
		for col := 0; col < 19; col++ {
			if broken && col >= 12 && col <= 17 && row >= 3 && row <= 6 {
				continue
			}
			k := openingEase(t*1.3 - float64(row)*.24 - float64(col)*.025)
			if k <= 0 {
				continue
			}
			x := float64(col)*108 + float64(row%2)*54 - 50
			y := 850 - float64(row)*65 - (1-k)*130
			fillPath(s, boxPath(u.X(x), u.Y(y), 103*u.S, 59*u.S, 4*u.S, false), rgba(181, 139, 169, 255), rgba(81, 56, 97, 255), vgrad(u.Y(y), u.Y(y+59)), false)
		}
	}
}
func (g *Game) openingLabel(s *ebiten.Image, title, subtitle string, x, y float64) {
	u := g.ui
	w := math.Max(660, math.Max(u.measure(title, fRune, 38), u.measure(subtitle, fReg, 23))+44)
	fillRect(s, u.X(x-22), u.Y(y-15), w*u.S, 118*u.S, rgba(17, 10, 30, 224))
	u.text(s, title, fRune, 38, x, y, cGold, alignLeft)
	u.text(s, subtitle, fReg, 23, x, y+58, cWhite, alignLeft)
}

func (g *Game) openingPortal(s *ebiten.Image, x, y, t float64, c color.RGBA) {
	if t < -.25 || t > 1.4 {
		return
	}
	u := g.ui
	k := openingEase((t+.25)/.4) * (1 - openingEase((t-.75)/.65))
	for i := 0; i < 24; i++ {
		a := float64(i) * math.Pi / 12
		b := float64(i+1) * math.Pi / 12
		strokeLine(s, u.X(x+math.Cos(a)*100), u.Y(y+math.Sin(a)*22), u.X(x+math.Cos(b)*100), u.Y(y+math.Sin(b)*22), 2*u.S, ca(c, k))
	}
	drawSprite(s, sprSoft, u.X(x-120), u.Y(y-550), 240*u.S, 560*u.S, ca(c, k*.32), true)
}

func (g *Game) drawOpeningScene(s *ebiten.Image) {
	o, u := g.opening, g.ui
	shot, t := openingShot(o.t)
	land := shot != "tower" && shot != "room" && shot != "scroll"
	if land {
		g.openingBackground(s, "land", o.t)
	} else {
		bg := "room"
		if shot == "tower" {
			bg = "tower"
		}
		if shot == "scroll" && t > 3.1 {
			bg = "clean-room"
		}
		g.openingBackground(s, bg, o.t)
	}
	if land {
		fillRect(s, u.X(0), u.Y(0), 1920*u.S, 1080*u.S, rgba(17, 8, 32, 85))
	}
	if shot == "tower" {
		g.openingLabel(s, "A small spell. A very large problem.", "Somewhere, just before date night.", 150, 160)
	}
	if shot == "room" || shot == "scroll" {
		fum := "fumble-0"
		if shot == "scroll" {
			fum = "fumble-1"
		}
		g.openingSprite(s, fum, 610, 885, 650, .018*math.Sin(t*2), 1)
		if shot == "room" {
			g.openingLabel(s, "ARCHMAGE FUMBLETON", "Has a date tonight. Has not tidied.", 960, 190)
			if t > 3 {
				g.openingSprite(s, "mirabel-0", 1430, 870, 660, .02*math.Sin(t), .45*openingEase((t-3)/.7))
			}
		}
		if shot == "scroll" {
			g.openingLabel(s, "INSTANT TIDY", "One scroll. No refunds.", 1100, 180)
			if t < 3 {
				fillRect(s, u.X(840), u.Y(380), 400*u.S, 230*u.S, rgba(237, 210, 163, 255))
				u.text(s, "TIDY!", fRune, 58, 1040, 440, rgba(68, 33, 81, 255), alignCenter)
			} else {
				for i, id := range []string{"boss-prism", "boss-crucible", "boss-warden"} {
					g.openingSprite(s, id, 1030+float64(i)*300, 875, 480, math.Sin(t*4+float64(i))*.025, openingEase((t-3)/.4))
				}
			}
		}
	}
	if shot == "outside" || shot == "bricks" {
		if shot == "outside" {
			for i, id := range []string{"boss-prism", "boss-crucible", "boss-warden"} {
				x := 850 + float64(i)*350
				h := 560 + 80*openingEase((t-6)/.4)
				g.openingSprite(s, id, x, 840+math.Abs(math.Sin(t*2+float64(i)))*8, h, math.Sin(t*2+float64(i))*.018, 1)
			}
			g.openingSprite(s, "fumble-0", 300+math.Sin(t*.8)*80, 865, 370, -.08, 1)
			g.openingLabel(s, "A little too helpful.", "The guardians take tidying very seriously.", 140, 115)
			g.openingBricks(s, t-1, 2)
			g.openingTransmutation(s, t)
		} else {
			g.openingBricks(s, t, 10)
			g.openingLabel(s, "Even the sky?", "That was not in the instructions.", 150, 120)
		}
	}
	if shot == "arrival" || shot == "conjure" || shot == "send" {
		g.openingBricks(s, 20, 3)
		x := 670.0
		if shot == "arrival" {
			x = -200 + 870*openingEase(t/1.5)
		}
		mira := "mirabel-0"
		if shot != "arrival" {
			mira = "mirabel-1"
		}
		g.openingSprite(s, mira, x, 880, 820, 0, 1)
		g.openingSprite(s, "fumble-1", 1390, 900, 440, .025*math.Sin(t*3), 1)
		if shot == "arrival" {
			g.openingLabel(s, "MIRABEL", "One look explains the situation.", 1030, 145)
		}
		if shot == "conjure" || shot == "send" {
			k := openingEase((t - .3) / 1.4)
			split := openingEase((t - 2.65) / 1.05)
			if shot == "send" {
				k = 1
				split = 1
			}
			if split < 1 {
				g.openingOrb(s, 895, 230, 55*k*(1-split), 1, cAcc)
			}
			for i := 0; i < 7; i++ {
				a := float64(i) * math.Pi * 2 / 7
				fly := 0.0
				if shot == "send" {
					fly = openingEase((t - float64(i)*.08) / 1.6)
				}
				r := 150*split + fly*1000
				if split > 0 {
					g.openingOrb(s, 895+math.Cos(a)*r, 230+math.Sin(a)*r, 23, 1-fly*.65, nodeColors[NCombat])
				}
			}
		}
	}
	if shot == "orb" {
		g.openingBricks(s, 20, 8)
		phase := math.Mod(t, 2) / 2
		g.openingOrb(s, 250+phase*1450, 420+math.Sin(phase*math.Pi)*180, 65, 1, cGold)
		if phase > .7 {
			p := (phase - .7) / .3
			for i := 0; i < 14; i++ {
				a := float64(i) * 2.4
				x := 1320 + math.Cos(a)*p*250
				y := 560 + math.Sin(a)*p*180 + p*p*100
				fillRectA(s, u.X(x), u.Y(y), (10+float64(i%3)*5)*u.S, 12*u.S, rgba(194, 151, 180, 255), 1-p)
			}
		}
		g.openingLabel(s, "THE ORB", "Pure magic. Excellent against bricks.", 150, 110)
	}
	if shot == "hero" {
		i := min(6, int(t/2))
		local := t - float64(i)*2
		a := archetypes[i]
		info := classInfo[a.ID]
		k := openingEase((local - []float64{.12, .30, .10, .22, .18, .12, .08}[i]) / []float64{.7, .34, .85, .6, .7, .72, .8}[i])
		rot := math.Sin(k*math.Pi) * []float64{-.015, -.035, .005, .018, -.018, .02, -.025}[i]
		g.openingSprite(s, a.ID, 1280, 900, 790, rot, 1)
		name := []string{"Dame Oriel", "Brakka", "Professor Vell", "Rivet", "Lucky Sol", "Cinder", "Zephyr"}[i]
		if local >= .28 {
			g.openingLabel(s, strings.ToUpper(name), a.Name, 190, 290)
			fillRect(s, u.X(168), u.Y(425), 670*u.S, 190*u.S, rgba(17, 10, 30, 214))
			u.paragraph(s, info.Lore, fReg, 28, 190, 440, 620, cWhite, alignLeft)
		}
		hx, hy := g.openingHand(a.ID, 1280, 900, 790, rot)
		swirl := 0.0
		if i == 2 || i == 4 || i == 6 {
			swirl = 60 * math.Sin(k*math.Pi*4) * (1 - k)
		}
		g.openingOrb(s, 220+(hx-220)*k+swirl, 600+(hy-600)*k-[]float64{110, 65, 175, 130, 160, 150, 195}[i]*math.Sin(k*math.Pi), 34, 1, info.Col)
	}
	if shot == "gather" || shot == "title" {
		tt := t
		if shot == "title" {
			tt = 7
		}
		fillRect(s, u.X(0), u.Y(0), 1920*u.S, 1080*u.S, rgba(17, 8, 32, 110))
		xs := []float64{960, 740, 1180, 500, 1420, 260, 1660}
		for _, i := range []int{5, 6, 3, 4, 1, 2, 0} {
			local := tt - []float64{0, .28, .55, .82, 1.65, 1.10, 1.35}[i]
			k := openingEase(local / .75)
			g.openingPortal(s, xs[i], 875, local, classInfo[archetypes[i].ID].Col)
			if k <= 0 {
				continue
			}
			a := archetypes[i]
			h := 540.0
			if i == 3 {
				h = 470
			}
			if i == 2 || i == 6 {
				h = 620
			}
			rot := math.Sin(k*math.Pi) * (1 - k) * .025
			g.openingSprite(s, a.ID, xs[i], 875, h, rot, k)
			hx, hy := g.openingHand(a.ID, xs[i], 875, h, rot)
			g.openingOrb(s, hx, hy+math.Sin(tt+float64(i))*3, 17, k, classInfo[a.ID].Col)
		}
		if shot == "title" {
			g.arcaneTitle(s, "Arcane Orb", 132, 960, 120)
			u.text(s, "BRING BACK THE SKY", fRune, 32, 960, 290, cGold, alignCenter)
		} else {
			g.openingLabel(s, "SEVEN HEROES. ONE VERY LARGE MESS.", "Most of them read the instructions.", 240, 130)
		}
	}
	// Short dips hide scene swaps without the old 12fps quantization.
	fade := 1 - openingEase(t/.2)
	if fade > 0 {
		fillRectA(s, 0, 0, float64(s.Bounds().Dx()), float64(s.Bounds().Dy()), rgba(9, 5, 18, 255), fade)
	}
}

func (g *Game) drawOpening(s *ebiten.Image) {
	o, u := g.opening, g.ui
	s.Fill(rgba(9, 5, 18, 255))
	if o.images == nil || o.err != "" {
		label := "Preparing the story…"
		if o.err != "" {
			label = "The opening could not load."
		}
		u.text(s, label, fRune, 40, 960, 460, cGold, alignCenter)
		if o.err != "" {
			u.paragraph(s, o.err, fReg, 20, 960, 540, 960, cDim, alignCenter)
		}
		if u.button(s, "filmclose", "Back", 810, 730, 300, 60, cAcc, true) {
			g.closeOpening()
		}
		return
	}
	g.drawOpeningScene(s)
	if o.captions {
		for _, line := range o.lines {
			if o.t >= line.Time && o.t < line.End+.18 {
				fillRect(s, u.X(150), u.Y(884), 1620*u.S, 100*u.S, rgba(12, 7, 25, 238))
				u.paragraph(s, line.Text, fReg, 32, 960, 898, 1540, cWhite, alignCenter)
				break
			}
		}
	}
	fillRect(s, u.X(0), u.Y(985), 1920*u.S, 95*u.S, rgba(10, 6, 21, 245))
	label := "Pause"
	if o.paused {
		label = "Play"
	}
	if u.button(s, "filmpause", label, 40, 1005, 170, 52, cAcc, true) {
		o.pause()
	}
	if u.button(s, "filmback", "−5s", 225, 1005, 110, 52, cAcc, true) {
		o.seek(o.t - 5)
	}
	if u.button(s, "filmnext", "+5s", 350, 1005, 110, 52, cAcc, true) {
		o.seek(o.t + 5)
	}
	cc := "CC on"
	if !o.captions {
		cc = "CC off"
	}
	if u.button(s, "filmcc", cc, 475, 1005, 150, 52, cAcc, true) {
		o.captions = !o.captions
	}
	gains := o.mix.gains()
	for i, name := range []string{"Music", "Effects", "Voice"} {
		x := 660 + float64(i)*300
		u.text(s, fmt.Sprintf("%s %d%%", name, int(gains[i]*100+.5)), fReg, 18, x, 989, cDim, alignLeft)
		if u.button(s, "filmlower"+itoa(i), "−", x, 1020, 60, 38, cAcc, true) {
			gains[i] = math.Max(0, gains[i]-.1)
			o.mix.setGains(gains[0], gains[1], gains[2])
		}
		if u.button(s, "filmraise"+itoa(i), "+", x+70, 1020, 60, 38, cAcc, true) {
			gains[i] = math.Min(1, gains[i]+.1)
			o.mix.setGains(gains[0], gains[1], gains[2])
		}
	}
	u.text(s, fmt.Sprintf("%02d:%02d / 01:22", int(o.t)/60, int(o.t)%60), fReg, 18, 1590, 1020, cDim, alignLeft)
	if u.button(s, "filmskip", "Skip", 1770, 1005, 120, 52, cGold, true) {
		g.closeOpening()
	}
}
