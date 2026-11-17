--!strict
-- Main.server.luau: the server side of the prototype. Chambers run on the
-- player's device (instant paddle); the server keeps the meta save (runs,
-- best act, wins, bricks broken) in a DataStore and is where badges,
-- leaderboards and run checks go next (docs/ROBLOX_PORT.md, Phase 3).

local Players = game:GetService("Players")
local ReplicatedStorage = game:GetService("ReplicatedStorage")
local DataStoreService = game:GetService("DataStoreService")
local RunService = game:GetService("RunService")

local remotes = Instance.new("Folder")
remotes.Name = "ArcaneOrbRemotes"
remotes.Parent = ReplicatedStorage
local runEnded = Instance.new("RemoteEvent")
runEnded.Name = "RunEnded"
runEnded.Parent = remotes

-- DataStores need "Enable Studio Access to API Services" in Studio;
-- without it the game still plays, it just doesn't save.
local store: DataStore? = nil
local ok = pcall(function()
	store = DataStoreService:GetDataStore("ArcaneOrbMeta_v1")
end)
if not ok then
	warn("Arcane Orb: DataStore unavailable; meta progress won't save this session")
end

type Meta = { runs: number, wins: number, bestAct: number, bricks: number, bestCombo: number, bestScore: number }

local meta: { [Player]: Meta } = {}

local function blank(): Meta
	return { runs = 0, wins = 0, bestAct = 0, bricks = 0, bestCombo = 0, bestScore = 0 }
end

local function load(p: Player)
	local m = blank()
	if store then
		local s = store :: DataStore
		local got, data = pcall(function()
			return s:GetAsync("u" .. p.UserId)
		end)
		if got and type(data) == "table" then
			for k, v in data do
				if (m :: any)[k] ~= nil and type(v) == "number" then
					(m :: any)[k] = v
				end
			end
		end
	end
	meta[p] = m
	local stats = Instance.new("Folder")
	stats.Name = "leaderstats"
	local wins = Instance.new("IntValue")
	wins.Name = "Wins"
	wins.Value = m.wins
	wins.Parent = stats
	local best = Instance.new("IntValue")
	best.Name = "Best Combo"
	best.Value = m.bestCombo
	best.Parent = stats
	stats.Parent = p
end

local function save(p: Player)
	local m = meta[p]
	if not m or not store then
		return
	end
	local s = store :: DataStore
	pcall(function()
		s:SetAsync("u" .. p.UserId, m)
	end)
end

-- sane: a client report can't claim impossible numbers
local function num(v: any, lo: number, hi: number): number
	if type(v) ~= "number" or v ~= v then
		return lo
	end
	return math.clamp(math.floor(v), lo, hi)
end

runEnded.OnServerEvent:Connect(function(p: Player, report: any)
	local m = meta[p]
	if not m or type(report) ~= "table" then
		return
	end
	m.runs += 1
	if report.won == true then
		m.wins += 1
	end
	m.bestAct = math.max(m.bestAct, num(report.act, 0, 3))
	m.bricks += num(report.bricks, 0, 20000)
	m.bestCombo = math.max(m.bestCombo, num(report.maxCombo, 0, 5000))
	m.bestScore = math.max(m.bestScore, num(report.score, 0, 10 ^ 9))
	local ls = p:FindFirstChild("leaderstats")
	if ls then
		(ls:FindFirstChild("Wins") :: IntValue).Value = m.wins;
		(ls:FindFirstChild("Best Combo") :: IntValue).Value = m.bestCombo
	end
	save(p)
end)

Players.PlayerAdded:Connect(load)
for _, p in Players:GetPlayers() do
	task.spawn(load, p)
end
Players.PlayerRemoving:Connect(function(p: Player)
	save(p)
	meta[p] = nil
end)
game:BindToClose(function()
	if RunService:IsStudio() then
		return
	end
	for _, p in Players:GetPlayers() do
		save(p)
	end
end)
