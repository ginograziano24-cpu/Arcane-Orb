@echo off
rem Arcanist: put the new lashes onto her texture, rebuild her figure (Base Form),
rem then rebuild her game model and HUD portraits.
cd /d "%~dp0"
set B="C:\Program Files\Blender Foundation\Blender 5.2\blender.exe"
%B% -b -P "%~dp0heroes\arcane_eyes_bake.py" > "%~dp0heroes\arcane_eyes_bake_log.txt" 2>&1
%B% -b -P "%~dp0heroes\arcane_baseform3.py" > "%~dp0heroes\arcane_baseform3_log.txt" 2>&1
call "%~dp0Export-HQ-Heroes.cmd" arcane
%B% -b -P "%~dp0heroes\render_heroes.py" -- busts arcane > "%~dp0heroes\render-log.txt" 2>&1
for %%p in (rest look fierce hurt) do if exist "%~dp0heroes\renders\arcane_%%p.png" copy /y "%~dp0heroes\renders\arcane_%%p.png" "%~dp0Source\assets\portraits\hud\arcane_%%p.png" >nul
echo ok > "%~dp0heroes\arcanist-eyes-body-done.txt"
