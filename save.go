package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Settings struct {
	Master     float64 `json:"master"`
	Music      float64 `json:"music"`
	SFX        float64 `json:"sfx"`
	Shake      bool    `json:"shake"`
	Flashes    bool    `json:"flashes"`
	Particles  int     `json:"particles"` // 0 reduced, 1 full
	Fullscreen bool    `json:"fullscreen"`
	VSync      bool    `json:"vsync"`
	ShowFPS    bool    `json:"show_fps"`
	NoBloom    bool    `json:"no_bloom"`
}

type SaveData struct {
	Version   int             `json:"version"`
	Shards    int             `json:"shards"`
	Upg       map[string]int  `json:"meta_upgrades"`
	Unlocked  map[string]bool `json:"unlocked"`
	Runs      int             `json:"runs"`
	Wins      int             `json:"wins"`
	BestAct   int             `json:"best_act"`
	BestScore int             `json:"best_score"`
	Bosses    int             `json:"bosses"`
	Bricks    int             `json:"bricks"`
	MaxCombo  int             `json:"max_combo"`
	Daily     map[string]int  `json:"daily_best"`
	Seen      map[string]bool `json:"seen_mods"`
	XP        int             `json:"xp"`
	AscMax    int             `json:"asc_max"` // highest ascension unlocked
	AscSel    int             `json:"asc"`     // ascension chosen for new runs
	Fusions   map[string]bool `json:"fusions"` // fusions ever formed
	TotalGold int             `json:"total_gold"`
	Settings  Settings        `json:"settings"`
}

func defaultSave() SaveData {
	return SaveData{Version: 2, Upg: map[string]int{}, Unlocked: map[string]bool{}, Daily: map[string]int{}, Seen: map[string]bool{}, Fusions: map[string]bool{},
		Settings: Settings{Master: 1, Music: 0.6, SFX: 0.8, Shake: true, Flashes: true, Particles: 1, VSync: true, Fullscreen: true}}
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
