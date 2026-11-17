--!strict
-- Renderer.luau: draws a Level (Sim/Level.luau) with plain Roblox parts.
-- The simulation's x runs across the arena and z runs away from the paddle;
-- in the world that is +X across and -Z away, on a floor at ORIGIN.
-- Parts are pooled: bricks, orbs, drops, coins and bolts reuse their
-- parts every frame, so a busy chamber creates nothing new while it plays.

local Workspace = game:GetService("Workspace")
local TweenService = game:GetService("TweenService")
local Debris = game:GetService("Debris")

local Shared = game:GetService("ReplicatedStorage"):WaitForChild("ArcaneOrb")
local Const = require(Shared.Sim.Const)

local Renderer = {}
Renderer.__index = Renderer

local ORIGIN = Vector3.new(0, 200, 0) -- the arena floats out of the way of the lobby
Renderer.Origin = ORIGIN

local BRICK_SIZE = Vector3.new(Const.CellW * 0.94, Const.LayerH * 0.92, Const.CellD * 0.9)

local function world(x: number, y: number, z: number): Vector3
	return ORIGIN + Vector3.new(x, y, -z)
end
Renderer.world = world

local function part(props: { [string]: any }): Part
	local p = Instance.new("Part")
	p.Anchored = true
	p.CanCollide = false
	p.CanQuery = false
	p.CanTouch = false
	p.CastShadow = false
	p.TopSurface = Enum.SurfaceType.Smooth
	p.BottomSurface = Enum.SurfaceType.Smooth
	for k, v in props do
		(p :: any)[k] = v
	end
	return p
end

export type Pool = { parts: { BasePart }, used: number, make: () -> BasePart, folder: Folder }

local function newPool(folder: Folder, make: () -> BasePart): Pool
	return { parts = {}, used = 0, make = make, folder = folder }
end

local function take(pool: Pool): BasePart
	pool.used += 1
	local p = pool.parts[pool.used]
	if not p then
		p = pool.make()
		p.Parent = pool.folder
		pool.parts[pool.used] = p
	end
	p.Transparency = 0
	return p
end

local function finish(pool: Pool)
	for i = pool.used + 1, #pool.parts do
		local p = pool.parts[i]
		if p.Transparency < 1 then
			p.Transparency = 1
			p.CFrame = CFrame.new(ORIGIN - Vector3.new(0, 50, 0))
		end
	end
	pool.used = 0
end

export type Renderer = typeof(setmetatable(
	{} :: {
		folder: Folder,
		static: Folder,
		bricks: Pool,
		orbs: Pool,
		drops: Pool,
		coins: Pool,
		bolts: Pool,
		paddle: Part,
		barrier: Part,
		hues: { Color3 },
	},
	Renderer
))

function Renderer.new(): Renderer
	local folder = Instance.new("Folder")
	folder.Name = "ArcaneOrbArena"
	folder.Parent = Workspace
	local static = Instance.new("Folder")
	static.Name = "Static"
	static.Parent = folder
	local function sub(name: string): Folder
		local f = Instance.new("Folder")
		f.Name = name
		f.Parent = folder
		return f
	end
	local self = setmetatable({
		folder = folder,
		static = static,
		bricks = newPool(sub("Bricks"), function(): BasePart
			return part({ Size = BRICK_SIZE, Material = Enum.Material.SmoothPlastic })
		end),
		orbs = newPool(sub("Orbs"), function(): BasePart
			local p = part({ Shape = Enum.PartType.Ball, Material = Enum.Material.Neon, Color = Color3.fromRGB(255, 240, 200) })
			local light = Instance.new("PointLight")
			light.Range = 8
			light.Brightness = 2
			light.Color = Color3.fromRGB(255, 220, 160)
			light.Parent = p
			return p
		end),
		drops = newPool(sub("Drops"), function(): BasePart
			local p = part({ Size = Vector3.new(1.4, 0.5, 0.7), Material = Enum.Material.Neon })
			local gui = Instance.new("BillboardGui")
			gui.Size = UDim2.fromScale(1.6, 1.6)
			gui.AlwaysOnTop = true
			gui.Name = "Tag"
			local label = Instance.new("TextLabel")
			label.Name = "Letter"
			label.BackgroundTransparency = 1
			label.Size = UDim2.fromScale(1, 1)
			label.Font = Enum.Font.GothamBlack
			label.TextScaled = true
			label.TextColor3 = Color3.new(1, 1, 1)
			label.TextStrokeTransparency = 0.3
			label.Parent = gui
			gui.Parent = p
			return p
		end),
		coins = newPool(sub("Coins"), function(): BasePart
			return part({
				Shape = Enum.PartType.Cylinder,
				Size = Vector3.new(0.12, 0.5, 0.5),
				Material = Enum.Material.Neon,
				Color = Color3.fromRGB(255, 205, 70),
			})
		end),
		bolts = newPool(sub("Bolts"), function(): BasePart
			return part({ Size = Vector3.new(0.16, 0.16, 1.2), Material = Enum.Material.Neon, Color = Color3.fromRGB(255, 80, 80) })
		end),
		paddle = part({ Name = "Paddle", Material = Enum.Material.Neon, Color = Color3.fromRGB(255, 214, 140) }),
		barrier = part({ Name = "Barrier", Material = Enum.Material.ForceField, Color = Color3.fromRGB(170, 140, 255) }),
		hues = {},
	}, Renderer)
	self.paddle.Parent = folder
	self.barrier.Parent = folder
	return self
end

-- setArena builds the floor and walls for a chamber
function Renderer.setArena(self: Renderer, walls: { { number } }, floorCol: Color3, wallCol: Color3)
	self.static:ClearAllChildren()
	part({
		Name = "Floor",
		Size = Vector3.new(Const.HalfW * 2 + 4, 1, Const.FarZ + 8),
		CFrame = CFrame.new(world(0, -0.5, (Const.FarZ - 4) / 2)),
		Color = floorCol,
		Material = Enum.Material.Slate,
		Parent = self.static,
	})
	for i = 1, #walls - 1 do
		local a, b = walls[i], walls[i + 1]
		local pa, pb = world(a[1], 0, a[2]), world(b[1], 0, b[2])
		local mid = (pa + pb) / 2 + Vector3.new(0, 1, 0)
		local len = (pb - pa).Magnitude
		if len > 0.01 then
			part({
				Name = "Wall",
				Size = Vector3.new(0.6, 2, len + 0.6),
				CFrame = CFrame.lookAt(mid, mid + (pb - pa)),
				Color = wallCol,
				Material = Enum.Material.Brick,
				Parent = self.static,
			})
		end
	end
end

function Renderer.draw(self: Renderer, L: any, t: number)
	-- bricks: every live brick, stacked by layer
	for _, b in L.bricks do
		if b.Alive then
			local p = take(self.bricks)
			local x, z = Const.cellX(b.C), Const.cellZ(b.R)
			local y = BRICK_SIZE.Y / 2 + b.Layer * Const.LayerH + b.FallY
			local appear = b.Appear
			p.CFrame = CFrame.new(world(x, y - (1 - appear) * 1.5, z))
			local col: Color3 = b.Col
			if b.Flash > 0 then
				col = col:Lerp(Color3.new(1, 1, 1), b.Flash * 0.7)
			end
			if b.Frozen > 0 then
				col = col:Lerp(Color3.fromRGB(170, 225, 255), 0.55)
			elseif b.Burn > 0 then
				col = col:Lerp(Color3.fromRGB(255, 120, 40), 0.35 + 0.15 * math.sin(t * 14 + b.Id))
			end
			if b.MaxHP > 1 and b.HP < b.MaxHP then
				col = col:Lerp(Color3.new(0, 0, 0), 0.35 * (1 - b.HP / b.MaxHP))
			end
			p.Color = col
			p.Material = if b.Kind == Const.Kind.Crystal or b.Kind == Const.Kind.Multiplier
				then Enum.Material.Neon
				elseif b.Kind == Const.Kind.Steel or b.Kind == Const.Kind.Armored then Enum.Material.DiamondPlate
				else Enum.Material.SmoothPlastic
		end
	end
	finish(self.bricks)
	for _, b in L.balls do
		local p = take(self.orbs)
		p.Size = Vector3.one * b.R * 2
		p.CFrame = CFrame.new(world(b.X, b.R, b.Z))
		p.Color = if b.Phantom > 0 then Color3.fromRGB(190, 140, 255)
			elseif L.timers[Const.Power.Fire] > 0 then Color3.fromRGB(255, 140, 50)
			else Color3.fromRGB(255, 240, 200)
	end
	finish(self.orbs)
	for _, d in L.drops do
		local p = take(self.drops)
		local def = Const.PowerDefs[d.P]
		p.Color = def.col
		p.CFrame = CFrame.new(world(d.X, 0.6 + 0.15 * math.sin(d.T * 6), d.Z)) * CFrame.Angles(0, d.T * 2, 0)
		local tag = p:FindFirstChild("Tag")
		local letter = tag and tag:FindFirstChild("Letter")
		if letter then
			(letter :: TextLabel).Text = def.letter
		end
	end
	finish(self.drops)
	for _, c in L.coins do
		local p = take(self.coins)
		p.CFrame = CFrame.new(world(c.X, c.Y, c.Z)) * CFrame.Angles(0, t * 8, math.pi / 2)
	end
	finish(self.coins)
	for _, bl in L.bolts do
		local p = take(self.bolts)
		p.CFrame = CFrame.new(world(bl.X, 0.6, bl.Z))
	end
	finish(self.bolts)
	self.paddle.Size = Vector3.new(L.paddleW, Const.PaddleH, Const.PaddleD)
	self.paddle.CFrame = CFrame.new(world(L.paddleX, Const.PaddleH / 2, Const.PaddleZ))
	if L.barrier > 0 then
		self.barrier.Transparency = 0.2
		self.barrier.Size = Vector3.new(Const.HalfW * 2, 0.8 + 0.3 * L.barrier, 0.15)
		self.barrier.CFrame = CFrame.new(world(0, 0.4, Const.BarrierZ))
	else
		self.barrier.Transparency = 1
	end
end

-- ------------------------------------------------------------------ effects

local function fade(p: BasePart, time: number, goal: { [string]: any })
	goal.Transparency = 1
	TweenService:Create(p, TweenInfo.new(time, Enum.EasingStyle.Quad, Enum.EasingDirection.Out), goal):Play()
	Debris:AddItem(p, time + 0.05)
end

function Renderer.floatText(self: Renderer, x: number, z: number, text: string, col: Color3, big: boolean)
	local anchor = part({ Size = Vector3.one * 0.1, Transparency = 1, CFrame = CFrame.new(world(x, 1.8, z)), Parent = self.folder })
	local gui = Instance.new("BillboardGui")
	gui.Size = UDim2.fromOffset(if big then 360 else 220, if big then 56 else 36)
	gui.AlwaysOnTop = true
	gui.LightInfluence = 0
	local label = Instance.new("TextLabel")
	label.BackgroundTransparency = 1
	label.Size = UDim2.fromScale(1, 1)
	label.Font = Enum.Font.GothamBlack
	label.TextScaled = true
	label.Text = text
	label.TextColor3 = col
	label.TextStrokeTransparency = 0.2
	label.Parent = gui
	gui.Parent = anchor
	TweenService:Create(anchor, TweenInfo.new(1.1), { CFrame = anchor.CFrame + Vector3.new(0, 2.5, 0) }):Play()
	TweenService:Create(label, TweenInfo.new(1.1, Enum.EasingStyle.Quad, Enum.EasingDirection.In), {
		TextTransparency = 1,
		TextStrokeTransparency = 1,
	}):Play()
	Debris:AddItem(anchor, 1.2)
end

function Renderer.burst(self: Renderer, x: number, z: number, col: Color3, n: number)
	for _ = 1, n do
		local p = part({
			Size = Vector3.one * (0.25 + math.random() * 0.25),
			Color = col,
			Material = Enum.Material.SmoothPlastic,
			CFrame = CFrame.new(world(x, 0.5, z)),
			Parent = self.folder,
		})
		local dir = Vector3.new(math.random() - 0.5, math.random() * 0.8 + 0.4, math.random() - 0.5).Unit
		fade(p, 0.45, { CFrame = p.CFrame + dir * (1.5 + math.random() * 2), Size = Vector3.one * 0.05 })
	end
end

function Renderer.ring(self: Renderer, x: number, z: number, r: number, col: Color3)
	local p = part({
		Shape = Enum.PartType.Cylinder,
		Size = Vector3.new(0.2, 0.5, 0.5),
		CFrame = CFrame.new(world(x, 0.3, z)) * CFrame.Angles(0, 0, math.pi / 2),
		Color = col,
		Material = Enum.Material.Neon,
		Transparency = 0.3,
		Parent = self.folder,
	})
	fade(p, 0.4, { Size = Vector3.new(0.2, r * 2, r * 2) })
end

function Renderer.arc(self: Renderer, x0: number, z0: number, x1: number, z1: number, col: Color3)
	local a, b = world(x0, 0.8, z0), world(x1, 0.8, z1)
	local len = (b - a).Magnitude
	if len < 0.05 then
		return
	end
	local p = part({
		Size = Vector3.new(0.18, 0.18, len),
		CFrame = CFrame.lookAt((a + b) / 2, b),
		Color = col,
		Material = Enum.Material.Neon,
		Parent = self.folder,
	})
	fade(p, 0.3, {})
end

-- handle turns simulation events into effects; returns sound names to play
function Renderer.handle(self: Renderer, events: { any })
	for _, e in events do
		local kind = e.type
		if kind == "break" then
			self:burst(e.x, e.z, e.brick.Col, 6)
		elseif kind == "text" then
			self:floatText(e.x, e.z, e.text, e.col, e.big)
		elseif kind == "explode" then
			self:ring(e.x, e.z, e.r, if e.cryo then Color3.fromRGB(150, 225, 255) else Color3.fromRGB(255, 170, 60))
		elseif kind == "ring" then
			self:ring(e.x, e.z, e.r, e.col)
		elseif kind == "blackhole" then
			self:ring(e.x, e.z, e.r, Color3.fromRGB(150, 80, 255))
		elseif kind == "arc" then
			self:arc(e.x0, e.z0, e.x1, e.z1, if e.plasma then Color3.fromRGB(255, 120, 220) else Color3.fromRGB(150, 210, 255))
		elseif kind == "crit" then
			self:ring(e.x, e.z, 1.2, Color3.fromRGB(255, 110, 90))
		elseif kind == "barrier" then
			self:ring(e.x, Const.BarrierZ, 2, Color3.fromRGB(170, 140, 255))
		end
	end
end

function Renderer.destroy(self: Renderer)
	self.folder:Destroy()
end

return Renderer
