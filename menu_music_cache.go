//go:build !js

package main

import (
	_ "embed"
	"orb/mus"
	"sync"
)

//go:generate go run ./tools/musiccache
//go:embed assets/music/groove.aopcm
var menuMusicBank []byte

var menuCacheOnce sync.Once
var menuCache *mus.PCMCache
var liveMenuSynthesis bool // -livemenu is the sound/CPU A/B fallback.

func cachedMenuStream() *mus.PCMStream {
	menuCacheOnce.Do(func() { menuCache, _ = mus.ParsePCMCache(menuMusicBank) })
	if menuCache == nil {
		return nil
	}
	return menuCache.NewStream()
}
