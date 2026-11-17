@echo off
rem Arcanist: re-project the new lashes (face only, not the cape behind her head),
rem rebuild her game model with the fix at full face resolution, re-render portraits.
rem Also puts back the second-pass Base Form (the third pass was worse).
cd /d "%~dp0"
set B="C:\Program Files\Blender Foundation\Blender 5.2\blender.exe"
set L=%~dp0heroes\arcane_layers2
if exist "%L%\arcanist_layers.blend1" copy /y "%L%\arcanist_layers.blend" "%L%\arcanist_layers_pass3.blend" >nul
if exist "%L%\arcanist_layers.blend1" copy /y "%L%\arcanist_layers.blend1" "%L%\arcanist_layers.blend" >nul
%B% -b -P "%~dp0heroes\arcane_eyes_bake.py" > "%~dp0heroes\arcane_eyes_bake_log.txt" 2>&1
call "%~dp0Export-HQ-Heroes.cmd" arcane
%B% -b -P "%~dp0heroes\render_heroes.py" -- busts arcane > "%~dp0heroes\render-log.txt" 2>&1
for %%p in (rest look fierce hurt) do if exist "%~dp0heroes\renders\arcane_%%p.png" copy /y "%~dp0heroes\renders\arcane_%%p.png" "%~dp0Source\assets\portraits\hud\arcane_%%p.png" >nul
echo ok > "%~dp0heroes\arcanist-eyes2-done.txt"
