@echo off
rem Arcanist: mirrored eye + lip gloss onto her texture, rebuild game model, portraits, showcase renders.
cd /d "%~dp0"
set B="C:\Program Files\Blender Foundation\Blender 5.2\blender.exe"
if exist "%~dp0heroes\arcane_layers2\show_done.txt" del "%~dp0heroes\arcane_layers2\show_done.txt"
%B% -b -P "%~dp0heroes\arcane_eyes_bake.py" > "%~dp0heroes\arcane_eyes_bake_log.txt" 2>&1
call "%~dp0Export-HQ-Heroes.cmd" arcane
%B% -b -P "%~dp0heroes\render_heroes.py" -- busts arcane > "%~dp0heroes\render-log.txt" 2>&1
for %%p in (rest look fierce hurt) do if exist "%~dp0heroes\renders\arcane_%%p.png" copy /y "%~dp0heroes\renders\arcane_%%p.png" "%~dp0Source\assets\portraits\hud\arcane_%%p.png" >nul
%B% -b -P "%~dp0heroes\arcane_showcase.py" > "%~dp0heroes\arcane_showcase_log.txt" 2>&1
