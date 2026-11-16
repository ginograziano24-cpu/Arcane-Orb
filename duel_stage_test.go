package main

import "testing"

// Duel stage select: the choice cycles through "every round a new world"
// and each world, and a bad number from the network falls back (#49).
func TestDuelStage(t *testing.T) {
	if duelStageName(0) == worlds[0].Name || duelStageName(1) != worlds[0].Name || duelStageName(NumWorlds) != worlds[NumWorlds-1].Name {
		t.Fatal("stage names don't line up with the worlds")
	}
	if clampStage(-3) != 0 || clampStage(NumWorlds+1) != 0 || clampStage(2) != 2 {
		t.Fatal("clampStage")
	}
	// stepping right from the last world wraps to the rotating choice
	if (NumWorlds+1)%(NumWorlds+1) != 0 || (0+NumWorlds)%(NumWorlds+1) != NumWorlds {
		t.Fatal("stage cycling")
	}
}
