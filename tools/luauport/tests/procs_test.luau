-- procs_test.luau: a run that owns every item at once (every proc, fusion
-- and synergy switched on), played for a few simulated seconds per
-- origin, so every effect path in Sim/Level.luau runs at least once.
local Shared = game:GetService("ReplicatedStorage"):WaitForChild("ArcaneOrb")
local Run = require(Shared.Sim.Run)
local Level = require(Shared.Sim.Level)
local Mods = require(Shared.Data.Mods)
local Origins = require(Shared.Data.Origins)
local Worlds = require(Shared.Data.Worlds)
local Arenas = require(Shared.LevelBank.Arenas)

local kinds = {}
for oi, origin in Origins.list do
	local run = Run.new(origin.ID, 50 + oi, false)
	for _, m in Mods.list do
		if not m.Curse then
			run:add(m.ID)
		end
	end
	local nf = 0
	for _ in run.fusions do
		nf += 1
	end
	local offers = run:offer(3, false, 0)
	local relics = run:offer(3, true, 2)
	run.act, run.actDepth = 1 + (oi % 3), 3
	local layout = run:nextLayout()
	local arena = Arenas[layout.shape]
	local L = Level.new(run, layout, arena.pts, Worlds.list[oi].Hues, run:chamberSeed(), run:hpMul(layout.elite))
	-- every power-up once
	for p = 0, 10 do
		L:applyPower(p)
	end
	local t = 0
	while t < 8 and not L.cleared and not L.lost do
		local x = 0
		for _, b in L.balls do
			if b.VZ < 0 then
				x = b.X
			end
		end
		L:step(1 / 30, { paddleX = x, fire = true })
		for _, e in L.events do
			kinds[e.type] = (kinds[e.type] or 0) + 1
		end
		table.clear(L.events)
		t += 1 / 30
	end
	print(string.format("%-11s fusions %d synergies %d offers %d/%d  t=%.1f left %d/%d balls %d gold %d %s",
		origin.Name, nf, #run.active, #offers, #relics, t, L.left, L.startLeft, #L.balls, run.gold, if L.cleared then "CLEARED" else ""))
end
local names = {}
for k, n in kinds do
	table.insert(names, k .. "=" .. n)
end
table.sort(names)
print("events: " .. table.concat(names, " "))
