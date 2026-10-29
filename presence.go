package main

// updatePresence tells Steam friends what the player is doing. The
// tokens and their wording live in store/richpresence.vdf, which is
// uploaded in Steamworks (see STEAM.md).
func (g *Game) updatePresence() {
	r := g.run
	inRun := r != nil && r.SeedCode != "MENU" && g.state != StMenu && g.state != StSummary
	if !inRun {
		steamPresence("#Status_Menu")
		return
	}
	class := r.Arch.Name
	act := itoa(r.Act)
	if d := g.duel; d != nil && r.Duel {
		steamPresence("#Status_Duel", "round", itoa(d.round), "opponent", fallback(d.oppName, "a friend"))
		return
	}
	switch {
	case g.state == StLevel && g.level != nil && g.level.Boss != nil:
		steamPresence("#Status_Boss", "class", class, "boss", g.level.Biome.BossName)
	case r.Loop() > 0:
		steamPresence("#Status_Endless", "class", class, "loop", itoa(r.Loop()))
	case r.Daily:
		steamPresence("#Status_Daily", "class", class, "act", act)
	case r.Challenge != "":
		name := r.Challenge
		if c := challengeByID[r.Challenge]; c != nil {
			name = c.Name
		}
		steamPresence("#Status_Challenge", "class", class, "act", act, "challenge", name)
	default:
		steamPresence("#Status_Run", "class", class, "act", act, "world", biomes[r.AI()].Name)
	}
}
