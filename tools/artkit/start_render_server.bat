@echo off
setlocal
cd /d "%~dp0"
set "BL="
for /d %%D in ("%ProgramFiles%\Blender Foundation\Blender*") do if exist "%%D\blender.exe" set "BL=%%D\blender.exe"
if not defined BL for /f "usebackq delims=" %%P in (`powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\find_blender.ps1"`) do set "BL=%%P"
if not defined BL (
  echo Could not find Blender.
  pause
  exit /b 1
)
title Arcane Orb render server
set ARTKIT_SERVER=1
"%BL%" -b --factory-startup -P "%~dp0scripts\watch.py"
pause
