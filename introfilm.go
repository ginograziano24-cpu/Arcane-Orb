package main

import (
	"bufio"
	"bytes"
	"image"
	_ "image/jpeg"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

// The 3D opening film: frames rendered in Blender (intro/build_intro.py)
// played back from intro/frames/f0001.jpg... at 24 fps, with the heroes'
// voice track (intro/voice.wav) and captions (intro/lines.txt). The frames
// stay on disk next to the game rather than inside the executable. When the
// folder is missing, "Watch opening" falls back to the old opening.

const introFPS = 24

type introLine struct {
	start, end float64
	who, text  string
}

type introFilm struct {
	dir    string
	frames int
	t      float64
	shown  int
	img    *ebiten.Image
	player *audio.Player
	lines  []introLine

	mu      sync.Mutex
	decoded map[int]image.Image
	want    int
	stop    chan struct{}
}

// introDir finds the rendered film beside the game (or one folder up, for
// a build run from Source/).
func introDir() string {
	var cands []string
	if exe, err := os.Executable(); err == nil {
		d := filepath.Dir(exe)
		cands = append(cands, filepath.Join(d, "intro"), filepath.Join(d, "..", "intro"))
	}
	if wd, err := os.Getwd(); err == nil {
		cands = append(cands, filepath.Join(wd, "intro"))
	}
	for _, c := range cands {
		if _, err := os.Stat(filepath.Join(c, "frames", "f0001.jpg")); err == nil {
			return c
		}
	}
	return ""
}

func newIntroFilm(g *Game, dir string) *introFilm {
	f := &introFilm{dir: dir, shown: -1, decoded: map[int]image.Image{}, stop: make(chan struct{})}
	for f.frames = 1; ; f.frames++ {
		if _, err := os.Stat(f.framePath(f.frames + 1)); err != nil {
			break
		}
	}
	f.lines = readIntroLines(filepath.Join(dir, "lines.txt"))
	if b, err := os.ReadFile(filepath.Join(dir, "voice.wav")); err == nil {
		if s, err := wav.DecodeWithSampleRate(sampleRate, bytes.NewReader(b)); err == nil {
			if p, err := g.snd.ctx.NewPlayer(s); err == nil {
				p.SetVolume(0.9)
				f.player = p
			}
		}
	}
	go f.decodeLoop()
	return f
}

func (f *introFilm) framePath(i int) string {
	return filepath.Join(f.dir, "frames", "f"+pad4(i)+".jpg")
}

func pad4(i int) string {
	s := strconv.Itoa(i)
	for len(s) < 4 {
		s = "0" + s
	}
	return s
}

func readIntroLines(path string) []introLine {
	fh, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer fh.Close()
	var out []introLine
	sc := bufio.NewScanner(fh)
	for sc.Scan() {
		p := strings.SplitN(sc.Text(), "|", 4)
		if len(p) != 4 || strings.HasPrefix(p[0], "#") {
			continue
		}
		a, e1 := strconv.ParseFloat(strings.TrimSpace(p[0]), 64)
		b, e2 := strconv.ParseFloat(strings.TrimSpace(p[1]), 64)
		if e1 != nil || e2 != nil {
			continue
		}
		out = append(out, introLine{a, b, strings.TrimSpace(p[2]), strings.TrimSpace(p[3])})
	}
	return out
}

// decodeLoop keeps the next few frames decoded ahead of playback.
func (f *introFilm) decodeLoop() {
	for {
		select {
		case <-f.stop:
			return
		default:
		}
		f.mu.Lock()
		want := f.want
		next := -1
		for i := want; i < want+6 && i <= f.frames; i++ {
			if _, ok := f.decoded[i]; !ok {
				next = i
				break
			}
		}
		for i := range f.decoded {
			if i < want-1 {
				delete(f.decoded, i)
			}
		}
		f.mu.Unlock()
		if next < 0 {
			time.Sleep(4 * time.Millisecond)
			continue
		}
		var im image.Image
		if fh, err := os.Open(f.framePath(next)); err == nil {
			im, _, _ = image.Decode(fh)
			fh.Close()
		}
		f.mu.Lock()
		f.decoded[next] = im
		f.mu.Unlock()
	}
}

func (f *introFilm) duration() float64 { return float64(f.frames) / introFPS }

func (f *introFilm) close() {
	close(f.stop)
	if f.player != nil {
		_ = f.player.Close()
	}
	if f.img != nil {
		f.img.Deallocate()
	}
}

func (f *introFilm) update(dt float64) bool {
	if f.player != nil && f.t == 0 && !f.player.IsPlaying() {
		f.player.Play()
	}
	if f.player != nil && f.player.IsPlaying() {
		f.t = f.player.Position().Seconds()
	} else {
		f.t += dt
	}
	i := int(f.t*introFPS) + 1
	if i > f.frames {
		return false
	}
	f.mu.Lock()
	f.want = i
	im, ok := f.decoded[i]
	f.mu.Unlock()
	if ok && im != nil && i != f.shown {
		if f.img == nil || f.img.Bounds() != im.Bounds() {
			if f.img != nil {
				f.img.Deallocate()
			}
			f.img = ebiten.NewImage(im.Bounds().Dx(), im.Bounds().Dy())
		}
		f.img.WritePixels(toRGBA(im).Pix)
		f.shown = i
	}
	return true
}

func toRGBA(im image.Image) *image.RGBA {
	if r, ok := im.(*image.RGBA); ok {
		return r
	}
	b := im.Bounds()
	out := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			out.Set(x, y, im.At(x, y))
		}
	}
	return out
}

func (f *introFilm) draw(g *Game, s *ebiten.Image) {
	s.Fill(rgba(4, 2, 10, 255))
	sw, sh := float64(s.Bounds().Dx()), float64(s.Bounds().Dy())
	if f.img != nil {
		iw, ih := float64(f.img.Bounds().Dx()), float64(f.img.Bounds().Dy())
		k := sw / iw
		if ih*k > sh {
			k = sh / ih
		}
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Scale(k, k)
		op.GeoM.Translate((sw-iw*k)/2, (sh-ih*k)/2)
		s.DrawImage(f.img, op)
	}
	u := g.ui
	for _, l := range f.lines {
		if f.t >= l.start && f.t < l.end {
			txt := l.text
			if l.who != "" {
				u.text(s, strings.ToUpper(l.who), fBold, 22, 960, 930, cGold, alignCenter)
			}
			u.text(s, txt, fReg, 30, 960, 970, cWhite, alignCenter)
		}
	}
	u.text(s, "Esc to skip", fReg, 16, 1880, 1050, cDim, alignRight)
}
