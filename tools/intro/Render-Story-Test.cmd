@echo off
rem Test stills of the new story opening into intro\test (a few minutes).
cd /d "%~dp0"
copy /y "%~dp0Source\assets\bg\dream_meadow_0.jpg" "%~dp0intro\sky.jpg" >nul
if exist "%~dp0intro\done.txt" del "%~dp0intro\done.txt"
"C:\Program Files\Blender Foundation\Blender 5.2\blender.exe" -b -P "%~dp0intro\build_story.py" -- test > "%~dp0intro\story-blender-log.txt" 2>&1
