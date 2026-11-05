# Game performance: what to try next

The renderer is already well optimised. It has:

- parallel side jobs split into pieces, with radix sort plus last-frame presorting
- a run planner and zero-copy `DrawTriangles32`
- the DX11 uniform-reuse patch
- dynamic resolution with an FSR1-style upscaler, and SDF UI
- background prefetch and audio warming

The perf4 numbers were 280–660 fps at 1440p. So the goal isn't raw fps on the
PC. It is:

1. steadier 1% lows
2. good frame pacing at 60–144 Hz
3. weak hardware (Steam Deck, iGPU laptops)
4. less heat and battery drain
5. shorter loads

Nothing here lowers quality on the High preset. The cheaper paths only
replace something on Low or Medium.

Order: **A** first (cheap and measurable), then **B**, then the bigger items in **C**.

## A. Free wins from the toolchain

| # | Change | Why | Effort |
|---|---|---|---|
| A1 | **Go 1.26** | The Green Tea GC is on by default and cuts GC overhead by 10–40% in GC-heavy programs (about 10% more on Zen 4 / Ice Lake+ through vector scanning). It also allocates more slice backing on the stack. Directly helps 1% lows. Opt out with `GOEXPERIMENT=nogreenteagc` if anything regresses. | Small: bump go.mod, rebuild, bench |
| A2 | **PGO** (`default.pgo`) | Go reports 2–14% CPU gains. The bench already writes CPU pprofs per scene. Merge equal-length profiles from the tour (`go tool pprof -proto a.pprof b.pprof … > default.pgo`), put the file next to main.go and commit it. `go build` picks it up automatically. Refresh it after big renderer changes, because matching is by line offset inside each function. | Small |
| A3 | **Ebitengine 2.10** (needs Go 1.25+) | It fixes a DirectX crash on start-up under Proton/DXVK, which matters for the Steam Deck. It also fixes turning VSync off, unbounded memory growth while the window is hidden or minimised (2.10.1), and lighter restoration bookkeeping. It adds shader precompilation (`exp/shaderprecomp`) and deferred glyph creation in `text/v2`. Patch 5 (`DrawTriangles32`) is upstream since 2.9, so drop it; check whether patch 2 still applies after the restoration rework. Re-apply the rest per `third_party/ebiten/ARCANE_PATCHES.md`. | Medium |
| A4 | **Native Linux build** | Since Ebitengine 2.10, desktop Linux builds no longer need Cgo, so the Linux build can be cross-compiled without the missing X11 headers. That gives a native SteamOS build instead of Proton. Still needs the user's yes (HANDOFF). | Small after A3 |

Bench A1–A3 one at a time against perf4: plain, `-nopieces` and `-noparmerge`.

## B. Frame pacing, 1% lows, heat

| # | Change | Why |
|---|---|---|
| B1 | **Frame-rate cap setting** (Off / 60 / 120 / 144 / monitor) | With `SyncWithFPS` and VSync off the game renders uncapped at 300–600 fps: the fans spin and laptop batteries drain for frames nobody sees. A cap uses a hybrid limiter (sleep until about 1 ms before the deadline, then spin). Default to VSync on; the cap applies when VSync is off. |
| B2 | **Idle throttling** | In menus, pause, the shop and the codex, cap at 60. When the window is unfocused or minimised, drop to about 10 fps or skip drawing. Costs no quality. |
| B3 | **GC tuning after A1** | Set `debug.SetMemoryLimit` to a soft ceiling (e.g. 1.5 GB) and raise `SetGCPercent` (e.g. 200) so collections happen less often. Check with the F3 overlay's alloc rate. Keep driving steady-state allocations per frame to zero: everything left in the alloc pprof is a target. |
| B4 | **Shader warm-up** | Compile every Kage shader at boot (or precompile with A3) and draw each one once offscreen, so the first bloom, upscale or SDF use mid-run doesn't hitch. |
| B5 | **Better measurements** | Add 0.1% lows and frame-time standard deviation to the bench report. Check real pacing on the PC with PresentMon or CapFrameX, because Ebitengine's own fps hides uneven presents. Add a "Deck" bench preset: 1280×800, Medium, 60 fps cap. |

## C. GPU and CPU work for weak hardware

| # | Change | Why |
|---|---|---|
| C1 | **Dual-Kawase bloom** | post.go blurs with two separable Gaussian passes, run twice. Dual filtering (ARM, SIGGRAPH 2015) gets a wide, soft bloom from a chain of downsample and upsample passes with 4–5 taps each, reading far fewer texels at lower resolutions. Unity URP and Android's compositor both moved to it for mobile. The look stays the same, so it can replace the Gaussian on every preset. |
| C2 | **Fewer full-screen passes** | Fold grade, vignette and the final upscale into one shader where possible. Each full-screen pass costs about 3.7 MPix of bandwidth at 1440p, and an iGPU shares that bandwidth with the CPU. |
| C3 | **Cheaper AA for Medium** | The custom 2× supersampled AA renders twice the pixels. Keep it on High; give Medium a single-pass post AA (FXAA- or SMAA-1x-style Kage shader) at native resolution. |
| C4 | **Cached distant layers (impostors)** | The camera shakes and sways, so the whole world can't be baked, but layers far behind the arena can. Render distant scenery and the backdrop into a texture and redraw it only when the camera moves past a threshold or every few frames. Small shakes become a 2D offset. This removes their quads from the sort and the flush. Check that nothing dynamic ever draws behind them. |
| C5 | **Retained static quads** | Walls, posts and the floor rarely change within a world. Cache their projected vertex arrays per camera pose, and re-emit only on a camera or brick change. This builds on the existing pieces and presorting. |
| C6 | **Particle budget by preset** | Cap live particles and glows per preset, and merge glows that overlap heavily. Bursts in multiball are the worst case for the 1% low (multiball 383 fps vs world1 550). |
| C7 | **Faster boot** | Store parsed models and built floors/backdrops in a binary cache (gob or flat structs, keyed by a content hash) instead of parsing at every boot. |

## Not recommended yet

- `GOEXPERIMENT=simd` for the projection maths: the API is experimental and amd64-only, so the gain isn't worth the risk before launch.
- `GOAMD64=v3`: small gains, and the game would refuse to start on older CPUs that Steam players still use.
- DirectX 12 (`EBITENGINE_DIRECTX=version=12`): only as a bench A/B; our DX11 uniform patch doesn't cover it.

## Sources

- [Go 1.26 release notes](https://tip.golang.org/doc/go1.26)
- [Profile-guided optimization](https://go.dev/doc/pgo)
- [Ebitengine 2.10 release notes](https://ebitengine.org/en/documents/2.10.html)
- [Ebitengine 2.9 release notes](https://ebitengine.org/en/documents/2.9.html)
- [Marius Bjørge, "Bandwidth-efficient rendering", SIGGRAPH 2015 (dual filtering)](https://community.arm.com:443/cfs-file/__key/communityserver-blogs-components-weblogfiles/00-00-00-20-66/siggraph2015_2D00_mmg_2D00_marius_2D00_notes.pdf)
- [Unity URP bloom filter modes](https://docs.unity.cn/Packages/com.unity.render-pipelines.universal@17.4/api/UnityEngine.Rendering.Universal.BloomFilterMode.html)
