//go:build !windows

package main

// Steam integration is Windows-only for now; elsewhere these are no-ops.

func steamInit() bool         { return false }
func steamTick()              {}
func steamUnlock(id string)   {}
func steamHas(id string) bool { return false }
func steamOnDeck() bool       { return false }
func steamActive() bool       { return false }

func steamSync(ids []string) {}
func steamShutdown()         {}
