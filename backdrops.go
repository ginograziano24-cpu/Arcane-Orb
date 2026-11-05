package main

import (
	"embed"
	"fmt"
	_ "image/jpeg"

	"github.com/hajimehoshi/ebiten/v2"
)

// Painted world backdrops (generated offline by tools/paint_backdrops.py).
//
//go:embed assets/bg/*.jpg
var backdropFS embed.FS

var backdropNames = map[int]string{
	WNeon: "neon", WMeadow: "meadow", WCaverns: "caverns",
	WRuins: "ruins", WClouds: "clouds", WMachine: "machine",
}

var backdropCache = map[string]*ebiten.Image{}

// backdrop returns the painted sky for a world; variant picks one of the
// paintings. Returns nil if the asset is missing (falls back to makeSky).
func backdrop(world int, variant int) *ebiten.Image {
	name := fmt.Sprintf("%s_%d", backdropNames[world], variant%2)
	if img, ok := backdropCache[name]; ok {
		return img
	}
	img := loadImage("bg/" + name).img
	backdropCache[name] = img
	return img
}
