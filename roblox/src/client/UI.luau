--!strict
-- UI.luau: the prototype's screens, built in code: origin select, HUD,
-- reward cards, banners and the end-of-run card. Sized with scale and
-- UIAspectRatio so it reads on phones as well as monitors.

local TweenService = game:GetService("TweenService")

local Run = require(game:GetService("ReplicatedStorage"):WaitForChild("ArcaneOrb").Sim.Run)

local UI = {}
UI.__index = UI

local GOLD = Color3.fromRGB(255, 214, 120)
local INK = Color3.fromRGB(18, 14, 34)
local PANEL = Color3.fromRGB(36, 26, 62)
local RARITY_COL = {
	[0] = Color3.fromRGB(200, 200, 210),
	[1] = Color3.fromRGB(110, 220, 140),
	[2] = Color3.fromRGB(90, 160, 255),
	[3] = Color3.fromRGB(200, 110, 255),
	[4] = Color3.fromRGB(255, 180, 60),
	[5] = Color3.fromRGB(255, 90, 120),
}

local function new(class: string, props: { [string]: any }, children: { Instance }?): any
	local o = Instance.new(class)
	for k, v in props do
		if k ~= "Parent" then
			(o :: any)[k] = v
		end
	end
	for _, c in children or {} do
		c.Parent = o
	end
	if props.Parent then
		o.Parent = props.Parent
	end
	return o
end

local function corner(r: number): UICorner
	return new("UICorner", { CornerRadius = UDim.new(0, r) })
end

local function stroke(col: Color3, t: number): UIStroke
	return new("UIStroke", { Color = col, Thickness = t, ApplyStrokeMode = Enum.ApplyStrokeMode.Border })
end

local function label(text: string, size: UDim2, pos: UDim2, col: Color3, font: Enum.Font?): TextLabel
	return new("TextLabel", {
		BackgroundTransparency = 1,
		Size = size,
		Position = pos,
		Text = text,
		TextColor3 = col,
		Font = font or Enum.Font.GothamBold,
		TextScaled = true,
		TextWrapped = true,
	})
end

export type UI = typeof(setmetatable(
	{} :: {
		gui: ScreenGui,
		hud: Frame,
		hudText: { [string]: TextLabel },
		layer: Frame,
		toasts: Frame,
		flashFrame: Frame,
	},
	UI
))

function UI.new(player: Player): UI
	local gui = new("ScreenGui", {
		Name = "ArcaneOrbUI",
		ResetOnSpawn = false,
		IgnoreGuiInset = true,
		ZIndexBehavior = Enum.ZIndexBehavior.Sibling,
		Parent = player:WaitForChild("PlayerGui"),
	})
	local hud = new("Frame", { BackgroundTransparency = 1, Size = UDim2.fromScale(1, 0.09), Visible = false, Parent = gui })
	local hudText = {}
	for i, key in { "gold", "lives", "chamber", "combo", "barrier" } do
		hudText[key] = label("", UDim2.fromScale(0.19, 0.7), UDim2.fromScale(0.01 + (i - 1) * 0.198, 0.15), GOLD)
		hudText[key].TextXAlignment = Enum.TextXAlignment.Left
		hudText[key].TextStrokeTransparency = 0.4
		hudText[key].Parent = hud
	end
	local layer = new("Frame", { BackgroundTransparency = 1, Size = UDim2.fromScale(1, 1), Parent = gui })
	local toasts = new("Frame", {
		BackgroundTransparency = 1,
		Size = UDim2.fromScale(0.5, 0.3),
		Position = UDim2.fromScale(0.25, 0.12),
		Parent = gui,
	}, { new("UIListLayout", { HorizontalAlignment = Enum.HorizontalAlignment.Center, Padding = UDim.new(0, 6) }) })
	local flashFrame = new("Frame", {
		BackgroundColor3 = Color3.new(1, 0, 0),
		BackgroundTransparency = 1,
		Size = UDim2.fromScale(1, 1),
		ZIndex = 50,
		Parent = gui,
	})
	return setmetatable({ gui = gui, hud = hud, hudText = hudText, layer = layer, toasts = toasts, flashFrame = flashFrame }, UI)
end

function UI.clear(self: UI)
	self.layer:ClearAllChildren()
end

function UI.showHud(self: UI, on: boolean)
	self.hud.Visible = on
end

function UI.updateHud(self: UI, run: any, level: any)
	local t = self.hudText
	t.gold.Text = string.format("◆ %d gold", run.gold)
	t.lives.Text = string.format("♥ %d", run.lives)
	t.chamber.Text = string.format("Act %d · %d/%d", run.act, run.actDepth, Run.ChambersPerAct)
	t.combo.Text = if level.combo >= 2 then string.format("Combo ×%d", level.combo) else ""
	t.barrier.Text = if level.barrier > 0 then string.format("Barrier %d", level.barrier) else ""
end

function UI.button(self: UI, parent: Instance, text: string, size: UDim2, pos: UDim2, col: Color3, onClick: () -> ()): TextButton
	local b = new("TextButton", {
		Size = size,
		Position = pos,
		BackgroundColor3 = PANEL,
		Text = text,
		TextColor3 = col,
		Font = Enum.Font.GothamBlack,
		TextScaled = true,
		AutoButtonColor = true,
		Parent = parent,
	}, { corner(10), stroke(col, 2), new("UIPadding", { PaddingLeft = UDim.new(0.05, 0), PaddingRight = UDim.new(0.05, 0) }) })
	b.Activated:Connect(onClick)
	return b
end

-- showMenu: pick an origin (and Quick Run or the full three acts)
function UI.showMenu(self: UI, origins: { any }, onPick: (string, boolean) -> ())
	self:clear()
	self:showHud(false)
	local root = new("Frame", { BackgroundColor3 = INK, BackgroundTransparency = 0.15, Size = UDim2.fromScale(1, 1), Parent = self.layer })
	label("ARCANE ORB", UDim2.fromScale(0.6, 0.12), UDim2.fromScale(0.2, 0.04), GOLD, Enum.Font.Fantasy).Parent = root
	label("Choose your origin", UDim2.fromScale(0.5, 0.05), UDim2.fromScale(0.25, 0.16), Color3.new(1, 1, 1)).Parent = root
	local quick = true
	local grid = new("ScrollingFrame", {
		BackgroundTransparency = 1,
		Size = UDim2.fromScale(0.9, 0.6),
		Position = UDim2.fromScale(0.05, 0.23),
		CanvasSize = UDim2.new(),
		AutomaticCanvasSize = Enum.AutomaticSize.Y,
		ScrollBarThickness = 6,
		Parent = root,
	}, {
		new("UIGridLayout", {
			CellSize = UDim2.fromScale(0.31, 0.45),
			CellPadding = UDim2.fromScale(0.02, 0.04),
			HorizontalAlignment = Enum.HorizontalAlignment.Center,
		}),
	})
	for _, o in origins do
		local card = new("TextButton", { BackgroundColor3 = PANEL, Text = "", AutoButtonColor = true, Parent = grid }, { corner(12), stroke(GOLD, 2) })
		label(o.Name, UDim2.fromScale(0.9, 0.18), UDim2.fromScale(0.05, 0.04), GOLD, Enum.Font.GothamBlack).Parent = card
		label("Passive: " .. (o.Passive or ""), UDim2.fromScale(0.9, 0.1), UDim2.fromScale(0.05, 0.23), Color3.fromRGB(180, 220, 255)).Parent = card
		local d = label(o.Desc or "", UDim2.fromScale(0.9, 0.6), UDim2.fromScale(0.05, 0.36), Color3.new(1, 1, 1), Enum.Font.Gotham)
		d.TextYAlignment = Enum.TextYAlignment.Top
		d.Parent = card
		card.Activated:Connect(function()
			self:clear()
			onPick(o.ID, quick)
		end)
	end
	local toggle: TextButton
	toggle = self:button(root, "Quick Run (1 act)", UDim2.fromScale(0.3, 0.07), UDim2.fromScale(0.35, 0.87), GOLD, function()
		quick = not quick
		toggle.Text = if quick then "Quick Run (1 act)" else "Full Run (3 acts)"
	end)
end

-- showReward: one item from a few cards (or skip)
function UI.showReward(self: UI, offers: { any }, run: any, onDone: (string?) -> ())
	self:clear()
	local root = new("Frame", { BackgroundColor3 = INK, BackgroundTransparency = 0.25, Size = UDim2.fromScale(1, 1), Parent = self.layer })
	label("Choose an upgrade", UDim2.fromScale(0.6, 0.08), UDim2.fromScale(0.2, 0.08), GOLD, Enum.Font.GothamBlack).Parent = root
	local n = #offers
	for i, m in offers do
		local rarity = m.Rarity or 0
		local col = RARITY_COL[rarity] or GOLD
		local w = 0.8 / math.max(n, 1)
		local card = new("TextButton", {
			BackgroundColor3 = PANEL,
			Text = "",
			Size = UDim2.fromScale(w - 0.02, 0.55),
			Position = UDim2.fromScale(0.1 + (i - 1) * w + 0.01, 0.2),
			AutoButtonColor = true,
			Parent = root,
		}, { corner(14), stroke(col, 3) })
		label(m.Name, UDim2.fromScale(0.9, 0.16), UDim2.fromScale(0.05, 0.05), col, Enum.Font.GothamBlack).Parent = card
		label(Run.RarityNames[rarity] or "", UDim2.fromScale(0.9, 0.07), UDim2.fromScale(0.05, 0.22), col).Parent = card
		local d = label(m.Desc or "", UDim2.fromScale(0.9, 0.5), UDim2.fromScale(0.05, 0.33), Color3.new(1, 1, 1), Enum.Font.Gotham)
		d.TextYAlignment = Enum.TextYAlignment.Top
		d.Parent = card
		local f = run:fusionWith(m.ID)
		if f then
			label("FUSES WITH " .. string.upper(f.A == m.ID and f.B or f.A), UDim2.fromScale(0.9, 0.08), UDim2.fromScale(0.05, 0.86), GOLD).Parent = card
		end
		card.Activated:Connect(function()
			self:clear()
			onDone(m.ID)
		end)
	end
	self:button(root, "Skip", UDim2.fromScale(0.2, 0.07), UDim2.fromScale(0.4, 0.82), Color3.fromRGB(200, 200, 210), function()
		self:clear()
		onDone(nil)
	end)
end

function UI.showOver(self: UI, won: boolean, run: any, onAgain: () -> ())
	self:clear()
	self:showHud(false)
	local root = new("Frame", { BackgroundColor3 = INK, BackgroundTransparency = 0.15, Size = UDim2.fromScale(1, 1), Parent = self.layer })
	label(if won then "VICTORY" else "THE ORB FADES", UDim2.fromScale(0.6, 0.12), UDim2.fromScale(0.2, 0.15), GOLD, Enum.Font.Fantasy).Parent = root
	label(
		string.format("Act %d · chamber %d  ·  %d bricks  ·  best combo ×%d  ·  score %d", run.act, run.actDepth, run.bricks, run.maxCombo, run.score),
		UDim2.fromScale(0.7, 0.06),
		UDim2.fromScale(0.15, 0.32),
		Color3.new(1, 1, 1)
	).Parent = root
	self:button(root, "Play again", UDim2.fromScale(0.3, 0.08), UDim2.fromScale(0.35, 0.5), GOLD, function()
		self:clear()
		onAgain()
	end)
end

function UI.banner(self: UI, title: string, sub: string)
	local f = new("Frame", {
		BackgroundColor3 = INK,
		BackgroundTransparency = 0.3,
		Size = UDim2.fromScale(0.6, 0.14),
		Position = UDim2.fromScale(0.2, 0.35),
		Parent = self.gui,
	}, { corner(12) })
	local t = label(title, UDim2.fromScale(0.9, 0.55), UDim2.fromScale(0.05, 0.05), GOLD, Enum.Font.GothamBlack)
	t.Parent = f
	local s = label(sub, UDim2.fromScale(0.9, 0.3), UDim2.fromScale(0.05, 0.62), Color3.new(1, 1, 1))
	s.Parent = f
	task.delay(1.6, function()
		local info = TweenInfo.new(0.5)
		TweenService:Create(f, info, { BackgroundTransparency = 1 }):Play()
		TweenService:Create(t, info, { TextTransparency = 1 }):Play()
		TweenService:Create(s, info, { TextTransparency = 1 }):Play()
		task.delay(0.55, function()
			f:Destroy()
		end)
	end)
end

function UI.toast(self: UI, text: string)
	local l = label(text, UDim2.fromScale(1, 0.18), UDim2.new(), GOLD, Enum.Font.GothamBlack)
	l.TextStrokeTransparency = 0.3
	l.Parent = self.toasts
	task.delay(2.5, function()
		l:Destroy()
	end)
end

function UI.flash(self: UI, col: Color3)
	self.flashFrame.BackgroundColor3 = col
	self.flashFrame.BackgroundTransparency = 0.6
	TweenService:Create(self.flashFrame, TweenInfo.new(0.4), { BackgroundTransparency = 1 }):Play()
end

return UI
