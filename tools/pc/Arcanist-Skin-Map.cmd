@echo off
rem Arcanist: shows what counts as skin and what counts as clothing on her original model (renders skinmap_*.png).
cd /d "%~dp0"
set B="C:\Program Files\Blender Foundation\Blender 5.2\blender.exe"
if exist "%~dp0heroes\arcane_layers2\skinmap_done.txt" del "%~dp0heroes\arcane_layers2\skinmap_done.txt"
%B% -b -P "%~dp0heroes\arcane_skinmap.py" > "%~dp0heroes\arcane_skinmap_out.txt" 2>&1
