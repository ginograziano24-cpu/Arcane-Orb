--!strict
-- Const.luau: the arena's measurements and the brick kinds, copied from the
-- Go game (play.go, levelgen). World units are the game's own: x runs across
-- the arena (-HalfW .. HalfW), z runs from the paddle (near) to the far wall.

local Const = {}

Const.HalfW = 14.0
Const.FarZ = 31.5
Const.GridTopZ = 30.3
Const.CellW = 2.0
Const.CellD = 1.1
Const.Cols = 14
Const.MaxRows = 16 -- rows the generator draws
Const.GridRows = 48 -- rows the grid can hold (bricks can be pushed down)
Const.PaddleZ = 2.2
Const.PaddleD = 0.7
Const.PaddleH = 0.7
Const.BarrierZ = 0.4
Const.LoseZ = -2.5
Const.BaseW = 3.6
Const.LayerH = 0.78
Const.EffCell = 2.0

-- loot and kill counters scale by this (small bricks vs the old big ones)
Const.KillWeight = 0.45
Const.OrbSplash = 0.2 -- how far past the orb's edge its impact cracks bricks
Const.OrbPlow = 0.35 -- chance a breaking small brick doesn't stop the orb

-- blast radii are in blast cells
Const.BlastCell = 1.35
Const.BaseExplodeR = 1.4
Const.BrickBlastR = 0.85
Const.BrickBlastDmg = 2.0

-- brick kinds, in levelgen order (the level bank stores them as a..p)
Const.Kind = {
	Empty = 0,
	Basic = 1,
	Steel = 2,
	Armored = 3,
	Explosive = 4,
	Chain = 5,
	Split = 6,
	Regen = 7,
	Moving = 8,
	Shield = 9,
	Teleport = 10,
	Spawner = 11,
	Multiplier = 12,
	Crystal = 13,
	Mystery = 14,
	Elite = 15,
}
local K = Const.Kind

-- required: must be destroyed to clear a chamber
function Const.required(kind: number): boolean
	return kind ~= K.Empty and kind ~= K.Steel
end

-- one-hit kinds always have 1 HP
Const.OneHit = {
	[K.Explosive] = true,
	[K.Mystery] = true,
	[K.Multiplier] = true,
	[K.Crystal] = true,
	[K.Chain] = true,
	[K.Split] = true,
}

function Const.cellX(c: number): number
	-- columns count from 0, like the Go grid
	return -Const.HalfW + (c + 0.5) * Const.CellW
end

function Const.cellZ(r: number, shift: number?): number
	return Const.GridTopZ - (shift or 0) - (r + 0.5) * Const.CellD
end

function Const.rowAt(z: number, shift: number?): number
	return math.floor((Const.GridTopZ - (shift or 0) - z) / Const.CellD)
end

-- power-ups (play.go powerDefs): letter, name, colour, good, weight, duration
Const.Power = {
	Multi = 0,
	Expand = 1,
	Shrink = 2,
	Slow = 3,
	Haste = 4,
	Fire = 5,
	Laser = 6,
	Barrier = 7,
	Glue = 8,
	Magnet = 9,
	Life = 10,
}
Const.NumPowers = 11

export type PowerDef = { letter: string, name: string, col: Color3, good: boolean, w: number, dur: number }

Const.PowerDefs = {
	[0] = { letter = "M", name = "Multi-Orb", col = Color3.fromRGB(255, 210, 60), good = true, w = 10, dur = 0 },
	[1] = { letter = "E", name = "Expand", col = Color3.fromRGB(70, 150, 255), good = true, w = 11, dur = 0 },
	[2] = { letter = "S", name = "Shrink", col = Color3.fromRGB(230, 60, 60), good = false, w = 5, dur = 0 },
	[3] = { letter = "W", name = "Slow", col = Color3.fromRGB(80, 220, 200), good = true, w = 8, dur = 10 },
	[4] = { letter = "H", name = "Haste", col = Color3.fromRGB(200, 40, 120), good = false, w = 5, dur = 10 },
	[5] = { letter = "F", name = "Fire Orb", col = Color3.fromRGB(255, 120, 30), good = true, w = 5, dur = 7 },
	[6] = { letter = "L", name = "Laser", col = Color3.fromRGB(255, 70, 70), good = true, w = 7, dur = 8 },
	[7] = { letter = "B", name = "Barrier", col = Color3.fromRGB(150, 120, 255), good = true, w = 7, dur = 0 },
	[8] = { letter = "G", name = "Glue", col = Color3.fromRGB(120, 220, 70), good = true, w = 7, dur = 11 },
	[9] = { letter = "$", name = "Magnet", col = Color3.fromRGB(255, 200, 80), good = true, w = 5, dur = 12 },
	[10] = { letter = "+", name = "Extra Life", col = Color3.fromRGB(255, 110, 160), good = true, w = 0, dur = 0 },
} :: { [number]: PowerDef }

-- brick colours by kind (biome.go kindColor); nil = use the palette colour
Const.KindColor = {
	[K.Steel] = Color3.fromRGB(128, 134, 146),
	[K.Armored] = Color3.fromRGB(86, 92, 108),
	[K.Explosive] = Color3.fromRGB(170, 40, 40),
	[K.Crystal] = Color3.fromRGB(60, 170, 230),
	[K.Elite] = Color3.fromRGB(60, 30, 80),
	[K.Mystery] = Color3.fromRGB(240, 200, 80),
	[K.Multiplier] = Color3.fromRGB(250, 120, 200),
	[K.Shield] = Color3.fromRGB(90, 200, 230),
	[K.Spawner] = Color3.fromRGB(110, 170, 70),
} :: { [number]: Color3 }

return Const
