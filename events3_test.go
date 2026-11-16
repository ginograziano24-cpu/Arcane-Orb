package main

import (
	"math/rand"
	"testing"
)

// The new events build, have choices and an illustration (#27).
func TestEvenMoreEvents(t *testing.T) {
	g := &Game{save: defaultSave()}
	g.run = NewRun("EVENTS", &archetypes[0], &g.save, false)
	g.run.Gold = 200
	built := 0
	for i, f := range evenMoreEvents(g, rand.New(rand.NewSource(7))) {
		ev := f()
		if ev == nil {
			continue
		}
		built++
		if len(ev.Choices) < 2 {
			t.Errorf("event %d (%s) offers no real choice", i, ev.Title)
		}
		if eventGlyphs[ev.Title] == "" {
			t.Errorf("event %q has no glyph", ev.Title)
		}
	}
	if built < 6 {
		t.Fatalf("only %d of the new events could happen in a fresh run", built)
	}
}
