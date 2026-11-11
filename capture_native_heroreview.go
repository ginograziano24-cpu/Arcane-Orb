//go:build heroreview && !js

package main

import (
	"bytes"
	"encoding/json"
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	"image/png"
	"os"
	"path/filepath"
)

// Offline developer capture build. Does not alter release behavior or write user saves.
var heroReviewFrame int
var reviewChambers bool
var reviewGraph []byte
var reviewNavigation bool
var reviewHUD bool
var navigationResults []map[string]interface{}

func init() {
	noPersist = true
	for _, a := range os.Args[1:] {
		if a == "-review-hud" {
			reviewHUD = true
		}
		if a == "-review-navigation" {
			reviewNavigation = true
			reviewChambers = true
		}
		if a == "-review-chambers" {
			reviewChambers = true
		}
	}
}
func captureUpdate(g *Game) bool {
	if reviewOpening {
		return captureOpeningUpdate(g)
	}
	i := heroReviewFrame / 60
	limit := len(archetypes)
	if reviewHUD {
		limit = 1
	}
	if reviewChambers {
		limit = numActs
	}
	if i >= limit {
		g.quit = true
		return true
	}
	if heroReviewFrame%60 == 0 {
		if heroReviewFrame == 0 {
			g.save = defaultSave()
			if benchWinW > 0 && benchWinH > 0 {
				scale := ebiten.Monitor().DeviceScaleFactor()
				ebiten.SetFullscreen(false)
				ebiten.SetWindowSize(int(float64(benchWinW)/scale), int(float64(benchWinH)/scale))
			}
		}
		benchConfig(g)
		for _, a := range archetypes {
			g.save.Unlocked[a.ID] = true
		}
		if reviewChambers {
			g.run = NewRun("CHAMBER-REVIEW", &archetypes[0], &g.save, false)
			g.run.Act = i + 1
			reach := g.run.Reachable()
			if len(reach) > 0 && i > 0 {
				g.run.Map.Nodes[reach[0]].Omen = []string{"", "fortified", "gilded"}[i]
			}
			g.setState(StMap)
			reviewGraph, _ = json.Marshal(g.run.Map)
		} else {
			g.setState(StArch)
		}
		g.archSel = i
		g.archShown = i
		g.archPrev = i
		g.archFade = 1
		g.stateT = 3
	}
	return false
}
func captureSkipDraw(g *Game, dst *ebiten.Image) bool {
	g.achQueue = nil
	g.achT = 0
	if reviewNavigation {
		switch heroReviewFrame % 60 {
		case 20, 22:
			g.navLatch = navInput{dx: 1}
		case 56:
			g.navLatch = navInput{confirm: true}
		}
	}
	return false
}
func captureAfterDraw(g *Game, dst *ebiten.Image) {
	if reviewOpening {
		captureOpeningAfterDraw(g, dst)
		return
	}
	if reviewHUD {
		dst.Fill(rgba(15, 12, 28, 255))
		g.ui.text(dst, "NATIVE HUD PORTRAIT REVIEW", fRune, 30, 960, 42, cGold, alignCenter)
		oldRun, oldHero := g.run, g.hero
		for i := range archetypes {
			g.run = NewRun("HUD-REVIEW", &archetypes[i], &g.save, false)
			x := 230 + float64(i)*230
			g.ui.text(dst, archetypes[i].Name, fReg, 20, x+75, 90, cGold, alignCenter)
			for j, pose := range []string{"rest", "look", "fierce", "hurt"} {
				y := 125 + float64(j)*227
				g.hero = heroPortrait{pose: pose, prev: pose, fade: 1}
				g.drawHero(dst, nil, x, y, 150, 186)
				g.ui.text(dst, pose, fReg, 16, x+75, y+198, cDim, alignCenter)
			}
		}
		g.run, g.hero = oldRun, oldHero
	}
	if heroReviewFrame%60 == 55 {
		exe, _ := os.Executable()
		dir := filepath.Join(filepath.Dir(exe), "Hero-Diversity", "runtime-captures")
		name := archetypes[heroReviewFrame/60].ID
		if reviewHUD {
			name = "hud-all-poses"
		}
		if reviewChambers {
			dir = filepath.Join(filepath.Dir(exe), "Chamber-Projection", "runtime-captures")
			name = "realm-" + itoa(heroReviewFrame/60+1)
			if reviewNavigation {
				name += "-navigation"
				valid := false
				for _, i := range g.run.Reachable() {
					if g.ui.nav.focus == "node"+itoa(i) {
						valid = true
					}
				}
				if !valid {
					panic("Navigation focus left the reachable chambers")
				}
			}
			if benchWinW > 0 {
				dir = filepath.Join(dir, itoa(benchWinW)+"x"+itoa(benchWinH))
			}
			after, _ := json.Marshal(g.run.Map)
			if !bytes.Equal(reviewGraph, after) {
				panic("Rendering mutated the chamber graph")
			}
		}
		_ = os.MkdirAll(dir, 0755)
		img := image.NewRGBA(dst.Bounds())
		dst.ReadPixels(img.Pix)
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			panic(err)
		}
		if err = png.Encode(f, img); err != nil {
			panic(err)
		}
		_ = f.Close()
	}
	if reviewNavigation && heroReviewFrame%60 == 56 {
		if g.state != StLevel {
			panic("Navigation confirmation did not enter a chamber")
		}
		navigationResults = append(navigationResults, map[string]interface{}{"act": heroReviewFrame/60 + 1, "focus": g.ui.nav.focus, "enteredChamber": true, "saveWritesDisabled": noPersist})
		exe, _ := os.Executable()
		b, _ := json.MarshalIndent(navigationResults, "", "  ")
		_ = os.WriteFile(filepath.Join(filepath.Dir(exe), "Chamber-Projection", "navigation-check.json"), b, 0644)
	}
	heroReviewFrame++
}
