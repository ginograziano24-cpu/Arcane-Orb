-- sim_test.luau: plays whole runs of the Roblox port's simulation with an
-- autopilot and a random build, for luaurun.py. It fails on any runtime
-- error and prints how each chamber went.
local Shared = game:GetService("ReplicatedStorage"):WaitForChild("ArcaneOrb")
local Run = require(Shared.Sim.Run)
local Level = require(Shared.Sim.Level)
local Origins = require(Shared.Data.Origins)
local Worlds = require(Shared.Data.Worlds)
local Arenas = require(Shared.LevelBank.Arenas)
local Mods = require(Shared.Data.Mods)

-- every item's Apply must run on a fresh build without erroring
local Stats = require(Shared.Data.Stats)
for _, m in Mods.list do
	local s = Stats.base()
	m.Apply(s)
end
print(string.format("items: %d apply cleanly", #Mods.list))

-- the Go balance autopilot (balance.go sloppy): reads the play every
-- ~0.25 s, predicts where the orb lands (folding off the side walls),
-- misjudges a little, and catches good drops when nothing is urgent
local rng = Random.new(7)
local ap = { target = 0, think = 0 }
local function fold(x: number, w: number): number
	local p = 4 * w
	local m = (x + w) % p
	if m > 2 * w then
		m = p - m
	end
	return m - w
end
local function autopilot(L, dt: number): number
	ap.think -= dt
	if ap.think <= 0 then
		ap.think = 0.25 * (0.7 + 0.6 * rng:NextNumber())
		local zp = 2.2 + 0.35
		local bt, x = math.huge, L.paddleX
		for _, b in L.balls do
			if not b.Stuck and b.VZ < 0 and b.Phantom <= 0 then
				local t = (b.Z - zp - b.R) / -b.VZ
				if t < bt then
					bt = t
					x = fold(b.X + b.VX * t, 14 - b.R)
				end
			end
		end
		if bt > 1.2 then
			for _, d in L.drops do
				if d.Z < 12 then
					x, bt = d.X, 0
					break
				end
			end
		end
		local err = (rng:NextNumber() * 2 - 1) * 0.8 * math.min(1.6, 0.4 + bt * 0.5)
		ap.target = x + err + (rng:NextNumber() * 2 - 1) * L.paddleW * 0.25
	end
	return ap.target
end

local totals = { runs = 0, wins = 0, chambers = 0 }
for runIdx = 1, 1 do
	local origin = Origins.list[((runIdx - 1) % #Origins.list) + 1]
	local run = Run.new(origin.ID, 1000 + runIdx, false)
	print(string.format("run %d: %s", runIdx, origin.Name))
	while not run.over do
		local layout = run:nextLayout()
		local world = Worlds.list[((run.act - 1) % #Worlds.list) + 1]
		local arena = Arenas[layout.shape] or Arenas[0]
		local L = Level.new(run, layout, arena.pts, world.Hues, run:chamberSeed(), run:hpMul(layout.elite))
		local t, dt = 0, 1 / 30
		local events = 0
		while not L.cleared and not L.lost and t < 150 do
			L:step(dt, { paddleX = autopilot(L, dt), fire = true })
			events += #L.events
			table.clear(L.events)
			t += dt
		end
		totals.chambers += 1
		print(string.format("  act %d chamber %d %-28s %s in %5.1fs  lives %d gold %d combo %d left %d events %d",
			run.act, run.actDepth, layout.name, if L.cleared then "cleared" elseif L.lost then "LOST" else "TIMEOUT",
			t, run.lives, run.gold, run.maxCombo, L.left, events))
		if L.lost then
			run.over = true
			break
		end
		if not L.cleared then
			-- an autopilot can stall on the last brick; count it and move on
			L:clear()
		end
		if not run:chamberDone() then
			break
		end
		local offers = run:offer(run:choices(), false, 0)
		if #offers > 0 then
			local pick = offers[1 + (run.depth % #offers)]
			local syn = run:add(pick.ID)
			print(string.format("    took %s%s", pick.Name, if syn then "  -> synergy " .. syn else ""))
		end
	end
	totals.runs += 1
	if run.won then
		totals.wins += 1
	end
	print(string.format("  run over: %s, act %d, bricks %d, score %d", if run.won then "WON" else "lost", run.act, run.bricks, run.score))
end
print(string.format("done: %d runs, %d won, %d chambers", totals.runs, totals.wins, totals.chambers))
