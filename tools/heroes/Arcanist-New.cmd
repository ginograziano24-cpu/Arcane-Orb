@echo off
rem New Arcanist from "fantasy witch 3d model.glb": compare, layered .blend
rem (robe/armor toggles + base form), then rebuild her game model and bust.
rem Waits for the intro render to finish first.
cd /d "%~dp0"
:wait
if exist "%~dp0intro\frames" if not exist "%~dp0intro\done.txt" (timeout /t 30 /nobreak >nul & goto wait)
if not exist "%~dp0heroes\glb\fantasy witch 3d model.glb" copy /y "%USERPROFILE%\Downloads\fantasy witch 3d model.glb" "%~dp0heroes\glb\" >nul
"C:\Program Files\Blender Foundation\Blender 5.2\blender.exe" -b -P "%~dp0heroes\arcane_layers.py" > "%~dp0heroes\arcane_layers_log.txt" 2>&1
call "%~dp0Export-HQ-Heroes.cmd" arcane
"C:\Program Files\Blender Foundation\Blender 5.2\blender.exe" -b -P "%~dp0heroes\render_heroes.py" -- busts arcane > "%~dp0heroes\render-log.txt" 2>&1
for %%p in (rest look fierce hurt) do if exist "%~dp0heroes\renders\arcane_%%p.png" copy /y "%~dp0heroes\renders\arcane_%%p.png" "%~dp0Source\assets\portraits\hud\arcane_%%p.png" >nul
echo ok > "%~dp0heroes\arcanist-new-done.txt"
