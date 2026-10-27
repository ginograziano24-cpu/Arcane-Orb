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

func steamLBAvailable() bool                                                                  { return false }
func steamLBFind(name string)                                                                 {}
func steamLBDownload(kind, from, to int)                                                      {}
func steamLBUpload(score int, details []int32)                                                {}
func steamLBPoll(onFound func(), onEntries func(tag string, rows []lbRow), onUploaded func()) {}
