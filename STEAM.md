# Arcane Orb on Steam

Everything the game needs for Steam is already in the code. This page is
the checklist for the Steamworks side.

## 1. App ID

1. Create the app in Steamworks and note its App ID.
2. Set `steamAppID` in `steam_windows.go` to that number and rebuild.
   While it is `0` the game never asks Steam to relaunch it, which is what
   you want for local testing.
3. For local testing without the Steam client launching the game, put a
   `steam_appid.txt` containing just the App ID next to `ArcaneOrb.exe`.
   Do **not** ship that file in the depot.

## 2. The Steamworks DLL

The game talks to Steam through `steam_api64.dll`, which is loaded only if
it sits next to the exe. Without it the game runs normally with no Steam
features (local achievements and popups still work).

- Download the Steamworks SDK (partner site → SDK).
- Copy `sdk/redistributable_bin/win64/steam_api64.dll` next to
  `ArcaneOrb.exe` in the depot.
- Any SDK from 1.57 onward works; the bridge falls back to older interface
  versions where the names changed.

What the bridge does (`steam_windows.go`):

| Call | When |
|---|---|
| `SteamAPI_RestartAppIfNecessary` | at start, only if `steamAppID != 0` |
| `SteamAPI_InitFlat` (or `SteamAPI_Init`) | at start |
| `SetAchievement` + `StoreStats` | the moment an achievement is earned |
| `GetAchievement` | at start, to push up any earned offline |
| `RunCallbacks` | every tick |
| `IsSteamRunningOnSteamDeck` | at start (switches prompts to pad glyphs) |
| `SteamAPI_Shutdown` | on quit |

When Steam is active its overlay shows the unlock toast, so the in-game
popup is skipped. Without Steam the game shows its own.

## 3. Achievements

There are 33. The full table (API name, display name, description, hidden
flag, icon files) is in `store/achievements.csv`, and the icons are in
`store/achievements/` as 256×256 JPGs: `<API name>.jpg` for unlocked and
`<API name>_locked.jpg` for locked.

In Steamworks → *Stats & Achievements → Achievements*, add one row per
line of the CSV. **The API names must match exactly.** Mark
`ACH_RAMPAGE` and `ACH_SUPERCHARGE` as hidden. Publish the changes.

Progress-style achievements (combo, loops, bricks, elites, rank, codex,
fusions, origins) are tracked in the save and unlock when the goal is
reached; there are no Steam *stats* to configure.

## 4. Cloud saves (Steam Auto-Cloud)

The save is two small JSON files, each written atomically (write to `.tmp`,
then rename): `save.json` (progression) and `run.json` (the run in
progress, so Save & quit carries between PC and Deck). Auto-Cloud is all
that's needed. In Steamworks →
*Steam Cloud*:

- Byte quota: `1048576` (1 MB), number of files: `10`
- Auto-Cloud root-file entry:

| Root | Subdirectory | Pattern | OS |
|---|---|---|---|
| `WinAppDataRoaming` | `ArcaneOrb` | `save.json` | Windows |
| `WinAppDataRoaming` | `ArcaneOrb` | `run.json` | Windows |

Optionally add a root override for Linux/Proton so Deck and PC share the
same save: Root `WinAppDataRoaming`, OS `Linux`, "Use Instead"
`LinuxHome`, path `.config/` — *only* if you ship a native Linux build;
under Proton the Windows path above already works.

## 5. Steam Deck

- Every menu works with the d-pad / stick, A to confirm, B to back, LB/RB
  to change pages and Codex tabs, Start to pause.
- In play: stick moves the paddle, A launches, X / RB / RT fires the
  ability.
- On a Deck the prompts start in controller glyphs.
- Text is sized for 1280×800; the game renders at any resolution.
- Suggested Deck Verified answers: controller support **full**, default
  config **gamepad**, no launcher, no anti-cheat.
- In Steamworks → *Steam Input*, choose "Gamepad" as the default
  configuration so the Deck's trackpads don't send mouse input.
- Play a Seed opens the Deck's floating keyboard over the seed box (and
  offers a Keyboard button), so seeds can be typed without a physical
  keyboard.
- On a Deck, body text starts at 115% (Settings → Access → Text size).
- Every gameplay key and the pad's launch / ability / scroll / pause
  buttons can be remapped in Settings → Controls.

## 5b. Rich Presence

`store/richpresence.vdf` holds the friends-list status lines. Upload it in
Steamworks → *Community* → *Rich Presence Localization* (English). The
game sets `steam_display` to one of these tokens and fills `%class%`,
`%act%`, `%world%`, `%boss%`, `%loop%` and `%challenge%`:

| Token | Shows as |
|---|---|
| `#Status_Menu` | In the main menu |
| `#Status_Run` | Berserker · Act 2 · Neon Grove |
| `#Status_Boss` | Berserker · Facing the Guardian |
| `#Status_Endless` | Berserker · Endless, loop 3 |
| `#Status_Daily` | Daily Challenge · Berserker · Act 1 |
| `#Status_Challenge` | Glass Paddle · Berserker · Act 2 |

## 5c. Crash reports

If the game ever crashes it writes `crash.log` next to the save
(`%APPDATA%\ArcaneOrb\crash.log`) with the version, the run (class, seed,
act, build), graphics settings and a stack trace, shows the player where
the file is, and exits. The run in progress was saved at its last safe
point, so Continue resumes it. Ask players to attach this file to bug
reports.

## 6. Depot layout

```
ArcaneOrb.exe
steam_api64.dll
```

Launch option: executable `ArcaneOrb.exe`, OS Windows, 64-bit.

## 7. Store assets

| Asset | Size | Source |
|---|---|---|
| Trailer | 1920×1080, 30 fps, H.264 | `store/trailer.mp4` |
| Screenshots (5+) | 1920×1080 | `store/screenshots/` |
| Header capsule | 920×430 | `store/capsules/` |
| Small capsule | 462×174 | `store/capsules/` |
| Main capsule | 1232×706 | `store/capsules/` |
| Vertical capsule | 748×896 | `store/capsules/` |
| Library capsule | 600×900 | `store/capsules/` |
| Library hero | 3840×1240 | `store/capsules/` |
| Library logo | 1280×720 (transparent PNG) | `store/capsules/` |
| Page background | 1438×810 | `store/capsules/` |

These sizes were current as of mid-2026. Check Steamworks →
*Graphical Assets* before uploading, since Valve adjusts them from time
to time.
