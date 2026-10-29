package main

import (
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// UI maps a virtual 1920×1080 canvas onto the real screen at native
// resolution, so text and panels stay razor-sharp at 1440p and 4K.
type UI struct {
	S, OX, OY   float64
	SW, SH      float64
	MX, MY      float64 // mouse in virtual coords
	Click       bool
	faces       map[faceKey]text.Face
	fonts       [5]*opentype.Font
	hoverID     string
	lastHover   string
	snd         *Sounds
	Th          *Theme             // framing for the current screen
	DT          float64            // frame delta for widget animation
	Held        bool               // left mouse button down
	anims       map[string]float64 // per-widget eased hover/press values
	nav         navState           // keyboard / gamepad focus
	time        float64            // seconds since start, for widget pulses
	rawText     bool               // text is already translated (wrapped lines)
	cyrFallback bool
	cjkFonts    [2]*opentype.Font // bold, regular
	nameFont    *opentype.Font    // for language names in their own scripts
	wrapCache   map[wrapKey][]string
}

type fontID int

const (
	fReg fontID = iota
	fBold
	fTitle
	fRune  // Cinzel Decorative: arcane titles
	fSerif // Cinzel: menu items
)

type faceKey struct {
	f  fontID
	px int
}

type align int

const (
	alignLeft align = iota
	alignCenter
	alignRight
)

func newUI(snd *Sounds) *UI {
	u := &UI{faces: map[faceKey]text.Face{}, snd: snd, anims: map[string]float64{}, wrapCache: map[wrapKey][]string{}}
	for i, data := range [][]byte{fontRegular, fontBold, fontDisplay, fontRune, fontSerif} {
		f, err := opentype.Parse(data)
		if err != nil {
			panic(err)
		}
		u.fonts[i] = f
	}
	return u
}

func (u *UI) Setup(sw, sh float64) {
	u.SW, u.SH = sw, sh
	u.S = math.Min(sw/1920, sh/1080)
	u.OX = (sw - 1920*u.S) / 2
	u.OY = (sh - 1080*u.S) / 2
	mx, my := ebiten.CursorPosition()
	u.MX = (float64(mx) - u.OX) / u.S
	u.MY = (float64(my) - u.OY) / u.S
	u.lastHover = u.hoverID
	u.hoverID = ""
}

func (u *UI) X(v float64) float64 { return u.OX + v*u.S }
func (u *UI) Y(v float64) float64 { return u.OY + v*u.S }

func (u *UI) face(f fontID, size float64) text.Face {
	px := int(math.Round(size * u.S))
	if px < 6 {
		px = 6
	}
	k := faceKey{f, px}
	if fc, ok := u.faces[k]; ok {
		return fc
	}
	mk := func(of *opentype.Font) text.Face {
		fc, err := opentype.NewFace(of, &opentype.FaceOptions{Size: float64(px), DPI: 72, Hinting: font.HintingNone})
		if err != nil {
			panic(err)
		}
		return text.NewGoXFace(fc)
	}
	var tf text.Face = mk(u.fonts[f])
	// other scripts fall back glyph by glyph: Fira for Cyrillic in the
	// display fonts, then the CJK face
	var chain []text.Face
	if u.cyrFallback && f != fReg && f != fBold {
		chain = append(chain, mk(u.fonts[fBold]))
	}
	if fb := u.cjkFonts[0]; fb != nil {
		if (f == fReg) && u.cjkFonts[1] != nil {
			fb = u.cjkFonts[1]
		}
		chain = append(chain, mk(fb))
	}
	if len(chain) > 0 {
		if mf, err := text.NewMultiFace(append([]text.Face{tf}, chain...)...); err == nil {
			tf = mf
		}
	}
	u.faces[k] = tf
	return tf
}

// langName draws a language's own name; it always has the glyphs it needs.
func (u *UI) langName(dst *ebiten.Image, s string, x, y float64, c color.RGBA) {
	raw := u.rawText
	u.rawText = true
	defer func() { u.rawText = raw }()
	if u.nameFont == nil {
		u.text(dst, s, fBold, 22, x, y, c, alignCenter)
		return
	}
	px := math.Round(22 * u.S)
	k := faceKey{fontID(100), int(px)}
	fc, ok := u.faces[k]
	if !ok {
		f1, _ := opentype.NewFace(u.fonts[fBold], &opentype.FaceOptions{Size: px, DPI: 72, Hinting: font.HintingNone})
		f2, _ := opentype.NewFace(u.nameFont, &opentype.FaceOptions{Size: px, DPI: 72, Hinting: font.HintingNone})
		mf, err := text.NewMultiFace(text.NewGoXFace(f1), text.NewGoXFace(f2))
		if err != nil {
			u.text(dst, s, fBold, 22, x, y, c, alignCenter)
			return
		}
		fc = mf
		u.faces[k] = fc
	}
	op := &text.DrawOptions{}
	op.PrimaryAlign = text.AlignCenter
	op.GeoM.Translate(u.X(x), u.Y(y)-2*u.S)
	op.ColorScale.ScaleWithColor(prem(c, 1))
	text.Draw(dst, s, fc, op)
}

// setScripts picks the fallback fonts a language needs and drops the
// cached faces so text is rebuilt with them.
func (u *UI) setScripts(code string) {
	u.cyrFallback = code == "ru"
	u.cjkFonts = [2]*opentype.Font{}
	if code == "zh" || code == "ja" {
		for i, w := range []string{"bold", "regular"} {
			if data, err := cjkFS.ReadFile("assets/fonts/cjk/" + code + "-" + w + ".ttf"); err == nil {
				if f, err := opentype.Parse(data); err == nil {
					u.cjkFonts[i] = f
				}
			}
		}
	}
	if u.nameFont == nil {
		if data, err := cjkFS.ReadFile("assets/fonts/cjk/zh-bold.ttf"); err == nil {
			u.nameFont, _ = opentype.Parse(data)
		}
	}
	u.faces = map[faceKey]text.Face{}
	u.wrapCache = map[wrapKey][]string{}
}

func (u *UI) drawText(dst *ebiten.Image, s string, f fontID, size, sx, sy float64, c color.RGBA, al align) {
	if !u.rawText {
		s = tr(s)
	}
	op := &text.DrawOptions{}
	switch al {
	case alignCenter:
		op.PrimaryAlign = text.AlignCenter
	case alignRight:
		op.PrimaryAlign = text.AlignEnd
	}
	op.GeoM.Translate(sx, sy)
	op.ColorScale.ScaleWithColor(prem(c, 1))
	text.Draw(dst, s, u.face(f, size), op)
}

// text draws at virtual coordinates (y = top of the line) with a soft shadow.
func (u *UI) text(dst *ebiten.Image, s string, f fontID, size, x, y float64, c color.RGBA, al align) {
	sh := math.Max(1, 2*u.S)
	u.drawText(dst, s, f, size, u.X(x)+sh, u.Y(y)+sh, color.RGBA{0, 0, 0, uint8(float64(c.A) * 0.6)}, al)
	u.drawText(dst, s, f, size, u.X(x), u.Y(y), c, al)
}

// textAt draws at screen coordinates.
func (u *UI) textAt(dst *ebiten.Image, s string, f fontID, size, sx, sy float64, c color.RGBA, al align) {
	u.drawText(dst, s, f, size, sx, sy, c, al)
}

func (u *UI) outline(dst *ebiten.Image, s string, f fontID, size, sx, sy float64, c color.RGBA, al align) {
	o := math.Max(1, 2.2*u.S)
	oc := color.RGBA{10, 6, 18, uint8(float64(c.A) * 0.85)}
	for _, d := range [][2]float64{{-o, 0}, {o, 0}, {0, -o}, {0, o}, {-o, -o}, {o, o}, {-o, o}, {o, -o}} {
		u.drawText(dst, s, f, size, sx+d[0], sy+d[1], oc, al)
	}
	u.drawText(dst, s, f, size, sx, sy, c, al)
}

func (u *UI) textOutlined(dst *ebiten.Image, s string, f fontID, size, x, y float64, c color.RGBA, al align) {
	u.outline(dst, s, f, size, u.X(x), u.Y(y), c, al)
}

func (u *UI) textOutlinedAt(dst *ebiten.Image, s string, f fontID, size, sx, sy float64, c color.RGBA, al align) {
	u.outline(dst, s, f, size, sx, sy, c, al)
}

func (u *UI) measure(s string, f fontID, size float64) float64 {
	if !u.rawText {
		s = tr(s)
	}
	w, _ := text.Measure(s, u.face(f, size), 0)
	return w / u.S
}

// wrap breaks text (translated first) into lines no wider than width.
// The lines it returns are already translated.
func (u *UI) wrap(s string, f fontID, size, width float64) []string {
	if !u.rawText {
		s = tr(s)
	}
	raw := u.rawText
	u.rawText = true
	defer func() { u.rawText = raw }()
	key := wrapKey{s, f, size, width}
	if l, ok := u.wrapCache[key]; ok {
		return l
	}
	var lines []string
	for _, para := range strings.Split(s, "\n") {
		toks := breakTokens(para)
		cur := ""
		for _, w := range toks {
			if w == " " && cur == "" {
				continue
			}
			try := cur + w
			if u.measure(strings.TrimRight(try, " "), f, size) > width && strings.TrimSpace(cur) != "" {
				lines = append(lines, strings.TrimRight(cur, " "))
				cur = strings.TrimLeft(w, " ")
			} else {
				cur = try
			}
		}
		lines = append(lines, strings.TrimRight(cur, " "))
	}
	if len(u.wrapCache) > 2000 {
		u.wrapCache = map[wrapKey][]string{}
	}
	u.wrapCache[key] = lines
	return lines
}

type wrapKey struct {
	s           string
	f           fontID
	size, width float64
}

func (u *UI) paragraph(dst *ebiten.Image, s string, f fontID, size, x, y, width float64, c color.RGBA, al align) float64 {
	lh := size * 1.35
	lines := u.wrap(s, f, size, width)
	raw := u.rawText
	u.rawText = true
	for _, l := range lines {
		u.text(dst, l, f, size, x, y, c, al)
		y += lh
	}
	u.rawText = raw
	return y
}

func (u *UI) rrect(dst *ebiten.Image, x, y, w, h, r float64, c color.RGBA) {
	roundRect(dst, u.X(x), u.Y(y), w*u.S, h*u.S, r*u.S, prem(c, 1))
}

func (u *UI) rstroke(dst *ebiten.Image, x, y, w, h, r, sw float64, c color.RGBA) {
	strokeRoundRect(dst, u.X(x), u.Y(y), w*u.S, h*u.S, r*u.S, sw*u.S, prem(c, 1))
}

// panel is the standard themed frame (see gui.go).
func (u *UI) panel(dst *ebiten.Image, x, y, w, h, a float64) {
	u.frame(dst, x, y, w, h, a)
}

func (u *UI) hover(x, y, w, h float64) bool {
	return u.MX >= x && u.MX <= x+w && u.MY >= y && u.MY <= y+h
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

// rawFace returns a face at an absolute pixel size (for offscreen art).
func (u *UI) rawFace(f fontID, px float64) text.Face {
	k := faceKey{f, -int(px)} // negative keys never collide with scaled faces
	if fc, ok := u.faces[k]; ok {
		return fc
	}
	fc, err := opentype.NewFace(u.fonts[f], &opentype.FaceOptions{Size: px, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		panic(err)
	}
	tf := text.NewGoXFace(fc)
	u.faces[k] = tf
	return tf
}

// drawRawText centres text horizontally at (cx, y) in pixel space.
func (u *UI) drawRawText(dst *ebiten.Image, s string, face text.Face, cx, y float64, c color.RGBA) {
	op := &text.DrawOptions{}
	op.PrimaryAlign = text.AlignCenter
	op.GeoM.Translate(cx, y)
	op.ColorScale.ScaleWithColor(prem(c, 1))
	text.Draw(dst, s, face, op)
}
