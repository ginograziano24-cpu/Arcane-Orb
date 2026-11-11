//go:build js

package main

import "orb/mus"

// Browser capture builds are muted by default. Keep their existing streaming
// fallback instead of adding an18.5MB bank to every WASM download.
var liveMenuSynthesis bool

func cachedMenuStream() *mus.PCMStream { return nil }
