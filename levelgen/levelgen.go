// Package levelgen builds seeded procedural brick layouts. It has no
// graphics dependencies so it can be unit-tested and reused by tools.
package levelgen

import (
	"fmt"
	"math"
	"math/rand"
)

const (
	lCols = 14
	lRows = 16
)

// DesignChance is how often a level uses a hand-drawn showcase design.
var DesignChance = 0.38

type Kind uint8

const (
	Empty      Kind = iota
	Basic           // 1-3 HP
	Steel           // indestructible
	Armored         // only damaged by crits, explosions, elements and void
	Explosive       // blows up its neighbours
	Chain           // arcs lightning to nearby bricks when destroyed
	Split           // breaks into smaller bricks
	Regen           // heals when left alone
	Moving          // slides between free cells
	Shield          // protects its neighbours
	Teleport        // hops to a random free cell
	Spawner         // grows new bricks around itself
	Multiplier      // triggers a frenzy (double gold and score)
	Crystal         // drops a burst of gold
	Mystery         // random effect
	Elite           // very tough, very rewarding
	NumKinds
)

var KindNames = [...]string{"Empty", "Basic", "Steel", "Armored", "Explosive", "Chain", "Split", "Regen",
	"Moving", "Shield", "Teleport", "Spawner", "Multiplier", "Crystal", "Mystery", "Elite"}

// Required reports whether a brick of this kind must be destroyed to clear a level.
func (k Kind) Required() bool { return k != Empty && k != Steel }

type Objective uint8

const (
	ObjClear Objective = iota
	ObjCrystals
)

type Cell struct {
	Kind  Kind
	HP    int
	Color int  // palette index 0..5
	Stack int  // bricks in this column (>=1 when not empty)
	Cap   Kind // optional special brick on top of a stack
}

type Params struct {
	Seed  int64
	Depth int // 1-based level index across the whole run
	Act   int // 1-based
	Elite bool
}

type coarse struct {
	Name       string
	Formation  string
	Structure  string
	Rows       int
	Cells      [lRows][lCols]Cell
	Outside    [lRows][lCols]bool // cells clipped away by the arena walls
	Objective  Objective
	Seed       int64
	Shape      int
	LayerShift int // colour step between stacked layers
}

func (l *coarse) Count(pred func(Cell) bool) int {
	n := 0
	for r := 0; r < lRows; r++ {
		for c := 0; c < lCols; c++ {
			if pred(l.Cells[r][c]) {
				n++
			}
		}
	}
	return n
}

func (l *coarse) Required() int {
	n := 0
	for r := 0; r < lRows; r++ {
		for c := 0; c < lCols; c++ {
			if cl := l.Cells[r][c]; cl.Kind.Required() {
				n += max(1, cl.Stack)
			}
		}
	}
	return n
}

// Mix derives a well-distributed seed from several integers.
func Mix(vals ...int64) int64 {
	x := uint64(0x9E3779B97F4A7C15)
	for _, v := range vals {
		x ^= uint64(v) + 0x9E3779B97F4A7C15 + (x << 6) + (x >> 2)
		x ^= x >> 30
		x *= 0xBF58476D1CE4E5B9
		x ^= x >> 27
		x *= 0x94D049BB133111EB
		x ^= x >> 31
	}
	return int64(x & 0x7FFFFFFFFFFFFFFF)
}

// generateCoarse deterministically builds a level from its parameters.
func generateCoarse(p Params) *coarse {
	for attempt := 0; attempt < 80; attempt++ {
		r := rand.New(rand.NewSource(Mix(p.Seed, int64(p.Depth), int64(attempt))))
		lv := genOnce(r, p)
		if lv.Required() >= 16 {
			lv.Seed = p.Seed
			return lv
		}
	}
	lv := &coarse{Name: "The Plain Wall", Formation: "wall", Rows: 6, Seed: p.Seed}
	for r := 1; r < 6; r++ {
		for c := 0; c < lCols; c++ {
			lv.Cells[r][c] = Cell{Kind: Basic, HP: 1, Color: r % 6, Stack: 1}
		}
	}
	return lv
}

var adjectives = []string{
	"Whispering", "Ember", "Frozen", "Gilded", "Hollow", "Shattered", "Moonlit",
	"Thorned", "Sunken", "Crimson", "Verdant", "Starfallen", "Howling", "Silent",
	"Obsidian", "Forgotten", "Glimmering", "Twisted", "Ancient", "Drifting",
	"Burning", "Azure", "Haunted", "Radiant", "Crumbling", "Veiled", "Humming", "Fractured",
}
var nouns = []string{
	"Vault", "Spire", "Labyrinth", "Sanctum", "Gate", "Grove", "Citadel",
	"Crypt", "Observatory", "Bastion", "Archive", "Forge", "Cloister",
	"Reliquary", "Keep", "Hollows", "Altar", "Library", "Garden",
	"Catacombs", "Tower", "Bridge", "Caverns", "Hall", "Orrery", "Engine", "Furnace",
}

type mask [lRows][lCols]bool

func (m *mask) count() int {
	n := 0
	for r := range m {
		for c := range m[r] {
			if m[r][c] {
				n++
			}
		}
	}
	return n
}

func mirror(m *mask) {
	for r := 0; r < lRows; r++ {
		for c := 0; c < 7; c++ {
			m[r][13-c] = m[r][c]
		}
	}
}

func combine(a, b mask, f func(a, b bool) bool) mask {
	var o mask
	for r := 0; r < lRows; r++ {
		for c := 0; c < lCols; c++ {
			o[r][c] = f(a[r][c], b[r][c])
		}
	}
	return o
}

func genOnce(r *rand.Rand, p Params) *coarse {
	d := math.Min(1, float64(p.Depth-1)/22) // difficulty 0..1
	lv := &coarse{}
	lv.Name = fmt.Sprintf("The %s %s", adjectives[r.Intn(len(adjectives))], nouns[r.Intn(len(nouns))])

	// showcase designs: hand-drawn castles, creatures, faces and objects,
	// mixed in among generated layouts
	if p.Depth > 1 && r.Float64() < DesignChance {
		idx := r.Intn(len(Designs))
		if applyDesign(r, lv, p, d, idx) {
			designShape(r, lv)
			sym := Designs[idx].Half
			pool := []Kind{Explosive, Crystal}
			if p.Depth >= 2 {
				pool = append(pool, Mystery, Multiplier)
			}
			if p.Depth >= 3 {
				pool = append(pool, Chain, Split)
			}
			if p.Depth >= 4 {
				pool = append(pool, Regen, Armored)
			}
			for i := 0; i < 1+int(d*4); i++ {
				k := pool[r.Intn(len(pool))]
				hp := 1
				if k == Armored || k == Regen {
					hp = 2
				}
				place(r, lv, Cell{Kind: k, HP: hp, Stack: 1}, sym)
			}
			if p.Elite {
				for i := 0; i < 2; i++ {
					place(r, lv, Cell{Kind: Elite, HP: 5 + p.Depth/3, Stack: 1}, sym)
				}
			}
			for rr := 0; rr < lRows; rr++ {
				for c := 0; c < lCols; c++ {
					if lv.Outside[rr][c] {
						lv.Cells[rr][c] = Cell{}
					}
				}
			}
			ensureReachable(lv)
			return lv
		}
		*lv = coarse{Name: lv.Name}
	}

	rows := 10 + r.Intn(3) + int(d*3)
	if p.Elite {
		rows += 2
	}
	if rows > 14 {
		rows = 14
	}
	sym := r.Float64() < 0.78
	name, m := formation(r, rows, sym)
	lv.Formation = name
	if r.Float64() < 0.35 {
		n2, m2 := formation(r, rows, sym)
		switch r.Intn(4) {
		case 0:
			m = combine(m, m2, func(a, b bool) bool { return a && b })
			lv.Formation += " ∩ " + n2
		case 1:
			m = combine(m, m2, func(a, b bool) bool { return a || b })
			lv.Formation += " + " + n2
		case 2:
			m = combine(m, m2, func(a, b bool) bool { return a != b })
			lv.Formation += " ⊕ " + n2
		default:
			m = combine(m, m2, func(a, b bool) bool { return a && !b })
			lv.Formation += " − " + n2
		}
	}
	top := r.Intn(3)
	lv.Rows = min(lRows, rows+top)
	if p.Depth <= 1 {
		lv.Shape = ShapeHall
	} else {
		lv.Shape = r.Intn(NumShapes)
	}
	poly := Arena(lv.Shape)
	for rr := 0; rr < lRows; rr++ {
		for c := 0; c < lCols; c++ {
			lv.Outside[rr][c] = !lCellInside(poly, rr, c)
		}
	}

	// ---- colours and hit points ----
	colorMode := r.Intn(6)
	hpMode := r.Intn(5)
	maxHP := 1 + int(math.Round(d*2.2))
	if p.Elite {
		maxHP++
	}
	if maxHP > 4 {
		maxHP = 4
	}
	cx, cy := 6.5, float64(rows-1)/2
	maxDist := math.Hypot(cx, cy) + 0.01
	var blockColor [lRows][lCols]int
	for rr := 0; rr < lRows; rr += 2 {
		for c := 0; c < lCols; c += 2 {
			v := r.Intn(6)
			for dr := 0; dr < 2; dr++ {
				for dc := 0; dc < 2; dc++ {
					if rr+dr < lRows && c+dc < lCols {
						blockColor[rr+dr][c+dc] = v
					}
				}
			}
		}
	}
	if sym {
		for rr := 0; rr < lRows; rr++ {
			for c := 0; c < 7; c++ {
				blockColor[rr][13-c] = blockColor[rr][c]
			}
		}
	}
	colorDiv := 1 + r.Intn(2)
	for rr := 0; rr < rows; rr++ {
		for c := 0; c < lCols; c++ {
			if !m[rr][c] {
				continue
			}
			dc := math.Abs(float64(c) - cx)
			dist := math.Hypot(dc, float64(rr)-cy) / maxDist
			col := 0
			switch colorMode {
			case 0:
				col = (rr / colorDiv) % 6
			case 1:
				col = int(dc/1.2) % 6
			case 2:
				col = int(dist*6) % 6
			case 3:
				col = blockColor[rr][c]
			case 4:
				col = (rr + int(dc)) / colorDiv % 6
			default:
				col = (rows - 1 - rr) / colorDiv % 6
			}
			hp := 1
			switch hpMode {
			case 0:
				if r.Float64() < 0.1+0.35*d {
					hp = 2
				}
			case 1:
				frac := 1 - float64(rr)/float64(rows)
				hp = 1 + int(frac*float64(maxHP)*(0.4+0.6*d)+0.3)
			case 2:
				hp = 1 + int((1-dist)*float64(maxHP)*(0.5+0.5*d))
			case 3:
				if edge(m, rr, c, rows) {
					hp = 1 + int(float64(maxHP-1)*(0.5+0.5*d)+0.5)
				}
			default:
				if (rr/2)%2 == 0 {
					hp = 1 + int(d*float64(maxHP-1)+0.5)
				}
			}
			hp = max(1, min(maxHP, hp))
			if rr+top < lRows && !lv.Outside[rr+top][c] {
				lv.Cells[rr+top][c] = Cell{Kind: Basic, HP: hp, Color: col, Stack: 1}
			}
		}
	}

	// ---- steel features ----
	if r.Float64() < 0.1+0.55*d {
		addSteel(r, lv, top, rows, d, sym)
	}

	// ---- special bricks, unlocked gradually ----
	type spec struct {
		k        Kind
		minDepth int
		weight   float64
	}
	specs := []spec{
		{Explosive, 1, 1.2}, {Crystal, 1, 1.0}, {Mystery, 2, 0.8}, {Multiplier, 2, 0.5},
		{Split, 3, 0.8}, {Chain, 3, 0.8}, {Regen, 4, 0.7}, {Armored, 4, 0.9},
		{Moving, 6, 0.7}, {Shield, 6, 0.6}, {Teleport, 8, 0.5}, {Spawner, 8, 0.4},
	}
	budget := 2 + r.Intn(3) + int(d*6)
	if p.Elite {
		budget += 3
	}
	for i := 0; i < budget; i++ {
		total := 0.0
		for _, s := range specs {
			if p.Depth >= s.minDepth {
				total += s.weight
			}
		}
		x := r.Float64() * total
		var pick Kind = Explosive
		for _, s := range specs {
			if p.Depth < s.minDepth {
				continue
			}
			if x < s.weight {
				pick = s.k
				break
			}
			x -= s.weight
		}
		hp := 1
		switch pick {
		case Armored:
			hp = 2
		case Regen:
			hp = 2 + int(d*2)
		case Shield, Spawner:
			hp = 2
		}
		place(r, lv, Cell{Kind: pick, HP: hp, Stack: 1}, sym)
	}
	// elites
	if p.Elite || (p.Depth >= 5 && r.Float64() < 0.25) {
		n := 1
		if p.Elite {
			n = 2 + r.Intn(2)
		}
		for i := 0; i < n; i++ {
			place(r, lv, Cell{Kind: Elite, HP: 5 + p.Depth/3, Stack: 1}, sym)
		}
	}
	// crystal hunt objective
	if p.Depth >= 3 && !p.Elite && r.Float64() < 0.2 {
		lv.Objective = ObjCrystals
		need := 4 + r.Intn(3) - lv.Count(func(c Cell) bool { return c.Kind == Crystal })
		for i := 0; i < need*3 && need > 0; i++ {
			if place(r, lv, Cell{Kind: Crystal, HP: 1, Stack: 1}, sym) {
				need -= 2
				if !sym {
					need++
				}
			}
		}
	}
	buildStacks(r, lv, p, d, sym)
	// never leave bricks or steel outside the walls
	for rr := 0; rr < lRows; rr++ {
		for c := 0; c < lCols; c++ {
			if lv.Outside[rr][c] {
				lv.Cells[rr][c] = Cell{}
			}
		}
	}
	ensureReachable(lv)
	return lv
}

func edge(m mask, r, c, rows int) bool {
	for _, d := range dirs4 {
		rr, cc := r+d[0], c+d[1]
		if rr < 0 || rr >= rows || cc < 0 || cc >= lCols || !m[rr][cc] {
			return true
		}
	}
	return false
}

var dirs4 = [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

func set(lv *coarse, r, c int, cell Cell, sym bool) {
	lv.Cells[r][c] = cell
	if sym {
		lv.Cells[r][13-c] = cell
	}
}

// place puts a special cell on top of an existing basic brick.
func place(r *rand.Rand, lv *coarse, cell Cell, sym bool) bool {
	for i := 0; i < 40; i++ {
		rr, c := r.Intn(lRows), r.Intn(lCols)
		if sym {
			c = r.Intn(7)
		}
		if lv.Cells[rr][c].Kind == Basic {
			cell.Color = lv.Cells[rr][c].Color
			switch cell.Kind {
			case Moving, Teleport, Spawner, Elite:
				cell.Stack = 1
			default:
				cell.Stack = max(1, lv.Cells[rr][c].Stack)
			}
			set(lv, rr, c, cell, sym)
			return true
		}
	}
	return false
}

func formation(r *rand.Rand, rows int, sym bool) (string, mask) {
	var m mask
	cx, cy := 6.5, float64(rows-1)/2
	half := 7
	if !sym {
		half = lCols
	}
	finish := func() {
		if sym {
			mirror(&m)
		}
	}
	switch r.Intn(16) {
	case 0:
		p := 0.5 + r.Float64()*0.15
		for rr := 0; rr < rows; rr++ {
			for c := 0; c < half; c++ {
				m[rr][c] = r.Float64() < p
			}
		}
		finish()
		for it := 0; it < 2; it++ {
			var o mask
			for rr := 0; rr < rows; rr++ {
				for c := 0; c < lCols; c++ {
					cnt := 0
					for dr := -1; dr <= 1; dr++ {
						for dc := -1; dc <= 1; dc++ {
							a, b := rr+dr, c+dc
							if (dr != 0 || dc != 0) && a >= 0 && a < rows && b >= 0 && b < lCols && m[a][b] {
								cnt++
							}
						}
					}
					o[rr][c] = (m[rr][c] && cnt >= 3) || (!m[rr][c] && cnt >= 5)
				}
			}
			m = o
		}
		return "cave", m
	case 1:
		scale := 1 + r.Intn(2)
		for rr := 0; rr < rows; rr += scale {
			for c := 0; c < half; c++ {
				v := r.Float64() < 0.55
				for k := 0; k < scale && rr+k < rows; k++ {
					m[rr+k][c] = v
				}
			}
		}
		finish()
		return "glyph", m
	case 2:
		metric := r.Intn(3)
		hollow := r.Float64() < 0.5
		inner := 0.35 + r.Float64()*0.3
		a := 6.5 + r.Float64()*1.5
		b := float64(rows)/2 + 0.3
		names := []string{"diamond", "circle", "square"}
		for rr := 0; rr < rows; rr++ {
			for c := 0; c < lCols; c++ {
				dx := math.Abs(float64(c)-cx) / a
				dy := math.Abs(float64(rr)-cy) / b
				var dd float64
				switch metric {
				case 0:
					dd = dx + dy
				case 1:
					dd = math.Hypot(dx, dy)
				default:
					dd = math.Max(dx, dy)
				}
				m[rr][c] = dd <= 1 && (!hollow || dd >= inner)
			}
		}
		n := names[metric]
		if hollow {
			n = "hollow " + n
		}
		return n, m
	case 3:
		freq := 0.35 + r.Float64()*0.6
		ph := r.Float64() * math.Pi * 2
		amp := 1 + r.Float64()*float64(rows)/3
		thick := 1.0 + r.Float64()*1.5
		bands := 1 + r.Intn(2)
		for c := 0; c < lCols; c++ {
			x := float64(c)
			if sym {
				x = math.Abs(float64(c) - cx)
			}
			for b := 0; b < bands; b++ {
				mid := float64(rows) * float64(b+1) / float64(bands+1)
				y0 := mid + amp*math.Sin(x*freq+ph+float64(b))
				for rr := 0; rr < rows; rr++ {
					if math.Abs(float64(rr)-y0) <= thick {
						m[rr][c] = true
					}
				}
			}
		}
		return "waves", m
	case 4:
		inv := r.Intn(2) == 0
		slope := 0.6 + r.Float64()*0.8
		for rr := 0; rr < rows; rr++ {
			k := float64(rr)
			if inv {
				k = float64(rows - 1 - rr)
			}
			w := 0.5 + k*slope
			for c := 0; c < lCols; c++ {
				m[rr][c] = math.Abs(float64(c)-cx) <= w
			}
		}
		return "pyramid", m
	case 5:
		v := r.Intn(4)
		k := 2 + r.Intn(2)
		for rr := 0; rr < rows; rr++ {
			for c := 0; c < lCols; c++ {
				switch v {
				case 0:
					m[rr][c] = (rr+c)%2 == 0
				case 1:
					m[rr][c] = (rr/2+c/2)%2 == 0
				case 2:
					m[rr][c] = rr%k != k-1
				default:
					m[rr][c] = c%k != k-1
				}
			}
		}
		finish()
		return "checkerboard", m
	case 6:
		mod := 2 + r.Intn(2)
		for rr := 0; rr < rows; rr++ {
			for c := 0; c < lCols; c++ {
				ring := min(min(rr, rows-1-rr), min(c, 13-c))
				m[rr][c] = ring%mod != mod-1
			}
		}
		return "concentric", m
	case 7:
		bw, bh := 2+r.Intn(2), 1+r.Intn(2)
		off := r.Intn(2)
		for rr := 0; rr < rows; rr++ {
			for c := 0; c < half; c++ {
				m[rr][c] = (c+off)%(bw+1) != bw && rr%(bh+1) != bh
			}
		}
		finish()
		return "rooms", m
	case 8:
		k := 2 + r.Intn(3)
		mm := 1 + r.Intn(k-1)
		up := r.Intn(2) == 0
		for rr := 0; rr < rows; rr++ {
			for c := 0; c < lCols; c++ {
				dc := int(math.Abs(float64(c) - cx))
				v := rr + dc
				if up {
					v = rr - dc + 100
				}
				m[rr][c] = v%k < mm
			}
		}
		return "chevrons", m
	case 9:
		cnt := 3 + r.Intn(4)
		for i := 0; i < cnt; i++ {
			w, h := 1+r.Intn(3), 1+r.Intn(3)
			x, y := r.Intn(half), r.Intn(max(1, rows-h+1))
			for rr := y; rr < y+h && rr < rows; rr++ {
				for c := x; c < x+w && c < half; c++ {
					m[rr][c] = true
				}
			}
		}
		finish()
		return "islands", m
	case 10: // spiral
		turns := 1.5 + r.Float64()*1.5
		thick := 0.55 + r.Float64()*0.35
		dir := 1.0
		if r.Intn(2) == 0 {
			dir = -1
		}
		for rr := 0; rr < rows; rr++ {
			for c := 0; c < lCols; c++ {
				dx := (float64(c) - cx) / 7
				dy := (float64(rr) - cy) / (float64(rows) / 2)
				rad := math.Hypot(dx, dy)
				ang := math.Atan2(dy, dx)*dir + math.Pi
				// distance to spiral arm r = ang/(2π*turns) + k/turns
				t := rad*turns - ang/(2*math.Pi)
				frac := t - math.Floor(t)
				m[rr][c] = rad <= 1.05 && frac < thick
			}
		}
		return "spiral", m
	case 11: // maze (walls of a recursive-backtracker maze)
		gw, gh := 7, (rows+1)/2
		var vis [lRows][lCols]bool
		for rr := 0; rr < rows; rr++ {
			for c := 0; c < lCols; c++ {
				m[rr][c] = true
			}
		}
		type pt struct{ x, y int }
		stack := []pt{{r.Intn(gw), r.Intn(gh)}}
		vis[stack[0].y][stack[0].x] = true
		carve := func(x, y int) {
			if y*2 < rows && x*2 < lCols {
				m[y*2][x*2] = false
			}
		}
		carve(stack[0].x, stack[0].y)
		for len(stack) > 0 {
			cur := stack[len(stack)-1]
			var nb []pt
			for _, d := range dirs4 {
				nx, ny := cur.x+d[1], cur.y+d[0]
				if nx >= 0 && nx < gw && ny >= 0 && ny < gh && !vis[ny][nx] {
					nb = append(nb, pt{nx, ny})
				}
			}
			if len(nb) == 0 {
				stack = stack[:len(stack)-1]
				continue
			}
			n := nb[r.Intn(len(nb))]
			vis[n.y][n.x] = true
			wy, wx := cur.y+n.y, cur.x+n.x // wall cell between (in doubled coords)
			if wy < rows && wx < lCols {
				m[wy][wx] = false
			}
			carve(n.x, n.y)
			stack = append(stack, n)
		}
		return "maze", m
	case 12: // stairs
		step := 1 + r.Intn(2)
		w := 2 + r.Intn(2)
		for rr := 0; rr < rows; rr++ {
			for c := 0; c < half; c++ {
				v := (c/step - rr + 100) % (w + 2)
				m[rr][c] = v < w
			}
		}
		finish()
		return "stairs", m
	case 13: // towers
		for c := 0; c < half; c++ {
			if r.Float64() < 0.35 {
				continue
			}
			h := 2 + r.Intn(rows-1)
			for rr := rows - h; rr < rows; rr++ {
				m[rr][c] = true
			}
		}
		finish()
		return "towers", m
	case 14: // corridors: full rows with aligned gaps
		gap := 1 + r.Intn(3)
		every := 2 + r.Intn(2)
		gc := r.Intn(half - gap + 1)
		for rr := 0; rr < rows; rr++ {
			if rr%every == every-1 {
				continue
			}
			for c := 0; c < half; c++ {
				m[rr][c] = !(c >= gc && c < gc+gap)
			}
		}
		finish()
		return "corridors", m
	default: // arch
		th := 1.5 + r.Float64()*1.5
		for rr := 0; rr < rows; rr++ {
			for c := 0; c < lCols; c++ {
				dx := (float64(c) - cx) / 7
				dy := float64(rows-1-rr) / float64(rows)
				rad := math.Hypot(dx, dy*1.1)
				m[rr][c] = rad <= 1 && rad >= 1-th/7
			}
		}
		return "arch", m
	}
}

func addSteel(r *rand.Rand, lv *coarse, top, rows int, d float64, sym bool) {
	steel := Cell{Kind: Steel, HP: 1, Stack: 1}
	switch r.Intn(5) {
	case 0:
		row := min(lRows-1, top+rows-1-r.Intn(max(1, rows/3)))
		gap := 3 + r.Intn(3)
		for c := 0; c < 7; c++ {
			if c%gap == gap-1 || c == 6 {
				lv.Cells[row][c], lv.Cells[row][13-c] = Cell{}, Cell{}
			} else {
				lv.Cells[row][c], lv.Cells[row][13-c] = steel, steel
			}
		}
		lv.Rows = max(lv.Rows, row+1)
	case 1:
		for rr := top; rr < top+rows && rr < lRows; rr++ {
			for c := 0; c < lCols; c++ {
				if lv.Cells[rr][c].Kind != Basic {
					continue
				}
				isEdge := false
				for _, dd := range dirs4 {
					a, b := rr+dd[0], c+dd[1]
					if a < 0 || a >= lRows || b < 0 || b >= lCols || lv.Cells[a][b].Kind == Empty {
						isEdge = true
					}
				}
				if isEdge && r.Float64() < 0.45+0.3*d {
					lv.Cells[rr][c] = steel
					if sym {
						lv.Cells[rr][13-c] = steel
					}
				}
			}
		}
	case 2:
		c := 1 + r.Intn(5)
		y0 := top + r.Intn(max(1, rows/2))
		y1 := y0 + 2 + r.Intn(max(1, rows/2))
		for rr := y0; rr < y1 && rr < lRows; rr++ {
			lv.Cells[rr][c], lv.Cells[rr][13-c] = steel, steel
		}
	case 3:
		p := 0.05 + 0.08*d
		for rr := top; rr < top+rows && rr < lRows; rr++ {
			for c := 0; c < lCols; c++ {
				if lv.Cells[rr][c].Kind == Basic && r.Float64() < p {
					set(lv, rr, c, steel, sym)
				}
			}
		}
	default:
		for c := 0; c < 7; c++ {
			if r.Float64() > 0.5 {
				continue
			}
			for rr := lRows - 1; rr >= 0; rr-- {
				if lv.Cells[rr][c].Kind == Basic {
					lv.Cells[rr][c], lv.Cells[rr][13-c] = steel, steel
					break
				}
			}
		}
	}
}

// ensureReachable converts steel into tough bricks wherever steel would
// seal destructible bricks away from the ball.
func ensureReachable(lv *coarse) {
	passable := func(r, c int) bool { return lv.Cells[r][c].Kind != Steel && !lv.Outside[r][c] }
	for iter := 0; iter < 300; iter++ {
		var reach [lRows][lCols]bool
		var queue [][2]int
		for c := 0; c < lCols; c++ {
			if passable(lRows-1, c) {
				reach[lRows-1][c] = true
				queue = append(queue, [2]int{lRows - 1, c})
			}
		}
		topReached := false
		for len(queue) > 0 {
			p := queue[0]
			queue = queue[1:]
			if p[0] == 0 && !topReached {
				topReached = true
				for c := 0; c < lCols; c++ {
					if passable(0, c) && !reach[0][c] {
						reach[0][c] = true
						queue = append(queue, [2]int{0, c})
					}
				}
			}
			for _, dd := range dirs4 {
				a, b := p[0]+dd[0], p[1]+dd[1]
				if a < 0 || a >= lRows || b < 0 || b >= lCols || reach[a][b] || !passable(a, b) {
					continue
				}
				reach[a][b] = true
				queue = append(queue, [2]int{a, b})
			}
		}
		stuck := false
		for r := 0; r < lRows && !stuck; r++ {
			for c := 0; c < lCols; c++ {
				if lv.Cells[r][c].Kind.Required() && !reach[r][c] {
					stuck = true
					break
				}
			}
		}
		if !stuck {
			return
		}
		best, bestScore := [2]int{-1, -1}, -1
		for r := 0; r < lRows; r++ {
			for c := 0; c < lCols; c++ {
				if lv.Cells[r][c].Kind != Steel {
					continue
				}
				touchR, touchU := false, false
				for _, dd := range dirs4 {
					a, b := r+dd[0], c+dd[1]
					if a >= lRows {
						touchR = true
						continue
					}
					if a < 0 {
						touchR = touchR || topReached
						continue
					}
					if b < 0 || b >= lCols {
						continue
					}
					if reach[a][b] {
						touchR = true
					} else if lv.Cells[a][b].Kind != Steel {
						touchU = true
					}
				}
				score := 0
				if touchR {
					score = 1
					if touchU {
						score = 2
					}
				}
				if score > bestScore {
					bestScore, best = score, [2]int{r, c}
				}
			}
		}
		if best[0] < 0 {
			return
		}
		lv.Cells[best[0]][best[1]] = Cell{Kind: Basic, HP: 3, Stack: 1}
	}
}
