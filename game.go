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
	StCodex
	StStory
)

type Input struct {
	MouseX  float64
	UseKeys bool
	KeyDir  float64
	Fast    bool
	Fire    bool
	Ability bool
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

	archSel      int
	archShown    int
	archPrev     int
	archFade     float64
	archGlow     color.RGBA
	archPar      [2]float64
	frontT       float64 // front-end animation clock
	clickLatch   bool
	navLatch     navInput
	stickT       float64
	stickHeld    bool
	archActivate bool
	padActive    bool // last input came from a gamepad: show pad glyphs
	helpPage     int
	helpShown    int
	helpT        float64
	lastCurX     int
	lastCurY     int
	keyLatch     map[ebiten.Key]bool
	drawnSince   bool
	seedText     string
	daily        bool
	rewards      []rewardSet
	shop         []shopItem
	rerollCost   int
	event        *Event
	eventMsg     string
	banner       string
	bannerDesc   string
	bannerT      float64
	victory      bool
	shardsWon    int
	pendingNode  int
	quit         bool
	tooltip      string

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

	codexTab    int
	endlessMode bool // next run starts in Endless
	storyIdx    int
	storySkip   float64
	storyTick   [8]bool
	boardRank   int // leaderboard placement of the run just ended
	xpBefore    int
	xpGained    int
	rankBefore  int
	ascUnlocked bool
	bannerIcon  *ebiten.Image
	bannerCol   color.RGBA
	shopFree    int

	// achievements
	achQueue  []string // popups waiting to show (when Steam's overlay is absent)
	achT      float64
	achTick   int
	achScroll float64
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
	// achievements earned before Steam was present catch up now
	var have []string
	for id, ok := range g.save.Ach {
		if ok {
			have = append(have, id)
		}
	}
	steamSync(have)
	if steamOnDeck() {
		g.padActive = true // the Deck is a controller first
	}
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
	g.consumeInput() // the press that changed screens must not act on the next one
	switch s {
	case StActIntro:
		if g.run != nil {
			g.snd.Play("actstart." + itoa(g.run.AI()))
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
		steamShutdown()
		return ebiten.Termination
	}
	if captureUpdate(g) {
		return nil // the trailer capture is waiting for its next frame
	}
	now := nowFn()
	dt := now.Sub(g.last).Seconds()
	g.last = now
	dt = math.Max(0, math.Min(dt, 1.0/30))
	g.stateT += dt
	g.bannerT -= dt
	steamTick()
	g.checkAchievements()
	g.lbTick()

	if inpututil.IsKeyJustPressed(ebiten.KeyF11) || (inpututil.IsKeyJustPressed(ebiten.KeyEnter) && ebiten.IsKeyPressed(ebiten.KeyAlt)) {
		g.settings.Fullscreen = !g.settings.Fullscreen
		g.applySettings()
		writeSave(&g.save)
	}
	g.ui.Click = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	g.latchInput()
	g.mouseHeld = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)

	// music
	switch g.state {
	case StMenu, StArch, StSeed, StSanctum, StHelp, StCodex, StSettings:
		if g.run == nil || g.state != StSettings || g.prevState == StMenu {
			g.snd.SetTrack(3)
		}
		g.snd.SetIntensity(0.25)
	default:
		if g.run != nil {
			g.snd.SetTrack(g.run.AI())
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
	case StMenu, StArch, StSeed, StSanctum, StHelp, StCodex:
		if g.state == StSeed {
			g.updateSeedInput()
		}
		debugMenuKeys(g)
		if backPressed() {
			if g.state == StMenu {
				// point at Quit rather than closing the game outright
				g.ui.nav.mode, g.ui.nav.focus = true, "menu8"
			} else {
				g.setState(StMenu)
			}
		}
	case StLevel:
		if pausePressed() {
			g.setState(StPause)
			return nil
		}
		in := g.readInput()
		if inputHook != nil {
			in = inputHook(g)
		}
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
	case StMap:
		debugMapKeys(g)
	case StPause:
		if pausePressed() || padJust(ebiten.StandardGamepadButtonRightRight) {
			g.setState(StLevel)
		}
	case StSettings:
		if backPressed() {
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
	in.Ability = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) || inpututil.IsKeyJustPressed(ebiten.KeyE)
	// gamepad: left stick or d-pad steers (analog), A launches, X / RB / RT fire the ability
	ax := padAxis(ebiten.StandardGamepadAxisLeftStickHorizontal)
	if padDown(ebiten.StandardGamepadButtonLeftLeft) {
		ax = -1
	}
	if padDown(ebiten.StandardGamepadButtonLeftRight) {
		ax = 1
	}
	if math.Abs(ax) > 0.15 {
		g.useKeys = true
		in.UseKeys = true
		// ease the dead zone so small tilts are precise
		in.KeyDir = math.Copysign((math.Abs(ax)-0.15)/0.85, ax)
		in.Fast = math.Abs(ax) > 0.92
	}
	if padJust(ebiten.StandardGamepadButtonRightBottom) {
		in.Fire = true
	}
	if padJust(ebiten.StandardGamepadButtonRightLeft) || padJust(ebiten.StandardGamepadButtonFrontTopRight) || padJust(ebiten.StandardGamepadButtonFrontBottomRight) {
		in.Ability = true
	}
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
	g.run.Endless = g.endlessMode
	g.run.EndlessStart = g.endlessMode
	g.save.Runs++
	writeSave(&g.save)
	if g.endlessMode || g.daily {
		g.setState(StActIntro)
	} else {
		g.startStory(0)
	}
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
		g.rewards = []rewardSet{{Title: "Treasure Room — choose a relic", Mods: r.Offer(r.choices(), true, Common, rng), Skip: 25}}
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
		g.rewards = append(g.rewards, rewardSet{Title: "Elite reward — choose a relic", Mods: r.Offer(r.choices(), true, Uncommon, rng), Skip: 30})
		g.rewards = append(g.rewards, rewardSet{Title: "Choose an upgrade", Mods: r.Offer(r.choices(), false, Common, rng), Skip: 15})
	case NBoss:
		g.rewards = append(g.rewards, rewardSet{Title: "Boss treasure — choose a relic", Mods: r.Offer(r.choices(), true, Rare, rng), Skip: 60})
	default:
		g.rewards = append(g.rewards, rewardSet{Title: "Choose an upgrade", Mods: r.Offer(r.choices(), false, Common, rng), Skip: 15})
	}
	g.setState(StReward)
}

func (g *Game) finishRewards() {
	r := g.run
	if r.Cur >= 0 && r.Map.Nodes[r.Cur].Type == NBoss {
		if !r.NextAct(&g.save) {
			if r.storyOn() {
				g.startStory(len(storyCards) - 1)
			} else {
				g.endRun(true)
			}
			return
		}
		if r.storyOn() && r.Act <= numActs && r.Act-1 < len(storyCards)-1 {
			g.startStory(r.Act - 1)
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
		g.toastImg("Relic acquired", m.Name+" — "+m.Desc, rarityColors[m.Rarity], modEmblem(m))
	} else {
		g.toastImg("Upgrade gained", m.Name+" — "+m.Desc, rarityColors[m.Rarity], modEmblem(m))
	}
	if syn := r.Add(m.ID, &g.save); syn != "" {
		for _, s := range synergies {
			if s.Name == syn {
				g.banner, g.bannerDesc, g.bannerT = "SYNERGY: "+strings.ToUpper(syn), s.Desc, 4
				g.bannerIcon, g.bannerCol = nil, cGold
				g.snd.PlayKeyed("synergy")
				g.snd.Duck(0.7, 3.5)
			}
		}
	}
	g.announceFusions()
}

func (g *Game) openShop(rng *rand.Rand) {
	r := g.run
	g.shop = nil
	for _, m := range r.Offer(3, false, Common, rng) {
		g.shop = append(g.shop, shopItem{Mod: m, Kind: "mod", Price: r.price(modPrice(m, rng))})
	}
	for _, m := range r.Offer(2, true, Common, rng) {
		g.shop = append(g.shop, shopItem{Mod: m, Kind: "mod", Price: r.price(modPrice(m, rng))})
	}
	g.shop = append(g.shop, shopItem{Kind: "heal", Price: r.price(35)})
	g.rerollCost = r.price(20)
	g.shopFree = r.Stats.ShopRerolls
	g.setState(StShop)
}

func (g *Game) endRun(win bool) {
	r := g.run
	g.victory = win
	won := win || (r.Endless && r.Loop() > 0)
	sh := r.Depth*3 + r.Bosses*20 + r.Elites*4
	if won {
		sh += 60
	}
	if win {
		g.save.Wins++
	}
	sh = int(float64(sh)*ascShardMult(r.Asc)) - r.shardsPaid
	r.shardsPaid += sh
	g.shardsWon = sh
	g.save.Shards += sh
	g.xpBefore = g.save.XP
	g.rankBefore = rankOf(g.save.XP)
	g.xpGained = runXP(r, won) - r.xpPaid
	r.xpPaid += g.xpGained
	g.save.XP += g.xpGained
	if r.Endless {
		g.save.BestLoop = max(g.save.BestLoop, r.Loop())
	}
	g.ascUnlocked = false
	if win && r.Asc >= g.save.AscMax && g.save.AscMax < len(ascensions) {
		g.save.AscMax++
		g.ascUnlocked = true
	}
	for _, f := range r.Fused {
		g.save.Fusions[f] = true
	}
	// a won run can continue into Endless and end again: only add what's new
	g.save.Bricks += r.Bricks - r.statsPaid[0]
	g.save.Bosses += r.Bosses - r.statsPaid[1]
	g.save.Elites += r.Elites - r.statsPaid[2]
	r.statsPaid = [3]int{r.Bricks, r.Bosses, r.Elites}
	g.save.BestAct = max(g.save.BestAct, r.Act)
	g.save.MaxCombo = max(g.save.MaxCombo, r.MaxCombo)
	score := r.Score + r.Gold*5 + r.Depth*200 + r.Loop()*2500
	r.FinalScore = score
	g.save.BestScore = max(g.save.BestScore, score)
	g.boardRank = 0
	if r.Endless && !win {
		g.boardRank = g.save.addToBoard(EndlessEntry{Score: score, Loop: r.Loop(), Depth: r.Depth, Class: r.Arch.Name, Asc: r.Asc, Date: time.Now().Format("2006-01-02")})
		lbSubmit(r, score)
	}
	if r.Daily {
		k := r.SeedCode
		g.save.Daily[k] = max(g.save.Daily[k], score)
	}
	g.onRunEnd(r, win)
	writeSave(&g.save)
	g.setState(StSummary)
}

// ---------------------------------------------------------------- draw

func (g *Game) Draw(screen *ebiten.Image) {
	if captureSkipDraw(g, screen) {
		return
	}
	sw, sh := float64(screen.Bounds().Dx()), float64(screen.Bounds().Dy())
	g.ui.Setup(sw, sh)
	now := nowFn()
	ddt := math.Min(0.05, now.Sub(g.lastDraw).Seconds())
	g.lastDraw = now
	g.ui.Th = &themes[g.themeWorld()]
	g.ui.DT = ddt
	g.ui.Held = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	g.bakeBadges()
	// clicks and key presses from every Update since the last frame
	g.ui.Click = g.clickLatch
	g.ui.time += ddt
	g.ui.beginNav(g.navLatch)
	g.drawnSince = true
	act := 1
	if g.run != nil {
		act = g.run.Act
	}
	switch g.state {
	case StMenu, StArch, StSeed, StSanctum, StHelp, StCodex:
		g.frontT += ddt
		switch g.state {
		case StMenu:
			g.drawHall(screen, 1290, 560, true)
		case StArch:
			g.drawHall(screen, 960, 300, true)
		default:
			g.drawHall(screen, 960, 200, false)
			fillRect(screen, 0, 0, sw, sh, color.RGBA{4, 2, 10, 90})
		}
	case StLevel, StPause:
		g.ensureSky(g.level.Biome, g.level.propSeed)
		tgt := g.post.Target(int(sw), int(sh))
		g.drawWorld(tgt, g.level)
		g.post.Apply(screen, worldGrades[g.level.Biome.ID], !g.settings.NoBloom)
		if !hideHUD || g.level.introT > 0 {
			g.drawHUD(screen, g.level)
		}
	case StSettings:
		if g.prevState == StPause && g.level != nil {
			g.drawWorld(screen, g.level)
		} else {
			g.frontT += ddt
			g.drawHall(screen, 960, 200, false)
		}
		fillRect(screen, 0, 0, sw, sh, color.RGBA{6, 4, 14, 170})
	case StStory:
		g.frontT += ddt
		g.drawHall(screen, 960, 140, false)
	default:
		g.ensureSky(&biomes[(act-1)%numActs], int64(act)*7)
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Scale(sw/float64(g.sky.Bounds().Dx()), sh/float64(g.sky.Bounds().Dy()))
		screen.DrawImage(g.sky, op)
		fillRect(screen, 0, 0, sw, sh, color.RGBA{6, 4, 14, 135})
		vignette(screen, sw, sh, 200)
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
	case StCodex:
		g.drawCodex(screen)
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
	case StStory:
		g.drawStory(screen)
	}
	// controller hint bar for menus
	if g.padActive && g.state != StLevel && g.state != StStory && g.state != StActIntro {
		u := g.ui
		x := 1880.0
		for _, hnt := range [][2]string{{"B", "Back"}, {"A", "Select"}} {
			w := u.measure(hnt[1], fBold, 18)
			x -= w
			u.text(screen, hnt[1], fBold, 18, x, 1036, rgba(220, 214, 236, 255), alignLeft)
			kw := u.measure(hnt[0], fBold, 16) + 28
			x -= kw + 8
			u.keycap(screen, hnt[0], x, 1030, 0.95)
			x -= 26
		}
	}
	if g.bannerT > 0 {
		u := g.ui
		a := ease(math.Min(g.bannerT, 4-g.bannerT) * 3)
		w := math.Min(1500, math.Max(760, math.Max(u.measure(g.banner, fRune, 40), u.measure(g.bannerDesc, fReg, 22))+300))
		y := 150 - (1-a)*24
		bc := g.bannerCol
		if bc.A == 0 {
			bc = cGold
		}
		drawSprite(screen, sprSoft, u.X(960-w*0.7), u.Y(y-90), w*1.4*u.S, 300*u.S, ca(bc, 0.25*a), true)
		u.frame(screen, 960-w/2, y, w, 128, 0.96*a)
		if g.bannerIcon != nil {
			drawSprite(screen, sprSoft, u.X(960-w/2+70-70), u.Y(y+64-70), 140*u.S, 140*u.S, ca(bc, 0.5*a), true)
			u.drawEmblem(screen, g.bannerIcon, 960-w/2+70, y+64, 104, a)
			u.drawEmblem(screen, g.bannerIcon, 960+w/2-70, y+64, 104, a)
		} else {
			u.icon(screen, icStar, 960-w/2+64, y+64, 58, a)
			u.icon(screen, icStar, 960+w/2-64, y+64, 58, a)
		}
		u.textOutlined(screen, g.banner, fRune, 40, 960, y+16, ca(lighten(bc, 0.2), a), alignCenter)
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
	g.drawAchievementPopup(screen)
	g.drawTransition(screen, ddt)
	debugOverlay(g, screen)
	captureAfterDraw(g, screen)
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
	const ox = 1290.0
	g.drawRunes(s, 22, ox, 900)
	g.drawTitleOrb(s, ox, 560, 140, g.frontT)
	// title block
	g.arcaneTitle(s, "Arcane Orb", 112, ox, 70)
	sub := "A  ROGUELITE  OF  ORB  AND  RUNE"
	sw := u.measure(sub, fSerif, 21)
	u.text(s, sub, fSerif, 21, ox, 214, rgba(190, 160, 255, 255), alignCenter)
	for _, sgn := range []float64{-1, 1} {
		x0 := ox + sgn*(sw/2+24)
		u.glowLine(s, math.Min(x0, x0+sgn*150), math.Max(x0, x0+sgn*150), 228, 1, rgba(196, 160, 92, 255), 0.8)
		fillPath(s, diamondPath(u.X(x0+sgn*156), u.Y(228), 5*u.S, 7*u.S), rgba(230, 200, 140, 255), rgba(150, 110, 60, 255), grad{}, false)
	}
	// menu column with a dark veil behind it
	fillPath(s, boxPath(u.X(0), u.Y(0), 640*u.S, 1080*u.S, 0, false), rgba(0, 0, 0, 170), rgba(0, 0, 0, 0), hgrad(u.X(0), u.X(640)), false)
	endlessLabel := "Endless Mode"
	if g.save.Wins == 0 {
		endlessLabel = "Endless Mode  ·  sealed"
	}
	items := []struct {
		label string
		do    func()
		on    bool
	}{
		{"Story Mode", func() { g.seedText = ""; g.daily = false; g.endlessMode = false; g.setState(StArch) }, true},
		{endlessLabel, func() {
			if g.save.Wins > 0 {
				g.seedText = ""
				g.daily = false
				g.endlessMode = true
				g.setState(StArch)
			} else {
				g.snd.Play("deny")
			}
		}, true},
		{"Daily Challenge", func() { g.daily = true; g.seedText = ""; g.endlessMode = false; g.setState(StArch) }, true},
		{"Play a Seed", func() { g.seedText = ""; g.endlessMode = false; g.setState(StSeed) }, true},
		{"Sanctum  ·  " + itoa(g.save.Shards) + " shards", func() { g.setState(StSanctum) }, true},
		{"Codex", func() { g.codexTab = 0; g.setState(StCodex) }, true},
		{"How to Play", func() { g.helpPage = 0; g.setState(StHelp) }, true},
		{"Settings", func() { g.setState(StSettings) }, true},
		{"Quit", func() { g.quit = true }, true},
	}
	y := 300.0
	for i, it := range items {
		if g.arcaneItem(s, "menu"+itoa(i), it.label, 120, y, it.on) {
			it.do()
		}
		y += 64
		if i == 3 || i == 6 {
			y += 12 // breathing room between groups
		}
	}
	g.drawRankBadge(s, 36, 32)
	g.drawLeaderboard(s, 1588, 300, 300)
	stats := fmt.Sprintf("Runs %d   ·   Wins %d   ·   Best act %d   ·   Best score %d   ·   Best combo %d",
		g.save.Runs, g.save.Wins, g.save.BestAct, g.save.BestScore, g.save.MaxCombo)
	if g.save.BestLoop > 0 {
		stats += fmt.Sprintf("   ·   Endless loop %d", g.save.BestLoop)
	}
	u.textOutlined(s, stats, fReg, 17, ox, 1010, rgba(150, 140, 175, 255), alignCenter)
	if best, ok := g.save.Daily[dailyCode()]; ok {
		u.textOutlined(s, fmt.Sprintf("Today's daily best: %d", best), fBold, 17, ox, 1038, cGold, alignCenter)
	}
	u.text(s, gameVersion, fReg, 16, 1890, 1048, rgba(120, 110, 140, 255), alignRight)
}

// archPreview draws a paddle silhouette and stat meters on a paddle card.
func (g *Game) archPreview(s *ebiten.Image, a *Archetype, x, y, w float64) {
	u := g.ui
	th := u.theme()
	S := u.S
	st := baseStats()
	a.Apply(&st)
	cx, py := x+w/2, y+236
	pw := 140 * st.PaddleWidth
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
		ry := y + 280 + float64(i)*27
		u.text(s, r.label, fBold, 15, x+24, ry, cDim, alignLeft)
		u.meter(s, x+104, ry+5, w-128, r.v, r.col)
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

func (g *Game) runPanel(s *ebiten.Image) {
	u := g.ui
	r := g.run
	u.panel(s, 30, 30, 360, 300, 0.85)
	u.text(s, r.Arch.Name, fRune, 34, 50, 42, cGold, alignLeft)
	u.iconLabel(s, icHeart, fmt.Sprintf("%.0f / %.0f", r.HP, r.Stats.MaxHP), fBold, 26, 52, 100, color.RGBA{255, 130, 140, 255}, alignLeft)
	u.iconLabel(s, icCoin, fmt.Sprintf("%d gold", r.Gold), fBold, 26, 52, 140, cGold, alignLeft)
	u.text(s, fmt.Sprintf("%s  ·  Act %d", biomes[r.AI()].Name, r.Act), fReg, 21, 50, 186, cDim, alignLeft)
	u.text(s, "Seed  "+r.SeedCode, fReg, 21, 50, 218, cDim, alignLeft)
	u.text(s, fmt.Sprintf("Damage %.1f  ·  Crit %.0f%%", r.Stats.Damage*r.Stats.DmgMult, r.Stats.CritChance*100), fReg, 19, 50, 256, cDim, alignLeft)
	if len(r.Active) > 0 {
		u.text(s, "Synergies: "+strings.Join(r.Active, ", "), fBold, 19, 50, 288, cAcc, alignLeft)
	}
	// inventory
	u.panel(s, 1530, 30, 360, 1020, 0.85)
	u.text(s, "Build", fRune, 34, 1550, 42, cGold, alignLeft)
	y := 100.0
	if ci, ok := classInfo[r.Arch.ID]; ok {
		fillPath(s, boxPath(u.X(1546), u.Y(y-6), 328*u.S, 58*u.S, 8*u.S, false), ca(darken(ci.Col, 0.3), 0.5), ca(darken(ci.Col, 0.15), 0.3), vgrad(u.Y(y-6), u.Y(y+52)), false)
		u.drawEmblem(s, glyphImage(ci.Glyph, ci.Col), 1574, y+23, 36, 1)
		u.text(s, strings.ToUpper(r.Arch.Passive), fBold, 15, 1600, y, lighten(ci.Col, 0.4), alignLeft)
		ab := "No active ability"
		if r.Arch.Ability != "" {
			ab = "Active: " + r.Arch.Ability
		}
		u.text(s, ab, fReg, 15, 1600, y+24, cDim, alignLeft)
		if u.hover(1546, y-6, 328, 58) {
			g.tooltip = r.Arch.Passive + " — " + r.Arch.Desc
		}
		y += 70
	}
	if len(r.Order) == 0 && len(r.Fused) == 0 {
		u.paragraph(s, "Upgrades, relics and fusions you collect will gather here.", fReg, 18, 1710, y+20, 300, ca(cDim, 0.7), alignCenter)
	}
	for _, id := range r.Fused {
		f := fusionByID[id]
		u.drawEmblem(s, fusionEmblem(f), 1562, y+13, 30, 1)
		u.text(s, f.Name, fBold, 19, 1582, y, f.Col, alignLeft)
		if u.hover(1540, y-4, 340, 28) {
			g.tooltip = "FUSION — " + f.Name + ": " + f.Desc
		}
		y += 30
	}
	for _, id := range r.Order {
		m := modByID[id]
		if m == nil || r.Mods[id] == 0 {
			continue
		}
		label := m.Name
		if r.Mods[id] > 1 {
			label += fmt.Sprintf(" ×%d", r.Mods[id])
		}
		u.drawEmblem(s, modEmblem(m), 1560, y+12, 26, 1)
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
	title := biomes[r.AI()].Name
	if l := r.Loop(); l > 0 {
		title = fmt.Sprintf("Endless %d  ·  %s", l, title)
	}
	// a dark pool behind the chart so routes read over any painting
	drawSprite(s, sprSoft, u.X(960-640), u.Y(40), 1280*u.S, 1060*u.S, rgba(0, 0, 0, 175), false)
	u.heading(s, title, 56, 30, cGold, 1)
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
			case (i == r.Cur && reach[j]) || (r.Cur < 0 && false):
				c := rgba(255, 222, 140, 255)
				strokeLine(s, u.X(x0), u.Y(y0), u.X(x1), u.Y(y1), 12*u.S, prem(c, 0.18))
				strokeLine(s, u.X(x0), u.Y(y0), u.X(x1), u.Y(y1), 4*u.S, prem(c, 0.95))
				// motes of light flowing along the open ley line
				for k := 0; k < 3; k++ {
					f := math.Mod(g.stateT*0.6+float64(k)/3, 1)
					px, py := x0+(x1-x0)*f, y0+(y1-y0)*f
					drawSprite(s, sprSoft, u.X(px-14), u.Y(py-14), 28*u.S, 28*u.S, ca(c, 0.9*math.Sin(f*math.Pi)), true)
				}
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
			// turning rune ring
			for k := 0; k < 16; k++ {
				an := g.stateT*0.8 + float64(k)/16*2*math.Pi
				r0, r1 := rad+10, rad+16
				if k%4 == 0 {
					r1 = rad + 20
				}
				strokeLine(s, u.X(x+math.Cos(an)*r0), u.Y(y+math.Sin(an)*r0), u.X(x+math.Cos(an)*r1), u.Y(y+math.Sin(an)*r1), 2*u.S, prem(lighten(c, 0.4), 0.8))
			}
		}
		u.medallion(s, x, y, rad, c)
		if i == r.Cur {
			strokePath(s, circlePath(u.X(x), u.Y(y), (rad+13)*u.S), 3*u.S, ca(cWhite, 0.9), false)
		}
		g.nodeIcon(s, n.Type, x, y, rad*1.1)
		var hv, clk bool
		if reach[i] {
			hv, clk = u.hit("node"+itoa(i), x-rad, y-rad, rad*2, rad*2)
			if hv && u.nav.mode {
				strokePath(s, circlePath(u.X(x), u.Y(y), (rad+24)*u.S), 3*u.S, ca(cWhite, 0.6+0.4*math.Sin(u.time*6)), false)
			}
		} else {
			hv = math.Hypot(u.MX-x, u.MY-y) < rad
		}
		if hv {
			g.tooltip = nodeNames[n.Type]
			if reach[i] {
				g.tooltip += " — click to enter"
			}
		}
		if clk {
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
	hv, clk := u.hit("card"+m.ID, x, y, w, h)
	if hv && u.nav.mode {
		u.focusRing(s, x, y-14, w, h, 20)
	}
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
	// emblem medallion (compact on short cards)
	ey0, er, nameTop := 130.0, 1.0, 212.0
	if h < 400 {
		ey0, er, nameTop = 88, 0.68, 148
	}
	ex, ey := u.X(x+w/2), u.Y(y+ey0)
	drawSprite(s, sprSoft, ex-90*er*S, ey-90*er*S, 180*er*S, 180*er*S, ca(tagc, 0.35), true)
	fillPath(s, circlePath(ex, ey+4*S, 66*er*S), ca(cBlack, 0.4), ca(cBlack, 0.4), grad{}, false)
	fillPath(s, circlePath(ex, ey, 66*er*S), lighten(th.Edge, 0.3), darken(th.Edge, 0.45), vgrad(ey-66*er*S, ey+66*er*S), false)
	fillPath(s, circlePath(ex, ey, 59*er*S), rgba(12, 9, 22, 255), rgba(12, 9, 22, 255), grad{}, false)
	fillPath(s, circlePath(ex, ey, 55*er*S), darken(tagc, 0.62), darken(tagc, 0.22), vgrad(ey-55*er*S, ey+55*er*S), false)
	fillPath(s, circlePath(ex, ey-20*er*S, 36*er*S), rgba(255, 255, 255, 50), rgba(255, 255, 255, 0), vgrad(ey-56*er*S, ey), false)
	if m.Relic {
		for i := 0; i < 8; i++ {
			a := float64(i)/8*2*math.Pi + math.Pi/8
			u.gem(s, x+w/2+math.Cos(a)*62.5*er, y+ey0+math.Sin(a)*62.5*er, 3.5, 3.5, rc, 1)
		}
	}
	u.drawEmblem(s, modEmblem(m), x+w/2, y+ey0, 104*er, 1)
	// the name shrinks to stay on one line; the description shrinks to
	// fit above the fusion hint and tags
	ns := 32.0
	for ns > 20 && u.measure(m.Name, fSerif, ns) > w-36 {
		ns -= 1
	}
	u.text(s, m.Name, fSerif, ns, x+w/2, y+nameTop+(32-ns)*0.5, cWhite, alignCenter)
	nameY := y + nameTop + 32*1.35
	u.glowLine(s, x+w*0.2, x+w*0.8, nameY+8, 1.5, rc, 0.7)
	limit := y + h - 62
	if g.run.fusionWith(m.ID) != nil {
		limit = y + h - 104
	} else if len(fusionPartners(m.ID)) > 0 {
		limit = y + h - 92
	}
	ds := 22.0
	for ds > 15 && nameY+20+float64(len(u.wrap(m.Desc, fReg, ds, w-50)))*ds*1.35 > limit {
		ds -= 1
	}
	u.paragraph(s, m.Desc, fReg, ds, x+w/2, nameY+20, w-50, cDim, alignCenter)
	// fusion hint
	if f := g.run.fusionWith(m.ID); f != nil {
		fy := y + h - 96
		pulse := 0.75 + 0.25*math.Sin(g.stateT*5)
		fp := boxPath(u.X(x+18), u.Y(fy), (w-36)*S, 36*S, 18*S, false)
		drawSprite(s, sprSoft, u.X(x), u.Y(fy-30), w*S, 96*S, ca(f.Col, 0.35*pulse), true)
		fillPath(s, fp, ca(f.Col, 0.3), ca(darken(f.Col, 0.4), 0.3), vgrad(u.Y(fy), u.Y(fy+36)), false)
		strokePath(s, fp, 1.6*S, ca(f.Col, pulse), false)
		u.drawEmblem(s, fusionEmblem(f), x+40, fy+18, 34, 1)
		u.text(s, "FORGES "+strings.ToUpper(f.Name), fBold, 15, x+62, fy+9, lighten(f.Col, 0.3), alignLeft)
	} else if ps := fusionPartners(m.ID); len(ps) > 0 {
		u.text(s, "Fuses with "+modByID[ps[0]].Name, fReg, 15, x+w/2, y+h-86, ca(cDim, 0.8), alignCenter)
	}
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
	return clk
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
	if n > 3 {
		w = math.Min(400, (1800-float64(n-1)*40)/float64(n))
	}
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
	if g.run.freeRerolls() > 0 && !rs.isRerolled() {
		if u.button(s, "rr", "Reroll (free)", 1180, 900, 260, 60, cGold, true) {
			g.run.Rerolls++
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
			u.text(s, "SOLD", fRune, 40, x+w/2, y+h/2-24, cDim, alignCenter)
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
	if g.shopFree > 0 || r.freeRerolls() > 0 {
		label = "Reroll stock   ·   free"
		cost = 0
	}
	if u.button(s, "reroll", label, 1160, 790, 330, 64, cGold, r.Gold >= cost) {
		switch {
		case g.shopFree > 0:
			g.shopFree--
		case cost == 0:
			r.Rerolls++
		default:
			r.Gold -= cost
			g.rerollCost += r.price(10)
		}
		rc, sf := g.rerollCost, g.shopFree
		g.openShop(rand.New(rand.NewSource(g.rng.Int63())))
		g.rerollCost, g.shopFree = rc, sf
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
	// event illustration: emblem floating in a turning sigil
	if g.eventMsg == "" {
		gl := eventGlyphs[ev.Title]
		if gl == "" {
			gl = "star"
		}
		col := u.theme().Accent
		bob := math.Sin(g.stateT*1.6) * 6
		g.drawSigil(s, 960, 462, 92, g.stateT*0.25, col, 0.55)
		drawSprite(s, sprSoft, u.X(960-110), u.Y(462-110), 220*u.S, 220*u.S, ca(col, 0.3), true)
		u.drawEmblem(s, glyphImage(gl, col), 960, 462+bob, 116, 1)
	}
	cy := math.Max(y+50, 590)
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

var eventGlyphs = map[string]string{
	"Mysterious Merchant": "flask", "Gambling Den": "clover", "Upgrade Shrine": "gem",
	"Cursed Room": "eye", "Abandoned Forge": "anvil", "Wandering Spirit": "feather",
}

// drawCampfire paints a small animated fire with logs and sparks.
func (g *Game) drawCampfire(s *ebiten.Image, cx, cy float64) {
	u := g.ui
	S := u.S
	t := g.stateT
	drawSprite(s, sprSoft, u.X(cx-320), u.Y(cy-260), 640*S, 400*S, rgba(255, 120, 40, 90), true)
	for _, l := range [][4]float64{{-70, 18, 60, -6}, {70, 18, -60, -6}, {-60, 6, 66, 10}} {
		strokePath(s, linePath(u.X(cx+l[0]), u.Y(cy+l[1]), u.X(cx+l[2]), u.Y(cy+l[3])), 20*S, rgba(52, 32, 22, 255), false)
		strokePath(s, linePath(u.X(cx+l[0]), u.Y(cy+l[1]-6), u.X(cx+l[2]), u.Y(cy+l[3]-6)), 5*S, rgba(110, 66, 40, 255), false)
	}
	for i := 0; i < 34; i++ {
		f := float64(i)
		life := math.Mod(t*(0.8+0.5*math.Mod(f*0.37, 1))+f*0.0731, 1)
		x := cx + (math.Mod(f*0.618, 1)-0.5)*70*(1-life) + math.Sin(t*6+f)*6*life
		y := cy - 6 - life*150
		c := mixc(rgba(255, 236, 160, 255), rgba(255, 70, 20, 255), life)
		r := (42 - 30*life)
		drawSprite(s, sprSoft, u.X(x-r), u.Y(y-r), 2*r*S, 2*r*S, ca(c, 0.7*(1-life)), true)
	}
	for i := 0; i < 10; i++ {
		f := float64(i)
		life := math.Mod(t*0.35+f*0.1, 1)
		x := cx + math.Sin(f*2.3+t)*40*life
		y := cy - 30 - life*260
		drawSprite(s, sprSoft, u.X(x-3), u.Y(y-3), 6*S, 6*S, ca(rgba(255, 190, 90, 255), math.Sin(life*math.Pi)), true)
	}
}

func (g *Game) drawRest(s *ebiten.Image) {
	u := g.ui
	r := g.run
	g.runPanel(s)
	u.heading(s, "A QUIET CAMPFIRE", 60, 200, cGold, 1)
	g.drawCampfire(s, 960, 790)
	u.text(s, "Rest your hands, or train your craft.", fReg, 26, 960, 300, cDim, alignCenter)
	healPct := 0.35
	if r.Asc >= 6 {
		healPct = 0.25
	}
	heal := math.Round(r.Stats.MaxHP * healPct)
	if u.button(s, "rest", fmt.Sprintf("Rest  (+%.0f health)", heal), 660, 420, 600, 80, cGreen, true) {
		r.HP = math.Min(r.Stats.MaxHP, r.HP+heal)
		g.snd.PlayKeyed("heal")
		g.setState(StMap)
	}
	if u.button(s, "train", "Train  (choose an Uncommon+ upgrade)", 660, 530, 600, 80, cAcc, true) {
		rng := rand.New(rand.NewSource(r.NodeSeed(r.Cur) + 5))
		g.rewards = []rewardSet{{Title: "Training — choose an upgrade", Mods: r.Offer(r.choices(), false, Uncommon, rng), Skip: 10}}
		g.setState(StReward)
	}
}

func (g *Game) drawActIntro(s *ebiten.Image) {
	u := g.ui
	S := u.S
	r := g.run
	b := biomes[r.AI()]
	t := g.stateT
	sw, sh := float64(s.Bounds().Dx()), float64(s.Bounds().Dy())
	// darken the world painting into a stage for the title
	fillRect(s, 0, 0, sw, sh, rgba(3, 2, 8, 150))
	drawSprite(s, sprSoft, u.X(960-900), u.Y(170), 1800*S, 560*S, rgba(0, 0, 0, 170), false)
	acc := lighten(u.theme().Accent, 0.25)
	// a great sigil turning behind the name
	sa := ease(clamp01(t * 0.9))
	g.drawSigil(s, 960, 450, 300*(0.9+0.1*sa), t*0.12, acc, 0.5*sa)
	drawSprite(s, sprSoft, u.X(960-500), u.Y(450-300), 1000*S, 600*S, ca(acc, 0.12*sa), true)

	sub := strings.ToUpper(b.Subtitle)
	if l := r.Loop(); l > 0 {
		sub = fmt.Sprintf("ENDLESS  ·  LOOP %d", l)
	} else if r.Endless {
		sub = "ENDLESS  ·  " + sub
	}
	a1 := ease(clamp01(t * 1.6))
	u.text(s, sub, fSerif, 28, 960, 300-(1-a1)*16, ca(acc, a1), alignCenter)
	a2 := ease(clamp01((t - 0.25) * 1.3))
	u.headingW(s, b.Name, 96, 360+(1-a2)*12, rgba(240, 222, 176, 255), a2, 940)
	a3 := ease(clamp01((t - 0.7) * 1.5))
	u.text(s, "GUARDIAN", fBold, 15, 960, 512, ca(cDim, a3), alignCenter)
	u.text(s, b.BossName, fRune, 34, 960, 534, ca(rgba(255, 214, 190, 255), a3), alignCenter)
	if l := r.Loop(); l > 0 {
		u.text(s, fmt.Sprintf("Bricks ×%.2f health   ·   orbs +%d%% speed   ·   orb loss +%d damage", math.Pow(1.25, float64(l)), 6*l, 3*l),
			fReg, 22, 960, 600, ca(rgba(255, 140, 120, 255), a3), alignCenter)
	}
	if (u.Click || g.keyHit(ebiten.KeySpace) || g.keyHit(ebiten.KeyEnter)) && t < 0.8 && t > 0.05 {
		g.stateT = 0.81 // first press finishes the reveal
		g.consumeInput()
		return
	}
	if t > 0.8 {
		u.text(s, "Click to begin", fSerif, 22, 960, 660, ca(cDim, 0.55+0.35*math.Sin(t*3)), alignCenter)
		if u.Click || g.keyHit(ebiten.KeySpace) || g.keyHit(ebiten.KeyEnter) {
			g.setState(StMap)
		}
	}
}

// drawSigil paints an arcane circle: rings, rune ticks, a hexagram and
// node gems, all additive.
func (g *Game) drawSigil(s *ebiten.Image, cx, cy, R, rot float64, col color.RGBA, a float64) {
	if a <= 0.01 {
		return
	}
	u := g.ui
	S := u.S
	X, Y := u.X(cx), u.Y(cy)
	ring := func(r, w, al float64) {
		strokePath(s, circlePath(X, Y, r*S), w*S, ca(col, al*a), true)
	}
	ring(R, 2, 0.8)
	ring(R*0.92, 1.2, 0.6)
	ring(R*0.6, 1.4, 0.6)
	ring(R*0.32, 1.2, 0.5)
	// rune ticks between the outer rings
	for i := 0; i < 48; i++ {
		an := rot + float64(i)/48*2*math.Pi
		l0, l1 := R*0.93, R*0.99
		if i%4 == 0 {
			l0 = R * 0.9
		}
		c, sn := math.Cos(an), math.Sin(an)
		strokePath(s, linePath(X+c*l0*S, Y+sn*l0*S, X+c*l1*S, Y+sn*l1*S), 1.3*S, ca(col, 0.7*a), true)
	}
	// hexagram turning the other way
	for tri := 0; tri < 2; tri++ {
		for k := 0; k < 3; k++ {
			a0 := -rot*1.4 + float64(tri)*math.Pi/3 + float64(k)*2*math.Pi/3
			a1 := a0 + 2*math.Pi/3
			r := R * 0.6
			strokePath(s, linePath(X+math.Cos(a0)*r*S, Y+math.Sin(a0)*r*S, X+math.Cos(a1)*r*S, Y+math.Sin(a1)*r*S), 1.4*S, ca(col, 0.55*a), true)
		}
	}
	for k := 0; k < 6; k++ {
		an := rot*2 + float64(k)*math.Pi/3
		px, py := X+math.Cos(an)*R*0.76*S, Y+math.Sin(an)*R*0.76*S
		fillPath(s, circlePath(px, py, 4*S), ca(lighten(col, 0.5), 0.9*a), ca(col, 0.9*a), grad{}, true)
		drawSprite(s, sprSoft, px-14*S, py-14*S, 28*S, 28*S, ca(col, 0.5*a), true)
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
	y += 180
	toggle := func(id, label string, v bool) bool {
		u.text(s, label, fBold, 22, 520, y+12, cWhite, alignLeft)
		if u.switchCtl(s, id, 1260, y+4, v) {
			v = !v
		}
		y += 58
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
	st.Classic = toggle("classic", "Classic backdrop (painted picture, no 3D scenery)", st.Classic)
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

var nodeGlyphs = map[NodeType]string{
	NCombat: "sword", NElite: "crown", NShop: "coin", NTreasure: "gem",
	NEvent: "eye", NRest: "flame", NBoss: "mask",
}

func (g *Game) nodeIcon(s *ebiten.Image, t NodeType, x, y, size float64) {
	g.ui.drawEmblemDark(s, glyphImage(nodeGlyphs[t], cWhite), x, y, size*1.25, 0.95)
}
