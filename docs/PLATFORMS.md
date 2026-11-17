# Arcane Orb on other platforms

Status on Nov 17, 2026. Ebitengine 2.10 (already in `go.mod`) builds every
desktop platform without Cgo, so all of these cross-compile from the
Windows PC with `Build-Platforms.cmd` (copy in `tools/pc/`). Output goes to
`builds\<os>-<arch>\` beside the script.

| Platform | Builds | Runs | What's left before shipping it |
|---|---|---|---|
| Windows x64 | yes | yes (the main build) | — |
| **Linux x64** (SteamOS, desktop Linux) | yes | **yes**: the screenshot tour ran under a virtual display in the cloud workspace (menu, origin select with the 3D hero, chambers, a Guardian; see below) | Steam features (below); a launch script and `.desktop` file; a Steam Linux depot |
| Linux ARM64 | yes | not tried | Only matters for ARM handhelds; low priority |
| macOS Apple silicon | yes | not tried | An `.app` bundle, an icon, and Apple signing + notarization (needs an Apple Developer account, $99/year), or players have to right-click → Open |
| macOS Intel | yes | not tried | Same as above; could ship as one universal app with `lipo` |
| Web (WebAssembly) | yes (134 MB) | not tried | An HTML page with `wasm_exec.js`; 134 MB is a heavy download, so a web demo should leave out the HD assets |

## Linux: what was checked

The Linux x64 build was copied into the cloud workspace and run with
`xvfb-run` (software OpenGL, no GPU, no sound card):

- It opened its window, loaded every asset and drew the menu, the origin
  select with the live 3D hero, chambers in several worlds and a Guardian
  fight correctly.
- With no sound device at all it **quit on start**, because Ebitengine
  stops the game when audio can't open. On a Steam Deck or a normal Linux
  desktop there is always a device, so players won't see this. The same
  thing can happen on Windows with every audio output disabled.
  `docs/patches/ebiten-silent-audio.patch` makes the game carry on
  silently instead. It changes the vendored Ebitengine
  (`third_party/ebiten/audio`). Those files are read-only on your PC, so I
  didn't apply it; it's your call.

## Steam on Linux and macOS

`steam_windows.go` talks to `steam_api64.dll` through its flat C API with
`syscall`. On Linux and macOS, `steam_other.go` turns Steam features off:
no achievements, leaderboards, rich presence or Steam Input text entry in
the native builds. The fix is the same bridge loading `libsteam_api.so` /
`libsteam_api.dylib` with `purego` (already a dependency, no Cgo). It's
about one session of work plus a test on a Steam Deck.

Until then, the Steam Deck can run the Windows build through Proton, with
all Steam features.

## Suggested order

1. **Steam Deck via Proton:** verify the Windows build on a Deck (Steam
   "Deck Verified" checks: 1280×800 text size, controller-only menus,
   on-screen keyboard). This is the biggest audience after Windows.
2. **Native Linux:** the purego Steam bridge, then a Linux depot. The
   build itself already works.
3. **macOS:** only once there's an Apple Developer account for signing.
4. **Web demo:** a cut-down build (first act, smaller assets) for itch.io
   or the Steam page.
