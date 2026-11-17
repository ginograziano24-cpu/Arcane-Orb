package main

// More lines per moment, so a hero doesn't repeat the same sentence every
// time a Guardian falls or a legendary card turns up (polish pass, phase 4).
// heroQuips keeps the original line for each moment; these are added to it,
// and quipLine picks from the pool without saying the same line twice in a
// row. Moments said once per run (start, win) also vary from run to run.
// Drafts for Gino to read: edit freely, they're plain strings.
var heroQuipsMore = map[string]map[quipMoment][]string{
	"guardian": {
		quipNearMiss:  {"Too close! Way too close.", "I had that. Mostly.", "My heart can't take this."},
		quipPick:      {"I brought plasters. For everyone.", "Right. Coats on, let's go."},
		quipStart:     {"Buddy system, everyone. Hold hands.", "Map? Check. Snacks? Check. Courage? Check.", "We'll be careful. Mostly me."},
		quipBoss:      {"And STAY down.", "That's for every door you blocked.", "Big wall, bigger fall."},
		quipLegendary: {"Oh, this is sturdy. I like sturdy.", "Now we're properly equipped.", "Sensible AND powerful. My favourite."},
		quipLastLife:  {"Nobody panic. Especially me.", "Deep breath. I've got you.", "I promised I'd get you home. I keep promises."},
	},
	"berserker": {
		quipNearMiss:  {"Ha! Not today!", "Nearly lost it. Didn't.", "That was fun. Again!"},
		quipPick:      {"Finally! Something to punch.", "Who wants to watch me hit a wall?"},
		quipStart:     {"Stretch, snack, SMASH.", "This one's for my kitchen.", "Bricks, I'm coming for you."},
		quipBoss:      {"That one had it coming.", "Ha! Who's napping NOW?", "Biggest brick yet. Best brick yet."},
		quipLegendary: {"Heavy! I love heavy!", "This is going straight in my fist.", "Ooh, that'll make a lovely crunch."},
		quipLastLife:  {"Okay. Now I'm annoyed.", "Not done. Not even a little bit.", "One life is plenty. Watch."},
	},
	"arcane": {
		quipNearMiss:  {"A narrow margin. Noted.", "Statistically, that should have failed.", "I'll be lecturing on that save."},
		quipPick:      {"Splendid. I've prepared notes.", "Shall we observe some physics?"},
		quipStart:     {"Today's lesson: everything explodes.", "Pens out. This will be on the exam.", "Remember, magic is just maths with sparkles."},
		quipBoss:      {"A fascinating specimen. Formerly.", "I believe that's what we call a result.", "Footnote: it lost."},
		quipLegendary: {"I wrote my thesis about these!", "Oh, the annotations I'll make.", "This rewrites at least three textbooks."},
		quipLastLife:  {"This is a teaching moment. For me.", "Hypothesis: we still win.", "Stay calm. Calm is very scientific."},
	},
	"engineer": {
		quipNearMiss:  {"Phew. Within tolerance.", "Barrier held. Good build.", "Close call. Recalibrating."},
		quipPick:      {"Tools packed. Let's build something.", "I've tuned the paddle. Twice."},
		quipStart:     {"Systems green. Let's go.", "I measured everything. We're fine.", "Small steps. Big results."},
		quipBoss:      {"Structural failure. As predicted.", "I knew where it was weak.", "Decommissioned. Nice."},
		quipLegendary: {"Look at the craftsmanship on this!", "I'm taking this apart later.", "Oh, this changes the whole design."},
		quipLastLife:  {"Recalculating...", "Okay. Plan B. I always have a Plan B.", "Steady hands. Steady hands."},
	},
	"lucky": {
		quipNearMiss:  {"Told you. Lucky.", "Living on the edge, baby!", "The house almost won that one."},
		quipPick:      {"Deal me in.", "Lucky's here. Relax."},
		quipStart:     {"Odds look great. They always do.", "Let's make some money, friends.", "Bet you a gold I break the first one."},
		quipBoss:      {"House always wins. I'm the house.", "Cash in that big lump!", "Called its bluff."},
		quipLegendary: {"Royal flush!", "Oh, Mama Sol's gonna hear about this.", "I KNEW that card was coming."},
		quipLastLife:  {"All in. Here we go.", "Last chip on the table. Exciting!", "Lady Luck, don't you dare."},
	},
	"pyro": {
		quipNearMiss:  {"Whoa, hot potato!", "Nearly singed my eyebrows.", "Still burning!"},
		quipPick:      {"Who's ready for warmth?", "I brought matches. Lots of matches."},
		quipStart:     {"Something's getting toasted today.", "Bricks burn. Trust me.", "Stand back. Further. Bit further."},
		quipBoss:      {"Look at it go! Look at it GO!", "Crispy.", "Best fire of the week."},
		quipLegendary: {"Ooh, it's warm already.", "This is going to be SO loud.", "Can I set it on fire? Just a bit?"},
		quipLastLife:  {"Still got a spark left.", "Okay, nobody blow on me.", "Not going out. Not yet."},
	},
	"storm": {
		quipNearMiss:  {"Whoosh! Saved it!", "Faster than that!", "Close one. Keep moving!"},
		quipPick:      {"Ready when you are. I'm ready now.", "Let's go let's go let's go."},
		quipStart:     {"Race you to the stars.", "Wind's up. Let's fly.", "Keep the paddle moving, friends."},
		quipBoss:      {"Zap! Gone!", "One less wall on the sky.", "I can almost see the stars."},
		quipLegendary: {"This is crackling!", "My hair's standing up. Good sign.", "Fast AND shiny? Yes please."},
		quipLastLife:  {"Don't stop moving. Never stop.", "Thunder's not done rolling.", "Little bit of static left. That's enough."},
	},
}

// quipLine picks the line a hero says for a moment: the original or one of
// the extras, never the same one twice in a row for that moment.
func quipLine(arch string, m quipMoment, last *[numQuipMoments]int, rng func(int) int) string {
	base, ok := heroQuips[arch]
	if !ok {
		return ""
	}
	pool := append([]string{}, heroQuipsMore[arch][m]...)
	if base[m] != "" {
		pool = append([]string{base[m]}, pool...)
	}
	if len(pool) == 0 {
		return ""
	}
	if len(pool) == 1 {
		return pool[0]
	}
	i := rng(len(pool) - 1)
	if last != nil {
		if prev := last[m] - 1; prev >= 0 && i >= prev {
			i++ // skip the last line said
		}
		last[m] = i + 1
	} else {
		i = rng(len(pool))
	}
	return pool[i]
}
