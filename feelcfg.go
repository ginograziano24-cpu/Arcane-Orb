package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// feelConfig gathers the game-feel numbers in one place so they can be tuned
// while playtesting. The defaults are the values the game shipped with.
//
// To try new values without rebuilding, put a feel.json next to the game
// (or in the save folder) with any of these fields, for example:
//
//	{"HitStopEliteBreak": 0.12, "RumbleScale": 0.6}
//
// Fields that are left out keep their defaults.
type feelConfig struct {
	// hit-stop: the whole chamber freezes for this many seconds
	HitStopCrit       float64 // a critical hit (at most every 0.35 s)
	HitStopEliteBreak float64 // an elite brick shatters
	HitStopHeavyBreak float64 // a brick with 3+ health shatters
	HitStopPaddleLost float64 // the paddle explodes (life lost)

	// screen shake added on those moments (scaled by the visual style and the Shake setting)
	ShakeCrit       float64
	ShakePaddleLost float64

	// camera knock when the orb breaks or hits a brick
	KickHeavyBreak float64 // elites and 3+ health
	KickBreak      float64
	KickHit        float64 // a hit that doesn't break
	KickSpring     float64 // stiffness of the spring bringing the camera back
	KickDamping    float64

	// controller rumble (Settings > Effects > Controller rumble)
	RumbleScale      float64 // multiplies every rumble below; 0 turns it off
	RumblePaddle     float64 // a hard paddle return
	RumbleHeavyBreak float64
	RumbleBossHit    float64
	RumbleBossKill   float64
	RumbleLifeLost   float64
}

var feel = feelConfig{
	HitStopCrit:       0.045,
	HitStopEliteBreak: 0.09,
	HitStopHeavyBreak: 0.035,
	HitStopPaddleLost: 0.12,
	ShakeCrit:         0.18,
	ShakePaddleLost:   1.1,
	KickHeavyBreak:    7,
	KickBreak:         3.5,
	KickHit:           1.5,
	KickSpring:        380,
	KickDamping:       30,
	RumbleScale:       1,
	RumblePaddle:      0.18,
	RumbleHeavyBreak:  0.3,
	RumbleBossHit:     0.45,
	RumbleBossKill:    0.9,
	RumbleLifeLost:    0.7,
}

// loadFeelOverrides reads feel.json from next to the game or the save folder.
func loadFeelOverrides() {
	var dirs []string
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(exe))
	}
	dirs = append(dirs, filepath.Dir(savePath()))
	for _, d := range dirs {
		b, err := os.ReadFile(filepath.Join(d, "feel.json"))
		if err != nil {
			continue
		}
		c := feel
		if json.Unmarshal(b, &c) == nil {
			feel = c
		}
		return
	}
}

// rumble buzzes every connected controller briefly. strength is 0..1; heavier
// moments lean on the low (strong) motor, light ones on the high (weak) one.
func (g *Game) rumble(strength float64, d time.Duration) {
	if g == nil || g.settings == nil || g.settings.NoRumble || feel.RumbleScale <= 0 || strength <= 0 {
		return
	}
	s := clamp01(strength * feel.RumbleScale)
	opt := &ebiten.VibrateGamepadOptions{Duration: d, StrongMagnitude: s * s, WeakMagnitude: s}
	rumbleIDs = ebiten.AppendGamepadIDs(rumbleIDs[:0])
	for _, id := range rumbleIDs {
		ebiten.VibrateGamepad(id, opt)
	}
}

var rumbleIDs []ebiten.GamepadID
