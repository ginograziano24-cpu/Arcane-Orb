//go:build !js

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMenuMusicBankMatchesCurrentSynthSources(t *testing.T) {
	b, err := os.ReadFile("assets/music/groove.aopcm.json")
	if err != nil {
		t.Fatal(err)
	}
	var meta struct {
		SHA256            string
		SynthSourceSHA256 string
	}
	if err = json.Unmarshal(b, &meta); err != nil {
		t.Fatal(err)
	}
	bankHash := sha256.Sum256(menuMusicBank)
	if meta.SHA256 != hex.EncodeToString(bankHash[:]) {
		t.Fatal("music bank/manifest mismatch; regenerate")
	}
	h := sha256.New()
	files, err := filepath.Glob("mus/*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range files {
		if !strings.HasSuffix(p, "_test.go") {
			data, e := os.ReadFile(p)
			if e != nil {
				t.Fatal(e)
			}
			h.Write([]byte(filepath.Base(p)))
			h.Write([]byte{0})
			h.Write(data)
		}
	}
	if meta.SynthSourceSHA256 != hex.EncodeToString(h.Sum(nil)) {
		t.Fatal("cached music is stale; run go generate .")
	}
}

func TestMenuCachePlayersHaveIndependentPositions(t *testing.T) {
	a, b := cachedMenuStream(), cachedMenuStream()
	if a == nil || b == nil {
		t.Fatal("embedded cache failed to load")
	}
	first, again := make([]byte, 4096), make([]byte, 4096)
	a.Read(first)
	a.Read(again)
	b.Read(again)
	if !bytes.Equal(first, again) {
		t.Fatal("menu players share a playhead")
	}
}
