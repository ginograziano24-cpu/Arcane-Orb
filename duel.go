package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"orb/levelgen"
	"orb/netplay"
)

// Duels: two players race through the same seeded chambers, best of
// three. Both carry the same origin and the same fixed kit, each sees the
// other's board live, and the host decides each round.

const (
	duelRounds    = 3 // best of three
	duelToWin     = 2
	duelCountdown = 3.0
	duelResultT   = 4.0
	duelTimeout   = 12.0 // seconds of silence before the opponent counts as gone
	duelSnapEvery = 1.0 / 12
)

// duelKit is what both players carry: each round adds two more items.
var duelKit = [duelRounds][]string{{"heavy", "wide"}, {"twin", "kindling"}, {"static", "blast"}}

// neutralMeta keeps Sanctum upgrades out of duels so both sides are equal.
var neutralMeta = defaultSave()

type duelRoom struct {
	Name, Code, Addr string
	id               uint64
}

type duelMsg struct {
	T  string    `json:"t"`
	V  int       `json:"v,omitempty"`
	N  string    `json:"n,omitempty"`
	A  string    `json:"a,omitempty"`
	R  bool      `json:"r,omitempty"`
	Rd int       `json:"rd,omitempty"`
	S  int64     `json:"s,omitempty"`
	W  int       `json:"w,omitempty"`
	Ws int       `json:"ws,omitempty"`
	L  int       `json:"l,omitempty"`
	St int       `json:"st,omitempty"`
	PX float32   `json:"px,omitempty"`
	PW float32   `json:"pw,omitempty"`
	B  []float32 `json:"b,omitempty"`
	C  string    `json:"c,omitempty"`
	K  string    `json:"k,omitempty"`
	Sh float32   `json:"sh,omitempty"`
	Cb int       `json:"cb,omitempty"`
	Tm float64   `json:"tm,omitempty"`
	Wn int       `json:"wn,omitempty"`
	Wi [2]int    `json:"wi,omitempty"`
	P  int64     `json:"p,omitempty"`
	F  bool      `json:"f,omitempty"`
}

// ghost is what we know of the opponent's board.
type ghost struct {
	wave, waves, left, start int
	px, pw, shift            float64
	balls                    []float32
	cells                    [levelgen.MaxRows * levelgen.Cols]byte
	colors                   [levelgen.MaxRows * levelgen.Cols]color.RGBA
	combo                    int
	done                     bool
	pulse                    float64
}

type duel struct {
	// connection
	link       netplay.Link
	via        string
	host       bool
	myName     string
	oppName    string
	oppVer     int
	lastRecv   time.Time
	helloAt    time.Time
	pingAt     time.Time
	setupAt    time.Time
	ping       float64
	server     *netplay.Server
	dialer     *netplay.Dialer
	browser    *netplay.Browser
	lanCode    string
	wanCode    string
	wanCh      chan string
	steamCode  string
	steamRooms []duelRoom
	status     string
	steamWant  bool // waiting on a Steam room
	inputs     [2]string
	focus      int // 0 none, 1 steam code, 2 direct code
	// lobby
	archIdx  int
	ready    bool
	oppReady bool
	// match
	phase      string // "connect", "lobby", "countdown", "play", "roundover", "over"
	t          float64
	phaseEnd   time.Time
	playStart  time.Time
	round      int
	seed       int64
	wins       [2]int
	times      [duelRounds][2]float64
	winners    [duelRounds]int
	resolved   bool
	finished   bool
	elapsed    float64
	snapT      float64
	waveSent   int
	layoutT    float64
	opp        ghost
	oppFinT    float64
	menuOpen   bool
	rematch    bool
	oppRematch bool
	note       string // why the match ended early
	lost       bool   // connection lost
}

func (d *duel) me() int {
	if d.host {
		return 0
	}
	return 1
}

func duelName() string {
	if steamActive() {
		if n := steamPersona(steamMyID()); n != "" && n != "Player" {
			return n
		}
	}
	for _, k := range []string{"USERNAME", "USER"} {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return fmt.Sprintf("Player %d", 100+rand.Intn(900))
}

// openDuel enters the duel screen.
func (g *Game) openDuel() {
	g.duel = &duel{phase: "connect", myName: duelName()}
	for i, a := range archetypes {
		if a.ID == "arcane" {
			g.duel.archIdx = i
		}
	}
	g.duel.browser = netplay.Browse()
	g.setState(StDuel)
}

// leaveDuel closes everything and returns to the menu.
func (g *Game) leaveDuel() {
	d := g.duel
	if d != nil {
		if d.link != nil {
			d.send(duelMsg{T: "bye"}, true)
			d.link.Close()
		}
		if d.server != nil {
			d.server.Close()
		}
		d.browser.Close()
		if d.via == "steam" || d.steamWant {
			steamDuelLeave()
		}
	}
	g.duel = nil
	g.run, g.level = nil, nil
	g.setState(StMenu)
}

func (d *duel) send(m duelMsg, reliable bool) {
	if d.link == nil {
		return
	}
	b, err := json.Marshal(m)
	if err != nil {
		return
	}
	d.link.Send(b, reliable)
}

// attach adopts a fresh link to the other player.
func (d *duel) attach(l netplay.Link, via string, host bool) {
	d.link, d.via, d.host = l, via, host
	d.lastRecv = time.Now()
	d.status = "Connected, saying hello…"
	if d.server != nil && !host {
		d.server.Close()
		d.server = nil
	}
}

// ---------------------------------------------------------------- hosting & joining

func (d *duel) hostDirect() {
	if d.server != nil {
		return
	}
	s, err := netplay.Host(netplay.DefaultPort, d.myName)
	if err != nil {
		d.status = "Couldn't open port 7777 (is another copy hosting?)"
		return
	}
	d.server = s
	if ips := netplay.LANAddrs(); len(ips) > 0 {
		d.lanCode = netplay.EncodeAddr(ips[0], s.Port)
	}
	d.status = "Waiting for a player to join…"
	d.wanCh = make(chan string, 1)
	go func(ch chan string, port int) { ch <- publicCode(port) }(d.wanCh, s.Port)
}

func (d *duel) joinDirect(target string) {
	if strings.TrimSpace(target) == "" {
		d.status = "Type a room code or an address first"
		return
	}
	d.dialer = netplay.Join(target)
	d.status = "Connecting…"
}

func (d *duel) hostRelay(code string) {
	if l := netplay.Relay(relayURL(code)); l != nil {
		d.attach(l, "relay", true)
		d.lanCode = code
		d.status = "Waiting on the relay for room " + code + "…"
	}
}

func (d *duel) joinRelay(code string) {
	if l := netplay.Relay(relayURL(code)); l != nil {
		d.attach(l, "relay", false)
	}
}

func relayURL(code string) string {
	return "ws://localhost:8768/room/" + strings.ToUpper(strings.TrimSpace(code))
}

// ---------------------------------------------------------------- update

func (g *Game) updateDuel(dt float64) {
	d := g.duel
	if d == nil {
		return
	}
	// background connection work
	if d.server != nil && d.link == nil {
		if l, err := d.server.Poll(); l != nil {
			d.attach(l, "direct", true)
		} else if err != nil {
			d.status = "Hosting stopped: " + err.Error()
		}
	}
	if d.dialer != nil && d.link == nil {
		if l, done, err := d.dialer.Poll(); done {
			d.dialer = nil
			if err != nil {
				d.status = "Couldn't connect: " + shortErr(err)
			} else {
				d.attach(l, "direct", false)
			}
		}
	}
	if d.wanCh != nil {
		select {
		case c := <-d.wanCh:
			d.wanCode, d.wanCh = c, nil
		default:
		}
	}
	if d.steamWant || d.via == "steam" {
		l, code, rooms, err := steamDuelPoll(d.myName)
		d.steamCode, d.steamRooms = code, rooms
		if err != "" && d.link == nil {
			d.status = err
		}
		if l != nil && d.link == nil {
			// the lobby owner hosts
			d.attach(l, "steam", sNetHost())
		}
	}
	if d.link == nil {
		return
	}
	// traffic
	for _, raw := range d.link.Recv() {
		var m duelMsg
		if json.Unmarshal(raw, &m) != nil {
			continue
		}
		d.lastRecv = time.Now()
		g.duelHandle(m)
		if g.duel == nil {
			return
		}
	}
	// heartbeats run on the wall clock so a slow frame can't starve them
	now := time.Now()
	if now.After(d.helloAt) {
		d.helloAt = now.Add(500 * time.Millisecond)
		if d.oppName == "" {
			d.send(duelMsg{T: "hello", V: netplay.Version, N: d.myName}, true)
		}
	}
	if now.After(d.pingAt) {
		d.pingAt = now.Add(time.Second)
		d.send(duelMsg{T: "ping", P: now.UnixMilli()}, false)
	}
	if d.host && d.phase == "lobby" && now.After(d.setupAt) {
		d.setupAt = now.Add(time.Second)
		d.send(duelMsg{T: "setup", A: archetypes[d.archIdx].ID}, true)
	}
	silent := time.Since(d.lastRecv).Seconds()
	if !d.link.Alive() || (d.oppName != "" && silent > duelTimeout) {
		duelLog("link lost: alive=%v silent=%.1fs phase=%s", d.link.Alive(), silent, d.phase)
		g.duelLost()
		return
	}
	g.updateDuelMatch(dt)
}

func shortErr(err error) string {
	s := err.Error()
	if i := strings.LastIndex(s, ": "); i >= 0 {
		s = s[i+2:]
	}
	return s
}

// duelLost handles the link dropping.
func (g *Game) duelLost() {
	d := g.duel
	d.link.Close()
	d.lost = true
	if d.phase == "countdown" || d.phase == "play" || d.phase == "roundover" {
		d.note = d.oppNameOr() + " disconnected. The match is yours."
		d.wins[d.me()] = duelToWin
		d.phase = "over"
		g.setState(StDuel)
		return
	}
	d.phase = "connect"
	d.oppName, d.oppReady, d.ready = "", false, false
	d.link = nil
	d.status = "The other player left"
}

func (d *duel) oppNameOr() string {
	if d.oppName == "" {
		return "Your opponent"
	}
	return d.oppName
}

func (g *Game) duelHandle(m duelMsg) {
	d := g.duel
	switch m.T {
	case "hello":
		if m.V != netplay.Version {
			d.status = "That player has a different version of the game"
			d.link.Close()
			return
		}
		first := d.oppName == ""
		d.oppName, d.oppVer = m.N, m.V
		if first {
			d.send(duelMsg{T: "hello", V: netplay.Version, N: d.myName}, true)
			if d.phase == "connect" {
				d.phase = "lobby"
				g.snd.Play("achievement")
			}
		}
	case "ping":
		d.send(duelMsg{T: "pong", P: m.P}, false)
	case "pong":
		d.ping = float64(time.Now().UnixMilli()-m.P) / 2
	case "setup":
		if !d.host {
			for i, a := range archetypes {
				if a.ID == m.A {
					d.archIdx = i
				}
			}
		}
	case "ready":
		d.oppReady = m.R
	case "start":
		if !d.host {
			for i, a := range archetypes {
				if a.ID == m.A {
					d.archIdx = i
				}
			}
			if m.Rd == 1 {
				d.wins = [2]int{}
			}
			g.duelBeginRound(m.Rd, m.S)
		}
	case "layout":
		d.opp.wave, d.opp.waves = m.W, m.Ws
		if b, err := hex.DecodeString(m.K); err == nil {
			for i := range d.opp.colors {
				if 2*i+1 < len(b) {
					d.opp.colors[i] = unpack565(uint16(b[2*i])<<8 | uint16(b[2*i+1]))
				}
			}
		}
	case "snap":
		if m.Rd != d.round {
			return
		}
		o := &d.opp
		if m.W > o.wave {
			o.pulse = 1
		}
		o.wave, o.waves, o.left, o.start = m.W, m.Ws, m.L, m.St
		o.px, o.pw, o.shift, o.balls, o.combo = float64(m.PX), float64(m.PW), float64(m.Sh), m.B, m.Cb
		for i := 0; i < len(m.C) && i < len(o.cells); i++ {
			o.cells[i] = hexVal(m.C[i])
		}
		o.done = m.F
	case "finish":
		o := &d.opp
		o.done = true
		d.oppFinT = m.Tm
		if d.host {
			g.duelResolve(1, m.Tm, m.Rd)
		}
	case "result":
		if !d.host {
			g.duelApplyResult(m.Rd, m.Wn, m.Wi, m.Tm)
		}
	case "over":
		if !d.host {
			d.wins = m.Wi
			g.duelMatchOver()
		}
	case "ff":
		d.note = d.oppNameOr() + " forfeited."
		d.wins[d.me()] = duelToWin
		g.duelMatchOver()
	case "rematch":
		d.oppRematch = m.R
	case "bye":
		duelLog("opponent said bye")
		g.duelLost()
	}
}

func hexVal(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	}
	return 0
}

func pack565(c color.RGBA) uint16 {
	return uint16(c.R>>3)<<11 | uint16(c.G>>2)<<5 | uint16(c.B>>3)
}

func unpack565(v uint16) color.RGBA {
	r, g, b := uint8(v>>11)<<3, uint8(v>>5&63)<<2, uint8(v&31)<<3
	return color.RGBA{r | r>>5, g | g>>6, b | b>>5, 255}
}

// ---------------------------------------------------------------- rounds

// duelBeginRound builds the round's run and chamber; both sides do the
// same thing from the same seed, so their chambers match.
func (g *Game) duelBeginRound(round int, seed int64) {
	d := g.duel
	d.round, d.seed = round, seed
	d.resolved, d.finished, d.elapsed = false, false, 0
	d.opp = ghost{waves: 2}
	d.oppFinT = 0
	d.waveSent = 0
	d.menuOpen = false
	d.rematch, d.oppRematch = false, false
	arch := &archetypes[d.archIdx]
	r := NewRun(fmt.Sprintf("DUEL%d", seed), arch, &neutralMeta, false)
	r.Duel = true
	r.startKit(&neutralMeta)
	for i := 0; i < round && i < duelRounds; i++ {
		for _, id := range duelKit[i] {
			if modByID[id] != nil {
				r.Add(id, &neutralMeta)
			}
		}
	}
	r.Act = 1 + (round-1)%numActs
	r.ActDepth = 3
	r.NewFusions = nil
	g.run = r
	g.hero = heroPortrait{}
	g.toasts = nil
	g.level = g.newLevel(NCombat, seed+int64(round)*7919)
	d.phase, d.t = "countdown", duelCountdown
	d.phaseEnd = time.Now().Add(duelCountdown * time.Second)
	g.setState(StLevel)
	g.snd.Play("mapenter")
}

// duelResolve is the host deciding a round (first finisher wins).
func (g *Game) duelResolve(who int, t float64, round int) {
	d := g.duel
	if d.resolved || round != d.round {
		return
	}
	d.wins[who]++
	d.send(duelMsg{T: "result", Rd: round, Wn: who, Wi: d.wins, Tm: t}, true)
	g.duelApplyResult(round, who, d.wins, t)
}

func (g *Game) duelApplyResult(round, who int, wins [2]int, t float64) {
	d := g.duel
	d.resolved = true
	d.wins = wins
	if round >= 1 && round <= duelRounds {
		d.winners[round-1] = who
		d.times[round-1][who] = t
	}
	d.phase, d.t = "roundover", duelResultT
	d.phaseEnd = time.Now().Add(duelResultT * time.Second)
	if who == d.me() {
		g.snd.PlayKeyed("clear")
		g.react(reactClear)
	} else {
		g.snd.Play("lose")
		g.react(reactHurt)
	}
}

func (g *Game) duelMatchOver() {
	d := g.duel
	d.phase = "over"
	d.rematch, d.oppRematch = false, false
	if d.wins[d.me()] >= duelToWin {
		g.snd.PlayKeyed("clear")
	}
	g.setState(StDuel)
}

// updateDuelMatch runs the round clock; it reports whether the level
// should be frozen this frame.
func (g *Game) updateDuelMatch(dt float64) {
	d := g.duel
	switch d.phase {
	case "countdown":
		// phase clocks run on wall time so both players stay in step
		was := d.t
		d.t = time.Until(d.phaseEnd).Seconds()
		if math.Ceil(was) != math.Ceil(d.t) && d.t > 0 {
			g.snd.PlayVol("thump", 0.9)
		}
		if d.t <= 0 {
			d.phase = "play"
			d.playStart = time.Now()
			g.snd.Play("launch")
			g.snd.PlayVol("abilready", 0.8)
		}
	case "play":
		d.elapsed = time.Since(d.playStart).Seconds()
		if L := g.level; L != nil && L.Cleared && !d.finished {
			d.finished = true
			d.send(duelMsg{T: "finish", Rd: d.round, Tm: d.elapsed}, true)
			if d.host {
				g.duelResolve(0, d.elapsed, d.round)
			}
		}
	case "roundover":
		d.t = time.Until(d.phaseEnd).Seconds()
		if d.t <= 0 && d.host {
			if d.wins[0] >= duelToWin || d.wins[1] >= duelToWin || d.round >= duelRounds {
				d.send(duelMsg{T: "over", Wi: d.wins}, true)
				g.duelMatchOver()
			} else {
				d.send(duelMsg{T: "start", Rd: d.round + 1, S: d.seed, A: archetypes[d.archIdx].ID}, true)
				g.duelBeginRound(d.round+1, d.seed)
			}
		}
	case "over":
		if d.host && d.rematch && d.oppRematch {
			seed := randSeed()
			d.wins = [2]int{}
			d.times = [duelRounds][2]float64{}
			d.note = ""
			d.send(duelMsg{T: "start", Rd: 1, S: seed, A: archetypes[d.archIdx].ID}, true)
			g.duelBeginRound(1, seed)
		}
	}
	// share the board
	if L := g.level; L != nil && (d.phase == "play" || d.phase == "countdown" || d.phase == "roundover") && g.state == StLevel {
		if L.wave != d.waveSent {
			d.waveSent = L.wave
			d.send(duelMsg{T: "layout", W: L.wave, Ws: L.waves, K: duelColors(L)}, true)
		}
		d.snapT -= dt
		if d.snapT <= 0 {
			d.snapT = duelSnapEvery
			d.send(duelSnap(d, L), false)
		}
	}
	d.opp.pulse = math.Max(0, d.opp.pulse-dt*2)
}

// duelFrozen reports whether the level should hold still (countdown,
// after a round is decided).
func (g *Game) duelFrozen() bool {
	d := g.duel
	if d == nil {
		return false
	}
	switch d.phase {
	case "countdown":
		return true
	case "roundover":
		return g.level == nil || !g.level.Cleared // the winner's finale plays out
	}
	return false
}

func duelColors(L *Level) string {
	b := make([]byte, 0, levelgen.MaxRows*levelgen.Cols*2)
	for r := 0; r < levelgen.MaxRows; r++ {
		for c := 0; c < levelgen.Cols; c++ {
			var v uint16
			if br := L.Grid[r][c]; br != nil {
				v = pack565(br.Col)
				if v == 0 {
					v = 1
				}
			}
			b = append(b, byte(v>>8), byte(v))
		}
	}
	return hex.EncodeToString(b)
}

func duelSnap(d *duel, L *Level) duelMsg {
	var cells strings.Builder
	for r := 0; r < levelgen.MaxRows; r++ {
		for c := 0; c < levelgen.Cols; c++ {
			n := 0
			for b := L.Grid[r][c]; b != nil; b = b.Up {
				if b.Alive {
					n++
				}
			}
			cells.WriteByte("0123456789abcdef"[min(n, 15)])
		}
	}
	m := duelMsg{T: "snap", Rd: d.round, W: L.wave, Ws: L.waves, L: L.Left, St: L.startLeft,
		PX: float32(L.PaddleX), PW: float32(L.PaddleW), C: cells.String(), Sh: float32(L.shift), Cb: L.Combo, F: L.Cleared}
	for i, b := range L.Balls {
		if i >= 6 {
			break
		}
		m.B = append(m.B, float32(b.X), float32(b.Z))
	}
	return m
}

// progress is how far through the chamber a board is (0..1).
func duelProgress(wave, waves, left, start int, done bool) float64 {
	if done {
		return 1
	}
	if waves <= 0 {
		waves = 1
	}
	f := 0.0
	if start > 0 {
		f = clamp01(float64(start-left) / float64(start))
	}
	return clamp01((float64(max(1, wave)-1) + f) / float64(waves))
}

// ---------------------------------------------------------------- input on the duel screen

func (g *Game) updateDuelScreen() {
	d := g.duel
	if d == nil {
		return
	}
	if d.focus > 0 {
		i := d.focus - 1
		for _, r := range ebiten.AppendInputChars(nil) {
			if len(d.inputs[i]) < 24 && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '.' || r == ':') {
				d.inputs[i] += strings.ToUpper(string(r))
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(d.inputs[i]) > 0 {
			d.inputs[i] = d.inputs[i][:len(d.inputs[i])-1]
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.duelJoinField(d.focus)
		}
	}
	if backPressed() {
		if d.focus > 0 {
			d.focus = 0
			return
		}
		g.leaveDuel()
	}
}

func (g *Game) duelJoinField(f int) {
	d := g.duel
	switch {
	case f == 1 && relayBuild:
		d.joinRelay(d.inputs[0])
	case f == 1:
		code := strings.ToUpper(strings.TrimSpace(d.inputs[0]))
		if code == "" {
			d.status = "Type the room code first"
			return
		}
		d.steamWant = true
		steamDuelList(code)
		d.status = "Looking for room " + code + "…"
	case f == 2:
		d.joinDirect(d.inputs[1])
	}
	d.focus = 0
}

func randSeed() int64 { return rand.Int63n(1 << 40) }

func duelLog(f string, a ...any) { fmt.Fprintf(os.Stderr, "duel: "+f+"\n", a...) }
