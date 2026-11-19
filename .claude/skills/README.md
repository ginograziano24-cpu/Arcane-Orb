# Ebitengine skills

Official agent skills from the Ebitengine project
(https://github.com/hajimehoshi/ebiten, `skills/` folder, commit dea10c2,
2026-11-19). Copied unmodified under the Apache License 2.0 — see
`LICENSE-ebitengine`.

| Skill | Use it for |
|---|---|
| `avoiding-blurry-ebitengine-rendering` | Sharp text, UI and scaling (high-DPI, pixel-art filters) |
| `efficient-ebitengine-rendering` | Draw batching, atlases, GPU performance |
| `writing-kage-shaders` | Writing or porting Kage shaders (retro/dark-fantasy restyle) |
| `run-ebitengine-app-headless` | Playtesting the game with no window: scripted input, frame capture, audio checks |

## Note for Arcane Orb

The game builds against a patched Ebitengine v2.8.8 in `third_party/ebiten`
(see `ARCANE_PATCHES.md` there). `run-ebitengine-app-headless` needs the
experimental `exp/vmhost` package, which that version does not have, so it
only works after upgrading the engine. The other three apply to v2.8.

Claude Code picks these up automatically from `.claude/skills/`. Other agents
(ChatGPT/Codex) can be pointed at this folder.
