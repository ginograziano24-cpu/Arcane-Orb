package main

// Effects intensity (Settings > Effects). One master dial over the
// decorative touches added in visfx9-17:
//   High   (0, default) everything, today's look
//   Medium (1)          ambient decoration off (drifting stars, motes,
//                       embers, shimmers, sheens); feedback flourishes stay
//   Low    (2)          flourishes off too (hover glows, pops, flashes,
//                       stamps, celebratory rings and sparks)
// Gameplay cues are never touched: pickup rings and smoke, crit ring,
// barrier ripple, last-life edge and heart ring, drop markers, ward
// barrier, familiar shots and dots, bond/partner links, banish and lock
// state, rarity pillars, affordability and parent hints. See ROADMAP AH.

const (
	fxHigh = iota
	fxMedium
	fxLow
)

var fxLevelNames = [...]string{"High", "Medium", "Low"}

func (g *Game) fxLevel() int {
	if g == nil || g.settings == nil {
		return fxHigh
	}
	return min(max(g.settings.FxLevel, 0), fxLow)
}

// fxAmount: 1 at High, 0.5 at Medium, 0 at Low.
func (g *Game) fxAmount() float64 { return [...]float64{1, 0.5, 0}[g.fxLevel()] }

// ambientOff: purely decorative motion is skipped at Medium and Low.
func (g *Game) ambientOff() bool { return g.fxLevel() >= fxMedium }

// flourishOff: one-shot feedback decoration is skipped at Low.
func (g *Game) flourishOff() bool { return g.fxLevel() >= fxLow }
