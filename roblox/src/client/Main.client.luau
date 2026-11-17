--!strict
-- Main.client.luau: the Arcane Orb prototype loop on the player's device.
-- Pick an origin, play chambers, take an item after each one, and push on
-- until the run is won or the last life is gone. The simulation runs here
-- so the paddle answers instantly; the server keeps the meta save
-- (Main.server.luau).
--
-- Controls: mouse or finger moves the paddle (drag anywhere on the lower
-- half on a phone); click, tap or Space serves and fires lasers; A/D or
-- the arrow keys steer too; a gamepad uses the left stick and A.

local Players = game:GetService("Players")
local RunService = game:GetService("RunService")
local UserInputService = game:GetService("UserInputService")
local Workspace = game:GetService("Workspace")
local ReplicatedStorage = game:GetService("ReplicatedStorage")

local Shared = ReplicatedStorage:WaitForChild("ArcaneOrb")
local Const = require(Shared.Sim.Const)
local Level = require(Shared.Sim.Level)
local Run = require(Shared.Sim.Run)
local Origins = require(Shared.Data.Origins)
local Worlds = require(Shared.Data.Worlds)
local Arenas = require(Shared.LevelBank.Arenas)
local Renderer = require(script.Parent:WaitForChild("Renderer"))
local UI = require(script.Parent:WaitForChild("UI"))

local player = Players.LocalPlayer
local camera = Workspace.CurrentCamera
local remotes = ReplicatedStorage:WaitForChild("ArcaneOrbRemotes")

local view = Renderer.new()
local ui = UI.new(player)

local run: any = nil
local level: any = nil
local state = "menu" -- menu | play | reward | over
local paddleTarget: number? = nil
local keyDir = 0
local fire = false
local world: any = Worlds.list[1]

-- ------------------------------------------------------------------ camera

local function setCamera()
	camera.CameraType = Enum.CameraType.Scriptable
	local eye = Renderer.world(0, 30, -9)
	local look = Renderer.world(0, 0, 13)
	camera.CFrame = CFrame.lookAt(eye, look)
	camera.FieldOfView = 50
end

-- floorX turns a screen point into an x on the arena floor at the paddle
local function floorX(screen: Vector2): number?
	local ray = camera:ViewportPointToRay(screen.X, screen.Y)
	local planeY = Renderer.Origin.Y
	if math.abs(ray.Direction.Y) < 1e-4 then
		return nil
	end
	local t = (planeY - ray.Origin.Y) / ray.Direction.Y
	if t < 0 then
		return nil
	end
	local hit = ray.Origin + ray.Direction * t
	return hit.X - Renderer.Origin.X
end

-- ------------------------------------------------------------------ input

UserInputService.InputChanged:Connect(function(input: InputObject, processed: boolean)
	if processed then
		return
	end
	if input.UserInputType == Enum.UserInputType.MouseMovement or input.UserInputType == Enum.UserInputType.Touch then
		paddleTarget = floorX(Vector2.new(input.Position.X, input.Position.Y))
	elseif input.KeyCode == Enum.KeyCode.Thumbstick1 then
		keyDir = if math.abs(input.Position.X) > 0.2 then input.Position.X else 0
		if keyDir ~= 0 then
			paddleTarget = nil
		end
	end
end)

UserInputService.InputBegan:Connect(function(input: InputObject, processed: boolean)
	if processed then
		return
	end
	local t = input.UserInputType
	if t == Enum.UserInputType.MouseButton1 or t == Enum.UserInputType.Touch then
		fire = true
		if t == Enum.UserInputType.Touch then
			paddleTarget = floorX(Vector2.new(input.Position.X, input.Position.Y))
		end
	elseif input.KeyCode == Enum.KeyCode.Space or input.KeyCode == Enum.KeyCode.ButtonA then
		fire = true
	elseif input.KeyCode == Enum.KeyCode.A or input.KeyCode == Enum.KeyCode.Left then
		keyDir, paddleTarget = -1, nil
	elseif input.KeyCode == Enum.KeyCode.D or input.KeyCode == Enum.KeyCode.Right then
		keyDir, paddleTarget = 1, nil
	end
end)

UserInputService.InputEnded:Connect(function(input: InputObject)
	local k = input.KeyCode
	if (k == Enum.KeyCode.A or k == Enum.KeyCode.Left) and keyDir < 0 then
		keyDir = 0
	elseif (k == Enum.KeyCode.D or k == Enum.KeyCode.Right) and keyDir > 0 then
		keyDir = 0
	end
end)

-- ------------------------------------------------------------------ flow

local startChamber: () -> ()

local function showReward()
	state = "reward"
	local offers = run:offer(run:choices(), false, 0)
	ui:showReward(offers, run, function(id: string?)
		if id then
			local syn = run:add(id)
			if syn then
				ui:toast("Synergy: " .. syn)
			end
			for _, f in run.newFusions do
				ui:toast("Fusion forged: " .. f)
			end
			run.newFusions = {}
		end
		startChamber()
	end)
end

local function endRun(won: boolean)
	state = "over"
	remotes:WaitForChild("RunEnded"):FireServer({
		won = won,
		act = run.act,
		depth = run.depth,
		score = run.score,
		bricks = run.bricks,
		maxCombo = run.maxCombo,
		origin = run.origin.ID,
	})
	ui:showOver(won, run, function()
		state = "menu"
		ui:showMenu(Origins.list, function(originId: string, quick: boolean)
			run = Run.new(originId, math.random(1, 2 ^ 30), quick)
			startChamber()
		end)
	end)
end

startChamber = function()
	local layout = run:nextLayout()
	world = Worlds.list[((run.act - 1) % #Worlds.list) + 1]
	local arena = Arenas[layout.shape] or Arenas[0]
	level = Level.new(run, layout, arena.pts, world.Hues, run:chamberSeed(), run:hpMul(layout.elite))
	level.guardian = run:isGuardianChamber()
	view:setArena(arena.pts, world.Floor:Lerp(Color3.new(0, 0, 0), 0.75), world.Wall)
	ui:showHud(true)
	ui:banner(
		if level.guardian then "GUARDIAN CHAMBER" else layout.name,
		string.format("%s  ·  Act %d  ·  Chamber %d of %d", world.Name, run.act, run.actDepth, Run.ChambersPerAct)
	)
	state = "play"
end

-- ------------------------------------------------------------------ frame

RunService.RenderStepped:Connect(function(dt: number)
	if state ~= "play" or not level then
		return
	end
	dt = math.min(dt, 1 / 30)
	level:step(dt, { paddleX = paddleTarget, keyDir = keyDir, fire = fire })
	fire = false
	view:handle(level.events)
	for _, e in level.events do
		if e.type == "lifelost" then
			ui:flash(Color3.fromRGB(255, 60, 60))
		end
	end
	table.clear(level.events)
	view:draw(level, level.T)
	ui:updateHud(run, level)
	if level.lost then
		endRun(false)
	elseif level.cleared then
		state = "between"
		task.delay(1.2, function()
			if run:chamberDone() then
				showReward()
			else
				endRun(true)
			end
		end)
	end
end)

setCamera()
ui:showMenu(Origins.list, function(originId: string, quick: boolean)
	run = Run.new(originId, math.random(1, 2 ^ 30), quick)
	startChamber()
end)
