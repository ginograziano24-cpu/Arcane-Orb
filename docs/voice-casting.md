# Arcane Orb opening: voice casting

Current build: free Piper voices (offline), rendered by `Make-Intro-Voices.cmd`
into `voices/intro/`, then pitched and mixed under the original score into
`intro/voice_story.wav`. Captions are in `intro/lines_story.txt`. Line timing is
in `intro/intro_cues.json`, which also drives the Blender acting (head nods
follow each line's loudness).

To upgrade to ElevenLabs later, record one file per line ID below. Keep the
names (`f01.wav` ... `r05.wav`) and drop them into `voices/intro/`, then re-run
the mix step. The film follows the cue file, so small timing changes are fine.

## Cast

| Character | Piper today | ElevenLabs direction |
|---|---|---|
| Fumbleton | en_GB alan, +3 semitones | Elderly British wizard, small and flustered, warm, comic timing. Think kindly grandpa who just set the kitchen on fire. Stability 35%, style 60%. |
| Mirabel | en_GB cori | Warm, confident, gently amused British witch. Affectionate exasperation with Fumbleton, then heroic resolve. Stability 50%, style 45%. |
| Merchant | Northern English male, phone-filtered | Fast, shouty market hawker. Over-sells everything. |
| Warden (guardian) | LibriTTS 135, -5 semitones, ring-mod | Slow, stony, mechanical. Each word lands like a stamp. Add a stone/metal filter in post. |
| Guardian (hero) | LibriTTS 285 | Steady, protective, few words. |
| Berserker | LibriTTS 90, -1 semitone | Gruff, grumpy, funny. Woken from a nap. |
| Arcanist | LibriTTS 195 | Precise, elegant, a little smug. |
| Engineer | LibriTTS 570 | Practical, upbeat, nerdy. |
| Gambler | LibriTTS 390 | Smooth, cocky, a grin you can hear. |
| Pyromancer | LibriTTS 315 | Bright and mischievous. Loves explosions. |
| Stormcaller | LibriTTS 615 | Calm, airy, cool. |

## Lines

| ID | Who | Line |
|---|---|---|
| f01 | Fumbleton | Date night with Mirabel. Seven o'clock. Candles, roses, my good hat... |
| f02 | Fumbleton | ...and three hundred years of laundry. Oh dear. Oh dear, oh dear. |
| m01 | Merchant | Instant Tidy! One scroll, zero scrubbing! Absolutely no refunds! |
| f03 | Fumbleton | Zero scrubbing? Sold! Tidy... instantly! |
| f04 | Fumbleton | Ah. That is... a lot of tidying. |
| g01 | Warden | Disorder detected. Order. Must. Be. Restored. |
| f05 | Fumbleton | Stop! Undo! Un-tidy! Why is there MORE of it? |
| f06 | Fumbleton | Right. Right. Nobody panic. Especially not me. |
| r01 | Mirabel | Fumbleton. Darling. Why is the sky made of bricks? |
| f07 | Fumbleton | Funny story... |
| r02 | Mirabel | Never mind. I know some people who love breaking things. |
| r03 | Mirabel | Go on, little lights. Find them. |
| r04 | Mirabel | One orb. One wall. One very satisfying crunch. |
| h01 | Guardian | Stay behind me. All of you. |
| h02 | Berserker | A wall went through my nap. Big mistake. |
| h03 | Arcanist | Fire, frost, storm, void. In that order. |
| h04 | Engineer | Hold still. Think. Then hit it really hard. |
| h05 | Gambler | Landed on my card table. That makes it mine. |
| h06 | Pyromancer | Honestly? I'm here for the explosions. |
| h07 | Stormcaller | Keep moving. The storm does the rest. |
| h08 | Berserker | So. Who gets to break the first wall? |
| h09 | Gambler | Double or nothing says it's me. |
| r05 | Mirabel | Arcane Orb. |
| f08 | Fumbleton | ...We're still getting dinner, right? |
