package main

import (
	"math"
	"math/rand"
	"testing"
)

func newTestRun() *Run {
	sv := defaultSave()
	return NewRun("S9", &archetypes[0], &sv, false)
}

func TestRarityGate(t *testing.T) {
	r := newTestRun()
	r.Depth = 1
	if r.rarityGate(Legendary) != 0 || r.rarityGate(Mythic) != 0 {
		t.Error("legendary and mythic items should not show up in the first chambers")
	}
	if g := r.rarityGate(Epic); g <= 0 || g >= 1 {
		t.Errorf("early epic gate %v, want a reduced chance", g)
	}
	if r.rarityGate(Rare) != 1 {
		t.Error("rare items are never gated")
	}
	r.Depth = 9
	if r.rarityGate(Epic) != 1 || r.rarityGate(Legendary) >= 1 {
		t.Error("late act 1: epics free, legendaries still a little rarer")
	}
	r.Act = 2
	if r.rarityGate(Mythic) != 1 {
		t.Error("from act 2 nothing is gated")
	}
	// an early relic hand still fills even when everything left is gated
	r.Act, r.Depth = 1, 0
	if got := r.Offer(3, true, Legendary, rand.New(rand.NewSource(1))); len(got) == 0 {
		t.Error("a promised legendary reward came back empty")
	}
}

func TestKeepsakesAndTrials(t *testing.T) {
	sv := defaultSave()
	for _, k := range keepsakes {
		if keepsakeByID[k.ID] == nil || k.Name == "" || k.Desc == nil || k.Ranks < 1 || k.Desc(k.Ranks) == "" {
			t.Errorf("keepsake %q incomplete", k.ID)
		}
		if k.Parent != "" && keepsakeByID[k.Parent] == nil {
			t.Errorf("keepsake %q has a missing parent", k.ID)
		}
	}
	if sv.keepOpen(keepsakeByID["knot"]) {
		t.Error("the Guardian's Knot needs ranks in the Spare Ward first")
	}
	r := newTestRun()
	r.Keepsake, r.KeepRank = "whetstone", 1
	base := r.Stats.Damage
	r.Recompute(&sv)
	one := r.Stats.Damage
	r.KeepRank = 5
	r.Recompute(&sv)
	if !(one > base && r.Stats.Damage > one) || math.Abs(r.Stats.Damage-base-0.2) > 1e-9 {
		t.Errorf("whetstone ranks: base %v, rank 1 %v, rank 5 %v", base, one, r.Stats.Damage)
	}
	r.Keepsake, r.KeepRank = "knot", 3
	r.Lives, r.Gold = 0, 100
	if !r.keepsakeSave() || r.Lives != 1 || r.Gold != 50 {
		t.Error("knot rank 3 should save the last life for half the gold")
	}
	if r.keepsakeSave() {
		t.Error("knot works once a run")
	}
	if trialPoints([]string{"hp", "choices", "fragile"}) != 5 {
		t.Error("trial points miscounted")
	}
	r.Trials = []string{"hp", "guardians"}
	if m := r.trialShardMult(); m < 1.17 || m > 1.19 {
		t.Errorf("3 trial points should give +18%% shards, got %v", m)
	}
	r.Trials = []string{"choices"}
	r.Recompute(&sv)
	if r.choices() != 2 {
		t.Errorf("Narrow Choice should leave 2 cards, got %d", r.choices())
	}
}

func TestFamiliarsAndBanish(t *testing.T) {
	r := newTestRun()
	bond := modByID["fam_bond"]
	if bond == nil || r.familiarOnly(bond) {
		t.Fatal("Kindred Bond should wait for a familiar")
	}
	r.Banished = map[string]bool{"fam_wisp": true}
	if r.offerAllowed(modByID["fam_wisp"]) {
		t.Error("a banished item was offered")
	}
	r.Banished = nil
	if !r.banish("fam_wisp") || r.banishesLeft() != baseBanishes-1 {
		t.Error("banish did not count")
	}
	if r.banish("fam_wisp") {
		t.Error("banished twice")
	}
	hand := r.Offer(3, false, Common, rand.New(rand.NewSource(3)))
	if len(hand) == 3 {
		if nm := r.replacementCard(hand, 1, rand.New(rand.NewSource(4))); nm != nil {
			for _, h := range hand {
				if h.ID == nm.ID {
					t.Error("replacement card duplicated the hand")
				}
			}
		}
	}
}

func TestShopLocksSurviveReroll(t *testing.T) {
	g := &Game{save: defaultSave(), settings: &Settings{NoTips: true}, ui: &UI{}}
	g.run = newTestRun()
	g.openShop(rand.New(rand.NewSource(5)))
	g.shop[1].Locked = true
	want := g.shop[1].Mod.ID
	kept := g.lockedStock()
	g.openShop(rand.New(rand.NewSource(6)))
	g.restoreLocked(kept)
	if g.shop[1].Mod.ID != want || !g.shop[1].Locked {
		t.Error("locked item lost on reroll")
	}
}

func TestCoopPartnerCatch(t *testing.T) {
	r := newTestRun()
	r.Coop = true
	sv := defaultSave()
	r.Recompute(&sv)
	solo := newTestRun()
	if r.Stats.PaddleWidth >= solo.Stats.PaddleWidth {
		t.Error("co-op paddles should be narrower")
	}
	L := &Level{p2On: true, p2X: 3, PaddleW: 2}
	if !L.partnerCatches(3.5, PaddleZ, 0) || L.partnerCatches(-3, PaddleZ, 0) {
		t.Error("partner catch area wrong")
	}
}

func TestKeepsakeTreeEconomy(t *testing.T) {
	sv := defaultSave()
	sv.migrateKeepsakes()
	if len(sv.KeepRanks) != 0 {
		t.Error("a new player should start the tree empty")
	}
	total := 0
	for _, k := range keepsakes {
		for r := 0; r < k.Ranks; r++ {
			total += k.cost(r)
		}
	}
	if total < 1500 {
		t.Errorf("the whole tree costs only %d shards", total)
	}
	old := defaultSave()
	old.Runs, old.Wins = 10, 1
	old.migrateKeepsakes()
	if old.keepRank("coin") != 1 || old.keepRank("clover") != 1 || old.keepRank("knot") != 0 {
		t.Errorf("migration ranks wrong: %v", old.KeepRanks)
	}
}
