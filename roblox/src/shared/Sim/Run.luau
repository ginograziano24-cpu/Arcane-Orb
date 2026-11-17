--!strict
-- Run.luau: one run of Arcane Orb: the origin, the build (items, synergies,
-- fusions), gold, lives and the chamber sequence. A port of run.go
-- (Recompute, Offer, rarityGate) and lives.go, with a simpler map: each
-- act is a line of chambers rather than the branching map, which suits the
-- shorter Roblox sessions (see docs/ROBLOX_PORT.md).

local Data = script.Parent.Parent.Data
local Stats = require(Data.Stats)
local Mods = require(Data.Mods)
local Fusions = require(Data.Fusions)
local Synergies = require(Data.Synergies)
local Origins = require(Data.Origins)
local Bank = script.Parent.Parent.LevelBank

type Stats = Stats.Stats

local Run = {}
Run.__index = Run

Run.BaseLives = 2
Run.Acts = 3
Run.ChambersPerAct = 8 -- the last one is the act's Guardian chamber
Run.RarityWeights = { [0] = 50, [1] = 30, [2] = 14, [3] = 5, [4] = 1.2, [5] = 0.25 }
Run.RarityNames = { [0] = "Common", [1] = "Uncommon", [2] = "Rare", [3] = "Epic", [4] = "Legendary", [5] = "Mythic" }

export type Run = typeof(setmetatable(
	{} :: {
		seed: number,
		rng: Random,
		origin: any,
		stats: Stats,
		mods: { [string]: number },
		order: { string },
		active: { string },
		fusions: { [string]: boolean },
		newFusions: { string },
		gold: number,
		lives: number,
		act: number,
		actDepth: number,
		depth: number,
		score: number,
		bricks: number,
		elites: number,
		maxCombo: number,
		quick: boolean,
		over: boolean,
		won: boolean,
	},
	Run
))

-- quick: a one-act Quick Run for short sessions
function Run.new(originId: string, seed: number, quick: boolean?): Run
	local origin = Origins.byId[originId] or Origins.list[1]
	local self = setmetatable({
		seed = seed,
		rng = Random.new(seed),
		origin = origin,
		stats = Stats.base(),
		mods = {},
		order = {},
		active = {},
		fusions = {},
		newFusions = {},
		gold = 0,
		lives = 0,
		act = 1,
		actDepth = 0,
		depth = 0,
		score = 0,
		bricks = 0,
		elites = 0,
		maxCombo = 0,
		quick = quick == true,
		over = false,
		won = false,
	}, Run)
	self:recompute()
	self.lives = self:maxLives()
	self.gold = math.floor(self.stats.StartGold)
	return self
end

function Run.maxLives(self: Run): number
	return math.max(1, Run.BaseLives + self.stats.ExtraLives)
end

function Run.owned(self: Run, id: string): boolean
	return (self.mods[id] or 0) > 0
end

-- recompute rebuilds the stats from scratch: base, origin, items in the
-- order they were taken, synergies (3 items sharing a tag), then fusions
function Run.recompute(self: Run)
	local s = Stats.base()
	if self.origin and self.origin.Apply then
		self.origin.Apply(s)
	end
	local tags: { [string]: number } = {}
	for _, id in self.order do
		local m = Mods.byId[id]
		if m then
			for _ = 1, self.mods[id] do
				m.Apply(s)
			end
			for _, t in m.Tags or {} do
				tags[t] = (tags[t] or 0) + self.mods[id]
			end
		end
	end
	for _, id in self.order do
		local m = Mods.byId[id]
		if m and m.Curse and self.mods[id] > 0 then
			s.CurseCount += 1
		end
	end
	if s.CurseWard then
		s.BarrierPerLevel += math.min(2, s.CurseCount)
	end
	self.active = {}
	for _, sy in Synergies.list do
		if (tags[sy.Tag] or 0) >= 3 then
			sy.Apply(s)
			table.insert(self.active, sy.Name)
		end
	end
	for _, f in Fusions.list do
		if self:owned(f.A) and self:owned(f.B) then
			f.Apply(s)
			if not self.fusions[f.ID] then
				self.fusions[f.ID] = true
				table.insert(self.newFusions, f.ID)
			end
		end
	end
	s.PaddleWidth = math.max(0.5, s.PaddleWidth)
	s.BallLossGold = math.max(0, s.BallLossGold)
	s.PowerDur = math.max(-0.6, s.PowerDur)
	self.stats = s
end

-- add grants an item; returns the name of a synergy it switched on, if any
function Run.add(self: Run, id: string): string?
	local before: { [string]: boolean } = {}
	for _, a in self.active do
		before[a] = true
	end
	if not self.mods[id] then
		self.mods[id] = 0
		table.insert(self.order, id)
	end
	self.mods[id] += 1
	local oldMax = self:maxLives()
	self:recompute()
	local gain = self:maxLives() - oldMax
	if gain > 0 then
		self.lives += gain -- an item that raises your lives grants them at once
	end
	for _, a in self.active do
		if not before[a] then
			return a
		end
	end
	return nil
end

function Run.fusionWith(self: Run, id: string): any
	for _, f in Fusions.list do
		if (f.A == id and self:owned(f.B)) or (f.B == id and self:owned(f.A)) then
			if not self.fusions[f.ID] then
				return f
			end
		end
	end
	return nil
end

-- the strongest items wait for later in a run (run.go rarityGate)
function Run.rarityGate(self: Run, rarity: number): number
	if self.act > 1 then
		return 1
	end
	local d = self.depth
	if rarity == 3 then
		if d < 3 then
			return 0.25
		elseif d < 6 then
			return 0.6
		end
	elseif rarity == 4 then
		if d < 4 then
			return 0
		elseif d < 8 then
			return 0.35
		end
		return 0.7
	elseif rarity == 5 then
		if d < 8 then
			return 0
		end
		return 0.4
	end
	return 1
end

-- offer draws n different items, weighted by rarity (run.go Offer)
function Run.offer(self: Run, n: number, relic: boolean, minRarity: number): { any }
	local pool = {}
	for _, m in Mods.list do
		local rarity = m.Rarity or 0
		local okOrigin = m.Origin == nil or m.Origin == "" or m.Origin == self.origin.ID
		local okNeeds = m.Needs == nil or m.Needs == "" or self:owned(m.Needs)
		local okMax = (m.Max or 0) == 0 or (self.mods[m.ID] or 0) < m.Max
		if not m.Retired and not m.Curse and (m.Relic == true) == relic and rarity >= minRarity and okMax and okOrigin and okNeeds then
			table.insert(pool, m)
		end
	end
	local luck = 1 + (self.stats.Luck - 1) * 0.3 + math.min(self.act - 1, 4) * 0.25
	local gated = true
	local function weight(m: any): number
		local rarity = m.Rarity or 0
		local w = Run.RarityWeights[rarity] * luck ^ rarity
		if gated and rarity > minRarity then
			w *= self:rarityGate(rarity)
		end
		if self:fusionWith(m.ID) then
			w *= 4 -- completing a fusion is exciting: offer it more
		end
		return w
	end
	local out = {}
	while #out < n and #pool > 0 do
		local total = 0
		for _, m in pool do
			total += weight(m)
		end
		if total <= 0 then
			if not gated then
				break
			end
			gated = false
		else
			local x = self.rng:NextNumber() * total
			for i, m in pool do
				local w = weight(m)
				if x < w then
					table.insert(out, m)
					table.remove(pool, i)
					break
				end
				x -= w
			end
		end
	end
	return out
end

function Run.choices(self: Run): number
	return math.max(2, 3 + self.stats.ExtraChoices)
end

-- brick health multiplier for this chamber (waves.go hpMul, no heat or ascension)
function Run.hpMul(self: Run, elite: boolean): number
	local m = (1 + (self.act - 1) * 0.15) * (1 + 0.025 * self.actDepth)
	if elite then
		m *= 1.15
	end
	return m * (1 + self.stats.BrickHPMult)
end

function Run.isGuardianChamber(self: Run): boolean
	return self.actDepth == Run.ChambersPerAct
end

-- nextLayout advances to the next chamber and picks its brick layout from
-- the bank: deeper chambers draw from deeper layouts; every fourth is elite
function Run.nextLayout(self: Run): any
	self.depth += 1
	self.actDepth += 1
	local bank = require(Bank:FindFirstChild("Act" .. math.min(self.act, 3)) :: ModuleScript) :: any
	local depthSet = bank[math.clamp(self.actDepth, 1, #bank)]
	local wantElite = self.actDepth % 4 == 0 or self:isGuardianChamber()
	local pick = {}
	for _, lay in depthSet do
		if lay.elite == wantElite then
			table.insert(pick, lay)
		end
	end
	if #pick == 0 then
		pick = depthSet
	end
	return pick[self.rng:NextInteger(1, #pick)]
end

-- chamberDone is called when a chamber is cleared; it reports whether the
-- run goes on (false = the run is won)
function Run.chamberDone(self: Run): boolean
	if self.actDepth >= Run.ChambersPerAct then
		if self.quick or self.act >= Run.Acts then
			self.over, self.won = true, true
			return false
		end
		self.act += 1
		self.actDepth = 0
		self.lives = math.max(self.lives, self:maxLives()) -- every act tops lives back up
	end
	return true
end

function Run.chamberSeed(self: Run): number
	return (self.seed * 31 + self.act * 977 + self.depth * 7919) % 2147483647
end

return Run
