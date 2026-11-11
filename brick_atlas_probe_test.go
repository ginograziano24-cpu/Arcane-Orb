package main

import (
	"crypto/sha256"
	"fmt"
	"image"
	"testing"
)

func TestBrickSheetPixelsMatchOriginal(t *testing.T) {
	atlas, h := composeBrickSheet()
	if atlas == nil || atlas.Bounds() != image.Rect(0, 0, 4096, 3072) || h != 1536 {
		t.Fatal("unexpected brick sheet layout")
	}
	const original = "01eb85080156158820450146b5a7c393b2e5aae076bb2a92460a377f645fd648"
	// Go 1.26 changed image/jpeg's integer rounding. The unchanged source
	// images yield this exact second hash, with a maximum delta of 3/255.
	const go126 = "32c170590ce2a9915b9f14e6e27b4c1b086a2a32dd3041acdc3c83977d6b0cde"
	switch hash := fmt.Sprintf("%x", sha256.Sum256(atlas.(*image.RGBA).Pix)); hash {
	case original, go126:
	default:
		t.Fatalf("brick sheet pixel hash changed: %s", hash)
	}
}
