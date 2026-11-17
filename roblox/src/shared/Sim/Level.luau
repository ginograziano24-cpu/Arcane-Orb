--!strict
-- Level.luau: one chamber of Arcane Orb, simulated without any Roblox
-- instances. It is a port of the Go game's play.go (Level.Update,
-- collideWalls, collideBricks, collidePaddle, ballHit, damage, destroy,
-- explode, lightning, drops, coins, powers, lasers, lives) with the same
-- numbers, so a chamber plays the way it does on Steam.
--
-- The renderer (client) reads the state (bricks, balls, paddle, drops,
-- coins, bolts) and drains `events` each frame for effects and sound.
-- Not ported yet: Guardians (boss fights), origin passives and abilities,
-- familiars, omens, waves; see docs/ROBLOX_PORT.md.

local Const = require(script.Parent.Const)
local Stats = require(script.Parent.Parent.Data.Stats)

type Stats = Stats.Stats

local K = Const.Kind
local P = Const.Power
local HalfW, FarZ, CellW, CellD = Const.HalfW, Const.FarZ, Const.CellW, Const.CellD
local PaddleZ, PaddleD = Const.PaddleZ, Const.PaddleD

export type Brick = {
	R: number,
	C: number,
	Kind: number,
	HP: number,
	MaxHP: number,
	Col: Color3,
	Alive: boolean,
	Flash: number,
	Burn: number,
	BurnTick: number,
	Frozen: number,
	SinceHit: number,
	Timer: number,
	Hits: number,
	Layer: number,
	Up: Brick?,
	FallY: number,
	Appear: number,
	Id: number,
}

export type Ball = {
	X: number,
	Z: number,
	VX: number,
	VZ: number,
	R: number,
	Stuck: boolean,
	Off: number,
	PierceLeft: number,
	Charged: boolean,
	Momentum: number,
	Phantom: number,
	Rico: number,
	Idle: number,
}

export type Hit = {
	Src: string?,
	Dmg: number,
	Crit: boolean?,
	Void: boolean?,
	Fire: boolean?,
	Blast: boolean?,
	Elemental: boolean?,
	Silent: boolean?,
}

export type Drop = { X: number, Z: number, P: number, T: number }
export type Coin = { X: number, Y: number, Z: number, VX: number, VY: number, VZ: number, Val: number }
export type Bolt = { X: number, Z: number, Last: Brick? }
export type Event = { [string]: any }

export type Input = {
	paddleX: number?, -- absolute paddle target (mouse / touch), world units
	keyDir: number?, -- -1 .. 1 when steering with keys or a stick
	fast: boolean?,
	fire: boolean?, -- serve / fire lasers
}

-- what a Level needs from the run around it
export type RunLike = {
	stats: Stats,
	gold: number,
	lives: number,
	act: number,
	actDepth: number,
	score: number,
	bricks: number,
	elites: number,
	maxCombo: number,
	active: { string },
	fusions: { [string]: boolean },
}

local Level = {}
Level.__index = Level

export type Level = typeof(setmetatable(
	{} :: {
		run: RunLike,
		rng: Random,
		grid: { [number]: { [number]: Brick? } },
		bricks: { Brick },
		balls: { Ball },
		drops: { Drop },
		coins: { Coin },
		bolts: { Bolt },
		walls: { { number } },
		outside: { [number]: { [number]: boolean } },
		events: { Event },
		T: number,
		paddleX: number,
		paddleW: number,
		barrier: number,
		left: number,
		startLeft: number,
		crystals: number,
		combo: number,
		comboT: number,
		frenzy: number,
		impacts: number,
		timers: { [number]: number },
		tempSize: number,
		laserT: number,
		hitStop: number,
		cleared: boolean,
		lost: boolean,
		goldEarned: number,
		killsSinceHeal: number,
		hydraCount: number,
		critsSinceBarrier: number,
		bhCount: number,
		rushBonus: number,
		fuses: { { x: number, z: number, t: number, rad: number, dmg: number } },
		nextId: number,
		depth: number,
		elite: boolean,
		guardian: boolean,
		lostOrbs: number,
	},
	Level
))

local function clamp(v: number, lo: number, hi: number): number
	return math.max(lo, math.min(hi, v))
end

local function hypot(a: number, b: number): number
	return math.sqrt(a * a + b * b)
end

local function lighten(c: Color3, k: number): Color3
	return c:Lerp(Color3.new(1, 1, 1), k)
end

local function fixAngle(b: Ball)
	-- never let an orb fly almost sideways (play.go fixAngle)
	local sp = hypot(b.VX, b.VZ)
	if sp == 0 then
		return
	end
	if math.abs(b.VZ) < 0.3 * sp then
		local sz = if b.VZ < 0 then -1 else 1
		b.VZ = sz * 0.3 * sp
		local sx = if b.VX < 0 then -1 else 1
		b.VX = sx * math.sqrt(sp * sp - b.VZ * b.VZ)
	end
end
Level.fixAngle = fixAngle

-- ------------------------------------------------------------------ setup

-- layout: a LevelBank entry; hues: the world's six brick colours;
-- arena: the wall polyline for layout.shape
function Level.new(run: RunLike, layout: any, arena: { { number } }, hues: { Color3 }, seed: number, hpMul: number): Level
	local self = setmetatable({
		run = run,
		rng = Random.new(seed),
		grid = {},
		bricks = {},
		balls = {},
		drops = {},
		coins = {},
		bolts = {},
		walls = arena,
		outside = {},
		events = {},
		T = 0,
		paddleX = 0,
		paddleW = Const.BaseW * run.stats.PaddleWidth,
		barrier = math.min(5, run.stats.BarrierPerLevel),
		left = 0,
		startLeft = 0,
		crystals = 0,
		combo = 0,
		comboT = 0,
		frenzy = 0,
		impacts = 0,
		timers = {},
		tempSize = 0,
		laserT = 0,
		hitStop = 0,
		cleared = false,
		lost = false,
		goldEarned = 0,
		killsSinceHeal = 0,
		hydraCount = 0,
		critsSinceBarrier = 0,
		bhCount = 0,
		rushBonus = 0,
		fuses = {},
		nextId = 0,
		depth = 0,
		elite = layout.elite == true,
		guardian = false,
		lostOrbs = 0,
	}, Level)
	for r = 0, Const.GridRows - 1 do
		self.grid[r] = {}
		self.outside[r] = {}
	end
	for i = 0, Const.NumPowers - 1 do
		self.timers[i] = 0
	end
	self:build(layout, hues, hpMul)
	self.startLeft = self.left
	self:spawnServe()
	return self
end

function Level.build(self: Level, layout: any, hues: { Color3 }, hpMul: number)
	local cells: string = layout.cells
	local shift: number = layout.layerShift or 0
	for r = 0, Const.MaxRows - 1 do
		for c = 0, Const.Cols - 1 do
			local i = (r * Const.Cols + c) * 5 + 1
			local kind = string.byte(cells, i) - 97
			if kind ~= K.Empty then
				local hp = string.byte(cells, i + 1) - 48
				local colour = string.byte(cells, i + 2) - 48
				local stack = string.byte(cells, i + 3) - 48
				local cap = string.byte(cells, i + 4) - 97
				local h = if kind == K.Steel then hp else math.ceil(hp * hpMul)
				local base = hues[(colour % 6) + 1]
				local bottom = self:addBrick(r, c, kind, h, base)
				local prev = bottom
				for k = 1, stack - 1 do
					local lk = K.Basic
					local lhp = math.ceil(math.max(1, hp) * hpMul)
					if kind == K.Steel then
						lk, lhp = K.Steel, 1
					end
					if k == stack - 1 and cap ~= K.Empty then
						lk = cap
					end
					local col = hues[((colour + k * shift) % 6) + 1]
					if shift == 0 then
						col = lighten(col, 0.08 * k)
					end
					local nb = self:newBrick(r, c, lk, lhp, col)
					nb.Layer = k
					if lk == K.Explosive or lk == K.Crystal then
						nb.HP, nb.MaxHP = 1, 1
					end
					table.insert(self.bricks, nb)
					if Const.required(lk) then
						self.left += 1
					end
					if lk == K.Crystal then
						self.crystals += 1
					end
					prev.Up = nb
					prev = nb
				end
			end
		end
	end
end

function Level.newBrick(self: Level, r: number, c: number, kind: number, hp: number, base: Color3): Brick
	self.nextId += 1
	return {
		R = r,
		C = c,
		Kind = kind,
		HP = hp,
		MaxHP = hp,
		Col = Const.KindColor[kind] or base,
		Alive = true,
		Flash = 0,
		Burn = 0,
		BurnTick = 0,
		Frozen = 0,
		SinceHit = 0,
		Timer = 0,
		Hits = 0,
		Layer = 0,
		Up = nil,
		FallY = 0,
		Appear = 0,
		Id = self.nextId,
	}
end

function Level.addBrick(self: Level, r: number, c: number, kind: number, hp: number, base: Color3): Brick
	local br = self:newBrick(r, c, kind, hp, base)
	if Const.OneHit[kind] then
		br.HP, br.MaxHP = 1, 1
	end
	self.grid[r][c] = br
	table.insert(self.bricks, br)
	if Const.required(kind) then
		self.left += 1
	end
	if kind == K.Crystal then
		self.crystals += 1
	end
	self:emit({ type = "spawn", brick = br })
	return br
end

function Level.emit(self: Level, e: Event)
	table.insert(self.events, e)
end

function Level.text(self: Level, x: number, z: number, s: string, col: Color3, big: boolean?)
	self:emit({ type = "text", x = x, z = z, text = s, col = col, big = big == true })
end

function Level.ballR(self: Level): number
	return math.min(0.75, 0.32 * self.run.stats.SizeMult)
end

function Level.speed(self: Level): number
	local r = self.run
	local base = math.min(22, 16 + r.actDepth * 0.3 + (r.act - 1) * 1.2)
	base += math.min(4, self.T * 0.035)
	base *= r.stats.SpeedMult
	if self.timers[P.Slow] > 0 then
		base *= 0.7
	end
	if self.timers[P.Haste] > 0 then
		base *= 1.3
	end
	return base
end

function Level.paddleTarget(self: Level): number
	local w = Const.BaseW * self.run.stats.PaddleWidth
	if self.tempSize > 0 then
		w *= 1 + 0.35 * self.tempSize
	elseif self.tempSize < 0 then
		w *= 0.7
	end
	return w
end

function Level.spawnServe(self: Level)
	table.insert(self.balls, {
		X = self.paddleX,
		Z = PaddleZ + 0.6,
		VX = 0,
		VZ = 0,
		R = self:ballR(),
		Stuck = true,
		Off = (self.rng:NextNumber() - 0.5) * 0.8,
		PierceLeft = self.run.stats.Pierce,
		Charged = false,
		Momentum = 0,
		Phantom = 0,
		Rico = 0,
		Idle = 0,
	})
end

function Level.newBall(self: Level, x: number, z: number, vx: number, vz: number, from: Ball?): Ball
	return {
		X = x,
		Z = z,
		VX = vx,
		VZ = vz,
		R = if from then from.R else self:ballR(),
		Stuck = false,
		Off = 0,
		PierceLeft = if from then from.PierceLeft else self.run.stats.Pierce,
		Charged = false,
		Momentum = 0,
		Phantom = if from then from.Phantom else 0,
		Rico = 0,
		Idle = 0,
	}
end

function Level.anyStuck(self: Level): boolean
	for _, b in self.balls do
		if b.Stuck then
			return true
		end
	end
	return false
end

-- release launches every orb resting on the paddle
function Level.release(self: Level): boolean
	local any = false
	local s = self.run.stats
	local sp = self:speed()
	for _, b in self.balls do
		if b.Stuck then
			b.Stuck = false
			local t = b.Off / (self.paddleW / 2)
			local ang = t * 50 * math.pi / 180 + (self.rng:NextNumber() - 0.5) * 0.1
			b.VX, b.VZ = sp * math.sin(ang), sp * math.cos(ang)
			if s.OpeningCrit > 0 then
				b.Charged = true
			end
			any = true
		end
	end
	if any then
		self:emit({ type = "sound", name = "launch" })
		-- the first launch of a serve brings the extra orbs
		if self.T < 0.5 or self:realBalls() == 1 then
			for i = 0, s.ExtraBalls - 1 do
				local a = (i + 1) * 0.35 * (1 - 2 * (i % 2))
				table.insert(self.balls, self:newBall(self.paddleX, PaddleZ + 1, sp * math.sin(a), sp * math.cos(a), nil))
			end
			if s.PrismOrb then
				local pb = self:newBall(self.paddleX, PaddleZ + 1, sp * math.sin(-0.5), sp * math.cos(-0.5), nil)
				pb.Phantom, pb.PierceLeft = 8, 2
				table.insert(self.balls, pb)
			end
		end
	end
	return any
end

function Level.realBalls(self: Level): number
	local n = 0
	for _, b in self.balls do
		if b.Phantom <= 0 then
			n += 1
		end
	end
	return n
end

function Level.cellFree(self: Level, r: number, c: number): boolean
	return r >= 0 and r < Const.GridRows and c >= 0 and c < Const.Cols and self.grid[r][c] == nil and not self.outside[r][c]
end

function Level.hasFusion(self: Level, id: string): boolean
	return self.run.fusions[id] == true
end

function Level.hasSynergy(self: Level, name: string): boolean
	for _, a in self.run.active do
		if a == name then
			return true
		end
	end
	return false
end

-- ------------------------------------------------------------------ update

function Level.step(self: Level, dt: number, input: Input)
	local r = self.run
	local s = r.stats
	if self.lost then
		return
	end
	if self.hitStop > 0 then
		self.hitStop -= dt
		dt *= 0.12
	end
	self.T += dt
	for i = 0, Const.NumPowers - 1 do
		if self.timers[i] > 0 then
			self.timers[i] -= dt
			if self.timers[i] <= 0 then
				self.timers[i] = 0
				if i == P.Glue then
					self:release()
				end
			end
		end
	end
	if self.frenzy > 0 then
		self.frenzy -= dt
	end
	self:updateFuses(dt)
	self.comboT -= dt
	if self.comboT <= 0 and self.combo > 0 then
		if not self:hasSynergy("Overdrive") or #self.balls == 0 then
			self.combo = 0
		end
	end

	-- paddle
	local tw = self:paddleTarget()
	self.paddleW += (tw - self.paddleW) * math.min(1, 10 * dt)
	local target = self.paddleX
	if input.paddleX then
		target = input.paddleX
	elseif input.keyDir then
		target += input.keyDir * (if input.fast then 46 else 30) * dt
	end
	local half = self.paddleW / 2
	self.paddleX = clamp(target, -HalfW + half, HalfW - half)
	for _, b in self.balls do
		if b.Stuck then
			local h = self.paddleW / 2 - 0.2
			b.Off = clamp(b.Off, -h, h)
			b.R = self:ballR()
			b.X, b.Z = self.paddleX + b.Off, PaddleZ + PaddleD / 2 + b.R + 0.02
		end
	end
	if not self.cleared then
		if input.fire then
			if not self:release() and self.timers[P.Laser] > 0 then
				self:fireLaser()
			end
		end
		local interval = s.LaserInterval
		if self.timers[P.Laser] > 0 and (interval == 0 or interval > 0.45) then
			interval = 0.45
		end
		if interval > 0 and not self:anyStuck() then
			self.laserT -= dt
			if self.laserT <= 0 then
				self.laserT = interval
				self:fireLaser()
			end
		end
	end

	-- orbs, sub-stepped so a fast orb never tunnels through a brick
	local sp = self:speed()
	for _, b in self.balls do
		if not b.Stuck then
			if b.Phantom > 0 then
				b.Phantom -= dt
				if b.Phantom <= 0 then
					b.Z = -100
				end
			end
			if b.Z > -100 then
				b.R += (self:ballR() - b.R) * math.min(1, dt * 6)
				b.Idle += dt
				local v = sp
				if s.Momentum then
					v *= 1 + 0.04 * math.min(15, b.Momentum)
				end
				local cur = hypot(b.VX, b.VZ)
				if cur > 0 then
					b.VX, b.VZ = b.VX / cur * v, b.VZ / cur * v
				end
				local steps = math.max(1, math.ceil(v * dt / (b.R * 0.45)))
				local h = dt / steps
				for _ = 1, steps do
					if b.Stuck then
						break
					end
					b.X += b.VX * h
					b.Z += b.VZ * h
					self:collideWalls(b)
					self:collideBricks(b)
					self:collidePaddle(b)
					if self.barrier > 0 and b.VZ < 0 and b.Z - b.R <= Const.BarrierZ and b.Z > Const.BarrierZ - 1 then
						b.Z = Const.BarrierZ + b.R
						b.VZ = math.abs(b.VZ)
						self.barrier -= 1
						self:emit({ type = "barrier", x = b.X })
						if s.Sanctuary then
							self:gainGold(3, b.X, Const.BarrierZ + 1, true)
						end
					end
				end
			end
		end
	end
	local alive = {}
	for _, b in self.balls do
		if b.Z > Const.LoseZ then
			table.insert(alive, b)
		end
	end
	self.balls = alive
	if not self.cleared and self:realBalls() == 0 then
		self:loseBall()
		if self.lost then
			return
		end
	end
	self:updateBricks(dt)
	self:updateDrops(dt)
	if not self.cleared and self.left <= 0 then
		self:clear()
	end
end

-- ------------------------------------------------------------------ collisions

function Level.collideWalls(self: Level, b: Ball)
	local hit = false
	local W = self.walls
	for i = 1, #W - 1 do
		local a, c = W[i], W[i + 1]
		local ax, az, cx, cz = a[1], a[2], c[1], c[2]
		local ex, ez = cx - ax, cz - az
		local l2 = ex * ex + ez * ez
		local t = clamp(((b.X - ax) * ex + (b.Z - az) * ez) / l2, 0, 1)
		local px, pz = ax + ex * t, az + ez * t
		local dx, dz = b.X - px, b.Z - pz
		local d = hypot(dx, dz)
		if d < b.R then
			local nx, nz = -ez, ex
			local l = hypot(nx, nz)
			nx, nz = nx / l, nz / l
			if nx * (0 - ax) + nz * (10 - az) < 0 then
				nx, nz = -nx, -nz
			end
			if d > 1e-6 and not (t > 0 and t < 1) then
				nx, nz = dx / d, dz / d -- a corner: push straight out
			end
			local pen = b.R - (dx * nx + dz * nz)
			if pen > 0 then
				b.X += nx * pen
				b.Z += nz * pen
				local dot = b.VX * nx + b.VZ * nz
				if dot < 0 then
					b.VX -= 2 * dot * nx
					b.VZ -= 2 * dot * nz
				end
				hit = true
			end
		end
	end
	if b.X - b.R < -HalfW then
		b.X = -HalfW + b.R
		b.VX = math.abs(b.VX)
		hit = true
	end
	if b.X + b.R > HalfW then
		b.X = HalfW - b.R
		b.VX = -math.abs(b.VX)
		hit = true
	end
	if b.Z + b.R > FarZ then
		b.Z = FarZ - b.R
		b.VZ = -math.abs(b.VZ)
		hit = true
	end
	if hit then
		fixAngle(b)
		self:stallNudge(b)
		b.Rico += 1
		b.PierceLeft = math.max(b.PierceLeft, self.run.stats.Pierce)
		self:emit({ type = "wall", x = b.X, z = b.Z })
	end
end

-- stallNudge: an orb that has gone 9 s without touching the paddle or
-- hurting a brick turns a little towards the paddle on every wall bounce,
-- so a loop between walls and steel can't stall a chamber (antistall.go)
function Level.stallNudge(self: Level, b: Ball)
	if b.Idle < 9 or b.Stuck then
		return
	end
	local sp = hypot(b.VX, b.VZ)
	if sp == 0 then
		return
	end
	local a = math.atan2(b.VX, b.VZ)
	if a >= 0 then
		a = math.min(math.pi * 0.8, a + 0.16)
	else
		a = math.max(-math.pi * 0.8, a - 0.16)
	end
	b.VX, b.VZ = sp * math.sin(a), sp * math.cos(a)
	if math.abs(b.VZ) < 0.3 * sp then
		b.VZ = -0.3 * sp
		local vx = math.sqrt(sp * sp - b.VZ * b.VZ)
		b.VX = if b.VX < 0 then -vx else vx
	end
end

function Level.collideBricks(self: Level, b: Ball)
	local reach = b.R + Const.OrbSplash
	local c0 = math.floor((b.X - reach + HalfW) / CellW)
	local c1 = math.floor((b.X + reach + HalfW) / CellW)
	local r0 = Const.rowAt(b.Z + reach)
	local r1 = Const.rowAt(b.Z - reach)
	local splash: { Brick } = {}
	local hits: { Brick } = {}
	local best: Brick? = nil
	local bestPen, nx, nz = -1, 0, 0
	for rr = math.max(r0, 0), math.min(r1, Const.GridRows - 1) do
		local row = self.grid[rr]
		for cc = math.max(c0, 0), math.min(c1, Const.Cols - 1) do
			local br = row[cc]
			if br and br.Alive then
				local bx, bz = Const.cellX(br.C), Const.cellZ(br.R)
				local px = clamp(b.X, bx - CellW / 2, bx + CellW / 2)
				local pz = clamp(b.Z, bz - CellD / 2, bz + CellD / 2)
				local dx, dz = b.X - px, b.Z - pz
				local d2 = dx * dx + dz * dz
				if d2 >= b.R * b.R then
					if d2 < reach * reach then
						table.insert(splash, br)
					end
				else
					table.insert(hits, br)
					local d = math.sqrt(d2)
					local ux, uz, pen
					if d < 1e-6 then
						local v = hypot(b.VX, b.VZ)
						ux, uz, pen = -b.VX / v, -b.VZ / v, b.R
					else
						ux, uz, pen = dx / d, dz / d, b.R - d
					end
					if pen > bestPen then
						best, bestPen, nx, nz = br, pen, ux, uz
					end
				end
			end
		end
	end
	if best == nil then
		return
	end
	local s = self.run.stats
	local reflect = true
	if #splash > 0 then
		local sd = math.max(1, s.Damage * s.DmgMult) * 0.75
		for _, br in splash do
			if br.Alive and br.Kind ~= K.Steel then
				self:damage(br, { Dmg = sd, Silent = true })
			end
		end
	end
	self:emit({ type = "impact", x = b.X, z = b.Z, nx = nx, nz = nz, col = best.Col })
	for _, br in hits do
		if br.Alive then
			local destroyed, solid = self:ballHit(b, br)
			if br == best then
				if not solid then
					reflect = false
				elseif destroyed and b.PierceLeft > 0 then
					b.PierceLeft -= 1
					reflect = false
				elseif destroyed and br.Kind ~= K.Elite and self.rng:NextNumber() < Const.OrbPlow then
					reflect = false -- small bricks don't stop a moving orb: it ploughs on
				elseif destroyed and b.Phantom > 0 then
					reflect = false
				end
			end
		end
	end
	if reflect then
		b.X += nx * bestPen
		b.Z += nz * bestPen
		local dot = b.VX * nx + b.VZ * nz
		if dot < 0 then
			b.VX -= 2 * dot * nx
			b.VZ -= 2 * dot * nz
		end
		if best.Kind == K.Steel then
			local a = (self.rng:NextNumber() - 0.5) * 0.08
			local ca, sa = math.cos(a), math.sin(a)
			b.VX, b.VZ = b.VX * ca - b.VZ * sa, b.VX * sa + b.VZ * ca
		end
		fixAngle(b)
	end
end

function Level.lastBrickAim(self: Level, b: Ball): number?
	-- the last few bricks: bounces lean toward the nearest one
	if self.left <= 0 or self.left > math.max(3, self.startLeft // 12) then
		return nil
	end
	local bestD, ang = math.huge, nil
	for _, br in self.bricks do
		if br.Alive and Const.required(br.Kind) then
			local dx, dz = Const.cellX(br.C) - b.X, Const.cellZ(br.R) - b.Z
			local d = dx * dx + dz * dz
			if d < bestD and dz > 0 then
				bestD, ang = d, math.atan2(dx, dz)
			end
		end
	end
	return ang
end

function Level.collidePaddle(self: Level, b: Ball)
	if b.VZ >= 0 then
		return
	end
	local half = self.paddleW / 2
	local px = clamp(b.X, self.paddleX - half, self.paddleX + half)
	local pz = clamp(b.Z, PaddleZ - PaddleD / 2, PaddleZ + PaddleD / 2)
	local dx, dz = b.X - px, b.Z - pz
	if dx * dx + dz * dz >= b.R * b.R or b.Z < PaddleZ - PaddleD * 0.2 then
		return
	end
	local s = self.run.stats
	local sp = hypot(b.VX, b.VZ)
	local t = clamp((b.X - self.paddleX) / half, -1, 1)
	local ang = t * 62 * math.pi / 180
	local aim = self:lastBrickAim(b)
	if aim then
		ang += (aim - ang) * 0.55
	end
	b.VX, b.VZ = sp * math.sin(ang), sp * math.cos(ang)
	b.Z = PaddleZ + PaddleD / 2 + b.R
	b.Idle = 0
	b.PierceLeft = s.Pierce
	b.Momentum = 0
	b.Rico = 0
	self:emit({ type = "paddle", x = b.X, edge = math.abs(t) >= 0.65 })
	if s.HeavyPaddle then
		b.Charged = true
	end
	if s.GoldenPaddle then
		self:gainGold(1, b.X, b.Z, false)
	end
	if self.timers[P.Glue] > 0 and b.Phantom <= 0 then
		b.Stuck = true
		b.Off = b.X - self.paddleX
	end
	if s.SplitChance > 0 and self.rng:NextNumber() < s.SplitChance and #self.balls < 14 then
		local a = ang + 0.45 * (1 - 2 * self.rng:NextInteger(0, 1))
		table.insert(self.balls, self:newBall(b.X, b.Z, sp * math.sin(a), sp * math.cos(a), nil))
	end
end

-- ------------------------------------------------------------------ hits

function Level.shielded(self: Level, b: Brick): boolean
	if b.Kind == K.Shield then
		return false
	end
	for dr = -1, 1 do
		for dc = -1, 1 do
			local r, c = b.R + dr, b.C + dc
			if r >= 0 and r < Const.GridRows and c >= 0 and c < Const.Cols and not (dr == 0 and dc == 0) then
				local n = self.grid[r][c]
				if n and n.Alive and n.Kind == K.Shield then
					return true
				end
			end
		end
	end
	return false
end

function Level.neighbours(self: Level, row: number, col: number): { Brick }
	local out = {}
	for _, d in { { 1, 0 }, { -1, 0 }, { 0, 1 }, { 0, -1 } } do
		local rr, cc = row + d[1], col + d[2]
		if rr >= 0 and rr < Const.GridRows and cc >= 0 and cc < Const.Cols then
			local n = self.grid[rr][cc]
			if n and n.Alive and n.Kind ~= K.Steel then
				table.insert(out, n)
			end
		end
	end
	return out
end

-- damage applies a hit; returns destroyed, solid (solid = false: the orb
-- passes through, as a fire orb does)
function Level.damage(self: Level, b: Brick, h: Hit): (boolean, boolean)
	local s = self.run.stats
	b.Flash = 1
	b.SinceHit = 0
	if b.Kind == K.Steel then
		if not h.Silent then
			self:emit({ type = "sound", name = "steel" })
		end
		return false, true
	end
	if self:shielded(b) and not h.Void then
		if not h.Silent then
			self:emit({ type = "sound", name = "shieldhit" })
		end
		return false, true
	end
	if b.Kind == K.Armored and not (h.Crit or h.Void or h.Blast or h.Elemental) then
		b.Hits += 1 + s.SunderMult
		if not h.Silent then
			self:emit({ type = "sound", name = "steel" })
		end
		if b.Hits >= 6 then
			b.Kind = K.Basic
			b.Col = lighten(b.Col, 0.25)
			self:text(Const.cellX(b.C), Const.cellZ(b.R), "ARMOR CRACKS", Color3.fromRGB(220, 220, 230))
		end
		return false, true
	end
	local dmg = h.Dmg
	if b.Frozen > 0 then
		if s.AbsoluteZero then
			dmg *= 3
		elseif s.FrozenBonus then
			dmg *= 2
		else
			dmg *= 1.5
		end
	end
	if b.Burn > 0 and s.PyroAsh then
		dmg *= 2
	end
	if h.Void and s.VoidDouble then
		dmg *= 3
	end
	b.HP -= dmg
	if b.HP > 0 and s.ExecuteBelow > 0 and b.HP <= b.MaxHP * s.ExecuteBelow then
		b.HP = 0
		self:text(Const.cellX(b.C), Const.cellZ(b.R), "EXECUTE", Color3.fromRGB(255, 90, 90))
	end
	if b.HP > 0 then
		if not h.Silent then
			self:emit({ type = "hit", brick = b })
		end
		return false, not h.Fire
	end
	self:destroy(b, h)
	return true, not h.Fire
end

function Level.comboCap(self: Level): number
	-- Perpetual Motion lifts the combo's damage cap to +200%
	return if self:hasFusion("perpetual") then 2 else 1
end

function Level.ballHit(self: Level, ball: Ball, b: Brick): (boolean, boolean)
	local r = self.run
	local s = r.stats
	self.impacts += 1
	if b.Kind ~= K.Steel and not self:shielded(b) then
		ball.Idle = 0
	end
	local charged = ball.Charged
	local crit = self.rng:NextNumber() < s.CritChance or charged
	ball.Charged = false
	local wasFrozen = b.Frozen > 0
	local dmg = s.Damage * s.DmgMult
	dmg *= 1 + math.min(self:comboCap(), self.combo * s.ComboDmg)
	if s.Momentum then
		dmg *= 1 + 0.04 * math.min(15, ball.Momentum)
	end
	if s.BloodMoon and r.lives <= 1 then
		dmg *= 1.75
	end
	if s.Berserker and r.lives <= 1 then
		dmg *= 1.5
	end
	if s.Ricochet > 0 then
		dmg *= 1 + s.Ricochet * math.min(ball.Rico, 12)
	end
	if s.BossDmg > 0 and b.Kind == K.Elite then
		dmg *= 1 + s.BossDmg
	end
	local extra = self:realBalls() - 1
	if s.HiveDmg > 0 and extra > 0 then
		dmg *= 1 + math.min(s.HiveDmg * 6, s.HiveDmg * extra)
	end
	if s.LoneDmg > 0 and self:realBalls() == 1 then
		dmg *= 1 + s.LoneDmg
	end
	if s.Avarice > 0 then
		dmg *= 1 + math.min(s.Avarice, r.gold / 1000)
	end
	if s.BattleHymn > 0 and self.T < s.BattleHymn then
		dmg *= 2
	end
	dmg *= 1 + self.rushBonus
	if ball.Phantom > 0 and s.VoidPrism then
		dmg *= 2
	end
	if charged and s.Haymaker then
		dmg *= 2.5
		self.combo += 3
	end
	if crit then
		dmg *= s.CritMult
	end
	local fire = self.timers[P.Fire] > 0
	local h: Hit = {
		Dmg = dmg,
		Crit = crit,
		Void = self.rng:NextNumber() < s.VoidChance or (ball.Phantom > 0 and s.VoidPrism),
		Fire = fire and b.Kind ~= K.Steel,
		Elemental = fire,
	}
	if fire then
		h.Dmg *= 2
	end
	local bx, bz = Const.cellX(b.C), Const.cellZ(b.R)
	local times = 1
	if (s.EchoEvery > 0 and self.impacts % s.EchoEvery == 0) or (s.EchoChance > 0 and self.rng:NextNumber() < s.EchoChance) then
		times = 2
		self:text(bx, bz, "ECHO", Color3.fromRGB(200, 200, 255))
		if s.Thunderclap then
			self:lightning(bx, bz, 3, math.max(1, dmg * 0.5), b, false)
		end
	end
	local destroyed, solid = false, true
	for _ = 1, times do
		if not b.Alive then
			break
		end
		destroyed, solid = self:damage(b, h)
	end
	if s.Crescendo > 0 and self.combo >= 2 and b.Alive then
		local note = dmg * math.min(s.Crescendo * self.combo, s.Crescendo * 20)
		self:damage(b, { Src = "Crescendo", Dmg = note, Silent = true, Void = h.Void, Crit = crit })
	end
	if crit then
		self.hitStop = math.max(self.hitStop, 0.035)
		self:emit({ type = "crit", x = bx, z = bz })
		if s.CritBarrier > 0 then
			self.critsSinceBarrier += 1
			if self.critsSinceBarrier >= s.CritBarrier then
				self.critsSinceBarrier = 0
				self:raiseBarrier(1, 3)
			end
		end
		if s.CritCoin > 0 and self.rng:NextNumber() < s.CritCoin then
			self:gainGold(1, bx, bz, true)
		end
		if s.Hawkstrike and b.R > 0 then
			local n = self.grid[b.R - 1][b.C]
			if n and n.Alive and n ~= b then
				self:damage(n, { Src = "Hawkstrike", Dmg = dmg * 0.75, Crit = true, Silent = true })
			end
		end
	end
	if b.Kind == K.Steel then
		return false, true
	end
	if s.FreezeSpread and wasFrozen then
		for _, n in self:neighbours(b.R, b.C) do
			n.Frozen = 4 + s.FreezeDur
		end
	end
	if s.DragonHeart and b.Alive and b.Burn > 0 then
		for _, n in self:neighbours(b.R, b.C) do
			n.Burn = 3.5
		end
	end
	if s.PlasmaStorm and b.Burn > 0 and self.rng:NextNumber() < 0.35 then
		self:lightning(bx, bz, 2, math.max(1, dmg * 0.5), b, false)
	end
	if crit and s.CritPierce and destroyed then
		solid = false
	end
	ball.Momentum += 1
	-- combo
	self.combo += 1
	self.comboT = 3 + s.ComboTime
	if self.combo > r.maxCombo then
		r.maxCombo = self.combo
	end
	if self.combo % 10 == 0 then
		self:text(bx, bz, string.format("COMBO ×%d", self.combo), Color3.fromRGB(255, 230, 120), true)
		for _ = 1, math.min(4, 1 + self.combo // 25) do
			self:gainGold(1, bx, bz, true)
		end
	end
	-- procs
	if b.Alive and (self.rng:NextNumber() < s.BurnChance or fire) then
		if b.Burn <= 0 then
			self:text(bx, bz, "BURN", Color3.fromRGB(255, 140, 40))
		end
		b.Burn = 3.5
	end
	if b.Alive and self.rng:NextNumber() < s.FreezeChance then
		b.Frozen = 4 + s.FreezeDur
		self:emit({ type = "sound", name = "freeze" })
	end
	if self.rng:NextNumber() < s.ChainChance or (crit and s.ThunderCrown) then
		self:lightning(bx, bz, s.ChainCount + 1, math.max(1, dmg * 0.6), b, s.StormCaller or s.ChainRecast)
	end
	if s.MeteorHeart or self.rng:NextNumber() < s.ExplodeChance then
		self:explode(bx, bz, s.ExplodeRadius, math.max(1.5, dmg), false)
	end
	if crit and s.EagleEye then
		for _, d in { { 1, 0 }, { -1, 0 }, { 0, 1 }, { 0, -1 } } do
			local rr, cc = b.R + d[1], b.C + d[2]
			if rr >= 0 and rr < Const.GridRows and cc >= 0 and cc < Const.Cols then
				local n = self.grid[rr][cc]
				if n and n.Alive then
					self:damage(n, { Src = "Eagle Eye", Dmg = dmg * 0.5, Silent = true })
				end
			end
		end
	end
	if s.MoltenEvery > 0 and self.impacts % s.MoltenEvery == 0 then
		self:text(bx, bz, "MOLTEN CORE!", Color3.fromRGB(255, 140, 40), true)
		self:explode(bx, bz, s.ExplodeRadius + 1.6, dmg * 3, true)
	end
	return destroyed, solid
end

function Level.removeFromColumn(self: Level, b: Brick)
	local above: Brick? = nil
	if self.grid[b.R][b.C] == b then
		self.grid[b.R][b.C] = b.Up
		above = b.Up
	else
		local n = self.grid[b.R][b.C]
		while n do
			if n.Up == b then
				n.Up = b.Up
				above = b.Up
				break
			end
			n = n.Up
		end
	end
	local n = above
	while n do
		n.Layer -= 1
		n.FallY += Const.LayerH
		n = n.Up
	end
	b.Up = nil
end

function Level.destroy(self: Level, b: Brick, h: Hit)
	if not b.Alive then
		return
	end
	local r = self.run
	local s = r.stats
	b.Alive = false
	self:removeFromColumn(b)
	if Const.required(b.Kind) then
		self.left -= 1
	end
	if b.Kind == K.Crystal then
		self.crystals -= 1
	end
	r.bricks += 1
	local x, z = Const.cellX(b.C), Const.cellZ(b.R)
	local pts = math.floor(10 * b.MaxHP)
	if self.frenzy > 0 then
		pts *= 2
	end
	r.score += pts * (1 + self.combo // 10)
	self:emit({ type = "break", brick = b, x = x, z = z, combo = self.combo })

	-- gold
	local coinP = (0.32 + s.CoinChance) * Const.KillWeight
	if s.Philosopher then
		coinP = (0.6 + s.CoinChance) * Const.KillWeight
	end
	if self.rng:NextNumber() < coinP then
		self:gainGold(1 + self.rng:NextInteger(0, 1), x, z, true)
	end
	if s.GoldPerBrick then
		self:gainGold(1, x, z, false)
	end
	if s.BarrierEvery > 0 then
		self.killsSinceHeal += 1
		if self.killsSinceHeal >= s.BarrierEvery then
			self.killsSinceHeal = 0
			self:raiseBarrier(1, 3)
		end
	end
	if s.Hydra then
		self.hydraCount += 1
		local need = math.floor(30 / Const.KillWeight)
		if self:hasFusion("brood") then
			need = math.floor(15 / Const.KillWeight)
		end
		if self.hydraCount >= need and #self.balls < 14 then
			self.hydraCount = 0
			local sp = self:speed()
			local a = (self.rng:NextNumber() - 0.5) * 1.2
			table.insert(self.balls, self:newBall(x, z - 1, sp * math.sin(a), -sp * math.cos(a), nil))
			self:text(x, z, "HYDRA!", Color3.fromRGB(120, 255, 160), true)
		end
	end
	-- power-up drop
	local dropP = 0.055 * Const.KillWeight * s.Luck
	if b.MaxHP >= 2 then
		dropP *= 1.4
	end
	if b.Kind == K.Elite then
		self:dropPower(x, z, true)
		self:gainGold(10, x, z, true)
		r.elites += 1
		self:text(x, z, "ELITE TOPPLED!", Color3.fromRGB(220, 160, 255), true)
	elseif b.Kind == K.Basic or b.Kind == K.Regen or b.Kind == K.Moving or b.Kind == K.Armored or b.Kind == K.Split then
		if self.rng:NextNumber() < dropP and #self.drops < 5 then
			self:dropPower(x, z, false)
		end
	end
	-- kind effects
	if b.Kind == K.Explosive then
		local rad = Const.BrickBlastR + math.max(0, s.ExplodeRadius - Const.BaseExplodeR) * 0.4
		if h.Blast then
			-- set off by another blast: a short fuse, so chains ripple
			table.insert(self.fuses, { x = x, z = z, t = 0.16, rad = rad, dmg = Const.BrickBlastDmg })
		else
			self:explode(x, z, rad, Const.BrickBlastDmg, false)
		end
	elseif b.Kind == K.Chain then
		self:lightning(x, z, 4, 2, b, s.StormCaller)
	elseif b.Kind == K.Split then
		local n = 0
		for _, d in { { 0, -1 }, { 0, 1 }, { 1, 0 }, { -1, 0 } } do
			local rr, cc = b.R + d[1], b.C + d[2]
			if n < 2 and self:cellFree(rr, cc) then
				self:addBrick(rr, cc, K.Basic, 1, lighten(b.Col, 0.2))
				n += 1
			end
		end
	elseif b.Kind == K.Multiplier then
		self.frenzy = 7
		self.combo += 5
		self:text(x, z, "FRENZY! ×2", Color3.fromRGB(255, 120, 220), true)
	elseif b.Kind == K.Crystal then
		for _ = 1, if s.Philosopher then 15 else 5 do
			self:gainGold(1, x, z, true)
		end
	elseif b.Kind == K.Mystery then
		self:mystery(x, z)
	end
	if b.Burn > 0 and s.Wildfire then
		for _, n in self:neighbours(b.R, b.C) do
			n.Burn = 3.5
		end
		if self:hasSynergy("Inferno") then
			self:explode(x, z, 1.2, 1.5, false)
		end
	end
	if s.KillSplash > 0 then
		for _, n in self:neighbours(b.R, b.C) do
			self:damage(n, { Src = "Shrapnel", Dmg = math.max(0.5, s.Damage * s.DmgMult * s.KillSplash), Blast = true, Silent = true })
		end
	end
	if s.BlackHole then
		self.bhCount += 1
		if self.bhCount >= 40 then
			self.bhCount = 0
			self:blackHole(x, z)
		end
	end
	if s.Volcano and b.Burn > 0 then
		self:explode(x, z, 1.0, math.max(1, s.Damage), false)
	end
	if b.Frozen > 0 and s.Shatter then
		local shR = if s.AbsoluteZero then 4.2 else 2.6
		self:emit({ type = "ring", x = x, z = z, r = shR + 0.4, col = Color3.fromRGB(170, 230, 255) })
		for _, n in self:near(x, z, shR) do
			if n ~= b then
				self:damage(n, { Src = "Shatter", Dmg = 1.5, Elemental = true, Silent = true })
			end
		end
	end
end

function Level.near(self: Level, x: number, z: number, rad: number): { Brick }
	local out = {}
	for _, b in self.bricks do
		if b.Alive and hypot(Const.cellX(b.C) - x, (Const.cellZ(b.R) - z) * 1.4) <= rad then
			table.insert(out, b)
		end
	end
	return out
end

function Level.updateFuses(self: Level, dt: number)
	if #self.fuses == 0 then
		return
	end
	local due, keep = {}, {}
	for _, f in self.fuses do
		f.t -= dt
		if f.t <= 0 then
			table.insert(due, f)
		else
			table.insert(keep, f)
		end
	end
	self.fuses = keep
	for _, f in due do
		self:explode(f.x, f.z, f.rad, f.dmg, false)
	end
end

function Level.explode(self: Level, x: number, z: number, radiusCells: number, dmg: number, big: boolean)
	local s = self.run.stats
	dmg *= 1 + s.BlastMult
	local rad = radiusCells * Const.BlastCell
	self:emit({ type = "explode", x = x, z = z, r = rad, big = big, cryo = s.CryoCharge })
	for _, b in self:near(x, z, rad) do
		if b.Alive and b.Layer <= 1 then
			if s.CryoCharge then
				b.Frozen = 4 + s.FreezeDur
			end
			if s.Volcano then
				b.Burn = 3.5
			end
			self:damage(b, { Src = "Explosions", Dmg = dmg, Blast = true, Silent = true })
		end
	end
end

function Level.lightning(self: Level, x: number, z: number, count: number, dmg: number, from: Brick?, recast: boolean)
	local plasma = self.run.stats.PlasmaStorm
	local cands = {}
	for _, b in self.bricks do
		if b.Alive and b ~= from and b.Layer == 0 and b.Kind ~= K.Steel and hypot(Const.cellX(b.C) - x, Const.cellZ(b.R) - z) < 9 then
			table.insert(cands, b)
		end
	end
	table.sort(cands, function(a: Brick, c: Brick): boolean
		return hypot(Const.cellX(a.C) - x, Const.cellZ(a.R) - z) < hypot(Const.cellX(c.C) - x, Const.cellZ(c.R) - z)
	end)
	if #cands == 0 then
		return
	end
	local px, pz = x, z
	for i = 1, math.min(count, #cands) do
		local t = cands[i]
		local tx, tz = Const.cellX(t.C), Const.cellZ(t.R)
		if plasma then
			t.Burn = 3.5
		end
		self:emit({ type = "arc", x0 = px, z0 = pz, x1 = tx, z1 = tz, plasma = plasma })
		self:damage(t, { Src = "Lightning", Dmg = dmg, Elemental = true, Silent = true })
		if recast and self.rng:NextNumber() < 0.5 then
			self:lightning(tx, tz, 1, dmg * 0.7, t, false)
		end
		px, pz = tx, tz
	end
end

function Level.blackHole(self: Level, x: number, z: number)
	local s = self.run.stats
	local rad = 3.4 * Const.EffCell
	self:emit({ type = "blackhole", x = x, z = z, r = rad })
	for _, b in self:near(x, z, rad) do
		if b.Layer <= 1 then
			self:damage(b, { Src = "Event Horizon", Dmg = 5 * s.Damage * s.DmgMult, Void = true, Blast = true, Silent = true })
		end
	end
end

function Level.mystery(self: Level, x: number, z: number)
	local roll = self.rng:NextInteger(0, 4)
	if roll == 0 then
		self:dropPower(x, z, true)
	elseif roll == 1 then
		for _ = 1, 6 do
			self:gainGold(1, x, z, true)
		end
	elseif roll == 2 then
		self:explode(x, z, 1.4, 2.5, false)
	elseif roll == 3 then
		self:lightning(x, z, 5, 2, nil, false)
	else
		self:raiseBarrier(1, 4)
	end
end

function Level.raiseBarrier(self: Level, n: number, cap: number)
	if self.barrier >= cap then
		return
	end
	self.barrier = math.min(cap, self.barrier + n)
	self:text(self.paddleX, PaddleZ + 1.6, "+BARRIER", Color3.fromRGB(190, 170, 255))
end

-- ------------------------------------------------------------------ loot

function Level.dropPower(self: Level, x: number, z: number, goodOnly: boolean)
	goodOnly = goodOnly or self.run.stats.Blessed
	local total = 0
	for i = 0, Const.NumPowers - 1 do
		local d = Const.PowerDefs[i]
		if not goodOnly or d.good then
			total += d.w
		end
	end
	local v = self.rng:NextNumber() * total
	for i = 0, Const.NumPowers - 1 do
		local d = Const.PowerDefs[i]
		if not goodOnly or d.good then
			if v < d.w then
				table.insert(self.drops, { X = x, Z = z, P = i, T = 0 })
				return
			end
			v -= d.w
		end
	end
end

function Level.gainGold(self: Level, n: number, x: number, z: number, asCoin: boolean)
	local r = self.run
	if asCoin then
		for _ = 1, n do
			local a = self.rng:NextNumber() * 2 * math.pi
			table.insert(self.coins, {
				X = x,
				Y = 0.8,
				Z = z,
				VX = math.cos(a) * 2,
				VZ = math.sin(a) * 2 - 1,
				VY = 4 + self.rng:NextNumber() * 3,
				Val = 1,
			})
		end
		return
	end
	local v = math.round(n * r.stats.GoldMult)
	if self.frenzy > 0 then
		v *= 2
	end
	v = math.max(v, 1)
	r.gold += v
	self.goldEarned += v
	self:emit({ type = "gold", amount = v })
end

function Level.applyPower(self: Level, p: number)
	local d = Const.PowerDefs[p]
	if p == P.Life then
		self.run.lives += 1
		self:text(self.paddleX, PaddleZ + 1, "+1 LIFE", d.col, true)
		return
	end
	self:text(self.paddleX, PaddleZ + 1, d.name, lighten(d.col, 0.3), true)
	self:emit({ type = "power", p = p, good = d.good })
	if d.dur > 0 then
		self.timers[p] = d.dur * (1 + self.run.stats.PowerDur)
	end
	if p == P.Expand then
		self.tempSize = math.min(2, math.max(0, self.tempSize) + 1)
	elseif p == P.Shrink then
		self.tempSize = -1
	elseif p == P.Multi then
		self:release()
		local add = {}
		for _, b in self.balls do
			if #self.balls + #add >= 9 then
				break
			end
			for _, a in { -0.4, 0.4 } do
				local ca, sa = math.cos(a), math.sin(a)
				local nb = self:newBall(b.X, b.Z, b.VX * ca - b.VZ * sa, b.VX * sa + b.VZ * ca, b)
				fixAngle(nb)
				table.insert(add, nb)
			end
		end
		for _, nb in add do
			table.insert(self.balls, nb)
		end
	elseif p == P.Slow then
		self.timers[P.Haste] = 0
	elseif p == P.Haste then
		self.timers[P.Slow] = 0
	elseif p == P.Barrier then
		self.barrier = math.min(4, self.barrier + 1)
	end
end

function Level.fireLaser(self: Level)
	local half = self.paddleW / 2 - 0.3
	table.insert(self.bolts, { X = self.paddleX - half, Z = PaddleZ + 0.5, Last = nil })
	table.insert(self.bolts, { X = self.paddleX + half, Z = PaddleZ + 0.5, Last = nil })
	self:emit({ type = "sound", name = "laser" })
end

function Level.updateDrops(self: Level, dt: number)
	local r = self.run
	local magnet = r.stats.Magnet or self.timers[P.Magnet] > 0
	local half = self.paddleW / 2
	local keep = {}
	for _, d in self.drops do
		d.T += dt
		d.Z -= 7 * dt
		if magnet and d.Z < 14 then
			d.X += (self.paddleX - d.X) * math.min(1, dt * 2.5)
		end
		local caught = d.Z - 0.4 <= PaddleZ + PaddleD / 2 and d.Z + 0.4 >= PaddleZ - PaddleD / 2 and math.abs(d.X - self.paddleX) <= half + 0.6
		if caught then
			self:applyPower(d.P)
		elseif d.Z >= Const.LoseZ then
			table.insert(keep, d)
		end
	end
	self.drops = keep
	local coins = {}
	for _, c in self.coins do
		c.VY -= 22 * dt
		c.X += c.VX * dt
		c.Y += c.VY * dt
		c.Z += c.VZ * dt
		if c.Y < 0.25 then
			c.Y = 0.25
			c.VY = math.abs(c.VY) * 0.4
			c.VX *= 0.7
		end
		c.VZ = math.max(c.VZ - 14 * dt, -9)
		if magnet or self.cleared then
			c.VX += (self.paddleX - c.X) * dt * 6
			c.VZ += (PaddleZ - c.Z) * dt * 3
		end
		c.X = clamp(c.X, -HalfW + 0.3, HalfW - 0.3)
		local caught = math.abs(c.Z - PaddleZ) < 0.7 and math.abs(c.X - self.paddleX) <= half + 0.4
		if caught then
			self:gainGold(c.Val, 0, 0, false)
			self:emit({ type = "coin", x = c.X })
			if r.stats.GoldRush then
				self.rushBonus = math.min(0.6, self.rushBonus + 0.02)
			end
		elseif c.Z >= Const.LoseZ then
			table.insert(coins, c)
		end
	end
	self.coins = coins
	local bolts = {}
	local s = r.stats
	for _, bl in self.bolts do
		bl.Z += 40 * dt
		local hit = bl.Z > FarZ
		local c = math.floor((bl.X + HalfW) / CellW)
		local rr = Const.rowAt(bl.Z)
		if not hit and rr >= 0 and rr < Const.GridRows and c >= 0 and c < Const.Cols then
			local b = self.grid[rr][c]
			if b and b.Alive and b ~= bl.Last then
				local dmg = math.max(1, s.Damage * s.DmgMult * (1 + s.LaserDmg))
				self:damage(b, { Src = "Laser", Dmg = dmg })
				if s.TeslaCoil then
					self:lightning(Const.cellX(b.C), Const.cellZ(b.R), 2, math.max(1, dmg * 0.5), b, false)
				end
				hit = true
			end
		end
		if not hit then
			table.insert(bolts, bl)
		end
	end
	self.bolts = bolts
end

-- ------------------------------------------------------------------ bricks

function Level.ballNear(self: Level, x: number, z: number): boolean
	for _, b in self.balls do
		if math.abs(b.X - x) < 2.2 and math.abs(b.Z - z) < 2.2 then
			return true
		end
	end
	return false
end

function Level.moveBrick(self: Level, b: Brick, r: number, c: number)
	self.grid[b.R][b.C] = nil
	b.R, b.C = r, c
	self.grid[r][c] = b
	local n = b.Up
	while n do
		n.R, n.C = r, c
		n = n.Up
	end
	self:emit({ type = "move", brick = b })
end

function Level.updateBricks(self: Level, dt: number)
	local s = self.run.stats
	for _, b in self.bricks do
		if b.Alive then
			b.Appear = math.min(1, b.Appear + dt * 4)
			if b.FallY > 0 then
				b.FallY = math.max(0, b.FallY - dt * 7)
			end
			b.Flash = math.max(0, b.Flash - dt * 6)
			b.SinceHit += dt
			if b.Frozen > 0 then
				b.Frozen -= dt
			end
			if b.Burn > 0 then
				b.Burn -= dt
				b.BurnTick += dt
				local tick = if s.DragonHeart then 0.35 else 0.7
				if s.SteamEngine and b.Frozen > 0 then
					b.Burn, b.Frozen = 0, 0
					local bx, bz = Const.cellX(b.C), Const.cellZ(b.R)
					self:text(bx, bz, "STEAM!", Color3.fromRGB(230, 240, 255))
					for _, n in self:near(bx, bz, 2.8) do
						self:damage(n, { Src = "Steam Engine", Dmg = 1.5 * s.Damage * s.DmgMult, Elemental = true, Silent = true })
					end
				end
				if b.Alive and b.BurnTick >= tick then
					b.BurnTick = 0
					self:damage(b, { Src = "Burning", Dmg = (0.5 + 0.25 * s.Damage) * (1 + s.BurnMult), Elemental = true, Silent = true })
				end
			end
			if b.Alive then
				local frozen = b.Frozen > 0 or b.Layer > 0 or b.Up ~= nil
				if b.Kind == K.Regen then
					if b.Frozen <= 0 and b.SinceHit > 3 and b.HP < b.MaxHP then
						b.HP = math.min(b.MaxHP, b.HP + dt)
					end
				elseif b.Kind == K.Moving and not frozen then
					b.Timer += dt
					if b.Timer > 1.6 then
						b.Timer = 0
						local dir = if self.rng:NextInteger(0, 1) == 0 then -1 else 1
						for _, d in { dir, -dir } do
							local nc = b.C + d
							if self:cellFree(b.R, nc) and not self:ballNear(Const.cellX(nc), Const.cellZ(b.R)) then
								self:moveBrick(b, b.R, nc)
								break
							end
						end
					end
				elseif b.Kind == K.Teleport and not frozen then
					b.Timer += dt
					if b.Timer > 5 then
						b.Timer = 0
						for _ = 1, 20 do
							local rr, cc = self.rng:NextInteger(0, Const.MaxRows + 1), self.rng:NextInteger(0, Const.Cols - 1)
							if self:cellFree(rr, cc) and not self:ballNear(Const.cellX(cc), Const.cellZ(rr)) then
								self:moveBrick(b, rr, cc)
								b.Appear = 0
								break
							end
						end
					end
				elseif b.Kind == K.Spawner and not frozen then
					b.Timer += dt
					if b.Timer > 7 then
						b.Timer = 0
						for _, d in { { 1, 0 }, { 0, 1 }, { 0, -1 }, { -1, 0 } } do
							local rr, cc = b.R + d[1], b.C + d[2]
							if self:cellFree(rr, cc) and not self:ballNear(Const.cellX(cc), Const.cellZ(rr)) then
								self:addBrick(rr, cc, K.Basic, 1, lighten(b.Col, 0.3))
								break
							end
						end
					end
				end
			end
		end
	end
	-- drop dead bricks from the list now and then, so it stays short
	if #self.bricks > 64 and self.rng:NextNumber() < dt * 2 then
		local live = {}
		for _, b in self.bricks do
			if b.Alive then
				table.insert(live, b)
			end
		end
		self.bricks = live
	end
end

-- ------------------------------------------------------------------ end of chamber

function Level.loseBall(self: Level)
	local r = self.run
	self.lostOrbs += 1
	local lose = math.min(r.gold, math.floor(r.stats.BallLossGold))
	if lose > 0 then
		r.gold -= lose
		self:text(self.paddleX, PaddleZ + 1.4, string.format("−%d GOLD", lose), Color3.fromRGB(255, 190, 90))
	end
	r.lives -= 1
	self:emit({ type = "lifelost", lives = r.lives })
	if r.lives <= 0 then
		self.lost = true
		self:emit({ type = "gameover" })
		return
	end
	if r.stats.DropBarrier > 0 then
		self.barrier = math.min(5, self.barrier + r.stats.DropBarrier)
	end
	for i = 0, Const.NumPowers - 1 do
		self.timers[i] = 0
	end
	self.tempSize = 0
	self.drops = {}
	self.bolts = {}
	self.combo = 0
	self:spawnServe()
end

-- clear pays the chamber's reward (play.go clear)
function Level.clear(self: Level)
	local r = self.run
	local s = r.stats
	self.cleared = true
	local reward = 12 + r.act * 4
	if self.elite then
		reward += 25
	end
	if self.guardian then
		reward += 60
	end
	self:gainGold(reward, 0, 0, false)
	if s.GoldOnClear > 0 then
		self:gainGold(math.floor(s.GoldOnClear), 0, 0, false)
	end
	if s.CleanClearGold > 0 and self.lostOrbs == 0 and not self.guardian then
		self:gainGold(math.floor(s.CleanClearGold), 0, 0, false)
		self:text(0, 8, string.format("FLAWLESS  +%d GOLD", math.floor(s.CleanClearGold)), Color3.fromRGB(255, 220, 120), true)
	end
	if s.Interest > 0 then
		local cap = if self:hasFusion("hoard") then 60 else 30
		local gain = math.floor(math.min(cap, r.gold * s.Interest))
		r.gold += gain
		self.goldEarned += gain
	end
	-- collect the coins still in the air
	for _, c in self.coins do
		self:gainGold(c.Val, 0, 0, false)
	end
	self.coins = {}
	for _, b in self.bricks do
		if b.Alive and b.Kind ~= K.Steel then
			b.Alive = false
			self:emit({ type = "break", brick = b, x = Const.cellX(b.C), z = Const.cellZ(b.R), combo = 0 })
		end
	end
	self.balls = {}
	self.drops = {}
	self:emit({ type = "cleared" })
end

return Level
