# Arcane Orb: hero quips (draft 1, now in game)

> **Status (2026-11-11):** implemented in `quips.go` with these exact lines,
> translated into all 7 languages (`tools/i18n/src/20.jsonl`). Each line also
> plays a short synthesised voice babble in the hero's register. Change a line
> here, in `heroQuips` (quips.go) and in 20.jsonl together. Settings > Audio >
> "Hero quips" turns them off.

Roadmap step 2. Short lines shown in a speech bubble next to the HUD
portrait. A cooldown keeps them rare (about one every 1–2 minutes), and a
setting turns them off. Corny on purpose.

Moments: **Pick** (chosen on class select) · **Start** (run begins) ·
**Combo** (big combo) · **Boss** (boss defeated) · **Legendary** (legendary
item picked up) · **Last life** · **Win** (story finished)

## Dame Oriel (Guardian)
- **Pick:** "Shield up. Snacks packed."
- **Start:** "Stay behind me. All of you."
- **Combo:** "Nothing gets through!"
- **Boss:** "That's what you get for building walls."
- **Legendary:** "Finally, proper equipment."
- **Last life:** "Steady... I've got this."
- **Win:** "Everyone home by bedtime."

## Brakka (Berserker)
- **Pick:** "Point me at a wall."
- **Start:** "Nap's over. Smashing time."
- **Combo:** "MORE BRICKS!"
- **Boss:** "Should've built it out of something tougher."
- **Legendary:** "Ooh. Shiny AND heavy."
- **Last life:** "I'm not done hitting things!"
- **Win:** "Now I can rebuild my house."

## Professor Vell (Arcanist)
- **Pick:** "Fascinating. Let us begin."
- **Start:** "Fire, frost, storm, void. In that order."
- **Combo:** "Textbook! Literally, chapter four."
- **Boss:** "I'll be writing a paper on that."
- **Legendary:** "I've only ever read about these!"
- **Last life:** "This was not in the syllabus."
- **Win:** "Class dismissed."

## Rivet (Engineer)
- **Pick:** "Paddle's calibrated."
- **Start:** "Hold still. Aim. Fire."
- **Combo:** "Running at peak efficiency!"
- **Boss:** "Shutting that one down."
- **Legendary:** "Ooh, I can upgrade this."
- **Last life:** "Okay. Think. Think."
- **Win:** "Told you it would work."

## Lucky Sol (Gambler)
- **Pick:** "Feeling lucky? I always am."
- **Start:** "House rules: I win."
- **Combo:** "Jackpot, baby!"
- **Boss:** "All in, and it paid off."
- **Legendary:** "Read 'em and weep."
- **Last life:** "Double or nothing..."
- **Win:** "Drinks are on Oriel."

## Cinder (Pyromancer)
- **Pick:** "Let's light this up."
- **Start:** "Who brought the marshmallows?"
- **Combo:** "Burn, baby, burn!"
- **Boss:** "Ooh, that's gonna leave a scorch mark."
- **Legendary:** "It's even hotter than it looks."
- **Last life:** "I'm not burning out yet!"
- **Win:** "Best. Bonfire. Ever."

## Zephyr (Stormcaller)
- **Pick:** "Let's get moving."
- **Start:** "Can't stop, won't stop."
- **Combo:** "Feel that static!"
- **Boss:** "Struck down. Literally."
- **Legendary:** "Now THAT is electric."
- **Last life:** "Keep moving, keep moving..."
- **Win:** "Look up. The stars are back."

## Combo commentary (added 2026-11-11)

A hot streak gets a running commentary. At a 50-hit combo the hero says one
of three lines (they take turns, at most once every 45 s); at 100 one of two
more; at 150 a last one. These live in `heroComboQuips` (quips.go).

| Hero | ×50 (take turns) | ×100 | ×150 |
|---|---|---|---|
| Oriel | Nothing gets through! / That's it, nice and steady! / Formation's holding. Good. | A hundred! Somebody write this down! / Okay. I'm actually impressed. | I'm telling everyone about this at dinner. |
| Brakka | MORE BRICKS! / Ha! Is that all you've got? / Keep 'em coming! | I could do this ALL DAY! / Best. Day. Of my LIFE! | Somebody stop me. Actually, don't. |
| Vell | Textbook! Literally, chapter four. / Mind the angles, mind the angles... / A most elegant chain reaction. | One hundred! I must take notes. / Remarkable. Utterly remarkable. | I'm rewriting the textbook. Again. |
| Rivet | Running at peak efficiency! / Huh. Working better than planned. / Okay, okay, the numbers love this. | A hundred? I'm logging that. / This is going straight in my notebook. | Nobody touch anything. It's perfect. |
| Sol | Jackpot, baby! / Hot streak! Don't jinx it. / Let it ride! | The house is SWEATING. / I'd bet on me. I DID bet on me. | Somebody call my bookie! |
| Cinder | Burn, baby, burn! / Ooh, it's catching! / Hehe. More. MORE. | It's so BEAUTIFUL! / Everything's on fire and I love it! | Best. Bonfire. Ever. Again! |
| Zephyr | Feel that static! / Faster! Faster! / Whoo! Don't stop now! | I can see the sky from up here! / Now we're flying! | So THIS is what lightning feels like! |
