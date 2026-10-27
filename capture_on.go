//go:build capture && js

package main

// Trailer capture mode. Build with:
//
//	GOOS=js GOARCH=wasm go build -tags capture -o game.wasm .
//
// The page then steps the game one frame at a time: it sets the global
// __capReq to the next frame number, waits for __capDone to match, and
// screenshots the canvas. A director (trailer.go) stages every shot, and
// every sound the game makes is logged with its time so the soundtrack
// can be mixed offline (__capEvents, __capMix).

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"math/rand"
	"syscall/js"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type sfxEvent struct {
	T    float64 `json:"t"`
	Name string  `json:"n"`
	V    int     `json:"v"`
	Vol  float64 `json:"g"`
}

var capState struct {
	started   bool
	frame     int
	t         float64
	fps       float64
	pending   bool
	recording bool
	events    []sfxEvent
}

var capEpoch = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func init() {
	capState.fps = 60
	nowFn = func() time.Time { return capEpoch.Add(time.Duration(capState.t * float64(time.Second))) }
	sfxHook = func(name string, v int, vol float64) bool {
		if capState.recording {
			capState.events = append(capState.events, sfxEvent{capState.t, name, v, vol})
		}
		return true
	}
	js.Global().Set("__capDone", 0)
	js.Global().Set("__capTotal", int(math.Round(trailerLength()*capState.fps)))
	js.Global().Set("__capEvents", js.FuncOf(func(this js.Value, args []js.Value) any {
		b, _ := json.Marshal(capState.events)
		return string(b)
	}))
	js.Global().Set("__capMix", js.FuncOf(func(this js.Value, args []js.Value) any {
		var ev []sfxEvent
		if len(args) > 0 {
			_ = json.Unmarshal([]byte(args[0].String()), &ev)
		}
		stems := 0
		if len(args) > 1 {
			stems = args[1].Int() // 1 = score only, 2 = sfx only
		}
		return base64.StdEncoding.EncodeToString(mixTrailer(theGame, ev, stems))
	}))
}

var theGame *Game

func capFPS() float64 {
	if v := js.Global().Get("__capFPS"); v.Type() == js.TypeNumber {
		return v.Float()
	}
	return 60
}

// captureUpdate holds the game still until the page asks for a frame.
func captureUpdate(g *Game) bool {
	theGame = g
	req := js.Global().Get("__capReq")
	if req.Type() != js.TypeNumber || req.Int() <= capState.frame {
		return true
	}
	if !capState.started {
		capState.started = true
		capState.fps = capFPS()
		ebiten.SetScreenClearedEveryFrame(false)
		setupTrailerGame(g)
	}
	capState.frame = req.Int() // the page may jump ahead to preview a shot
	capState.t = float64(capState.frame-1) / capState.fps
	capState.pending = true
	capState.recording = true
	directorStep(g, capState.t)
	if v := js.Global().Get("__showHUD"); v.Type() == js.TypeBoolean {
		hideHUD = !v.Bool()
	}
	return false
}

// captureSkipDraw leaves the last frame on screen between requests, and
// lets the director draw whole shots of its own.
func captureSkipDraw(g *Game, dst *ebiten.Image) bool {
	if !capState.pending {
		return true
	}
	if v := js.Global().Get("__capDraw"); v.Type() == js.TypeBoolean && !v.Bool() {
		// preview runs skip rendering frames nobody will look at
		capState.pending = false
		js.Global().Set("__capDone", capState.frame)
		return true
	}
	return directorDraw(g, dst, capState.t)
}

func captureAfterDraw(g *Game, dst *ebiten.Image) {
	if !capState.pending {
		return
	}
	directorOverlay(g, dst, capState.t)
	capState.pending = false
	js.Global().Set("__capDone", capState.frame)
}

// setupTrailerGame gives the trailer a fully unlocked, deterministic game.
func setupTrailerGame(g *Game) {
	g.rng = rand.New(rand.NewSource(7))
	g.save = defaultSave()
	g.settings = &g.save.Settings
	g.save.XP = 99999
	g.save.Wins = 3
	g.save.AscMax = 10
	g.save.StorySeen = true
	for _, a := range archetypes {
		g.save.Unlocked[a.ID] = true
	}
	for _, m := range modList {
		g.save.Seen[m.ID] = true
	}
	for _, a := range achievements {
		g.save.Ach[a.ID] = true // no unlock popups in the trailer
	}
	s := g.settings
	s.Particles, s.Shake, s.Flashes, s.NoBloom = 1, true, true, false
	g.snd.Music = 0
	g.snd.SFX = 1
	hideHUD = true
}
