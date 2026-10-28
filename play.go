package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"strings"

	"orb/levelgen"
)

// World layout (units): x across, z away from the player, y up.
const (
	HalfW    = 14.0
	FarZ     = 31.5
	GridTopZ = 30.3
	CellW    = 2.0
	CellD    = 1.1
	PaddleZ  = 2.2
	PaddleD  = 0.7
	PaddleH  = 0.7
	BarrierZ = 0.4
	LoseZ    = -2.5
	baseW    = 3.6
	LayerH   = 0.85
)

// forceWorld pins the world for testing (-1 = normal rotation).
var forceWorld = -1

func cellX(c int) float64 { return -HalfW + (float64(c)+0.5)*CellW }
func cellZ(r int) float64 { return GridTopZ - (float64(r)+0.5)*CellD }

// ---------------------------------------------------------------- entities

type Brick struct {
	R, C        int
	Kind        levelgen.Kind
	HP, MaxHP   float64
	Col         color.RGBA
	Alive       bool
	Flash       float64
	Burn        float64
	burnTick    float64
	Frozen      float64
	SinceHit    float64
	Timer       float64
	OffX, OffZ  float64 // visual slide offset
	Hits        int
	BossPart    bool
	Core        bool
	Appear      float64
	shieldFlash float64
	Layer       int
	Up          *Brick
	FallY       float64
}

func (b *Brick) X() float64 { return cellX(b.C) }
func (b *Brick) Z() float64 { return cellZ(b.R) }

type Ball struct {
	X, Z, VX, VZ, R float64
	Stuck           bool
	Off             float64
	Trail           [][2]float64
	PierceLeft      int
	Charged         bool
	Momentum        int
	Phantom         float64 // >0: temporary orb with this many seconds left
	Rico            int     // wall bounces since the paddle (Ricochet)
	Storm           float64 // Stormcaller supercharge seconds left
}

type Power int

const (
	PMulti Power = iota
	PExpand
	PShrink
	PSlow
	PHaste
	PFire
	PLaser
	PBarrier
	PGlue
	PMagnet
	PLife // a restored life (never in the normal drop table)
	numPowers
)

type powerDef struct {
	Letter, Name string
	Col          color.RGBA
	Good         bool
	W            float64
	Dur          float64
}

var powerDefs = [numPowers]powerDef{
	PMulti:   {"M", "Multi-Orb", color.RGBA{255, 210, 60, 255}, true, 12, 0},
	PExpand:  {"E", "Expand", color.RGBA{70, 150, 255, 255}, true, 11, 0},
	PShrink:  {"S", "Shrink", color.RGBA{230, 60, 60, 255}, false, 5, 0},
	PSlow:    {"W", "Slow", color.RGBA{80, 220, 200, 255}, true, 8, 12},
	PHaste:   {"H", "Haste", color.RGBA{200, 40, 120, 255}, false, 5, 10},
	PFire:    {"F", "Fire Orb", color.RGBA{255, 120, 30, 255}, true, 5, 9},
	PLaser:   {"L", "Laser", color.RGBA{255, 70, 70, 255}, true, 7, 10},
	PBarrier: {"B", "Barrier", color.RGBA{150, 120, 255, 255}, true, 7, 0},
	PGlue:    {"G", "Glue", color.RGBA{120, 220, 70, 255}, true, 7, 14},
	PMagnet:  {"$", "Magnet", color.RGBA{255, 200, 80, 255}, true, 5, 15},
	PLife:    {"+", "Extra Life", color.RGBA{255, 110, 160, 255}, true, 0, 0},
}

type Drop struct {
	X, Z float64
	P    Power
	T    float64
}

type Coin struct {
	X, Y, Z, VX, VY, VZ float64
	Val                 int
	T                   float64
}

type Bolt struct {
	X, Z   float64
	Pierce int     // bricks it can pass through (overclocked beam)
	Mult   float64 // damage multiplier
	last   *Brick
}

type Rock struct {
	X, Z, TX float64
	Warn     float64
	Spin     float64
}

const (
	pDebris = iota
	pSpark
	pSmoke
	pAmbient
)

type Particle struct {
	P, V      V3
	Life, Max float64
	Size      float64
	Col       color.RGBA
	Kind      int
	Grav      float64
	Rot, Spin float64 // debris tumble
}

type Ring struct {
	X, Z, R, MaxR, Life, Max float64
	Col                      color.RGBA
}

type Arc struct {
	Pts  []V3
	Life float64
	Col  color.RGBA
}

type FloatText struct {
	P     V3
	S     string
	Col   color.RGBA
	Life  float64
	Big   bool
	Scale float64 // >0: a damage number at this size
}

type Boss struct {
	Name        string
	HP, MaxHP   float64
	coreT, atkT float64
	rebuildT    float64
	coreHits    int
	body        [][2]int
	Dead        bool
	deathT      float64
	Hurt        float64
}

// ---------------------------------------------------------------- level

type Level struct {
	introT, introMax      float64 // boss intro cinematic
	hitStop               float64
	critStopT             float64 // last crit freeze (they're rationed)
	finaleT               float64 // >0 while the last brick's slow-motion plays
	finX, finZ            float64 // where the last brick broke
	hadFinale             bool
	lastBrkX, lastBrkZ    float64
	lostOrbs              int // orbs dropped this chamber (for Untouchable)
	startLeft             int
	paddleGone            float64 // >0 while the exploded paddle reforms
	gameOverT             float64 // >0 while the last life's blast plays out
	lifeFlash             float64 // required bricks at the start, for the last-brick assist
	timeWarp, midasT      float64 // scroll effects
	omen                  *omen
	peakCombo             int
	titanT, magnetT       float64
	cls                   classState
	lastPadX              float64
	igniteN               int
	abilReadyT, abilFlash float64
	g                     *Game
	Data                  *levelgen.Level
	Biome                 *Biome
	Node                  NodeType
	Grid                  [levelgen.MaxRows][levelgen.Cols]*Brick
	Bricks                []*Brick
	Left                  int
	Crystals              int
	Balls                 []*Ball
	Drops                 []*Drop
	Coins                 []*Coin
	Bolts                 []*Bolt
	Rocks                 []*Rock
	Parts                 []*Particle
	Rings                 []*Ring
	Arcs                  []*Arc
	Floats                []*FloatText
	Boss                  *Boss
	rng                   *rand.Rand

	PaddleX, PaddleW float64
	tempSize         int
	Timers           [numPowers]float64
	Barrier          int
	barrierRegenT    float64
	laserT           float64
	stun             float64

	T, Shake, Flash float64
	Combo           int
	comboT          float64
	Frenzy          float64
	impacts         int
	killsSinceHeal  int
	hydraCount      int
	GoldEarned      int
	Cleared, Done   bool
	clearT          float64
	propSeed        int64
	ambientT        float64
	propsX          []V3
	paddleBounce    float64
	hurtFlash       float64
	Walls           []levelgen.Pt
	Vis             Visuals
	Designed        bool
	rushBonus       float64 // Gold Rush damage bonus
	bhCount         int     // bricks towards the next black hole
	eruptDepth      int     // guards chained eruptions
}

func (g *Game) newLevel(node NodeType, seed int64) *Level {
	r := g.run
	act := r.Act
	wi := r.pickWorld(seed)
	if forceWorld >= 0 {
		wi = forceWorld
	}
	if node == NBoss {
		wi = acts[r.AI()].Signature
	}
	r.LastWorld = wi
	g.snd.World = wi
	b, vis := levelLook(&worlds[wi], seed)
	if node == NBoss {
		bc := worlds[wi]
		bc.BossName = acts[r.AI()].BossName
		b, vis = &bc, Visuals{}
	}
	L := &Level{g: g, Biome: b, Node: node, Vis: vis, rng: rand.New(rand.NewSource(seed ^ 0x5eed))}
	r.Depth++
	if node == NBoss {
		L.Data = &levelgen.Level{Name: b.BossName, Formation: "boss", Seed: seed}
	} else {
		L.Data = pacedLevel(levelgen.Params{Seed: seed, Depth: r.Depth + (act-1)*2, Act: act, Elite: node == NElite})
		L.Designed = strings.HasPrefix(L.Data.Formation, "design")
	}
	hpMul := (1.0 + float64(act-1)*0.35) * (1 + 0.02*float64(r.Depth)) * ascHP(r.Asc, node == NElite, false) * math.Pow(1.25, float64(r.Loop()))
	if r.Cursed && node != NBoss {
		hpMul *= 1.5
	}
	hpMul *= 1 + r.Stats.BrickHPMult
	for row := 0; row < levelgen.MaxRows; row++ {
		for c := 0; c < levelgen.Cols; c++ {
			cell := L.Data.Cells[row][c]
			if cell.Kind == levelgen.Empty {
				continue
			}
			hp := float64(cell.HP)
			if cell.Kind != levelgen.Steel {
				hp = math.Ceil(hp * hpMul)
			}
			L.addColumn(row, c, cell, hp, hpMul)
		}
	}
	if node == NBoss {
		L.setupBoss()
		L.introT, L.introMax = 5.2, 5.2
		g.snd.Play("bossintro")
		g.snd.Duck(0.8, 4)
	}
	L.startLeft = L.Left
	if r.hasAbility() && node != NBoss {
		// abilities come in partly charged, so they shape a chamber
		// rather than open it
		r.AbilT = math.Max(r.AbilT, r.Arch.Cooldown*0.4)
	}
	L.Walls = levelgen.Arena(L.Data.Shape)
	L.PaddleX = 0
	L.PaddleW = baseW * r.Stats.PaddleWidth
	L.Barrier = r.Stats.BarrierPerLevel
	L.propSeed = seed
	pr := rand.New(rand.NewSource(seed))
	for i := 0; i < 14; i++ {
		side := float64(pr.Intn(2)*2 - 1)
		L.propsX = append(L.propsX, V3{side * (24 + pr.Float64()*30), 2 + pr.Float64()*6, 14 + pr.Float64()*70})
	}
	L.spawnServe()
	if r.Stats.OpeningPower {
		opts := []Power{PExpand, PSlow, PFire, PLaser, PBarrier, PMagnet}
		L.applyPower(opts[L.rng.Intn(len(opts))])
	}
	return L
}

func (L *Level) addBrick(r, c int, k levelgen.Kind, hp float64, base color.RGBA) *Brick {
	br := &Brick{R: r, C: c, Kind: k, HP: hp, MaxHP: hp, Col: kindColor(k, base), Alive: true, Appear: 0}
	if k == levelgen.Explosive || k == levelgen.Mystery || k == levelgen.Multiplier || k == levelgen.Crystal || k == levelgen.Chain || k == levelgen.Split {
		br.HP, br.MaxHP = 1, 1
	}
	L.Grid[r][c] = br
	L.Bricks = append(L.Bricks, br)
	if k.Required() {
		L.Left++
	}
	if k == levelgen.Crystal {
		L.Crystals++
	}
	return br
}

// addColumn builds a stack of bricks on one cell.
func (L *Level) addColumn(r, c int, cell levelgen.Cell, hp, hpMul float64) {
	bio := L.Biome
	bottom := L.addBrick(r, c, cell.Kind, hp, bio.Hues[cell.Color%6])
	prev := bottom
	for k := 1; k < cell.Stack; k++ {
		kind := levelgen.Basic
		lhp := math.Ceil(float64(max(1, cell.HP)) * hpMul)
		if cell.Kind == levelgen.Steel {
			kind = levelgen.Steel
			lhp = 1
		}
		if k == cell.Stack-1 && cell.Cap != levelgen.Empty {
			kind = cell.Cap
		}
		col := bio.Hues[(cell.Color+k*L.Data.LayerShift)%6]
		if L.Data.LayerShift == 0 {
			col = lighten(col, 0.08*float64(k))
		}
		nb := &Brick{R: r, C: c, Kind: kind, HP: lhp, MaxHP: lhp, Col: kindColor(kind, col), Alive: true, Layer: k}
		if kind == levelgen.Explosive || kind == levelgen.Crystal {
			nb.HP, nb.MaxHP = 1, 1
		}
		L.Bricks = append(L.Bricks, nb)
		if kind.Required() {
			L.Left++
		}
		if kind == levelgen.Crystal {
			L.Crystals++
		}
		prev.Up = nb
		prev = nb
	}
}

// removeFromColumn unlinks a brick from its stack; bricks above it drop.
func (L *Level) removeFromColumn(b *Brick) {
	var above *Brick
	if L.Grid[b.R][b.C] == b {
		L.Grid[b.R][b.C] = b.Up
		above = b.Up
	} else {
		for n := L.Grid[b.R][b.C]; n != nil; n = n.Up {
			if n.Up == b {
				n.Up = b.Up
				above = b.Up
				break
			}
		}
	}
	for n := above; n != nil; n = n.Up {
		n.Layer--
		n.FallY += LayerH
	}
	b.Up = nil
}

func (L *Level) top(b *Brick) *Brick {
	for b != nil && b.Up != nil {
		b = b.Up
	}
	return b
}

func (L *Level) cellFree(r, c int) bool {
	return r >= 0 && r < levelgen.MaxRows && c >= 0 && c < levelgen.Cols && L.Grid[r][c] == nil && !L.Data.Outside[r][c]
}

func (L *Level) spawnServe() {
	r := L.g.run
	n := 1
	for i := 0; i < n; i++ {
		L.Balls = append(L.Balls, &Ball{R: L.ballR(), Stuck: true, Off: (L.rng.Float64() - 0.5) * 0.8, X: L.PaddleX, Z: PaddleZ + 0.6, PierceLeft: r.Stats.Pierce})
	}
}

func (L *Level) ballR() float64 {
	r := math.Min(0.75, 0.32*L.g.run.Stats.SizeMult)
	if L.titanT > 0 {
		r *= 2
	}
	return r
}

func (L *Level) speed() float64 {
	r := L.g.run
	base := math.Min(22, 16+float64(r.Depth)*0.22+float64(r.Act-1)*1.0)
	base += math.Min(4, L.T*0.035)
	base *= r.Stats.SpeedMult
	if L.Timers[PSlow] > 0 {
		base *= 0.7
	}
	if L.Timers[PHaste] > 0 {
		base *= 1.3
	}
	if L.timeWarp > 0 {
		base *= 0.55
	}
	if L.hasOmen("frenzy") {
		base *= 1.15
	}
	return base
}

func (L *Level) paddleTarget() float64 {
	w := baseW * L.g.run.Stats.PaddleWidth
	switch {
	case L.tempSize > 0:
		w *= 1 + 0.35*float64(L.tempSize)
	case L.tempSize < 0:
		w *= 0.7
	}
	return w
}

// ---------------------------------------------------------------- update

func (L *Level) Update(dt float64, in Input) {
	g := L.g
	r := g.run
	if L.updateLives(dt) {
		return // the last life's explosion plays out
	}
	// boss intro: the arena holds still while the Guardian assembles
	if L.introT > 0 {
		L.T += dt
		was := L.introT
		L.introT -= dt
		if in.Fire && L.introMax-L.introT > 0.4 {
			L.introT = math.Min(L.introT, 0.25)
		}
		L.updateBricks(dt)
		L.updateEffects(dt)
		L.Shake = math.Max(0, L.Shake-dt*2)
		for _, b := range L.Bricks {
			if b.BossPart && b.Appear > 0 && b.Appear-dt*4 <= 0 {
				L.Shake = math.Max(L.Shake, 0.12)
			}
		}
		if was > 1.0 && L.introT <= 1.0 {
			L.g.snd.Play("boom")
			L.Shake = math.Max(L.Shake, 0.6)
			L.Flash = math.Max(L.Flash, 0.3)
		}
		if L.introT <= 0 {
			for _, b := range L.Bricks {
				b.Appear, b.FallY = 1, 0
			}
		}
		return
	}
	// hit-stop: a few frames of near-freeze sell the big impacts
	if L.hitStop > 0 {
		L.hitStop -= dt
		dt *= 0.12
	}
	// the last brick: time crawls, then eases back to speed
	if L.finaleT > 0 {
		L.finaleT -= dt
		p := clamp01(1 - L.finaleT/finaleDur)
		dt *= 0.16 + 0.84*p*p
	}
	L.T += dt
	for i := range L.Timers {
		if L.Timers[i] > 0 {
			L.Timers[i] -= dt
			if L.Timers[i] <= 0 {
				L.Timers[i] = 0
				if Power(i) == PGlue {
					L.release()
				}
			}
		}
	}
	if L.Frenzy > 0 {
		L.Frenzy -= dt
	}
	L.timeWarp = math.Max(0, L.timeWarp-dt)
	L.midasT = math.Max(0, L.midasT-dt)
	L.titanT = math.Max(0, L.titanT-dt)
	L.magnetT = math.Max(0, L.magnetT-dt)
	L.comboT -= dt
	if L.comboT <= 0 && L.Combo > 0 {
		overdrive := false
		for _, a := range r.Active {
			if a == "Overdrive" {
				overdrive = true
			}
		}
		if !overdrive || len(L.Balls) == 0 {
			L.Combo = 0
		}
	}
	if r.Stats.BarrierRegen {
		L.barrierRegenT += dt
		if L.barrierRegenT > 20 {
			L.barrierRegenT = 0
			if L.Barrier < 3 {
				L.Barrier++
			}
		}
	}
	L.updateAbility(dt, in)
	L.updatePassives(dt)
	L.abilFlash = math.Max(0, L.abilFlash-dt*2)
	L.stun -= dt
	L.hurtFlash = math.Max(0, L.hurtFlash-dt*2)
	L.paddleBounce = math.Max(0, L.paddleBounce-dt*6)

	// paddle
	tw := L.paddleTarget()
	L.PaddleW += (tw - L.PaddleW) * math.Min(1, 10*dt)
	if L.stun <= 0 {
		target := L.PaddleX
		if in.UseKeys {
			sp := 30.0
			if in.Fast {
				sp = 46
			}
			target += in.KeyDir * sp * dt
		} else {
			target = g.cam.ScreenToFloorX(in.MouseX, PaddleZ)
		}
		if math.IsNaN(target) || math.IsInf(target, 0) {
			target = L.PaddleX
		}
		half := L.PaddleW / 2
		L.PaddleX = math.Max(-HalfW+half, math.Min(HalfW-half, target))
	}
	for _, b := range L.Balls {
		if b.Stuck {
			half := L.PaddleW/2 - 0.2
			b.Off = math.Max(-half, math.Min(half, b.Off))
			b.R = L.ballR()
			b.X, b.Z = L.PaddleX+b.Off, PaddleZ+PaddleD/2+b.R+0.02
		}
	}
	if !L.Cleared {
		if in.Fire {
			if !L.release() && L.Timers[PLaser] > 0 {
				L.fireLaser()
			}
		}
		// lasers
		interval := r.Stats.LaserInterval
		if L.Timers[PLaser] > 0 && (interval == 0 || interval > 0.45) {
			interval = 0.45
		}
		if interval > 0 && !L.anyStuck() {
			L.laserT -= dt
			if L.laserT <= 0 {
				L.laserT = interval
				L.fireLaser()
			}
		}
	}

	// balls
	sp := L.speed()
	for _, b := range L.Balls {
		if b.Stuck {
			continue
		}
		if b.Phantom > 0 {
			b.Phantom -= dt
			if b.Phantom <= 0 {
				b.Z = -100 // fade out
				continue
			}
		}
		b.R += (L.ballR() - b.R) * math.Min(1, dt*6)
		if L.titanT > 0 && b.PierceLeft < 2 {
			b.PierceLeft = 2
		}
		s := sp
		if r.Stats.Momentum {
			s *= 1 + 0.04*math.Min(15, float64(b.Momentum))
		}
		cur := math.Hypot(b.VX, b.VZ)
		if cur > 0 {
			b.VX, b.VZ = b.VX/cur*s, b.VZ/cur*s
		}
		steps := int(math.Ceil(s * dt / (b.R * 0.45)))
		h := dt / float64(steps)
		for k := 0; k < steps && !b.Stuck; k++ {
			b.X += b.VX * h
			b.Z += b.VZ * h
			L.collideWalls(b)
			L.collideBricks(b)
			L.collidePaddle(b)
			L.collideRocks(b)
			if L.Barrier > 0 && b.VZ < 0 && b.Z-b.R <= BarrierZ && b.Z > BarrierZ-1 {
				b.Z = BarrierZ + b.R
				b.VZ = math.Abs(b.VZ)
				L.Barrier--
				g.snd.Play("shield")
				L.sparks(V3{b.X, 0.3, BarrierZ}, 16, color.RGBA{170, 140, 255, 255}, 6)
				L.barrierSaved(b.X)
			}
		}
		b.Trail = append(b.Trail, [2]float64{b.X, b.Z})
		if len(b.Trail) > 12 {
			b.Trail = b.Trail[1:]
		}
		if L.Timers[PFire] > 0 && g.settings.Particles > 0 {
			L.Parts = append(L.Parts, &Particle{P: V3{b.X, b.R, b.Z}, V: V3{(L.rng.Float64() - 0.5) * 1.5, 1.5 + L.rng.Float64()*2, (L.rng.Float64() - 0.5) * 1.5},
				Life: 0.35, Max: 0.35, Size: 0.35, Col: color.RGBA{255, uint8(110 + L.rng.Intn(120)), 30, 255}, Kind: pSpark})
		}
	}
	alive := L.Balls[:0]
	lostReal := false
	for _, b := range L.Balls {
		if b.Z > LoseZ {
			alive = append(alive, b)
		} else if b.Phantom <= 0 {
			lostReal = true
		}
	}
	L.Balls = alive
	_ = lostReal
	if !L.Cleared && L.realBalls() == 0 {
		L.loseBall()
		if g.state != StLevel {
			return
		}
	}

	L.updateBricks(dt)
	if L.Boss != nil {
		L.updateBoss(dt)
	}
	L.updateDrops(dt)
	L.updateEffects(dt)

	if !L.Cleared {
		done := false
		switch {
		case L.Boss != nil:
			done = L.Boss.Dead && L.Boss.deathT > 2.2
		case L.Data.Objective == levelgen.ObjCrystals:
			done = L.Crystals <= 0
		default:
			done = L.Left <= 0
		}
		if done {
			L.clear()
		}
	} else {
		L.clearT += dt
	}
}

func (L *Level) realBalls() int {
	n := 0
	for _, b := range L.Balls {
		if b.Phantom <= 0 {
			n++
		}
	}
	return n
}

func (L *Level) anyStuck() bool {
	for _, b := range L.Balls {
		if b.Stuck {
			return true
		}
	}
	return false
}

func (L *Level) release() bool {
	r := L.g.run
	if L.paddleGone > 0 {
		return false // wait for the paddle to reform
	}
	sp := L.speed()
	released := false
	for _, b := range L.Balls {
		if !b.Stuck {
			continue
		}
		t := b.Off / (L.PaddleW / 2)
		ang := t*50*math.Pi/180 + (L.rng.Float64()-0.5)*0.1
		b.VX, b.VZ = sp*math.Sin(ang), sp*math.Cos(ang)
		b.Stuck = false
		if r.Stats.OpeningCrit > 0 {
			b.Charged = true
		}
		released = true
	}
	if released {
		L.g.snd.Play("launch")
		// first launch of a serve: extra orbs
		if L.T < 0.5 || L.realBalls() == 1 {
			for i := 0; i < r.Stats.ExtraBalls; i++ {
				a := (float64(i) + 1) * 0.35 * float64(1-2*(i%2))
				L.Balls = append(L.Balls, &Ball{X: L.PaddleX, Z: PaddleZ + 1, R: L.ballR(), VX: sp * math.Sin(a), VZ: sp * math.Cos(a), PierceLeft: r.Stats.Pierce})
			}
			if r.Stats.PrismOrb {
				L.Balls = append(L.Balls, &Ball{X: L.PaddleX, Z: PaddleZ + 1, R: L.ballR(), VX: sp * math.Sin(-0.5), VZ: sp * math.Cos(-0.5), Phantom: 8, PierceLeft: 2})
			}
		}
	}
	return released
}

func (L *Level) fireLaser() {
	half := L.PaddleW / 2
	pierce, mult := 0, 1.0
	if L.cls.overclock {
		L.cls.overclock = false
		L.cls.stillT = 0
		pierce, mult = 3, 2
		L.g.snd.PlayVol("zap", 0.6)
	}
	L.Bolts = append(L.Bolts, &Bolt{X: L.PaddleX - half + 0.4, Z: PaddleZ + 0.5, Pierce: pierce, Mult: mult}, &Bolt{X: L.PaddleX + half - 0.4, Z: PaddleZ + 0.5, Pierce: pierce, Mult: mult})
	L.g.snd.Play("laser")
}

func fixAngle(b *Ball) {
	sp := math.Hypot(b.VX, b.VZ)
	if sp == 0 {
		return
	}
	if math.Abs(b.VZ) < 0.3*sp {
		sz := 1.0
		if b.VZ < 0 {
			sz = -1
		}
		b.VZ = sz * 0.3 * sp
		sx := 1.0
		if b.VX < 0 {
			sx = -1
		}
		b.VX = sx * math.Sqrt(sp*sp-b.VZ*b.VZ)
	}
}

func (L *Level) collideWalls(b *Ball) {
	hit := false
	// angled / shaped arena walls
	for i := 0; i+1 < len(L.Walls); i++ {
		a, c := L.Walls[i], L.Walls[i+1]
		ex, ez := c.X-a.X, c.Z-a.Z
		l2 := ex*ex + ez*ez
		t := math.Max(0, math.Min(1, ((b.X-a.X)*ex+(b.Z-a.Z)*ez)/l2))
		px, pz := a.X+ex*t, a.Z+ez*t
		dx, dz := b.X-px, b.Z-pz
		d := math.Hypot(dx, dz)
		if d >= b.R {
			continue
		}
		// inward normal (towards the arena interior)
		nx, nz := -ez, ex
		l := math.Hypot(nx, nz)
		nx, nz = nx/l, nz/l
		if nx*(0-a.X)+nz*(10-a.Z) < 0 {
			nx, nz = -nx, -nz
		}
		if d > 1e-6 && (t > 0 && t < 1) == false {
			nx, nz = dx/d, dz/d // corner: push straight out
		}
		pen := b.R - (dx*nx + dz*nz)
		if pen <= 0 {
			continue
		}
		b.X += nx * pen
		b.Z += nz * pen
		dot := b.VX*nx + b.VZ*nz
		if dot < 0 {
			b.VX -= 2 * dot * nx
			b.VZ -= 2 * dot * nz
		}
		hit = true
	}
	if b.X-b.R < -HalfW {
		b.X = -HalfW + b.R
		b.VX = math.Abs(b.VX)
		hit = true
	}
	if b.X+b.R > HalfW {
		b.X = HalfW - b.R
		b.VX = -math.Abs(b.VX)
		hit = true
	}
	if b.Z+b.R > FarZ {
		b.Z = FarZ - b.R
		b.VZ = -math.Abs(b.VZ)
		hit = true
	}
	if hit {
		fixAngle(b)
		b.Rico++
		b.PierceLeft = max(b.PierceLeft, L.g.run.Stats.Pierce)
		L.g.snd.Play("wall")
	}
}

func (L *Level) collideBricks(b *Ball) {
	c0 := int(math.Floor((b.X - b.R + HalfW) / CellW))
	c1 := int(math.Floor((b.X + b.R + HalfW) / CellW))
	r0 := int(math.Floor((GridTopZ - (b.Z + b.R)) / CellD))
	r1 := int(math.Floor((GridTopZ - (b.Z - b.R)) / CellD))
	var best *Brick
	bestPen := -1.0
	var nx, nz float64
	var hits []*Brick
	for rr := max(r0, 0); rr <= min(r1, levelgen.MaxRows-1); rr++ {
		for cc := max(c0, 0); cc <= min(c1, levelgen.Cols-1); cc++ {
			br := L.Grid[rr][cc]
			if br == nil || !br.Alive {
				continue
			}
			x0, x1 := br.X()-CellW/2, br.X()+CellW/2
			z0, z1 := br.Z()-CellD/2, br.Z()+CellD/2
			px := math.Max(x0, math.Min(b.X, x1))
			pz := math.Max(z0, math.Min(b.Z, z1))
			dx, dz := b.X-px, b.Z-pz
			d2 := dx*dx + dz*dz
			if d2 >= b.R*b.R {
				continue
			}
			hits = append(hits, br)
			d := math.Sqrt(d2)
			var ux, uz, pen float64
			if d < 1e-6 {
				v := math.Hypot(b.VX, b.VZ)
				ux, uz, pen = -b.VX/v, -b.VZ/v, b.R
			} else {
				ux, uz, pen = dx/d, dz/d, b.R-d
			}
			if pen > bestPen {
				best, bestPen, nx, nz = br, pen, ux, uz
			}
		}
	}
	if best == nil {
		return
	}
	reflect := true
	for _, br := range hits {
		if !br.Alive {
			continue
		}
		destroyed, solid := L.ballHit(b, br)
		if br == best {
			if !solid {
				reflect = false
			} else if destroyed && b.PierceLeft > 0 {
				b.PierceLeft--
				reflect = false
			} else if destroyed && b.Phantom > 0 {
				reflect = false
			}
		}
	}
	if reflect {
		b.X += nx * bestPen
		b.Z += nz * bestPen
		dot := b.VX*nx + b.VZ*nz
		if dot < 0 {
			b.VX -= 2 * dot * nx
			b.VZ -= 2 * dot * nz
		}
		if best.Kind == levelgen.Steel {
			a := (L.rng.Float64() - 0.5) * 0.08
			ca, sa := math.Cos(a), math.Sin(a)
			b.VX, b.VZ = b.VX*ca-b.VZ*sa, b.VX*sa+b.VZ*ca
		}
		fixAngle(b)
	}
}

func (L *Level) collidePaddle(b *Ball) {
	if b.VZ >= 0 {
		return
	}
	half := L.PaddleW / 2
	x0, x1 := L.PaddleX-half, L.PaddleX+half
	z0, z1 := PaddleZ-PaddleD/2, PaddleZ+PaddleD/2
	px := math.Max(x0, math.Min(b.X, x1))
	pz := math.Max(z0, math.Min(b.Z, z1))
	dx, dz := b.X-px, b.Z-pz
	if dx*dx+dz*dz >= b.R*b.R || b.Z < PaddleZ-PaddleD*0.2 {
		return
	}
	r := L.g.run
	sp := math.Hypot(b.VX, b.VZ)
	t := math.Max(-1, math.Min(1, (b.X-L.PaddleX)/half))
	ang := t * 62 * math.Pi / 180
	if a, ok := L.lastBrickAim(b); ok {
		// the last few bricks: bounces lean toward them so a chamber
		// never ends in a long hunt for one brick
		ang += (a - ang) * 0.55
	}
	b.VX, b.VZ = sp*math.Sin(ang), sp*math.Cos(ang)
	b.Z = z1 + b.R
	L.passiveOnPaddle(b, t)
	b.PierceLeft = r.Stats.Pierce
	b.Momentum = 0
	b.Rico = 0
	L.paddleBounce = 1
	if r.Stats.StaticPaddle {
		L.zapNearest(b.X, b.Z, math.Max(1, r.Stats.Damage*(1+r.Stats.ChainMult)))
	}
	if r.Stats.HeavyPaddle {
		b.Charged = true
	}
	if r.Stats.GoldenPaddle {
		L.gainGold(1, b.X, b.Z, false)
	}
	if L.Timers[PGlue] > 0 && b.Phantom <= 0 {
		b.Stuck = true
		b.Off = b.X - L.PaddleX
	}
	if r.Stats.SplitChance > 0 && L.rng.Float64() < r.Stats.SplitChance && len(L.Balls) < 14 {
		a := ang + 0.45*float64(1-2*L.rng.Intn(2))
		L.Balls = append(L.Balls, &Ball{X: b.X, Z: b.Z, R: b.R, VX: sp * math.Sin(a), VZ: sp * math.Cos(a), PierceLeft: r.Stats.Pierce})
		L.g.snd.PlayKeyed("split")
	}
	L.g.snd.PlayKeyed("pad")
}

func (L *Level) collideRocks(b *Ball) {
	for i := 0; i < len(L.Rocks); i++ {
		rk := L.Rocks[i]
		if rk.Warn > 0 {
			continue
		}
		dx, dz := b.X-rk.X, b.Z-rk.Z
		d := math.Hypot(dx, dz)
		if d < b.R+0.6 && d > 0 {
			nx, nz := dx/d, dz/d
			dot := b.VX*nx + b.VZ*nz
			if dot < 0 {
				b.VX -= 2 * dot * nx
				b.VZ -= 2 * dot * nz
			}
			L.debris(V3{rk.X, 0.5, rk.Z}, 14, color.RGBA{120, 110, 100, 255})
			L.g.snd.Play("steel")
			L.gainGold(1, rk.X, rk.Z, true)
			L.Rocks = append(L.Rocks[:i], L.Rocks[i+1:]...)
			return
		}
	}
}

// ---------------------------------------------------------------- damage

type Hit struct {
	Src                                string // what dealt it, for the run recap ("" = the orb)
	Dmg                                float64
	Crit, Void, Blast, Elemental, Fire bool
	Silent                             bool
}

func (L *Level) shielded(b *Brick) bool {
	if b.Kind == levelgen.Shield {
		return false
	}
	for dr := -1; dr <= 1; dr++ {
		for dc := -1; dc <= 1; dc++ {
			r, c := b.R+dr, b.C+dc
			if r < 0 || r >= levelgen.MaxRows || c < 0 || c >= levelgen.Cols || (dr == 0 && dc == 0) {
				continue
			}
			if n := L.Grid[r][c]; n != nil && n.Alive && n.Kind == levelgen.Shield {
				return true
			}
		}
	}
	return false
}

// damage applies a hit and reports (destroyed, solid). solid=false means the
// orb should pass through (fire orb).
func (L *Level) damage(b *Brick, h Hit) (bool, bool) {
	g := L.g
	r := g.run
	b.Flash = 1
	b.SinceHit = 0
	if b.Core && L.Boss != nil {
		if L.Boss.Dead {
			return false, true
		}
		dmg := h.Dmg
		if h.Crit {
			L.text(V3{b.X(), 2, b.Z()}, fmt.Sprintf("%.0f!", dmg), color.RGBA{255, 90, 90, 255}, true)
		}
		r.recordDmg(h.Src, math.Min(dmg, math.Max(0, L.Boss.HP)))
		L.Boss.HP -= dmg
		L.Boss.Hurt = 1
		L.Boss.coreHits++
		g.snd.Play("bosshit")
		L.sparks(V3{b.X(), 1.2, b.Z()}, 16, color.RGBA{255, 220, 120, 255}, 7)
		L.Shake = math.Max(L.Shake, 0.25)
		if L.Boss.HP <= 0 {
			L.killBoss()
		} else if L.Boss.coreHits >= 4 {
			L.moveCore()
		}
		return false, true
	}
	switch {
	case b.Kind == levelgen.Steel:
		if !h.Silent {
			g.snd.Play("steel")
		}
		return false, true
	case L.shielded(b) && !h.Void:
		b.shieldFlash = 1
		if !h.Silent {
			g.snd.Play("shieldhit")
		}
		return false, true
	case b.Kind == levelgen.Armored && !(h.Crit || h.Void || h.Blast || h.Elemental):
		b.Hits += 1 + r.Stats.SunderMult
		if !h.Silent {
			g.snd.Play("steel")
		}
		if b.Hits >= 6 {
			b.Kind = levelgen.Basic
			b.Col = lighten(b.Col, 0.25)
			L.text(V3{b.X(), 1.5, b.Z()}, "ARMOR CRACKS", color.RGBA{220, 220, 230, 255}, false)
		}
		return false, true
	}
	dmg := h.Dmg
	if b.Frozen > 0 {
		switch {
		case r.Stats.AbsoluteZero:
			dmg *= 3
		case r.Stats.FrozenBonus:
			dmg *= 2
		default:
			dmg *= 1.5
		}
	}
	if b.Burn > 0 && r.Stats.PyroAsh {
		dmg *= 2
	}
	if h.Void && r.Stats.VoidDouble {
		dmg *= 3
	}
	r.recordDmg(h.Src, math.Min(dmg, math.Max(0, b.HP)))
	b.HP -= dmg
	if b.HP > 0 && r.Stats.ExecuteBelow > 0 && !b.Core && b.HP <= b.MaxHP*r.Stats.ExecuteBelow {
		b.HP = 0
		L.text(V3{b.X(), 1.4, b.Z()}, "EXECUTE", color.RGBA{255, 90, 90, 255}, false)
	}
	if b.HP > 0 {
		if !h.Silent {
			g.snd.BrickHit()
		}
		return false, !h.Fire
	}
	L.destroy(b, h)
	return true, !h.Fire
}

func (L *Level) ballHit(ball *Ball, b *Brick) (bool, bool) {
	g := L.g
	r := g.run
	s := &r.Stats
	L.impacts++
	charged := ball.Charged
	crit := L.rng.Float64() < s.CritChance || charged
	ball.Charged = false
	wasFrozen := b.Frozen > 0
	dmg := s.Damage * s.DmgMult
	dmg *= 1 + math.Min(L.comboCap(), float64(L.Combo)*s.ComboDmg)
	if s.Momentum {
		dmg *= 1 + 0.04*math.Min(15, float64(ball.Momentum))
	}
	if s.BloodMoon && r.HP < s.MaxHP/2 {
		dmg *= 1.75
	}
	if s.Berserker && r.HP < s.MaxHP/2 {
		dmg *= 1.5
	}
	if s.Ricochet > 0 {
		dmg *= 1 + s.Ricochet*float64(min(ball.Rico, 12))
	}
	if s.BossDmg > 0 && (L.Boss != nil || b.Kind == levelgen.Elite) {
		dmg *= 1 + s.BossDmg
	}
	if extra := L.realBalls() - 1; s.HiveDmg > 0 && extra > 0 {
		dmg *= 1 + math.Min(s.HiveDmg*6, s.HiveDmg*float64(extra))
	}
	if s.LoneDmg > 0 && L.realBalls() == 1 {
		dmg *= 1 + s.LoneDmg
	}
	if s.Avarice > 0 {
		dmg *= 1 + math.Min(s.Avarice, float64(r.Gold)/1000)
	}
	if s.BattleHymn > 0 && L.T < s.BattleHymn {
		dmg *= 2
	}
	dmg *= 1 + L.rushBonus
	if ball.Phantom > 0 && s.VoidPrism {
		dmg *= 2
	}
	if charged && s.Haymaker {
		dmg *= 2.5
		L.Combo += 3
		L.Rings = append(L.Rings, &Ring{X: b.X(), Z: b.Z(), MaxR: 2.2, Life: 0.3, Max: 0.3, Col: color.RGBA{255, 150, 90, 255}})
	}
	dmg = L.passiveDamage(b, dmg)
	if crit {
		dmg *= s.CritMult
	}
	fire := L.Timers[PFire] > 0
	h := Hit{Dmg: dmg, Crit: crit, Void: L.rng.Float64() < s.VoidChance || (ball.Phantom > 0 && s.VoidPrism), Fire: fire && b.Kind != levelgen.Steel, Elemental: fire}
	if fire {
		h.Dmg *= 2
	}
	bx, bz := b.X(), b.Z()
	times := 1
	if (s.EchoEvery > 0 && L.impacts%s.EchoEvery == 0) || (s.EchoChance > 0 && L.rng.Float64() < s.EchoChance) {
		times = 2
		L.text(V3{bx, 1.6, bz}, "ECHO", color.RGBA{200, 200, 255, 255}, false)
		if s.Thunderclap {
			L.skyStrike(bx, bz, dmg*0.8, color.RGBA{170, 220, 255, 255})
			L.lightning(bx, bz, 3, math.Max(1, dmg*0.5), b, false)
		}
	}
	var destroyed, solid bool
	for i := 0; i < times; i++ {
		if !b.Alive {
			break
		}
		destroyed, solid = L.damage(b, h)
	}
	if s.Crescendo > 0 {
		note := 0.0
		if L.Combo >= 2 && b.Alive {
			note = dmg * math.Min(s.Crescendo*float64(L.Combo), s.Crescendo*20)
			L.damage(b, Hit{Src: "Crescendo", Dmg: note, Silent: true, Void: h.Void, Crit: crit})
		}
		L.dmgNumber(bx, bz, dmg*float64(times)+note, crit)
	}
	if crit && L.T-L.critStopT > 0.35 && !g.settings.FastAnim {
		L.critStopT = L.T
		L.hitStop = math.Max(L.hitStop, 0.045)
		L.Shake = math.Max(L.Shake, 0.18)
	}
	L.passiveOnHit(ball, b, dmg)
	if L.midasT > 0 && L.rng.Float64() < 0.6 {
		L.gainGold(1, b.X(), b.Z(), true)
	}
	if b.Kind == levelgen.Steel {
		return false, true
	}
	if crit {
		L.g.snd.Play("crit")
		if s.CritHeal > 0 {
			r.HP = math.Min(s.MaxHP, r.HP+s.CritHeal)
		}
		if s.CritCoin > 0 && L.rng.Float64() < s.CritCoin {
			L.gainGold(1, bx, bz, true)
		}
		if s.Hawkstrike && b.R > 0 {
			if n := L.Grid[b.R-1][b.C]; n != nil && n.Alive && n != b {
				L.damage(n, Hit{Src: "Hawkstrike", Dmg: dmg * 0.75, Crit: true, Silent: true})
				L.sparks(V3{n.X(), 1, n.Z()}, 8, color.RGBA{255, 200, 120, 255}, 6)
			}
		}
	}
	if s.FreezeSpread && wasFrozen {
		for _, n := range L.neighbours(b.R, b.C) {
			n.Frozen = 4 + s.FreezeDur
		}
	}
	if s.DragonHeart && b.Alive && b.Burn > 0 {
		for _, n := range L.neighbours(b.R, b.C) {
			n.Burn = 3.5
		}
	}
	if s.PlasmaStorm && b.Burn > 0 && L.rng.Float64() < 0.35 {
		L.lightning(bx, bz, 2, math.Max(1, dmg*0.5), b, false)
	}
	if s.Cataclysm && L.impacts%12 == 0 {
		L.meteor()
	}
	if crit && !b.Core {
		L.text(V3{bx, 1.4, bz}, "CRIT", color.RGBA{255, 90, 90, 255}, false)
	}
	if crit && s.CritPierce && destroyed {
		solid = false
	}
	ball.Momentum++
	// combo
	L.Combo++
	L.comboT = 3 + s.ComboTime
	if L.Combo > r.MaxCombo {
		r.MaxCombo = L.Combo
	}
	L.peakCombo = max(L.peakCombo, L.Combo)
	if L.Combo%10 == 0 {
		L.text(V3{bx, 2.2, bz}, fmt.Sprintf("COMBO ×%d", L.Combo), color.RGBA{255, 230, 120, 255}, true)
		L.g.snd.ComboMilestone(L.Combo)
	}
	// procs
	if b.Alive && (L.rng.Float64() < s.BurnChance || fire || L.classIgnite()) {
		if b.Burn <= 0 {
			L.text(V3{bx, 1.2, bz}, "BURN", color.RGBA{255, 140, 40, 255}, false)
		}
		b.Burn = 3.5
	}
	if b.Alive && L.rng.Float64() < s.FreezeChance {
		b.Frozen = 4 + s.FreezeDur
		L.g.snd.Play("freeze")
	}
	if L.rng.Float64() < s.ChainChance || (crit && s.ThunderCrown) {
		L.lightning(bx, bz, s.ChainCount+1, math.Max(1, dmg*0.6), b, s.StormCaller || s.ChainRecast)
	}
	if s.MeteorHeart || L.rng.Float64() < s.ExplodeChance {
		L.explode(bx, bz, s.ExplodeRadius, math.Max(1.5, dmg), false)
	}
	if crit && s.EagleEye {
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			rr, cc := b.R+d[0], b.C+d[1]
			if rr >= 0 && rr < levelgen.MaxRows && cc >= 0 && cc < levelgen.Cols {
				if n := L.Grid[rr][cc]; n != nil && n.Alive {
					L.damage(n, Hit{Src: "Eagle Eye", Dmg: dmg * 0.5, Silent: true})
				}
			}
		}
	}
	if s.MoltenEvery > 0 && L.impacts%s.MoltenEvery == 0 {
		L.text(V3{bx, 2.5, bz}, "MOLTEN CORE!", color.RGBA{255, 140, 40, 255}, true)
		L.explode(bx, bz, s.ExplodeRadius+1.6, dmg*3, true)
	}
	if s.StaticEvery > 0 && L.impacts%s.StaticEvery == 0 {
		if t := L.top(L.randomBrick()); t != nil {
			L.Arcs = append(L.Arcs, &Arc{Pts: jag(V3{t.X(), 12, t.Z()}, V3{t.X(), 0.8, t.Z()}, L.rng), Life: 0.35, Col: color.RGBA{150, 210, 255, 255}})
			L.damage(t, Hit{Src: "Static Field", Dmg: 3, Elemental: true})
			L.g.snd.Play("zap")
		}
	}
	if !destroyed && b.Alive {
		L.sparks(V3{ball.X, 0.6, ball.Z}, 5, lighten(b.Col, 0.5), 4)
	}
	return destroyed, solid
}

func (L *Level) randomBrick() *Brick {
	var c []*Brick
	for _, b := range L.Bricks {
		if b.Alive && b.Layer == 0 && b.Kind != levelgen.Steel && !b.BossPart {
			c = append(c, b)
		}
	}
	if len(c) == 0 {
		return nil
	}
	return c[L.rng.Intn(len(c))]
}

func (L *Level) destroy(b *Brick, h Hit) {
	if !b.Alive {
		return
	}
	L.lastBrkX, L.lastBrkZ = b.X(), b.Z()
	g := L.g
	r := g.run
	s := &r.Stats
	b.Alive = false
	L.passiveOnBreak()
	L.removeFromColumn(b)
	if bo := L.Boss; b.BossPart && !b.Core && bo != nil && !bo.Dead {
		// every slab of armour torn off wounds the Guardian a little
		bo.HP -= bo.MaxHP * 0.02
		bo.Hurt = math.Max(bo.Hurt, 0.4)
		if bo.HP <= 0 {
			L.killBoss()
		}
	}
	if b.Kind.Required() && !b.BossPart {
		L.Left--
	}
	if b.Kind == levelgen.Crystal {
		L.Crystals--
	}
	r.Bricks++
	x, z := b.X(), b.Z()
	pts := int(10 * b.MaxHP)
	if L.Frenzy > 0 {
		pts *= 2
	}
	r.Score += pts * (1 + L.Combo/10)
	L.debris(V3{x, 0.5, z}, 10, b.Col)
	L.breakPop(b, x, z)
	L.g.snd.BrickBreak(L.Combo, b.MaxHP >= 2 || b.Kind == levelgen.Elite)

	// gold
	coinP := 0.32 + s.CoinChance
	if s.Philosopher {
		coinP = 0.6 + s.CoinChance
	}
	if L.rng.Float64() < coinP {
		L.gainGold(1+L.rng.Intn(2), x, z, true)
	}
	if s.GoldPerBrick {
		L.gainGold(1, x, z, false)
	}
	// lifesteal / hydra
	if s.Lifesteal > 0 {
		L.killsSinceHeal++
		if L.killsSinceHeal >= s.Lifesteal {
			L.killsSinceHeal = 0
			r.HP = math.Min(s.MaxHP, r.HP+1)
		}
	}
	if s.Hydra {
		L.hydraCount++
		need := 30
		if r.HasFusion("brood") {
			need = 15
		}
		if L.hydraCount >= need && len(L.Balls) < 14 {
			L.hydraCount = 0
			sp := L.speed()
			a := (L.rng.Float64() - 0.5) * 1.2
			L.Balls = append(L.Balls, &Ball{X: x, Z: z - 1, R: L.ballR(), VX: sp * math.Sin(a), VZ: -sp * math.Cos(a), PierceLeft: s.Pierce})
			L.text(V3{x, 2, z}, "HYDRA!", color.RGBA{120, 255, 160, 255}, true)
		}
	}
	// power-up drop
	dropP := 0.055 * s.Luck
	if L.hasOmen("bountiful") {
		dropP *= 2
	}
	if b.MaxHP >= 2 {
		dropP *= 1.4
	}
	switch b.Kind {
	case levelgen.Elite:
		L.dropPower(x, z, true)
		L.gainGold(10, x, z, true)
		r.Elites++
		L.text(V3{x, 2, z}, "ELITE DOWN", color.RGBA{220, 160, 255, 255}, true)
		L.Shake = math.Max(L.Shake, 0.5)
	case levelgen.Basic, levelgen.Regen, levelgen.Moving, levelgen.Armored, levelgen.Split:
		if L.rng.Float64() < dropP && len(L.Drops) < 5 {
			L.dropPower(x, z, false)
		}
	}
	L.maybeDropLife(x, z)
	// kind effects
	switch b.Kind {
	case levelgen.Explosive:
		L.explode(x, z, 1.6+s.ExplodeRadius*0.4, 4, false)
	case levelgen.Chain:
		L.lightning(x, z, 4, 2, b, s.StormCaller)
	case levelgen.Split:
		n := 0
		for _, d := range [][2]int{{0, -1}, {0, 1}, {1, 0}, {-1, 0}} {
			rr, cc := b.R+d[0], b.C+d[1]
			if n < 2 && L.cellFree(rr, cc) {
				nb := L.addBrick(rr, cc, levelgen.Basic, 1, lighten(b.Col, 0.2))
				nb.Appear = 0
				n++
			}
		}
	case levelgen.Multiplier:
		L.Frenzy = 7
		L.Combo += 5
		L.text(V3{x, 2.4, z}, "FRENZY! ×2", color.RGBA{255, 120, 220, 255}, true)
		g.snd.PlayKeyed("frenzy")
	case levelgen.Crystal:
		n := 5
		if s.Philosopher {
			n = 15
		}
		for i := 0; i < n; i++ {
			L.gainGold(1, x, z, true)
		}
		L.sparks(V3{x, 1, z}, 20, crystalGlow, 8)
		g.snd.PlayKeyed("crystal")
	case levelgen.Mystery:
		L.mystery(x, z, b)
	}
	if b.Burn > 0 && s.Wildfire {
		for _, d := range [][2]int{{0, -1}, {0, 1}, {1, 0}, {-1, 0}} {
			rr, cc := b.R+d[0], b.C+d[1]
			if rr >= 0 && rr < levelgen.MaxRows && cc >= 0 && cc < levelgen.Cols {
				if n := L.Grid[rr][cc]; n != nil && n.Alive && n.Kind != levelgen.Steel {
					n.Burn = 3.5
				}
			}
		}
		for _, a := range r.Active {
			if a == "Inferno" {
				L.explode(x, z, 1.2, 1.5, false)
			}
		}
	}
	if s.KillSplash > 0 {
		for _, n := range L.neighbours(b.R, b.C) {
			L.damage(n, Hit{Src: "Shrapnel", Dmg: math.Max(0.5, s.Damage*s.DmgMult*s.KillSplash), Blast: true, Silent: true})
		}
		L.sparks(V3{x, 0.8, z}, 6, lighten(b.Col, 0.3), 7)
	}
	if s.BlackHole {
		L.bhCount++
		if L.bhCount >= 40 {
			L.bhCount = 0
			L.blackHole(x, z)
		}
	}
	if s.Volcano && b.Burn > 0 && L.eruptDepth < 3 {
		L.eruptDepth++
		L.explode(x, z, 1.0, math.Max(1, s.Damage), false)
		L.eruptDepth--
	}
	if b.Frozen > 0 && s.Shatter {
		shR := 2.6
		if s.AbsoluteZero {
			shR = 4.2
		}
		L.Rings = append(L.Rings, &Ring{X: x, Z: z, MaxR: shR + 0.4, Life: 0.35, Max: 0.35, Col: color.RGBA{170, 230, 255, 255}})
		for _, n := range L.near(x, z, shR) {
			if n != b {
				L.damage(n, Hit{Src: "Shatter", Dmg: 1.5, Elemental: true, Silent: true})
			}
		}
		g.snd.Play("shatter")
	}
}

var crystalGlow = color.RGBA{120, 220, 255, 255}

func (L *Level) near(x, z, rad float64) []*Brick {
	var out []*Brick
	for _, b := range L.Bricks {
		if b.Alive && math.Hypot(b.X()-x, (b.Z()-z)*1.4) <= rad {
			out = append(out, b)
		}
	}
	return out
}

func (L *Level) explode(x, z, radiusCells, dmg float64, big bool) {
	g := L.g
	st := &g.run.Stats
	dmg *= 1 + st.BlastMult
	rad := radiusCells * CellW * 0.95
	rc := color.RGBA{255, 170, 60, 255}
	if st.CryoCharge {
		rc = color.RGBA{150, 225, 255, 255}
	}
	L.Rings = append(L.Rings, &Ring{X: x, Z: z, MaxR: rad, Life: 0.45, Max: 0.45, Col: rc})
	L.sparks(V3{x, 0.8, z}, 26, color.RGBA{255, 190, 80, 255}, 10)
	if g.settings.Particles > 0 {
		for i := 0; i < 8; i++ {
			L.Parts = append(L.Parts, &Particle{P: V3{x + (L.rng.Float64()-0.5)*rad, 0.6, z + (L.rng.Float64()-0.5)*rad},
				V: V3{0, 1.5 + L.rng.Float64(), 0}, Life: 1.1, Max: 1.1, Size: 1.6 + L.rng.Float64(), Col: color.RGBA{60, 50, 50, 255}, Kind: pSmoke})
		}
	}
	L.Shake = math.Max(L.Shake, 0.35)
	if big {
		L.Shake = 0.8
		L.Flash = 0.5
	}
	g.snd.Play("boom")
	for _, b := range L.near(x, z, rad) {
		if b.Alive && b.Layer <= 1 {
			if st.CryoCharge {
				b.Frozen = 4 + st.FreezeDur
			}
			if st.Volcano {
				b.Burn = 3.5
			}
			L.damage(b, Hit{Src: "Explosions", Dmg: dmg, Blast: true, Silent: true})
		}
	}
}

func jag(a, b V3, rng *rand.Rand) []V3 {
	pts := []V3{a}
	n := 6
	for i := 1; i < n; i++ {
		t := float64(i) / float64(n)
		p := V3{lerp(a.X, b.X, t), lerp(a.Y, b.Y, t), lerp(a.Z, b.Z, t)}
		p.X += (rng.Float64() - 0.5) * 0.7
		p.Y += (rng.Float64() - 0.5) * 0.5
		p.Z += (rng.Float64() - 0.5) * 0.7
		pts = append(pts, p)
	}
	return append(pts, b)
}

func (L *Level) lightning(x, z float64, count int, dmg float64, from *Brick, recast bool) {
	L.lightningRaw(x, z, count, dmg*(1+L.g.run.Stats.ChainMult), from, recast)
}

func (L *Level) lightningRaw(x, z float64, count int, dmg float64, from *Brick, recast bool) {
	plasma := L.g.run.Stats.PlasmaStorm
	var cands []*Brick
	for _, b := range L.Bricks {
		if b.Alive && b != from && b.Layer == 0 && b.Kind != levelgen.Steel && math.Hypot(b.X()-x, b.Z()-z) < 9 {
			cands = append(cands, b)
		}
	}
	// nearest first
	for i := 0; i < len(cands); i++ {
		for j := i + 1; j < len(cands); j++ {
			di := math.Hypot(cands[i].X()-x, cands[i].Z()-z)
			dj := math.Hypot(cands[j].X()-x, cands[j].Z()-z)
			if dj < di {
				cands[i], cands[j] = cands[j], cands[i]
			}
		}
	}
	if len(cands) > count {
		cands = cands[:count]
	}
	if len(cands) == 0 {
		return
	}
	L.g.snd.Play("zap")
	prev := V3{x, 0.8, z}
	for _, t := range cands {
		p := V3{t.X(), 0.8, t.Z()}
		ac := color.RGBA{150, 210, 255, 255}
		if plasma {
			ac = color.RGBA{255, 120, 220, 255}
			t.Burn = 3.5
		}
		L.Arcs = append(L.Arcs, &Arc{Pts: jag(prev, p, L.rng), Life: 0.3, Col: ac})
		tx, tz := t.X(), t.Z()
		L.damage(t, Hit{Src: "Lightning", Dmg: dmg, Elemental: true, Silent: true})
		if recast && L.rng.Float64() < 0.5 {
			L.lightningRaw(tx, tz, 1, dmg*0.7, t, false)
		}
		prev = p
	}
}

func (L *Level) mystery(x, z float64, b *Brick) {
	r := L.g.run
	switch L.rng.Intn(5) {
	case 0:
		L.dropPower(x, z, true)
		L.text(V3{x, 2, z}, "GIFT!", color.RGBA{255, 230, 120, 255}, true)
	case 1:
		for i := 0; i < 8; i++ {
			L.gainGold(1, x, z, true)
		}
		L.text(V3{x, 2, z}, "GOLD SHOWER!", color.RGBA{255, 210, 70, 255}, true)
	case 2:
		L.explode(x, z, 2, 3, false)
	case 3:
		if len(L.Balls) < 14 {
			sp := L.speed()
			L.Balls = append(L.Balls, &Ball{X: x, Z: z - 1, R: L.ballR(), VX: sp * 0.4, VZ: -sp * 0.9, PierceLeft: r.Stats.Pierce})
			L.text(V3{x, 2, z}, "BONUS ORB!", color.RGBA{180, 255, 200, 255}, true)
		}
	default:
		n := 0
		for _, d := range [][2]int{{0, -1}, {0, 1}, {1, 0}, {-1, 0}, {1, 1}, {1, -1}} {
			rr, cc := b.R+d[0], b.C+d[1]
			if n < 3 && L.cellFree(rr, cc) {
				L.addBrick(rr, cc, levelgen.Basic, 1, L.Biome.Hues[L.rng.Intn(6)])
				n++
			}
		}
		L.text(V3{x, 2, z}, "TRICKED!", color.RGBA{255, 140, 140, 255}, true)
	}
}

func (L *Level) dropPower(x, z float64, goodOnly bool) {
	goodOnly = goodOnly || L.g.run.Stats.Blessed
	total := 0.0
	for _, p := range powerDefs {
		if !goodOnly || p.Good {
			total += p.W
		}
	}
	v := L.rng.Float64() * total
	for i, p := range powerDefs {
		if goodOnly && !p.Good {
			continue
		}
		if v < p.W {
			L.Drops = append(L.Drops, &Drop{X: x, Z: z, P: Power(i)})
			return
		}
		v -= p.W
	}
}

func (L *Level) gainGold(n int, x, z float64, asCoin bool) {
	r := L.g.run
	if asCoin {
		for i := 0; i < n; i++ {
			a := L.rng.Float64() * 2 * math.Pi
			L.Coins = append(L.Coins, &Coin{X: x, Y: 0.8, Z: z, VX: math.Cos(a) * 2, VZ: math.Sin(a)*2 - 1, VY: 4 + L.rng.Float64()*3, Val: 1})
		}
		return
	}
	v := int(math.Round(float64(n) * r.Stats.GoldMult))
	if L.Frenzy > 0 {
		v *= 2
	}
	if r.Cursed {
		v *= 2
	}
	if L.hasOmen("gilded") {
		v *= 2
	}
	v = max(v, 1)
	r.Gold += v
	L.GoldEarned += v
}

func (L *Level) applyPower(p Power) {
	g := L.g
	d := powerDefs[p]
	if p == PLife {
		L.gainLife()
		return
	}
	if d.Good {
		g.snd.Play("pw." + itoa(int(p)) + "." + itoa(g.snd.Key))
	} else {
		g.snd.Play("pwbad." + itoa(int(p)))
	}
	L.text(V3{L.PaddleX, 1.8, PaddleZ + 1}, d.Name, lighten(d.Col, 0.3), true)
	if d.Dur > 0 {
		L.Timers[p] = d.Dur * (1 + g.run.Stats.PowerDur)
	}
	switch p {
	case PExpand:
		L.tempSize = min(2, max(0, L.tempSize)+1)
	case PShrink:
		L.tempSize = -1
	case PMulti:
		L.release()
		var add []*Ball
		for _, b := range L.Balls {
			if len(L.Balls)+len(add) >= 14 {
				break
			}
			for _, a := range []float64{-0.4, 0.4} {
				ca, sa := math.Cos(a), math.Sin(a)
				nb := &Ball{X: b.X, Z: b.Z, R: b.R, VX: b.VX*ca - b.VZ*sa, VZ: b.VX*sa + b.VZ*ca, PierceLeft: b.PierceLeft, Phantom: b.Phantom}
				fixAngle(nb)
				add = append(add, nb)
			}
		}
		L.Balls = append(L.Balls, add...)
	case PSlow:
		L.Timers[PHaste] = 0
	case PHaste:
		L.Timers[PSlow] = 0
	case PBarrier:
		L.Barrier = min(5, L.Barrier+2)
	}
}

func (L *Level) loseBall() {
	g := L.g
	r := g.run
	L.lostOrbs++
	if dmg := r.Stats.BallLossDmg; dmg > 0 {
		// some curses and relics make a drop sting twice
		r.HP = math.Max(1, r.HP-dmg)
		L.text(V3{L.PaddleX, 1.2, PaddleZ + 1.4}, fmt.Sprintf("−%.0f", dmg), color.RGBA{255, 100, 100, 255}, false)
	}
	L.loseLife("ORB LOST")
	if g.state != StLevel || L.gameOverT > 0 {
		return
	}
	if r.Stats.DropBarrier > 0 {
		L.Barrier = min(5, L.Barrier+r.Stats.DropBarrier)
	}
	if r.Stats.LifeHeal > 0 {
		r.HP = math.Min(r.Stats.MaxHP, r.HP+r.Stats.LifeHeal)
	}
	L.Timers = [numPowers]float64{}
	L.tempSize = 0
	L.Drops = nil
	L.Bolts = nil
	L.Combo = 0
	L.passiveOnLoss()
	L.spawnServe()
}

func (L *Level) hurt(dmg float64, label string) {
	g := L.g
	r := g.run
	r.HP -= dmg
	L.hurtFlash = 1
	L.Shake = math.Max(L.Shake, 0.6)
	L.text(V3{L.PaddleX, 1.5, PaddleZ + 2}, fmt.Sprintf("%s  −%.0f", label, dmg), color.RGBA{255, 100, 100, 255}, true)
	if r.HP <= 0 {
		if r.Stats.Phoenix && !r.PhoenixUsed {
			r.PhoenixUsed = true
			r.HP = 50
			L.text(V3{L.PaddleX, 2.5, PaddleZ + 3}, "PHOENIX REBIRTH!", color.RGBA{255, 160, 60, 255}, true)
			g.snd.PlayKeyed("heal")
			g.snd.Duck(0.5, 1.5)
			return
		}
		// out of health: it costs a life, and the health comes back
		r.HP = r.Stats.MaxHP
		L.loseLife("HEALTH LOST")
		if L.gameOverT <= 0 {
			L.Balls = nil
			L.spawnServe()
		}
		return
	}
	g.react(reactHurt)
	g.snd.Play("lose")
}

func (L *Level) clear() {
	g := L.g
	r := g.run
	L.Cleared = true
	L.clearT = 0
	if L == g.level && r.SeedCode != "MENU" {
		g.unlock("ACH_FIRST_CHAMBER")
		L.omenCleared()
		if L.Boss == nil {
			L.startFinale()
		}
	}
	reward := 12 + r.Act*4
	if L.Node == NElite {
		reward += 25
		if r.is("gauntlet") {
			reward *= 2
		}
	}
	if L.Node == NBoss {
		reward += 60
	}
	L.gainGold(reward, 0, 0, false)
	if r.Stats.HealOnClear > 0 && r.HP < r.Stats.MaxHP {
		heal := math.Min(r.Stats.MaxHP-r.HP, r.Stats.HealOnClear)
		r.HP += heal
		L.text(V3{0, 2.5, 8}, fmt.Sprintf("+%.0f HEALTH", heal), color.RGBA{120, 255, 150, 255}, true)
	}
	if r.Stats.Interest > 0 {
		cap := 30.0
		if r.HasFusion("hoard") {
			cap = 60
		}
		in := int(math.Min(cap, float64(r.Gold)*r.Stats.Interest))
		r.Gold += in
		L.GoldEarned += in
	}
	// collect remaining coins instantly
	for _, c := range L.Coins {
		L.gainGold(c.Val, 0, 0, false)
	}
	L.Coins = nil
	for _, b := range L.Bricks {
		if b.Alive && b.Kind != levelgen.Steel {
			b.Alive = false
			L.debris(V3{b.X(), 0.5, b.Z()}, 6, b.Col)
		}
	}
	r.Cursed = false
	g.snd.PlayKeyed("clear")
	g.snd.Duck(0.6, 2.5)
	for i := 0; i < 5; i++ {
		L.sparks(V3{(L.rng.Float64() - 0.5) * 24, 1, 20 + L.rng.Float64()*15}, 30, L.Biome.Hues[i], 12)
	}
	L.Balls = nil
	L.Drops = nil
	L.Rocks = nil
}

// ---------------------------------------------------------------- per-frame brick behaviour

func (L *Level) updateBricks(dt float64) {
	r := L.g.run
	for _, b := range L.Bricks {
		if !b.Alive {
			continue
		}
		b.Appear = math.Min(1, b.Appear+dt*4)
		if b.FallY > 0 && b.Appear > 0 {
			b.FallY = math.Max(0, b.FallY-dt*7)
		}
		b.Flash = math.Max(0, b.Flash-dt*6)
		b.shieldFlash = math.Max(0, b.shieldFlash-dt*4)
		b.SinceHit += dt
		b.OffX *= math.Pow(0.0005, dt)
		b.OffZ *= math.Pow(0.0005, dt)
		if b.Frozen > 0 {
			b.Frozen -= dt
		}
		if b.Burn > 0 {
			b.Burn -= dt
			b.burnTick += dt
			tick := 0.7
			if r.Stats.DragonHeart {
				tick = 0.35
			}
			if r.Stats.SteamEngine && b.Frozen > 0 {
				L.steamBurst(b)
				if !b.Alive {
					continue
				}
			}
			if b.burnTick >= tick {
				b.burnTick = 0
				L.damage(b, Hit{Src: "Burning", Dmg: (0.5 + 0.25*r.Stats.Damage) * (1 + r.Stats.BurnMult), Elemental: true, Silent: true})
				if !b.Alive {
					continue
				}
			}
			if L.g.settings.Particles > 0 && L.rng.Float64() < dt*12 {
				L.Parts = append(L.Parts, &Particle{P: V3{b.X() + (L.rng.Float64()-0.5)*1.6, 1, b.Z()}, V: V3{0, 2 + L.rng.Float64()*2, 0},
					Life: 0.5, Max: 0.5, Size: 0.5, Col: color.RGBA{255, uint8(100 + L.rng.Intn(120)), 30, 255}, Kind: pSpark})
			}
		}
		frozen := b.Frozen > 0 || b.Layer > 0 || b.Up != nil
		switch b.Kind {
		case levelgen.Regen:
			if b.Frozen <= 0 && b.SinceHit > 3 && b.HP < b.MaxHP {
				b.HP = math.Min(b.MaxHP, b.HP+dt)
			}
		case levelgen.Moving:
			if frozen {
				break
			}
			b.Timer += dt
			if b.Timer > 1.6 {
				b.Timer = 0
				dir := 1
				if L.rng.Intn(2) == 0 {
					dir = -1
				}
				for _, d := range []int{dir, -dir} {
					nc := b.C + d
					if L.cellFree(b.R, nc) && !L.ballNear(cellX(nc), b.Z()) {
						L.moveBrick(b, b.R, nc)
						break
					}
				}
			}
		case levelgen.Teleport:
			if frozen {
				break
			}
			b.Timer += dt
			if b.Timer > 5 {
				b.Timer = 0
				for i := 0; i < 20; i++ {
					rr, cc := L.rng.Intn(max(1, L.Data.Rows+2)), L.rng.Intn(levelgen.Cols)
					if L.cellFree(rr, cc) && !L.ballNear(cellX(cc), cellZ(rr)) {
						L.sparks(V3{b.X(), 0.7, b.Z()}, 12, color.RGBA{180, 120, 255, 255}, 5)
						L.Grid[b.R][b.C] = nil
						b.R, b.C = rr, cc
						L.Grid[rr][cc] = b
						b.Appear = 0
						L.g.snd.Play("warp")
						break
					}
				}
			}
		case levelgen.Spawner:
			if frozen {
				break
			}
			b.Timer += dt
			if b.Timer > 7 {
				b.Timer = 0
				for _, d := range [][2]int{{1, 0}, {0, 1}, {0, -1}, {-1, 0}} {
					rr, cc := b.R+d[0], b.C+d[1]
					if L.cellFree(rr, cc) && !L.ballNear(cellX(cc), cellZ(rr)) {
						L.addBrick(rr, cc, levelgen.Basic, 1, lighten(b.Col, 0.3))
						break
					}
				}
			}
		}
	}
	// compact dead bricks occasionally
	if len(L.Bricks) > 64 {
		alive := L.Bricks[:0]
		for _, b := range L.Bricks {
			if b.Alive {
				alive = append(alive, b)
			}
		}
		L.Bricks = alive
	}
}

func (L *Level) ballNear(x, z float64) bool {
	for _, b := range L.Balls {
		if math.Abs(b.X-x) < CellW && math.Abs(b.Z-z) < CellD+0.6 {
			return true
		}
	}
	return false
}

func (L *Level) moveBrick(b *Brick, r, c int) {
	b.OffX += cellX(b.C) - cellX(c)
	b.OffZ += cellZ(b.R) - cellZ(r)
	L.Grid[b.R][b.C] = nil
	b.R, b.C = r, c
	L.Grid[r][c] = b
}

// ---------------------------------------------------------------- drops, coins, effects

func (L *Level) updateDrops(dt float64) {
	r := L.g.run
	magnet := r.Stats.Magnet || L.Timers[PMagnet] > 0 || L.magnetT > 0
	half := L.PaddleW / 2
	for i := 0; i < len(L.Drops); {
		d := L.Drops[i]
		d.T += dt
		d.Z -= 7 * dt
		if magnet && d.Z < 14 {
			d.X += (L.PaddleX - d.X) * math.Min(1, dt*2.5)
		}
		caught := d.Z-0.4 <= PaddleZ+PaddleD/2 && d.Z+0.4 >= PaddleZ-PaddleD/2 && math.Abs(d.X-L.PaddleX) <= half+0.6
		if caught {
			L.applyPower(d.P)
		}
		if caught || d.Z < LoseZ {
			L.Drops = append(L.Drops[:i], L.Drops[i+1:]...)
			continue
		}
		i++
	}
	for i := 0; i < len(L.Coins); {
		c := L.Coins[i]
		c.T += dt
		c.VY -= 22 * dt
		c.X += c.VX * dt
		c.Y += c.VY * dt
		c.Z += c.VZ * dt
		if c.Y < 0.25 {
			c.Y = 0.25
			c.VY = math.Abs(c.VY) * 0.4
			c.VX *= 0.7
		}
		c.VZ = math.Max(c.VZ-14*dt, -9)
		if magnet || L.Cleared {
			dx, dz := L.PaddleX-c.X, PaddleZ-c.Z
			c.VX += dx * dt * 6
			c.VZ += dz * dt * 3
		}
		c.X = math.Max(-HalfW+0.3, math.Min(HalfW-0.3, c.X))
		caught := math.Abs(c.Z-PaddleZ) < 0.7 && math.Abs(c.X-L.PaddleX) <= half+0.4
		if caught {
			L.gainGold(c.Val, 0, 0, false)
			L.g.snd.Coin()
			if L.g.run.Stats.GoldRush {
				L.rushBonus = math.Min(0.6, L.rushBonus+0.02)
				L.sparks(V3{c.X, 0.8, PaddleZ}, 6, color.RGBA{255, 220, 90, 255}, 5)
			}
		}
		if caught || c.Z < LoseZ {
			L.Coins = append(L.Coins[:i], L.Coins[i+1:]...)
			continue
		}
		i++
	}
	for i := 0; i < len(L.Bolts); {
		bl := L.Bolts[i]
		bl.Z += 40 * dt
		hit := bl.Z > FarZ
		c := int(math.Floor((bl.X + HalfW) / CellW))
		rr := int(math.Floor((GridTopZ - bl.Z) / CellD))
		if !hit && rr >= 0 && rr < levelgen.MaxRows && c >= 0 && c < levelgen.Cols {
			if b := L.Grid[rr][c]; b != nil && b.Alive && b != bl.last {
				st := &L.g.run.Stats
				bm := bl.Mult
				if bm == 0 {
					bm = 1
				}
				L.damage(b, Hit{Src: "Lasers", Dmg: math.Max(0.6, st.Damage*st.DmgMult*0.6) * (1 + st.LaserDmg) * bm, Silent: true, Void: bl.Pierce > 0})
				L.sparks(V3{bl.X, 0.6, bl.Z}, 5, color.RGBA{255, 100, 100, 255}, 4)
				if st.TeslaCoil {
					L.lightning(bl.X, bl.Z, 2, math.Max(1, st.Damage*0.5), b, false)
				}
				hit = true
				if bl.Pierce > 0 {
					bl.Pierce--
					bl.last = b
					hit = false
				}
			}
		}
		if hit {
			L.Bolts = append(L.Bolts[:i], L.Bolts[i+1:]...)
			continue
		}
		i++
	}
	for i := 0; i < len(L.Rocks); {
		rk := L.Rocks[i]
		if rk.Warn > 0 {
			rk.Warn -= dt
			i++
			continue
		}
		rk.Z -= 10 * dt
		rk.X += (rk.TX - rk.X) * math.Min(1, dt*0.9)
		rk.Spin += dt * 5
		if math.Abs(rk.Z-PaddleZ) < 0.9 && math.Abs(rk.X-L.PaddleX) < L.PaddleW/2+0.5 {
			L.hurt(8, "CRUSHED")
			L.stun = 0.6
			L.debris(V3{rk.X, 0.5, rk.Z}, 16, color.RGBA{120, 110, 100, 255})
			L.Rocks = append(L.Rocks[:i], L.Rocks[i+1:]...)
			continue
		}
		if rk.Z < LoseZ {
			L.Rocks = append(L.Rocks[:i], L.Rocks[i+1:]...)
			continue
		}
		i++
	}
}

func (L *Level) updateEffects(dt float64) {
	for i := 0; i < len(L.Parts); {
		p := L.Parts[i]
		p.Life -= dt
		if p.Life <= 0 {
			L.Parts[i] = L.Parts[len(L.Parts)-1]
			L.Parts = L.Parts[:len(L.Parts)-1]
			continue
		}
		p.V.Y -= p.Grav * dt
		p.P = p.P.Add(p.V.Mul(dt))
		p.Rot += p.Spin * dt
		if p.Kind == pDebris && p.P.Y < p.Size/2 {
			p.P.Y = p.Size / 2
			p.V.Y = math.Abs(p.V.Y) * 0.35
			p.V.X *= 0.6
			p.V.Z *= 0.6
			p.Spin *= 0.5
		}
		i++
	}
	for i := 0; i < len(L.Rings); {
		rg := L.Rings[i]
		rg.Life -= dt
		if rg.Life <= 0 {
			L.Rings = append(L.Rings[:i], L.Rings[i+1:]...)
			continue
		}
		i++
	}
	for i := 0; i < len(L.Arcs); {
		if L.Arcs[i].Life -= dt; L.Arcs[i].Life <= 0 {
			L.Arcs = append(L.Arcs[:i], L.Arcs[i+1:]...)
			continue
		}
		i++
	}
	for i := 0; i < len(L.Floats); {
		f := L.Floats[i]
		f.Life -= dt
		f.P.Y += dt * 1.5
		if f.Life <= 0 {
			L.Floats = append(L.Floats[:i], L.Floats[i+1:]...)
			continue
		}
		i++
	}
	L.Shake = math.Max(0, L.Shake-dt*2.5)
	L.Flash = math.Max(0, L.Flash-dt*2)
	// ambient biome particles
	if L.g.settings.Particles > 0 {
		L.ambientT += dt
		for L.ambientT > 0.06 {
			L.ambientT -= 0.06
			var p Particle
			rf := func() float64 { return L.rng.Float64() }
			switch L.Biome.ID {
			case WNeon: // drifting motes
				p = Particle{P: V3{(rf() - 0.5) * 34, 0.5 + rf()*4, rf() * 42}, V: V3{(rf() - 0.5) * 0.6, (rf() - 0.3) * 0.5, (rf() - 0.5) * 0.6},
					Life: 3, Max: 3, Size: 0.35, Col: L.Biome.Hues[L.rng.Intn(6)], Kind: pAmbient}
			case WMeadow: // dandelion seeds on the breeze
				p = Particle{P: V3{-30 + rf()*10, 1 + rf()*6, rf() * 44}, V: V3{2 + rf()*1.5, (rf() - 0.4) * 0.4, (rf() - 0.5) * 0.5},
					Life: 9, Max: 9, Size: 0.28, Col: color.RGBA{255, 250, 235, 255}, Kind: pAmbient}
			case WCaverns: // sparkling motes
				p = Particle{P: V3{(rf() - 0.5) * 40, rf() * 8, rf() * 46}, V: V3{0, 0.2 + rf()*0.3, 0},
					Life: 4, Max: 4, Size: 0.25, Col: L.Biome.Hues[L.rng.Intn(6)], Kind: pAmbient}
			case WRuins: // dust motes in the sunlight
				p = Particle{P: V3{(rf() - 0.5) * 40, rf() * 6, rf() * 46}, V: V3{0.4 + rf()*0.4, (rf() - 0.5) * 0.2, 0},
					Life: 6, Max: 6, Size: 0.2, Col: color.RGBA{255, 240, 200, 255}, Kind: pAmbient}
			case WClouds: // twinkles
				p = Particle{P: V3{(rf() - 0.5) * 50, rf() * 9, rf() * 50}, V: V3{0.6, 0, 0},
					Life: 3, Max: 3, Size: 0.3, Col: color.RGBA{255, 245, 210, 255}, Kind: pAmbient}
			default: // falling embers
				p = Particle{P: V3{(rf() - 0.5) * 40, 9, rf() * 46}, V: V3{(rf() - 0.5) * 0.6, -1.6 - rf(), 0},
					Life: 5, Max: 5, Size: 0.2, Col: color.RGBA{255, uint8(120 + L.rng.Intn(100)), 40, 255}, Kind: pAmbient}
			}
			L.Parts = append(L.Parts, &p)
		}
	}
	if len(L.Parts) > 900 {
		L.Parts = L.Parts[len(L.Parts)-900:]
	}
}

func (L *Level) sparks(p V3, n int, c color.RGBA, spd float64) {
	if L.g.settings.Particles == 0 {
		n = n / 3
	}
	for i := 0; i < n; i++ {
		a := L.rng.Float64() * 2 * math.Pi
		e := L.rng.Float64() * 1.2
		s := spd * (0.3 + L.rng.Float64()*0.7)
		L.Parts = append(L.Parts, &Particle{P: p, V: V3{math.Cos(a) * s, math.Sin(e) * s * 0.8, math.Sin(a) * s},
			Life: 0.35 + L.rng.Float64()*0.3, Max: 0.65, Size: 0.25 + L.rng.Float64()*0.25, Col: c, Kind: pSpark, Grav: 8})
	}
}

func (L *Level) debris(p V3, n int, c color.RGBA) {
	if L.g.settings.Particles == 0 {
		n = n / 3
	}
	rnd := func() float64 { return L.rng.Float64() - 0.5 }
	add := func(pp Particle) {
		if pp.Kind == pDebris {
			pp.Rot = L.rng.Float64() * 6.28
			pp.Spin = (L.rng.Float64() - 0.5) * 18
		}
		L.Parts = append(L.Parts, &pp)
	}
	world := WNeon
	if L.Biome != nil {
		world = L.Biome.ID
	}
	for i := 0; i < n; i++ {
		base := p.Add(V3{rnd() * 1.6, L.rng.Float64() * 0.5, rnd() * 0.8})
		v := V3{rnd() * 9, 3 + L.rng.Float64()*6, rnd() * 7}
		switch world {
		case WNeon: // energy fragments + glowing motes
			add(Particle{P: base, V: v, Life: 0.9, Max: 1.2, Size: 0.16 + L.rng.Float64()*0.2, Col: lighten(c, 0.3), Kind: pDebris, Grav: 22})
			add(Particle{P: base, V: v.Mul(0.6), Life: 0.6, Max: 0.6, Size: 0.5, Col: c, Kind: pSpark, Grav: 2})
		case WMeadow: // wooden splinters and a few petals
			add(Particle{P: base, V: v, Life: 1.1, Max: 1.4, Size: 0.14 + L.rng.Float64()*0.15, Col: mixc(c, color.RGBA{150, 100, 60, 255}, 0.5), Kind: pDebris, Grav: 26})
			if i%3 == 0 {
				add(Particle{P: base, V: V3{rnd() * 3, 2 + L.rng.Float64()*2, rnd() * 3}, Life: 1.6, Max: 1.6, Size: 0.35, Col: color.RGBA{255, 200, 210, 255}, Kind: pSmoke, Grav: 1})
			}
		case WCaverns: // crystal shards and sparkle
			add(Particle{P: base, V: v.Mul(1.2), Life: 1, Max: 1.2, Size: 0.12 + L.rng.Float64()*0.18, Col: lighten(c, 0.4), Kind: pDebris, Grav: 20})
			add(Particle{P: base, V: v.Mul(0.4), Life: 0.7, Max: 0.7, Size: 0.35, Col: color.RGBA{220, 250, 255, 255}, Kind: pSpark, Grav: 0})
		case WRuins: // stone chunks and dust
			add(Particle{P: base, V: v.Mul(0.8), Life: 1.1, Max: 1.4, Size: 0.2 + L.rng.Float64()*0.25, Col: darken(c, 0.9), Kind: pDebris, Grav: 28})
			if i%2 == 0 {
				add(Particle{P: base, V: V3{rnd() * 2, 0.8 + L.rng.Float64(), rnd() * 2}, Life: 1.3, Max: 1.3, Size: 1.4, Col: color.RGBA{220, 196, 150, 255}, Kind: pSmoke})
			}
		case WClouds: // soft cloud puffs and sparkles
			add(Particle{P: base, V: v.Mul(0.4), Life: 0.9, Max: 0.9, Size: 1.1 + L.rng.Float64()*0.6, Col: color.RGBA{255, 255, 255, 255}, Kind: pSmoke, Grav: -1})
			add(Particle{P: base, V: v.Mul(0.7), Life: 0.7, Max: 0.7, Size: 0.35, Col: lighten(c, 0.5), Kind: pSpark, Grav: 4})
		case WMachine: // metal fragments and hot sparks
			add(Particle{P: base, V: v, Life: 1, Max: 1.3, Size: 0.14 + L.rng.Float64()*0.18, Col: mixc(c, color.RGBA{120, 124, 136, 255}, 0.5), Kind: pDebris, Grav: 30})
			add(Particle{P: base, V: V3{rnd() * 12, 4 + L.rng.Float64()*6, rnd() * 10}, Life: 0.6, Max: 0.6, Size: 0.22, Col: color.RGBA{255, 190, 80, 255}, Kind: pSpark, Grav: 22})
		}
	}
}

func (L *Level) text(p V3, s string, c color.RGBA, big bool) {
	if len(L.Floats) > 40 {
		return
	}
	L.Floats = append(L.Floats, &FloatText{P: p, S: s, Col: c, Life: 1.1, Big: big})
}

// ---------------------------------------------------------------- boss: the golem

func (L *Level) setupBoss() {
	act := L.g.run.Act
	b := &Boss{Name: L.Biome.BossName, MaxHP: (40 + 30*float64(act-1) + 10*float64((act-1)*(act-1))) * ascHP(L.g.run.Asc, false, true)}
	b.HP = b.MaxHP
	L.Boss = b
	shape := []string{
		"....######....",
		"...########...",
		"..##########..",
		"..###.##.###..",
		"..##########..",
		"...########...",
		"..##......##..",
		".###......###.",
	}
	body := bossBody(L.Biome)
	for r, row := range shape {
		for c := 0; c < len(row); c++ {
			if row[c] == '#' {
				br := L.addBrick(r+2, c, levelgen.Basic, 2+float64(act), body)
				br.BossPart = true
				// assemble row by row from the top during the intro
				br.Appear = -(1.6 + float64(len(shape)-r)*0.95 + math.Abs(float64(c)-6.5)*0.12)
				br.FallY = 4
				L.Left--
				b.body = append(b.body, [2]int{r + 2, c})
			}
		}
	}
	L.moveCore()
	b.atkT = 4
}

func (L *Level) moveCore() {
	b := L.Boss
	b.coreHits = 0
	b.coreT = 0
	// clear old core
	for _, br := range L.Bricks {
		if br.Alive && br.Core {
			br.Core = false
			br.Kind = levelgen.Basic
			br.Col = bossBody(L.Biome)
		}
	}
	// prefer armour the orb can reach: nothing in front of it
	var cands, open []*Brick
	for _, br := range L.Bricks {
		if br.Alive && br.BossPart {
			cands = append(cands, br)
			if br.R+1 >= levelgen.MaxRows || L.Grid[br.R+1][br.C] == nil {
				open = append(open, br)
			}
		}
	}
	if len(open) > 0 && L.rng.Float64() < 0.8 {
		cands = open
	}
	if len(cands) == 0 {
		// golem fully stripped: rebuild a core in the chest
		br := L.addBrick(5, 6, levelgen.Elite, 1, L.Biome.Trim)
		br.BossPart = true
		L.Left--
		cands = []*Brick{br}
	}
	c := cands[L.rng.Intn(len(cands))]
	c.Core = true
	c.Kind = levelgen.Elite
	c.Col = L.Biome.Trim
	c.Appear = 0.5
	L.g.snd.Play("warp")
}

func (L *Level) updateBoss(dt float64) {
	b := L.Boss
	b.Hurt = math.Max(0, b.Hurt-dt*3)
	if b.Dead {
		b.deathT += dt
		if L.rng.Float64() < dt*10 {
			for _, br := range L.Bricks {
				if br.Alive && br.BossPart {
					br.Alive = false
					L.Grid[br.R][br.C] = nil
					L.explode(br.X(), br.Z(), 0.5, 0, false)
					break
				}
			}
		}
		return
	}
	b.coreT += dt
	if b.coreT > 7 {
		L.moveCore()
	}
	rage := 1 + (1-b.HP/b.MaxHP)*0.8
	b.rebuildT += dt * rage
	if b.rebuildT > 3.5 {
		b.rebuildT = 0
		for _, p := range b.body {
			if L.Grid[p[0]][p[1]] == nil && !L.ballNear(cellX(p[1]), cellZ(p[0])) {
				br := L.addBrick(p[0], p[1], levelgen.Basic, 2+float64(L.g.run.Act), bossBody(L.Biome))
				br.BossPart = true
				L.Left--
				break
			}
		}
	}
	b.atkT -= dt * rage
	if b.atkT <= 0 {
		b.atkT = 6
		n := 2 + L.rng.Intn(2)
		for i := 0; i < n; i++ {
			tx := (L.rng.Float64() - 0.5) * 2 * (HalfW - 2)
			if i == 0 {
				tx = L.PaddleX
			}
			L.Rocks = append(L.Rocks, &Rock{X: (L.rng.Float64() - 0.5) * 8, Z: 20, TX: tx, Warn: 1.0 + 0.2*float64(i)})
		}
		L.g.snd.Play("roar")
		L.Shake = math.Max(L.Shake, 0.4)
	}
}

func (L *Level) killBoss() {
	b := L.Boss
	b.Dead = true
	b.HP = 0
	L.g.run.Bosses++
	L.g.run.log("boss", b.Name+" defeated", "crown")
	if L == L.g.level {
		if L.g.run.Act <= numActs {
			L.g.unlock("ACH_GUARDIAN_" + itoa(L.g.run.Act))
		}
		if L.lostOrbs == 0 {
			L.g.unlock("ACH_FLAWLESS")
		}
	}
	L.Rocks = nil
	L.text(V3{0, 4, 28}, b.Name+" FALLS!", color.RGBA{255, 220, 120, 255}, true)
	L.Shake = 1
	L.Flash = 0.8
	L.g.snd.PlayKeyed("bossdie")
	L.g.snd.Duck(0.75, 4)
}

// breakPop is the instant of a brick breaking: a flash in its colour, a
// quick ring on the floor and, for big bricks, a hit-stop and shockwave.
func (L *Level) breakPop(b *Brick, x, z float64) {
	c := lighten(b.Col, 0.45)
	L.Parts = append(L.Parts, &Particle{P: V3{x, 0.7, z}, Life: 0.16, Max: 0.16, Size: 2.8, Col: c, Kind: pSpark})
	L.Rings = append(L.Rings, &Ring{X: x, Z: z, MaxR: 0.9, Life: 0.22, Max: 0.22, Col: c})
	switch {
	case b.Kind == levelgen.Elite:
		L.hitStop = math.Max(L.hitStop, 0.09)
		L.Rings = append(L.Rings, &Ring{X: x, Z: z, MaxR: 4, Life: 0.5, Max: 0.5, Col: color.RGBA{255, 210, 120, 255}})
		L.sparks(V3{x, 1, z}, 26, color.RGBA{255, 220, 140, 255}, 11)
		L.Flash = math.Max(L.Flash, 0.25)
	case b.MaxHP >= 3:
		L.hitStop = math.Max(L.hitStop, 0.035)
		L.sparks(V3{x, 0.8, z}, 8, c, 7)
	}
}

// bossBody is the Guardian's armour: the world's stone, touched by its trim.
func bossBody(b *Biome) color.RGBA { return mixc(darken(b.Wall, 0.6), b.Trim, 0.16) }

// ---------------------------------------------------------------- pacing

// levelUnits is a chamber's work: the brick health it holds before the
// act and depth multipliers (steel doesn't count; stacks do).
func levelUnits(d *levelgen.Level) float64 {
	u := 0.0
	for _, row := range d.Cells {
		for _, c := range row {
			if c.Kind.Required() {
				u += float64(max(1, c.HP) * max(1, c.Stack))
			}
		}
	}
	return u
}

// chamberBudget is how much work a chamber at this depth should be, so a
// fresh run opens with short, readable chambers and they grow from there.
func chamberBudget(p levelgen.Params) float64 {
	u := (26 + 4.5*float64(min(p.Depth, 26))) * (1 + 0.25*float64(p.Act-1))
	if p.Elite {
		u *= 1.3
	}
	return u
}

// pacedLevel generates a few candidates and keeps the one closest to the
// budget; anything still far too big is trimmed.
func pacedLevel(p levelgen.Params) *levelgen.Level {
	want := chamberBudget(p)
	var best *levelgen.Level
	bestD := math.Inf(1)
	for k := 0; k < 8; k++ {
		q := p
		q.Seed = p.Seed + int64(k)*7919
		d := levelgen.Generate(q)
		u := levelUnits(d)
		dist := math.Abs(math.Log(u / want))
		if dist < bestD {
			best, bestD = d, dist
		}
		if u > want*0.7 && u < want*1.3 {
			break
		}
	}
	if u := levelUnits(best); u > want*1.4 {
		// thin it out: lower health first, then shorter stacks
		k := want * 1.25 / u
		for r := range best.Cells {
			for c := range best.Cells[r] {
				cl := &best.Cells[r][c]
				if !cl.Kind.Required() {
					continue
				}
				cl.HP = max(1, int(math.Round(float64(cl.HP)*math.Sqrt(k))))
				if cl.Stack > 1 {
					cl.Stack = max(1, int(math.Round(float64(cl.Stack)*math.Sqrt(k))))
				}
			}
		}
	}
	return best
}

// lastBrickAim gives the launch angle toward the nearest remaining brick
// once only a handful are left.
func (L *Level) lastBrickAim(b *Ball) (float64, bool) {
	if L.Boss != nil || L.Left <= 0 || L.Left > max(3, L.startLeft/12) {
		return 0, false
	}
	best := math.Inf(1)
	ang := 0.0
	for _, br := range L.Bricks {
		if !br.Alive || !br.Kind.Required() {
			continue
		}
		dx, dz := br.X()-b.X, br.Z()-b.Z
		if d := dx*dx + dz*dz; d < best {
			best = d
			ang = math.Atan2(dx, dz)
		}
	}
	lim := 62 * math.Pi / 180
	return math.Max(-lim, math.Min(lim, ang)), true
}
