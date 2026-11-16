package main

// The scenery audit (-shots -audit): every world photographed from the
// play camera and from four outside angles, so clipping, unfinished backs
// and pop-in show up in one folder of screenshots.

var auditOn bool

// heroAuditZoom enlarges the class select hero for face close-ups (1 = normal).
var heroAuditZoom = 1.0

type auditCam struct {
	name     string
	eye, tgt V3
}

var auditCams = []auditCam{
	{"left", V3{-40, 22, -6}, V3{0, 2, 24}},
	{"right", V3{36, 7, 4}, V3{0, 3, 28}},
	{"back", V3{0, 15, 74}, V3{0, 2, 12}},
	{"high", V3{0, 64, -34}, V3{0, 0, 32}},
}

func auditScenes() []benchScene {
	out := []benchScene{{name: "audit-classsel", setup: func(g *Game) { camHook = nil; g.setState(StArch) }}}
	// every hero in the class select, then a close-up of the face (faces
	// and edges are where the 3D heroes go wrong)
	for i := range archetypes {
		i := i
		id := archetypes[i].ID
		for _, z := range []float64{1, 3.2} {
			z := z
			nm := "audit-hero-" + id
			if z > 1 {
				nm += "-face"
			}
			out = append(out, benchScene{name: nm, setup: func(g *Game) {
				camHook = nil
				heroAuditZoom = z
				if g.save.Unlocked == nil {
					g.save.Unlocked = map[string]bool{}
				}
				g.save.Unlocked[id] = true
				g.archSel = i
				g.setState(StArch)
			}})
		}
	}
	// the in-game HUD portrait of every hero (top left), in the first world
	for i := range archetypes {
		i := i
		out = append(out, benchScene{name: "audit-hud-" + archetypes[i].ID, level: true, setup: func(g *Game) {
			camHook = nil
			heroAuditZoom = 1
			g.run = NewRun("BENCH", &archetypes[i], &g.save, false)
			g.run.Lives = g.run.maxLives()
			g.settings.Classic = false
			forceWorld = 0
			g.level = g.newLevel(NCombat, 4242)
			forceWorld = -1
			g.setState(StLevel)
			g.level.introT = 0
		}})
	}
	// the letters between worlds, one from each writer, fully written out
	for _, from := range []string{"fumbleton", "mirabel"} {
		from := from
		out = append(out, benchScene{name: "audit-letter-" + from, setup: func(g *Game) {
			camHook = nil
			benchRunFor(g)
			g.setState(StMap)
			for i := range letterDefs {
				d := &letterDefs[i]
				if d.From == from && len(d.Body) >= 2 {
					g.letter = &letterAnim{def: d, hero: "Seraphine", ps: letterPS(g.run), photo: from == "fumbleton", art: map[string]string{"fumbleton": "fumble-1", "mirabel": "mirabel-0"}[from]}
					g.letter.t = g.letter.doneAt() + 1
					g.letter.full = true
					g.letter.lines = g.letter.lineCount()
					g.letter.kissd = true
					break
				}
			}
		}})
	}
	for w := 0; w < NumWorlds; w++ {
		w := w
		place := func(g *Game) {
			g.letter, g.pendingLetter = nil, nil // the letter shots come first
			benchRunFor(g)
			g.settings.Classic = false
			forceWorld = w
			g.level = g.newLevel(NCombat, int64(4242+w))
			forceWorld = -1
			g.setState(StLevel)
			g.level.introT = 0
		}
		out = append(out, benchScene{name: "audit-w" + itoa(w+1) + "-play", level: true, setup: func(g *Game) {
			camHook = nil
			heroAuditZoom = 1
			place(g)
		}})
		for _, c := range auditCams {
			c := c
			out = append(out, benchScene{name: "audit-w" + itoa(w+1) + "-" + c.name, level: true, setup: func(g *Game) {
				camHook = func(cam *Camera, L *Level) { cam.Eye, cam.Target = c.eye, c.tgt }
				place(g)
			}})
		}
	}
	return out
}
