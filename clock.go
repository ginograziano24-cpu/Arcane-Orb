package main

import "time"

// Hooks that let the trailer capture build (capture_on.go) drive the game
// frame by frame. In normal builds they stay at these defaults.
var (
	// nowFn is the game's clock.
	nowFn = time.Now
	// sfxHook, when set, receives every sound instead of the speakers;
	// returning true swallows it.
	sfxHook func(name string, variant int, vol float64) bool
	// camHook may adjust the gameplay camera after it is placed.
	camHook func(c *Camera, L *Level)
	// inputHook, when set, replaces player input in a level.
	inputHook func(g *Game) Input
	// hideHUD hides the in-level HUD (the boss intro still shows).
	hideHUD bool
)
