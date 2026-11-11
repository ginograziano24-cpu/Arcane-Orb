package main

import (
	"bytes"
	"encoding/json"
	"image"
	"math"
	"testing"
)

// Cross-asset contracts prevent portrait stretching, displaced effects and missing HUD poses.
func TestHeroPortraitAssetContracts(t *testing.T) {
	for _, a := range archetypes {
		for _, suffix := range []string{".jpg", "_a.png"} {
			b, err := portraitFS.ReadFile("assets/portraits/" + a.ID + suffix)
			if err != nil {
				t.Fatal(err)
			}
			c, _, err := image.DecodeConfig(bytes.NewReader(b))
			if err != nil {
				t.Fatal(err)
			}
			if c.Width != 1200 || c.Height != 1600 {
				t.Errorf("%s%s: %dx%d violates the 3:4 class selection frame", a.ID, suffix, c.Width, c.Height)
			}
		}
		b, err := portraitFS.ReadFile("assets/portraits/" + a.ID + ".json")
		if err != nil {
			t.Fatal(err)
		}
		var anchors []pAnchor
		if err = json.Unmarshal(b, &anchors); err != nil {
			t.Fatal(err)
		}
		if len(anchors) == 0 {
			t.Errorf("%s: no signature effect anchor", a.ID)
		}
		for _, p := range anchors {
			if math.IsNaN(p.X) || math.IsNaN(p.Y) || p.X < 0 || p.X > 1 || p.Y < 0 || p.Y > 1 {
				t.Errorf("%s: invalid anchor %#v", a.ID, p)
			}
		}
		for _, pose := range []string{"rest", "look", "fierce", "hurt"} {
			b, err := portraitFS.ReadFile("assets/portraits/hud/" + a.ID + "_" + pose + ".png")
			if err != nil {
				t.Fatal(err)
			}
			c, _, err := image.DecodeConfig(bytes.NewReader(b))
			if err != nil {
				t.Fatal(err)
			}
			if c.Width != 536 || c.Height != 680 {
				t.Errorf("%s/%s: inconsistent HUD framing", a.ID, pose)
			}
		}
	}
}
