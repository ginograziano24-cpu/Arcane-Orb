package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Settings struct {
	Master     float64              `json:"master"`
	Music      float64              `json:"music"`
	SFX        float64              `json:"sfx"`
	Shake      bool                 `json:"shake"`
	Flashes    bool                 `json:"flashes"`
	Particles  int                  `json:"particles"` // 0 reduced, 1 full
	Fullscreen bool                 `json:"fullscreen"`
	VSync      bool                 `json:"vsync"`
	ShowFPS    bool                 `json:"show_fps"`
	DX12       bool                 `json:"dx12,omitempty"` // Windows: DirectX 12 instead of 11 (next launch)
	NoBloom    bool                 `json:"no_bloom"`
	Classic    bool                 `json:"classic_backdrop"` // flat painted picture instead of 3D scenery
	Quality    int                  `json:"quality"`          // 0 low, 1 medium, 2 high, 3 custom
	RenderRes  float64              `json:"render_scale"`     // 3D resolution, 0.5..1 (0 = full)
	DynRes     bool                 `json:"dynamic_res"`      // lower the 3D resolution when frames run long
	FastAnim   bool                 `json:"fast_animations"`  // shorter transitions, no slow-motion or hit-stop
	NoTips     bool                 `json:"no_tips"`          // hide first-time coach tips
	CenterHUD  bool                 `json:"center_hud"`       // ultrawide: keep the HUD in the 16:9 centre
	TextScale  float64              `json:"text_scale"`       // body text size, 1 = default (0 = default)
	GameSpeed  float64              `json:"game_speed"`       // chamber speed assist, 0.5..1 (0 = 1)
	ColorBlind int                  `json:"color_blind"`      // 0 off, 1 protanopia, 2 deuteranopia, 3 tritanopia
	Borderless bool                 `json:"borderless"`       // windowed without a frame, filling the monitor
	Binds      map[string]savedBind `json:"binds,omitempty"`
	Lang       string               `json:"lang,omitempty"` // language code; "" = pick from Steam or the system
}

// EndlessEntry is one line of the local Endless leaderboard.
type EndlessEntry struct {
	Score int    `json:"score"`
	Loop  int    `json:"loop"`
	Depth int    `json:"depth"`
	Class string `json:"class"`
	Asc   int    `json:"asc"`
	Date  string `json:"date"`
}

// addToBoard inserts an entry and returns its 1-based rank (0 = didn't place).
func (s *SaveData) addToBoard(e EndlessEntry) int {
	pos := len(s.Board)
	for i, b := range s.Board {
		if e.Score > b.Score {
			pos = i
			break
		}
	}
	if pos >= 10 {
		return 0
	}
	s.Board = append(s.Board, EndlessEntry{})
	copy(s.Board[pos+1:], s.Board[pos:])
	s.Board[pos] = e
	if len(s.Board) > 10 {
		s.Board = s.Board[:10]
	}
	return pos + 1
}

type SaveData struct {
	Version    int             `json:"version"`
	Shards     int             `json:"shards"`
	Upg        map[string]int  `json:"meta_upgrades"`
	Unlocked   map[string]bool `json:"unlocked"`
	Runs       int             `json:"runs"`
	Wins       int             `json:"wins"`
	BestAct    int             `json:"best_act"`
	BestScore  int             `json:"best_score"`
	Bosses     int             `json:"bosses"`
	Bricks     int             `json:"bricks"`
	MaxCombo   int             `json:"max_combo"`
	Daily      map[string]int  `json:"daily_best"`
	Seen       map[string]bool `json:"seen_mods"`
	XP         int             `json:"xp"`
	AscMax     int             `json:"asc_max"` // highest ascension unlocked
	AscSel     int             `json:"asc"`     // ascension chosen for new runs
	Fusions    map[string]bool `json:"fusions"` // fusions ever formed
	TotalGold  int             `json:"total_gold"`
	BestLoop   int             `json:"best_endless_loop"`
	Board      []EndlessEntry  `json:"endless_board"` // local top-10
	StorySeen  bool            `json:"story_seen"`
	Ach        map[string]bool `json:"achievements"`           // unlocked achievement API names
	Elites     int             `json:"elites"`                 // elite bricks broken, all runs
	History    []runHistory    `json:"history,omitempty"`      // finished runs, newest first
	Challenges map[string]int  `json:"challenges"`             // best act reached per challenge (4 = sealed)
	Hints      map[string]bool `json:"hints"`                  // coach tips already shown
	Mastery    map[string]int  `json:"mastery,omitempty"`      // XP earned per origin
	MOpts      map[string]int  `json:"mastery_opts,omitempty"` // mastery options switched on, per origin
	Settings   Settings        `json:"settings"`
}

func defaultSave() SaveData {
	return SaveData{Version: 2, Upg: map[string]int{}, Unlocked: map[string]bool{}, Daily: map[string]int{}, Seen: map[string]bool{}, Fusions: map[string]bool{}, Ach: map[string]bool{}, Challenges: map[string]int{},
		Settings: Settings{Master: 1, Music: 0.6, SFX: 0.8, Shake: true, Flashes: true, Particles: 1, VSync: true, Fullscreen: true, Quality: 2, RenderRes: 1, DynRes: true}}
}

func savePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "arcane_orb_save.json"
	}
	return filepath.Join(dir, "ArcaneOrb", "save.json")
}

// loadSave never fails: a missing or corrupted file yields defaults
// (a corrupt file is kept as save.json.bak for recovery).
func loadSave() SaveData {
	s := defaultSave()
	b, err := os.ReadFile(savePath())
	if err != nil {
		return s
	}
	if err := json.Unmarshal(b, &s); err != nil {
		_ = os.WriteFile(savePath()+".bak", b, 0o644)
		return defaultSave()
	}
	if s.Upg == nil {
		s.Upg = map[string]int{}
	}
	if s.Unlocked == nil {
		s.Unlocked = map[string]bool{}
	}
	if s.Daily == nil {
		s.Daily = map[string]int{}
	}
	if s.Seen == nil {
		s.Seen = map[string]bool{}
	}
	if s.Fusions == nil {
		s.Fusions = map[string]bool{}
	}
	if s.Challenges == nil {
		s.Challenges = map[string]int{}
	}
	if s.Ach == nil {
		s.Ach = map[string]bool{}
	}
	s.AscSel = max(0, min(s.AscSel, s.AscMax))
	return s
}

func writeSave(s *SaveData) {
	p := savePath()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	b, _ := json.MarshalIndent(s, "", "  ")
	tmp := p + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		_ = os.Rename(tmp, p)
	}
}
