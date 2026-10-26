package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"orb/levelgen"
)

type State int

const (
	StMenu State = iota
	StArch
	StSeed
	StMap
	StLevel
	StPause
	StReward
	StShop
	StEvent
	StRest
	StSummary
	StSettings
	StSanctum
	StActIntro
	StHelp
)

type Input struct {
	MouseX  float64
	UseKeys bool
	KeyDir  float64
	Fast    bool
	Fire    bool
}

type rewardSet struct {
	Title string
	Mods  []*Mod
	Skip  int // gold for skipping
}

type shopItem struct {
	Mod   *Mod
	Kind  string // "mod", "heal"
	Price int
	Sold  bool
}

type Game struct {
	state, prevState State
	stateT           float64
	run              *Run
	level            *Level
	save             SaveData
	settings         *Settings
	snd              *Sounds
	ui               *UI
	cam              Camera
	batch            *Batch
	rng              *rand.Rand
	sky              *ebiten.Image
	skyAct           int
	menuL            *Level
	menu             *menuScene
	post             *Post
	titleCache       map[string]*ebiten.Image
	lastDraw         time.Time

	last       time.Time
	lastMouseX int
	useKeys    bool
	mouseHeld  bool

	archSel     int
	seedText    string
	daily       bool
	rewards     []rewardSet
	shop        []shopItem
	rerollCost  int
	event       *Event
	eventMsg    string
	banner      string
	bannerDesc  string
	bannerT     float64
	victory     bool
	shardsWon   int
	pendingNode int
	quit        bool
	tooltip     string

	// HUD animation state
	hudLevel   *Level
	hudHP      float64
	hudBoss    float64
	hudGold    float64
	goldPop    float64
	lastGold   int
	comboShown int
	toasts     []toastMsg
	fadeT      float64
	heartT     float64
}

func NewGame() *Game {
	g := &Game{rng: rand.New(rand.NewSource(time.Now().UnixNano())), last: time.Now(), skyAct: -1}
	g.save = loadSave()
	g.settings = &g.save.Settings
	g.snd = NewSounds()
	g.ui = newUI(g.snd)
	g.batch = newBatch(&g.cam)
	g.post = newPost()
	g.applySettings()
	g.titleCache = map[string]*ebiten.Image{}
	return g
}

func (g *Game) applySettings() {
	s := g.settings
	g.snd.SFX = s.SFX * s.Master
	g.snd.Music = s.Music * s.Master
	ebiten.SetFullscreen(s.Fullscreen)
	ebiten.SetVsyncEnabled(s.VSync)
}

// demoLevel is an attract-mode arena shown behind the main menu.
func (g *Game) demoLevel() *Level {
	saved := g.run
	g.run = NewRun("MENU", &archetypes[0], &g.save, false)
	g.run.Act = 1 + g.rng.Intn(3)
	L := g.newLevel(NCombat, g.rng.Int63())
	L.release()
	demoRun := g.run
	g.run = saved
	if g.run == nil {
		g.run = demoRun
	}
	return L
}

func (g *Game) setState(s State) {
	if s == StSettings && g.state != StSettings {
		g.prevState = g.state
	}
	pauseToggle := (g.state == StLevel && s == StPause) || (g.state == StPause && s == StLevel)
	if !pauseToggle && s != g.state {
		g.fadeT = 1
	}
	g.state = s
	g.stateT = 0
	switch s {
	case StActIntro:
		if g.run != nil {
			g.snd.Play("actstart." + itoa(g.run.Act-1))
			g.snd.Duck(0.6, 3)
		}
	case StReward:
		g.snd.Play("deal")
	case StEvent:
		g.snd.PlayKeyed("event")
	case StRest:
		g.snd.PlayKeyed("rest")
	}
	if s == StLevel {
		ebiten.SetCursorMode(ebiten.CursorModeHidden)
	} else {
		ebiten.SetCursorMode(ebiten.CursorModeVisible)
	}
}

func (g *Game) Layout(ow, oh int) (int, int) {
	s := ebiten.Monitor().DeviceScaleFactor()
	return int(float64(ow) * s), int(float64(oh) * s)
}

// ---------------------------------------------------------------- update

func (g *Game) Update() error {
	if g.quit {
		writeSave(&g.save)
		return ebiten.Termination
	}
	now := time.Now()
	dt := now.Sub(g.last).Seconds()
	g.last = now
	dt = math.Max(0, math.Min(dt, 1.0/30))
	g.stateT += dt
	g.bannerT -= dt

	if inpututil.IsKeyJustPressed(ebiten.KeyF11) || (inpututil.IsKeyJustPressed(ebiten.KeyEnter) && ebiten.IsKeyPressed(ebiten.KeyAlt)) {
		g.settings.Fullscreen = !g.settings.Fullscreen
		g.applySettings()
		writeSave(&g.save)
	}
	g.ui.Click = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	g.mouseHeld = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)

	// music
	switch g.state {
	case StMenu, StArch, StSeed, StSanctum, StHelp, StSettings:
		if g.run == nil || g.state != StSettings || g.prevState == StMenu {
			g.snd.SetTrack(3)
		}
		g.snd.SetIntensity(0.25)
	default:
		if g.run != nil {
			g.snd.SetTrack(g.run.Act - 1)
		}
		if g.state == StLevel && g.level != nil {
			v := math.Min(1, float64(g.level.Combo)/25)
			if g.level.Boss != nil {
				v = math.Max(v, 0.8)
			}
			g.snd.SetIntensity(v)
		} else {
			g.snd.SetIntensity(0.1)
		}
	}

	switch g.state {
	case StMenu, StArch, StSeed, StSanctum, StHelp:
		if g.state == StSeed {
			g.updateSeedInput()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			if g.state == StMenu {
				g.quit = true
			} else {
				g.setState(StMenu)
			}
		}
	case StLevel:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyP) {
			g.setState(StPause)
			return nil
		}
		in := g.readInput()
		debugKeys(g)
		g.level.Update(dt, in)
		if r := g.run; r != nil && r.HP > 0 && r.HP < r.Stats.MaxHP*0.3 && !g.level.Cleared {
			g.heartT -= dt
			if g.heartT <= 0 {
				g.snd.PlayVol("heartbeat", 0.7)
				g.heartT = 0.55 + 2*r.HP/r.Stats.MaxHP
			}
		}
		if g.state == StLevel && g.level.Cleared && g.level.clearT > 1.2 && (g.ui.Click || inpututil.IsKeyJustPressed(ebiten.KeySpace) || g.level.clearT > 4) {
			g.afterLevel()
		}
	case StPause:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyP) {
			g.setState(StLevel)
		}
	case StSettings:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			writeSave(&g.save)
			g.setState(g.prevState)
		}
	}
	return nil
}

func (g *Game) updateDemo(dt float64) {
	L := g.menuL
	if L == nil {
		return
	}
	g.run = L.g.run
	if g.run == nil {
		return
	}
	// simple autopilot: follow the lowest ball heading toward us
	target := 0.0
	best := 1e9
	for _, b := range L.Balls {
		if b.VZ < 0 && b.Z < best {
			best = b.Z
			target = b.X
		}
	}
	in := Input{UseKeys: true}
	if math.Abs(target-L.PaddleX) > 0.4 {
		in.KeyDir = math.Copysign(1, target-L.PaddleX)
	}
	L.Update(dt, in)
	if L.anyStuck() {
		L.release()
	}
	g.run.HP = g.run.Stats.MaxHP
	if L.Cleared && L.clearT > 2 {
		g.menuL = g.demoLevel()
	}
}

func (g *Game) readInput() Input {
	mx, _ := ebiten.CursorPosition()
	if mx != g.lastMouseX {
		g.useKeys = false
		g.lastMouseX = mx
	}
	in := Input{MouseX: float64(mx)}
	left := ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA)
	right := ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyD)
	if left || right {
		g.useKeys = true
	}
	if left {
		in.KeyDir -= 1
	}
	if right {
		in.KeyDir += 1
	}
	in.UseKeys = g.useKeys
	in.Fast = ebiten.IsKeyPressed(ebiten.KeyShift)
	in.Fire = g.ui.Click || inpututil.IsKeyJustPressed(ebiten.KeySpace)
	return in
}

func (g *Game) updateSeedInput() {
	for _, r := range ebiten.AppendInputChars(nil) {
		if len(g.seedText) >= 16 {
			break
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			g.seedText += strings.ToUpper(string(r))
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(g.seedText) > 0 {
		g.seedText = g.seedText[:len(g.seedText)-1]
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && g.seedText != "" {
		g.daily = false
		g.setState(StArch)
	}
}

func randomSeedCode(r *rand.Rand) string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = chars[r.Intn(len(chars))]
	}
	return string(b)
}

func dailyCode() string { return "DAILY-" + time.Now().Format("2006-01-02") }

// ---------------------------------------------------------------- run flow

func (g *Game) startRun(arch *Archetype) {
	code := g.seedText
	if g.daily {
		code = dailyCode()
	}
	if code == "" {
		code = randomSeedCode(g.rng)
	}
	g.run = NewRun(code, arch, &g.save, g.daily)
	g.save.Runs++
	writeSave(&g.save)
	g.setState(StActIntro)
}

func (g *Game) enterNode(i int) {
	r := g.run
	n := r.Map.Nodes[i]
	r.Cur = i
	n.Visited = true
	rng := rand.New(rand.NewSource(r.NodeSeed(i)))
	switch n.Type {
	case NCombat, NElite, NBoss:
		g.snd.Play("mapenter")
		if n.Type == NBoss {
			g.snd.Play("roar")
		}
		g.level = g.newLevel(n.Type, r.NodeSeed(i))
		g.setState(StLevel)
	case NShop:
		g.openShop(rng)
	case NTreasure:
		g.rewards = []rewardSet{{Title: "Treasure Room — choose a relic", Mods: r.Offer(3, true, Common, rng), Skip: 25}}
		g.setState(StReward)
	case NEvent:
		g.event = makeEvent(g, rng)
		g.eventMsg = ""
		g.setState(StEvent)
	case NRest:
		g.setState(StRest)
	}
}

func (g *Game) afterLevel() {
	r := g.run
	L := g.level
	rng := rand.New(rand.NewSource(r.NodeSeed(r.Cur) + 99))
	g.rewards = nil
	switch L.Node {
	case NElite:
		g.rewards = append(g.rewards, rewardSet{Title: "Elite reward — choose a relic", Mods: r.Offer(3, true, Uncommon, rng), Skip: 30})
		g.rewards = append(g.rewards, rewardSet{Title: "Choose an upgrade", Mods: r.Offer(3, false, Common, rng), Skip: 15})
	case NBoss:
		g.rewards = append(g.rewards, rewardSet{Title: "Boss treasure — choose a relic", Mods: r.Offer(3, true, Rare, rng), Skip: 60})
	default:
		g.rewards = append(g.rewards, rewardSet{Title: "Choose an upgrade", Mods: r.Offer(3, false, Common, rng), Skip: 15})
	}
	g.setState(StReward)
}

func (g *Game) finishRewards() {
	r := g.run
	if r.Cur >= 0 && r.Map.Nodes[r.Cur].Type == NBoss {
		if !r.NextAct(&g.save) {
			g.endRun(true)
			return
		}
		g.setState(StActIntro)
		return
	}
	g.setState(StMap)
}

func (g *Game) take(m *Mod) {
	r := g.run
	g.save.Seen[m.ID] = true
	if m.Relic {
		g.toast("Relic acquired", m.Name+" — "+m.Desc, rarityColors[m.Rarity], icShard)
	} else {
		g.toast("Upgrade gained", m.Name+" — "+m.Desc, rarityColors[m.Rarity], icStar)
	}
	if syn := r.Add(m.ID, &g.save); syn != "" {
		for _, s := range synergies {
			if s.Name == syn {
				g.banner, g.bannerDesc, g.bannerT = "SYNERGY: "+strings.ToUpper(syn), s.Desc, 4
				g.snd.PlayKeyed("synergy")
				g.snd.Duck(0.7, 3.5)
			}
		}
	}
}

func (g *Game) openShop(rng *rand.Rand) {
	r := g.run
	g.shop = nil
	for _, m := range r.Offer(3, false, Common, rng) {
		g.shop = append(g.shop, shopItem{Mod: m, Kind: "mod", Price: modPrice(m, rng)})
	}
	for _, m := range r.Offer(2, true, Common, rng) {
		g.shop = append(g.shop, shopItem{Mod: m, Kind: "mod", Price: modPrice(m, rng)})
	}
	g.shop = append(g.shop, shopItem{Kind: "heal", Price: 35})
	g.rerollCost = 20
	g.setState(StShop)
}

func (g *Game) endRun(win bool) {
	r := g.run
	g.victory = win
	sh := r.Depth*2 + r.Bosses*15 + r.Elites*3
	if win {
		sh += 50
		g.save.Wins++
	}
	g.shardsWon = sh
	g.save.Shards += sh
	g.save.Bosses += r.Bosses
	g.save.Bricks += r.Bricks
	g.save.BestAct = max(g.save.BestAct, r.Act)
	g.save.MaxCombo = max(g.save.MaxCombo, r.MaxCombo)
	score := r.Score + r.Gold*5 + r.Depth*200
	r.Score = score
	g.save.BestScore = max(g.save.BestScore, score)
	if r.Daily {
		k := r.SeedCode
		g.save.Daily[k] = max(g.save.Daily[k], score)
	}
	writeSave(&g.save)
	g.setState(StSummary)
}

// ---------------------------------------------------------------- draw

func (g *Game) Draw(screen *ebiten.Image) {
	sw, sh := float64(screen.Bounds().Dx()), float64(screen.Bounds().Dy())
	g.ui.Setup(sw, sh)
	now := time.Now()
	ddt := math.Min(0.05, now.Sub(g.lastDraw).Seconds())
	g.lastDraw = now
	g.ui.Th = &themes[g.themeWorld()]
	g.ui.DT = ddt
	g.ui.Held = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	act := 1
	if g.run != nil {
		act = g.run.Act
	}
	switch g.state {
	case StMenu, StArch, StSeed, StSanctum, StHelp:
		tgt := g.post.Target(int(sw), int(sh))
		g.drawMenuScene(tgt, ddt)
		g.post.Apply(screen, worldGrades[WNeon], !g.settings.NoBloom)
		if g.state != StMenu {
			fillRect(screen, 0, 0, sw, sh, color.RGBA{6, 4, 14, 150})
		}
	case StLevel, StPause:
		g.ensureSky(g.level.Biome, g.level.propSeed)
		tgt := g.post.Target(int(sw), int(sh))
		g.drawWorld(tgt, g.level)
		g.post.Apply(screen, worldGrades[g.level.Biome.ID], !g.settings.NoBloom)
		g.drawHUD(screen, g.level)
	case StSettings:
		if g.prevState == StPause && g.level != nil {
			g.drawWorld(screen, g.level)
		} else {
			g.drawMenuScene(screen, ddt)
		}
		fillRect(screen, 0, 0, sw, sh, color.RGBA{6, 4, 14, 170})
	default:
		g.ensureSky(&biomes[act-1], int64(act)*7)
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Scale(sw/float64(g.sky.Bounds().Dx()), sh/float64(g.sky.Bounds().Dy()))
		screen.DrawImage(g.sky, op)
		fillRect(screen, 0, 0, sw, sh, color.RGBA{6, 4, 14, 120})
	}
	switch g.state {
	case StMenu:
		g.drawMenu(screen)
	case StArch:
		g.drawArch(screen)
	case StSeed:
		g.drawSeed(screen)
	case StSanctum:
		g.drawSanctum(screen)
	case StHelp:
		g.drawHelp(screen)
	case StMap:
		g.drawMap(screen)
	case StPause:
		g.drawPause(screen)
	case StReward:
		g.drawReward(screen)
	case StShop:
		g.drawShop(screen)
	case StEvent:
		g.drawEvent(screen)
	case StRest:
		g.drawRest(screen)
	case StSummary:
		g.drawSummary(screen)
	case StSettings:
		g.drawSettings(screen)
	case StActIntro:
		g.drawActIntro(screen)
	}
	if g.bannerT > 0 {
		u := g.ui
		a := ease(math.Min(g.bannerT, 4-g.bannerT) * 3)
		w := math.Max(760, u.measure(g.banner, fTitle, 44)+260)
		y := 150 - (1-a)*24
		drawSprite(screen, sprSoft, u.X(960-w*0.7), u.Y(y-90), w*1.4*u.S, 300*u.S, ca(cGold, 0.25*a), true)
		u.frame(screen, 960-w/2, y, w, 128, 0.96*a)
		u.icon(screen, icStar, 960-w/2+64, y+64, 58, a)
		u.icon(screen, icStar, 960+w/2-64, y+64, 58, a)
		u.textOutlined(screen, g.banner, fTitle, 44, 960, y+14, ca(rgba(255, 216, 120, 255), a), alignCenter)
		u.text(screen, g.bannerDesc, fReg, 22, 960, y+76, ca(cWhite, a), alignCenter)
	}
	g.drawToasts(screen, ddt)
	if g.tooltip != "" {
		u := g.ui
		lines := u.wrap(g.tooltip, fReg, 19, 380)
		h := float64(len(lines))*26 + 20
		x := math.Min(u.MX+24, 1900-410)
		y := math.Min(u.MY+16, 1070-h)
		u.panel(screen, x, y, 410, h, 0.95)
		for i, l := range lines {
			u.text(screen, l, fReg, 19, x+16, y+10+float64(i)*26, color.RGBA{235, 235, 250, 255}, alignLeft)
		}
		g.tooltip = ""
	}
	g.drawTransition(screen, ddt)
}

func (g *Game) ensureSky(b *Biome, seed int64) {
	variant := int(uint64(seed) % 2)
	key := b.SkyKind*10 + variant
	if g.sky == nil || g.skyAct != key {
		if img := backdrop(b.SkyKind, variant); img != nil {
			g.sky = img
		} else {
			g.sky = makeSky(b, seed)
		}
		g.skyAct = key
	}
}

// ---------------------------------------------------------------- screens

var (
	cGold  = color.RGBA{255, 214, 120, 255}
	cWhite = color.RGBA{240, 238, 252, 255}
	cDim   = color.RGBA{180, 176, 205, 255}
	cAcc   = color.RGBA{150, 120, 255, 255}
	cRed   = color.RGBA{255, 110, 110, 255}
	cGreen = color.RGBA{120, 230, 150, 255}
)

func (g *Game) drawMenu(s *ebiten.Image) {
	u := g.ui
	bob := math.Sin(g.stateT*1.4) * 5
	glow := color.RGBA{255, 60, 200, 255}
	if g.menu != nil {
		glow = g.menu.biome.Trim
	}
	g.chromeTitle(s, "ARCANE ORB", 150, 960, 70+bob, glow)
	u.textOutlined(s, "~ a roguelite brick-breaker ~", fBold, 30, 960, 262, color.RGBA{120, 240, 255, 255}, alignCenter)
	items := []struct {
		label string
		do    func()
	}{
		{"New Run", func() { g.seedText = ""; g.daily = false; g.setState(StArch) }},
		{"Daily Challenge", func() { g.daily = true; g.seedText = ""; g.setState(StArch) }},
		{"Play a Seed", func() { g.seedText = ""; g.setState(StSeed) }},
		{"Sanctum  ·  " + itoa(g.save.Shards) + " shards", func() { g.setState(StSanctum) }},
		{"How to Play", func() { g.setState(StHelp) }},
		{"Settings", func() { g.setState(StSettings) }},
		{"Quit", func() { g.quit = true }},
	}
	y := 360.0
	for i, it := range items {
		if g.neonItem(s, "menu"+itoa(i), it.label, 960, y, glow, true) {
			it.do()
		}
		y += 66
	}
	stats := fmt.Sprintf("Runs %d   ·   Wins %d   ·   Best act %d   ·   Best score %d   ·   Best combo %d",
		g.save.Runs, g.save.Wins, g.save.BestAct, g.save.BestScore, g.save.MaxCombo)
	u.textOutlined(s, stats, fReg, 18, 960, 1012, cDim, alignCenter)
	if best, ok := g.save.Daily[dailyCode()]; ok {
		u.textOutlined(s, fmt.Sprintf("Today's daily best: %d", best), fBold, 18, 960, 1040, cGold, alignCenter)
	}
	u.text(s, "v0.4", fReg, 16, 1890, 1048, cDim, alignRight)
}

func (g *Game) drawArch(s *ebiten.Image) {
	u := g.ui
	title := "Choose your paddle"
	if g.daily {
		title = "Daily Challenge — " + dailyCode()
	} else if g.seedText != "" {
		title = "Seeded run — " + g.seedText
	}
	u.heading(s, title, 60, 70, cGold, 1)
	n := len(archetypes)
	w := 330.0
	x0 := 960 - (float64(n)*w+float64(n-1)*20)/2
	for i := range archetypes {
		a := &archetypes[i]
		x := x0 + float64(i)*(w+20)
		y := 230.0
		locked := a.Cost > 0 && !g.save.Unlocked[a.ID]
		hv := u.hover(x, y, w, 500)
		k := u.anim("arch"+a.ID, hv, 12)
		if k > 0.01 {
			drawSprite(s, sprSoft, u.X(x-70), u.Y(y-70), (w+140)*u.S, 640*u.S, ca(u.theme().Accent, 0.28*k), true)
		}
		u.panel(s, x, y-8*k, w, 500, 0.9)
		if k > 0.01 {
			r, ch := styleShape(u.theme().Style)
			strokePath(s, boxPath(u.X(x), u.Y(y-8*k), w*u.S, 500*u.S, r*u.S, ch), 3*u.S, ca(lighten(u.theme().Accent, 0.3), k), false)
		}
		u.text(s, a.Name, fTitle, 44, x+w/2, y+30-8*k, cWhite, alignCenter)
		u.paragraph(s, a.Desc, fReg, 21, x+w/2, y+104-8*k, w-40, cDim, alignCenter)
		g.archPreview(s, a, x, y-8*k, w)
		if locked {
			can := g.save.Shards >= a.Cost
			if u.button(s, "unlock"+a.ID, fmt.Sprintf("Unlock · %d shards", a.Cost), x+30, y+410, w-60, 60, cGold, can) {
				g.save.Shards -= a.Cost
				g.save.Unlocked[a.ID] = true
				writeSave(&g.save)
				g.snd.Play("buy")
			}
		} else if u.button(s, "pick"+a.ID, "Begin", x+30, y+410, w-60, 60, cGreen, true) {
			g.startRun(a)
		}
	}
	if u.button(s, "back", "Back", 860, 900, 200, 56, cAcc, true) {
		g.setState(StMenu)
	}
}

// archPreview draws a paddle silhouette and stat meters on a paddle card.
func (g *Game) archPreview(s *ebiten.Image, a *Archetype, x, y, w float64) {
	u := g.ui
	th := u.theme()
	S := u.S
	st := baseStats()
	a.Apply(&st)
	cx, py := x+w/2, y+238
	pw := 170 * st.PaddleWidth
	drawSprite(s, sprSoft, u.X(cx-pw*0.8), u.Y(py-40), pw*1.6*S, 80*S, ca(th.Accent, 0.35), true)
	fillPath(s, boxPath(u.X(cx-pw/2), u.Y(py-10), pw*S, 20*S, 6*S, false), rgba(120, 104, 180, 255), rgba(46, 38, 84, 255), vgrad(u.Y(py-10), u.Y(py+10)), false)
	for _, ex := range []float64{cx - pw/2 - 4, cx + pw/2 - 22} {
		fillPath(s, boxPath(u.X(ex), u.Y(py-14), 26*S, 28*S, 6*S, false), rgba(255, 220, 140, 255), rgba(176, 116, 40, 255), vgrad(u.Y(py-14), u.Y(py+14)), false)
	}
	if st.LaserInterval > 0 {
		for _, ex := range []float64{cx - pw/2 + 9, cx + pw/2 - 9} {
			fillPath(s, boxPath(u.X(ex-3), u.Y(py-26), 6*S, 14*S, 3*S, false), rgba(255, 120, 120, 255), rgba(160, 30, 40, 255), vgrad(u.Y(py-26), u.Y(py-12)), false)
		}
	}
	fillPath(s, circlePath(u.X(cx), u.Y(py-26), 11*S), rgba(255, 255, 255, 255), rgba(190, 200, 230, 255), vgrad(u.Y(py-37), u.Y(py-15)), false)
	rows := []struct {
		label string
		v     float64
		col   color.RGBA
	}{
		{"Health", (st.MaxHP - 60) / 80, rgba(236, 70, 100, 255)},
		{"Width", (st.PaddleWidth - 0.6) / 0.8, rgba(110, 170, 255, 255)},
		{"Power", (st.Damage*st.DmgMult - 0.4) / 1.6, rgba(255, 170, 70, 255)},
		{"Fortune", (st.GoldMult - 0.8) / 0.8, rgba(255, 214, 90, 255)},
	}
	for i, r := range rows {
		ry := y + 290 + float64(i)*28
		u.text(s, r.label, fBold, 16, x+34, ry, cDim, alignLeft)
		u.meter(s, x+124, ry+5, w-158, r.v, r.col)
	}
}

func (g *Game) drawSeed(s *ebiten.Image) {
	u := g.ui
	u.heading(s, "PLAY A SEED", 80, 200, cGold, 1)
	u.text(s, "Every seed creates its own run: the same maps, levels, shops and rewards.", fReg, 24, 960, 330, cDim, alignCenter)
	u.text(s, "Type a code from a friend (or any word), then press Enter.", fReg, 24, 960, 366, cDim, alignCenter)
	u.panel(s, 660, 440, 600, 100, 0.9)
	cur := ""
	if math.Mod(g.stateT, 1) < 0.5 {
		cur = "_"
	}
	u.text(s, g.seedText+cur, fTitle, 56, 960, 456, cWhite, alignCenter)
	if u.button(s, "seedgo", "Continue", 760, 600, 400, 64, cGreen, g.seedText != "") {
		g.daily = false
		g.setState(StArch)
	}
	if u.button(s, "seedback", "Back", 860, 690, 200, 56, cAcc, true) {
		g.setState(StMenu)
	}
}

var metaUpgrades = []struct {
	ID, Name, Desc string
	Cost, Max      int
}{
	{"vitality", "Vitality", "+10 starting max health per rank.", 40, 3},
	{"purse", "Heavy Purse", "+20 starting gold per rank.", 30, 3},
	{"reroll", "Fortune's Favour", "+1 free shop reroll each run.", 80, 2},
}

func (g *Game) drawSanctum(s *ebiten.Image) {
	u := g.ui
	u.heading(s, "THE SANCTUM", 80, 80, cGold, 1)
	u.iconLabel(s, icShard, fmt.Sprintf("%d shards   ·   earned at the end of every run", g.save.Shards), fBold, 26, 960, 200, rgba(200, 180, 255, 255), alignCenter)
	y := 290.0
	for _, m := range metaUpgrades {
		rank := g.save.Upg[m.ID]
		u.panel(s, 460, y, 1000, 110, 0.85)
		u.text(s, fmt.Sprintf("%s  (%d/%d)", m.Name, rank, m.Max), fBold, 30, 490, y+18, cWhite, alignLeft)
		u.text(s, m.Desc, fReg, 22, 490, y+62, cDim, alignLeft)
		can := rank < m.Max && g.save.Shards >= m.Cost
		label := fmt.Sprintf("%d shards", m.Cost)
		if rank >= m.Max {
			label = "Maxed"
		}
		if u.button(s, "meta"+m.ID, label, 1220, y+25, 210, 60, cGold, can) {
			g.save.Shards -= m.Cost
			g.save.Upg[m.ID]++
			writeSave(&g.save)
			g.snd.Play("buy")
		}
		y += 130
	}
	u.text(s, "New paddles can be unlocked from the paddle selection screen.", fReg, 22, 960, y+20, cDim, alignCenter)
	if u.button(s, "sback", "Back", 860, 900, 200, 56, cAcc, true) {
		g.setState(StMenu)
	}
}

func (g *Game) drawHelp(s *ebiten.Image) {
	u := g.ui
	u.panel(s, 200, 60, 1520, 960, 0.92)
	u.heading(s, "HOW TO PLAY", 64, 80, cGold, 1)
	lines := []string{
		"Move the paddle with the mouse (or A/D, arrow keys; hold Shift to move faster). Click or Space launches the orb.",
		"Each act is a branching map. Pick your route: battles, elites, shops, events, treasure and rest sites, then a boss.",
		"Losing the orb costs health. Reach zero and the run ends — but you keep the shards you earned for the Sanctum.",
		"After each battle choose an upgrade. Elites, treasure and bosses give relics. Own 3 items with the same tag to awaken a SYNERGY.",
		"Brick types: gold-trimmed tall bricks take extra hits · steel can't break · armoured bricks need crits, explosions, elements or void ·",
		"bombs explode · bolts chain lightning · crystals rain gold · shields protect neighbours · ? bricks are a gamble · crowned elites pay big.",
		"Esc pauses. F11 toggles fullscreen. Every level shows its seed so you can share and replay it.",
	}
	y := 200.0
	for _, l := range lines {
		y = u.paragraph(s, l, fReg, 23, 260, y, 1400, cWhite, alignLeft) + 14
	}
	y += 10
	u.text(s, "Power-ups", fBold, 28, 260, y, cGold, alignLeft)
	y += 50
	for i, p := range powerDefs {
		x := 260 + float64(i%2)*720
		yy := y + float64(i/2)*44
		u.rrect(s, x, yy, 50, 30, 15, p.Col)
		u.text(s, p.Letter, fBold, 20, x+25, yy+3, color.RGBA{20, 20, 30, 255}, alignCenter)
		c := cWhite
		if !p.Good {
			c = cRed
		}
		u.text(s, p.Name, fReg, 22, x+66, yy+2, c, alignLeft)
	}
	if u.button(s, "hback", "Back", 860, 940, 200, 56, cAcc, true) {
		g.setState(StMenu)
	}
}

func (g *Game) runPanel(s *ebiten.Image) {
	u := g.ui
	r := g.run
	u.panel(s, 30, 30, 360, 300, 0.85)
	u.text(s, r.Arch.Name, fTitle, 36, 50, 42, cGold, alignLeft)
	u.iconLabel(s, icHeart, fmt.Sprintf("%.0f / %.0f", r.HP, r.Stats.MaxHP), fBold, 26, 52, 100, color.RGBA{255, 130, 140, 255}, alignLeft)
	u.iconLabel(s, icCoin, fmt.Sprintf("%d gold", r.Gold), fBold, 26, 52, 140, cGold, alignLeft)
	u.text(s, fmt.Sprintf("%s  ·  Act %d", biomes[r.Act-1].Name, r.Act), fReg, 21, 50, 186, cDim, alignLeft)
	u.text(s, "Seed  "+r.SeedCode, fReg, 21, 50, 218, cDim, alignLeft)
	u.text(s, fmt.Sprintf("Damage %.1f  ·  Crit %.0f%%", r.Stats.Damage*r.Stats.DmgMult, r.Stats.CritChance*100), fReg, 19, 50, 256, cDim, alignLeft)
	if len(r.Active) > 0 {
		u.text(s, "Synergies: "+strings.Join(r.Active, ", "), fBold, 19, 50, 288, cAcc, alignLeft)
	}
	// inventory
	u.panel(s, 1530, 30, 360, 1020, 0.85)
	u.text(s, "Build", fTitle, 36, 1550, 42, cGold, alignLeft)
	y := 100.0
	for _, id := range r.Order {
		m := modByID[id]
		if m == nil || r.Mods[id] == 0 {
			continue
		}
		label := m.Name
		if r.Mods[id] > 1 {
			label += fmt.Sprintf(" ×%d", r.Mods[id])
		}
		if m.Relic {
			u.gem(s, 1560, y+12, 6, 8, rarityColors[m.Rarity], 1)
		} else {
			fillPath(s, circlePath(u.X(1560), u.Y(y+12), 4*u.S), lighten(rarityColors[m.Rarity], 0.3), darken(rarityColors[m.Rarity], 0.6), vgrad(u.Y(y+8), u.Y(y+16)), false)
		}
		u.text(s, label, fBold, 19, 1576, y, rarityColors[m.Rarity], alignLeft)
		if u.hover(1540, y-4, 340, 28) {
			g.tooltip = m.Name + " (" + rarityNames[m.Rarity] + ") — " + m.Desc + "  [" + strings.Join(m.Tags, ", ") + "]"
		}
		y += 28
		if y > 1020 {
			break
		}
	}
}

func (g *Game) drawMap(s *ebiten.Image) {
	u := g.ui
	r := g.run
	g.runPanel(s)
	u.heading(s, biomes[r.Act-1].Name, 56, 30, cGold, 1)
	pos := func(n *MapNode) (float64, float64) {
		x := 960 + (float64(n.Lane)-1.5)*230
		if n.Type == NBoss {
			x = 960
		}
		y := 960 - float64(n.Row)*112
		return x, y
	}
	reach := map[int]bool{}
	for _, i := range r.Reachable() {
		reach[i] = true
	}
	for i, n := range r.Map.Nodes {
		x0, y0 := pos(n)
		for _, j := range n.Next {
			x1, y1 := pos(r.Map.Nodes[j])
			switch {
			case i == r.Cur && reach[j]:
				c := rgba(255, 222, 140, 255)
				strokeLine(s, u.X(x0), u.Y(y0), u.X(x1), u.Y(y1), 12*u.S, prem(c, 0.18))
				strokeLine(s, u.X(x0), u.Y(y0), u.X(x1), u.Y(y1), 4*u.S, prem(c, 0.95))
			case n.Visited && r.Map.Nodes[j].Visited:
				strokeLine(s, u.X(x0), u.Y(y0), u.X(x1), u.Y(y1), 4*u.S, prem(u.theme().Accent, 0.85))
			default:
				// dashed: a route not (yet) taken
				d := math.Hypot(x1-x0, y1-y0)
				for t := 0.0; t < d; t += 22 {
					t1 := math.Min(d, t+11)
					strokeLine(s, u.X(x0+(x1-x0)*t/d), u.Y(y0+(y1-y0)*t/d), u.X(x0+(x1-x0)*t1/d), u.Y(y0+(y1-y0)*t1/d), 3*u.S, prem(rgba(150, 140, 190, 255), 0.5))
				}
			}
		}
	}
	for i, n := range r.Map.Nodes {
		x, y := pos(n)
		rad := 34.0
		if n.Type == NBoss {
			rad = 52
		}
		c := nodeColors[n.Type]
		if !reach[i] && !n.Visited {
			c = darken(c, 0.5)
		}
		if reach[i] {
			p := 0.5 + 0.5*math.Sin(g.stateT*5)
			drawSprite(s, sprSoft, u.X(x-rad*2.2), u.Y(y-rad*2.2), rad*4.4*u.S, rad*4.4*u.S, ca(c, 0.35+0.3*p), true)
		}
		u.medallion(s, x, y, rad, c)
		if i == r.Cur {
			strokePath(s, circlePath(u.X(x), u.Y(y), (rad+13)*u.S), 3*u.S, ca(cWhite, 0.9), false)
		}
		g.nodeIcon(s, n.Type, x, y, rad*1.1)
		hv := math.Hypot(u.MX-x, u.MY-y) < rad
		if hv {
			g.tooltip = nodeNames[n.Type]
			if reach[i] {
				g.tooltip += " — click to enter"
			}
		}
		if hv && reach[i] && u.Click {
			g.snd.Play("click")
			g.enterNode(i)
			return
		}
	}
	// legend
	ly := 380.0
	u.panel(s, 30, ly, 360, 330, 0.85)
	u.text(s, "Legend", fBold, 24, 50, ly+16, cGold, alignLeft)
	for i, nm := range nodeNames {
		yy := ly + 60 + float64(i)*36
		u.medallion(s, 68, yy+12, 12, nodeColors[i])
		g.nodeIcon(s, NodeType(i), 68, yy+12, 15)
		u.text(s, nm, fReg, 21, 94, yy, cWhite, alignLeft)
	}
	if u.button(s, "mapmenu", "Abandon run", 60, 990, 300, 56, cRed, true) {
		g.endRun(false)
	}
}

func (g *Game) drawPause(s *ebiten.Image) {
	u := g.ui
	fillRect(s, 0, 0, u.SW, u.SH, color.RGBA{6, 4, 14, 170})
	u.heading(s, "PAUSED", 80, 200, cGold, 1)
	L := g.level
	u.text(s, fmt.Sprintf("%s  ·  %s  ·  level seed %d", L.Data.Name, L.Data.Formation, L.Data.Seed%100000000), fReg, 22, 960, 310, cDim, alignCenter)
	if u.button(s, "resume", "Resume", 760, 380, 400, 64, cGreen, true) {
		g.setState(StLevel)
	}
	if u.button(s, "psettings", "Settings", 760, 460, 400, 64, cAcc, true) {
		g.setState(StSettings)
	}
	if u.button(s, "pabandon", "Abandon run", 760, 540, 400, 64, cRed, true) {
		g.endRun(false)
	}
	g.runPanel(s)
}

func (g *Game) drawCard(s *ebiten.Image, m *Mod, x, y, w, h float64, price int, afford bool) bool {
	u := g.ui
	th := u.theme()
	S := u.S
	hv := u.hover(x, y, w, h)
	if hv {
		u.hoverID = "card" + m.ID
		if u.lastHover != u.hoverID {
			g.snd.Play("cardhover")
		}
	}
	k := u.anim("card"+m.ID, hv, 12)
	y -= 14 * k
	rc := rarityColors[m.Rarity]
	tagc := rc
	if len(m.Tags) > 0 {
		tagc = tagColors[m.Tags[0]]
	}
	r, ch := styleShape(th.Style)
	r = math.Min(r+4, 26)
	sx, sy, sw, sh := u.X(x), u.Y(y), w*S, h*S
	fillPath(s, boxPath(sx+6*S, sy+(12+10*k)*S, sw, sh, r*S, ch), ca(cBlack, 0.42), ca(cBlack, 0.42), grad{}, false)
	if k > 0.01 {
		drawSprite(s, sprSoft, sx-60*S, sy-60*S, sw+120*S, sh+120*S, ca(rc, 0.3*k), true)
	}
	body := boxPath(sx, sy, sw, sh, r*S, ch)
	top := mixc(th.BgTop, rc, 0.16)
	top.A = 250
	bot := th.BgBot
	bot.A = 252
	fillPath(s, body, top, bot, vgrad(sy, sy+sh), false)
	fillPath(s, body, ca(rc, 0.32), ca(rc, 0), vgrad(sy, sy+170*S), false)
	strokePath(s, body, 3*S, mixc(rc, cWhite, 0.25*k), false)
	strokePath(s, boxPath(sx+7*S, sy+7*S, sw-14*S, sh-14*S, math.Max(0, r-7)*S, ch), S, ca(th.Edge2, 1), false)
	u.ornaments(s, th, x, y, w, h, 1)
	kind := "UPGRADE"
	if m.Relic {
		kind = "RELIC"
	}
	u.text(s, kind+"   ·   "+strings.ToUpper(rarityNames[m.Rarity]), fBold, 16, x+w/2, y+24, lighten(rc, 0.15), alignCenter)
	// emblem medallion
	ex, ey := u.X(x+w/2), u.Y(y+130)
	drawSprite(s, sprSoft, ex-90*S, ey-90*S, 180*S, 180*S, ca(tagc, 0.35), true)
	fillPath(s, circlePath(ex, ey+4*S, 66*S), ca(cBlack, 0.4), ca(cBlack, 0.4), grad{}, false)
	fillPath(s, circlePath(ex, ey, 66*S), lighten(th.Edge, 0.3), darken(th.Edge, 0.45), vgrad(ey-66*S, ey+66*S), false)
	fillPath(s, circlePath(ex, ey, 59*S), rgba(12, 9, 22, 255), rgba(12, 9, 22, 255), grad{}, false)
	fillPath(s, circlePath(ex, ey, 55*S), darken(tagc, 0.62), darken(tagc, 0.22), vgrad(ey-55*S, ey+55*S), false)
	fillPath(s, circlePath(ex, ey-20*S, 36*S), rgba(255, 255, 255, 50), rgba(255, 255, 255, 0), vgrad(ey-56*S, ey), false)
	if m.Relic {
		for i := 0; i < 8; i++ {
			a := float64(i)/8*2*math.Pi + math.Pi/8
			u.gem(s, x+w/2+math.Cos(a)*62.5, y+130+math.Sin(a)*62.5, 3.5, 3.5, rc, 1)
		}
	}
	u.textOutlined(s, string([]rune(m.Name)[0]), fTitle, 66, x+w/2, y+88, lighten(tagc, 0.55), alignCenter)
	nameY := u.paragraph(s, m.Name, fTitle, 34, x+w/2, y+212, w-40, cWhite, alignCenter)
	u.glowLine(s, x+w*0.2, x+w*0.8, nameY+8, 1.5, rc, 0.7)
	u.paragraph(s, m.Desc, fReg, 22, x+w/2, nameY+20, w-50, cDim, alignCenter)
	// tags
	tx := x + 22
	for _, t := range m.Tags {
		tw := u.measure(t, fBold, 15) + 24
		tc := tagColors[t]
		p := boxPath(u.X(tx), u.Y(y+h-52), tw*S, 28*S, 14*S, false)
		fillPath(s, p, ca(tc, 0.28), ca(darken(tc, 0.5), 0.28), vgrad(u.Y(y+h-52), u.Y(y+h-24)), false)
		strokePath(s, p, 1.4*S, ca(tc, 0.8), false)
		u.text(s, t, fBold, 15, tx+tw/2, y+h-47, lighten(tc, 0.3), alignCenter)
		tx += tw + 8
	}
	if g.run.Mods[m.ID] > 0 {
		u.text(s, fmt.Sprintf("owned ×%d", g.run.Mods[m.ID]), fReg, 16, x+w-22, y+h-47, cDim, alignRight)
	}
	if price > 0 {
		pc := cGold
		if !afford {
			pc = cRed
		}
		pw := u.measure(itoa(price), fBold, 24) + 74
		px, py := x+w/2-pw/2, y+h+14
		p := boxPath(u.X(px), u.Y(py), pw*S, 44*S, 22*S, false)
		fillPath(s, boxPath(u.X(px), u.Y(py+4), pw*S, 44*S, 22*S, false), ca(cBlack, 0.4), ca(cBlack, 0.4), grad{}, false)
		fillPath(s, p, th.BgTop, th.BgBot, vgrad(u.Y(py), u.Y(py+44)), false)
		strokePath(s, p, 2*S, ca(pc, 0.9), false)
		u.iconLabel(s, icCoin, itoa(price), fBold, 24, x+w/2, py+7, pc, alignCenter)
	}
	return hv && u.Click
}

func (g *Game) drawReward(s *ebiten.Image) {
	u := g.ui
	if len(g.rewards) == 0 {
		g.finishRewards()
		return
	}
	rs := g.rewards[0]
	u.heading(s, strings.TrimSpace(rs.Title), 56, 90, cGold, 1)
	g.statLine(s, 960, 172)
	n := len(rs.Mods)
	w, h := 400.0, 560.0
	x0 := 960 - (float64(n)*w+float64(n-1)*40)/2
	for i, m := range rs.Mods {
		if g.drawCard(s, m, x0+float64(i)*(w+40), 250, w, h, 0, true) && g.stateT > 0.3 {
			g.take(m)
			g.snd.Pick(m.Rarity)
			g.rewards = g.rewards[1:]
			g.stateT = 0
			return
		}
	}
	label := fmt.Sprintf("Skip  (+%d gold)", rs.Skip)
	if u.button(s, "skip", label, 760, 900, 400, 60, cAcc, true) && g.stateT > 0.3 {
		g.run.Gold += rs.Skip
		g.rewards = g.rewards[1:]
		g.stateT = 0
	}
	if g.run.Stats.FreeRerolls > 0 && !rs.isRerolled() {
		if u.button(s, "rr", "Reroll (free)", 1180, 900, 260, 60, cGold, true) {
			g.run.Stats.FreeRerolls--
			rng := rand.New(rand.NewSource(g.rng.Int63()))
			relic := len(rs.Mods) > 0 && rs.Mods[0].Relic
			g.rewards[0].Mods = g.run.Offer(len(rs.Mods), relic, Common, rng)
			g.rewards[0].Title += " "
		}
	}
}

func (r rewardSet) isRerolled() bool { return strings.HasSuffix(r.Title, " ") }

// statLine shows the run's gold and health side by side, centred on cx.
func (g *Game) statLine(s *ebiten.Image, cx, y float64) {
	u := g.ui
	a := itoa(g.run.Gold) + " gold"
	b := fmt.Sprintf("%.0f / %.0f", g.run.HP, g.run.Stats.MaxHP)
	is := 24 * 1.05
	wa := is + 8 + u.measure(a, fBold, 24)
	wb := is + 8 + u.measure(b, fBold, 24)
	x := cx - (wa+48+wb)/2
	u.iconLabel(s, icCoin, a, fBold, 24, x, y, cGold, alignLeft)
	u.iconLabel(s, icHeart, b, fBold, 24, x+wa+48, y, rgba(255, 130, 140, 255), alignLeft)
}

func (g *Game) drawShop(s *ebiten.Image) {
	u := g.ui
	r := g.run
	g.runPanel(s)
	u.heading(s, "THE WANDERING SHOP", 56, 40, cGold, 1)
	u.iconLabel(s, icCoin, fmt.Sprintf("%d gold", r.Gold), fBold, 30, 960, 118, cGold, alignCenter)
	w, h := 330.0, 430.0
	xs := []float64{430, 795, 1160}
	for i, it := range g.shop {
		if it.Kind != "mod" {
			continue
		}
		x := xs[i%3]
		y := 180.0
		if i >= 3 {
			y = 670
			x = xs[i-3]
			h = 300
		}
		if it.Sold {
			u.panel(s, x, y, w, h, 0.4)
			u.text(s, "SOLD", fTitle, 40, x+w/2, y+h/2-24, cDim, alignCenter)
			continue
		}
		afford := r.Gold >= it.Price
		clicked := g.drawCard(s, it.Mod, x, y, w, h, it.Price, afford)
		if clicked && !afford {
			g.snd.Play("deny")
		}
		if clicked && afford {
			r.Gold -= it.Price
			g.take(it.Mod)
			g.shop[i].Sold = true
			g.snd.Play("buy")
			g.snd.Pick(it.Mod.Rarity)
		}
		h = 430
	}
	// heal + reroll
	for i, it := range g.shop {
		if it.Kind != "heal" {
			continue
		}
		label := fmt.Sprintf("Heal 30   ·   %d gold", it.Price)
		if it.Sold {
			label = "Healed"
		}
		if u.button(s, "heal", label, 1160, 700, 330, 64, cGreen, !it.Sold && r.Gold >= it.Price && r.HP < r.Stats.MaxHP) {
			r.Gold -= it.Price
			r.HP = math.Min(r.Stats.MaxHP, r.HP+30)
			g.shop[i].Sold = true
			g.snd.PlayKeyed("heal")
		}
	}
	cost := g.rerollCost
	label := fmt.Sprintf("Reroll stock   ·   %d gold", cost)
	if r.Stats.FreeRerolls > 0 {
		label = "Reroll stock   ·   free"
		cost = 0
	}
	if u.button(s, "reroll", label, 1160, 790, 330, 64, cGold, r.Gold >= cost) {
		if cost == 0 {
			r.Stats.FreeRerolls--
		} else {
			r.Gold -= cost
			g.rerollCost += 10
		}
		rc := g.rerollCost
		g.openShop(rand.New(rand.NewSource(g.rng.Int63())))
		g.rerollCost = rc
		g.snd.Play("buy")
	}
	if u.button(s, "leave", "Leave shop", 1160, 900, 330, 64, cAcc, true) {
		g.setState(StMap)
	}
}

func (g *Game) drawEvent(s *ebiten.Image) {
	u := g.ui
	ev := g.event
	g.runPanel(s)
	u.panel(s, 460, 160, 1000, 760, 0.9)
	u.heading(s, ev.Title, 52, 192, cGold, 1)
	y := u.paragraph(s, ev.Text, fReg, 26, 960, 300, 860, cWhite, alignCenter)
	if g.eventMsg != "" {
		u.paragraph(s, g.eventMsg, fBold, 26, 960, y+40, 860, cAcc, alignCenter)
		if u.button(s, "evdone", "Continue", 760, 820, 400, 64, cGreen, true) {
			g.setState(StMap)
		}
		return
	}
	cy := math.Max(y+50, 560)
	for i, c := range ev.Choices {
		ok := c.Can == nil || c.Can(g)
		if u.button(s, "ev"+itoa(i), c.Label, 560, cy, 800, 64, cAcc, ok) && g.stateT > 0.3 {
			hp, gold, nm := g.run.HP, g.run.Gold, len(g.run.Order)
			g.eventMsg = c.Do(g)
			if g.run.HP < hp || (g.run.Gold < gold && len(g.run.Order) == nm) {
				g.snd.PlayKeyed("bad")
			} else {
				g.snd.PlayKeyed("good")
			}
		}
		cy += 84
	}
}

func (g *Game) drawRest(s *ebiten.Image) {
	u := g.ui
	r := g.run
	g.runPanel(s)
	u.heading(s, "A QUIET CAMPFIRE", 60, 200, cGold, 1)
	u.text(s, "Rest your hands, or train your craft.", fReg, 26, 960, 300, cDim, alignCenter)
	heal := math.Round(r.Stats.MaxHP * 0.35)
	if u.button(s, "rest", fmt.Sprintf("Rest  (+%.0f health)", heal), 660, 420, 600, 80, cGreen, true) {
		r.HP = math.Min(r.Stats.MaxHP, r.HP+heal)
		g.snd.PlayKeyed("heal")
		g.setState(StMap)
	}
	if u.button(s, "train", "Train  (choose an Uncommon+ upgrade)", 660, 530, 600, 80, cAcc, true) {
		rng := rand.New(rand.NewSource(r.NodeSeed(r.Cur) + 5))
		g.rewards = []rewardSet{{Title: "Training — choose an upgrade", Mods: r.Offer(3, false, Uncommon, rng), Skip: 10}}
		g.setState(StReward)
	}
}

func (g *Game) drawActIntro(s *ebiten.Image) {
	u := g.ui
	r := g.run
	b := biomes[r.Act-1]
	a := math.Min(1, g.stateT*1.5)
	u.textOutlined(s, strings.ToUpper(b.Subtitle), fBold, 30, 960, 330-(1-a)*20, alpha(lighten(u.theme().Accent, 0.3), a), alignCenter)
	u.heading(s, strings.ToUpper(b.Name), 110, 384, cGold, a)
	u.text(s, "Guardian:  "+b.BossName, fBold, 26, 960, 548, alpha(cWhite, a), alignCenter)
	if g.stateT > 0.8 {
		u.text(s, "Click to continue", fReg, 22, 960, 640, alpha(cDim, 0.6+0.4*math.Sin(g.stateT*4)), alignCenter)
		if u.Click || inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.setState(StMap)
		}
	}
}

func (g *Game) drawSummary(s *ebiten.Image) {
	u := g.ui
	r := g.run
	title, col := "RUN OVER", cRed
	if g.victory {
		title, col = "VICTORY!", cGold
	}
	u.panel(s, 460, 100, 1000, 860, 0.9)
	u.heading(s, title, 96, 130, col, 1)
	lines := []string{
		fmt.Sprintf("Paddle: %s   ·   Seed: %s", r.Arch.Name, r.SeedCode),
		fmt.Sprintf("Reached %s (Act %d), %d chambers deep", biomes[r.Act-1].Name, r.Act, r.Depth),
		fmt.Sprintf("Bricks broken: %d   ·   Elites: %d   ·   Bosses: %d", r.Bricks, r.Elites, r.Bosses),
		fmt.Sprintf("Highest combo: ×%d   ·   Gold held: %d", r.MaxCombo, r.Gold),
		fmt.Sprintf("Final score: %d", r.Score),
	}
	y := 300.0
	for _, l := range lines {
		u.text(s, l, fReg, 28, 960, y, cWhite, alignCenter)
		y += 50
	}
	u.iconLabel(s, icShard, fmt.Sprintf("+%d shards earned for the Sanctum", g.shardsWon), fBold, 32, 960, y+30, rgba(200, 180, 255, 255), alignCenter)
	if len(r.Active) > 0 {
		u.text(s, "Synergies: "+strings.Join(r.Active, ", "), fBold, 24, 960, y+90, cGold, alignCenter)
	}
	if u.button(s, "summenu", "Main menu", 760, 840, 400, 64, cAcc, g.stateT > 0.8) {
		g.run = nil
		g.level = nil
		g.setState(StMenu)
	}
}

func (g *Game) drawSettings(s *ebiten.Image) {
	u := g.ui
	st := g.settings
	u.panel(s, 460, 80, 1000, 900, 0.92)
	u.heading(s, "SETTINGS", 64, 100, cGold, 1)
	y := 220.0
	st.Master = u.slider(s, "Master volume", 520, y, 880, st.Master, g.mouseHeld)
	st.Music = u.slider(s, "Music", 520, y+60, 880, st.Music, g.mouseHeld)
	st.SFX = u.slider(s, "Sound effects", 520, y+120, 880, st.SFX, g.mouseHeld)
	g.snd.SFX = st.SFX * st.Master
	g.snd.Music = st.Music * st.Master
	g.snd.updateVolumes()
	y += 200
	toggle := func(id, label string, v bool) bool {
		u.text(s, label, fBold, 22, 520, y+12, cWhite, alignLeft)
		if u.switchCtl(s, id, 1260, y+4, v) {
			v = !v
		}
		y += 66
		return v
	}
	st.Shake = toggle("shake", "Screen shake", st.Shake)
	st.Flashes = toggle("flash", "Flash effects", st.Flashes)
	p := st.Particles == 1
	p = toggle("parts", "Full particles", p)
	if p {
		st.Particles = 1
	} else {
		st.Particles = 0
	}
	fs := toggle("fs", "Fullscreen (F11)", st.Fullscreen)
	vs := toggle("vs", "VSync", st.VSync)
	st.ShowFPS = toggle("fps", "Show FPS", st.ShowFPS)
	st.NoBloom = !toggle("bloom", "Bloom & colour grading", !st.NoBloom)
	if fs != st.Fullscreen || vs != st.VSync {
		st.Fullscreen, st.VSync = fs, vs
		g.applySettings()
	}
	if u.button(s, "setback", "Back", 760, 900, 400, 60, cGreen, true) {
		writeSave(&g.save)
		g.setState(g.prevState)
	}
	_ = levelgen.Cols
}

var nodeIconName = []string{"bolt", "crown", "", "gem", "q", "plus", "rock"}

func (g *Game) nodeIcon(s *ebiten.Image, t NodeType, x, y, size float64) {
	u := g.ui
	dark := color.RGBA{20, 14, 30, 255}
	if t == NShop {
		u.text(s, "$", fBold, size*0.9, x, y-size*0.58, dark, alignCenter)
		return
	}
	spr := sprIcons[nodeIconName[t]]
	drawSprite(s, spr, u.X(x-size/2), u.Y(y-size/2), size*u.S, size*u.S, dark, false)
}
