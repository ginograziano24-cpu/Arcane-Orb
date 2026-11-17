@echo off
cd /d "%~dp0"
"C:\Program Files\Blender Foundation\Blender 5.2\blender.exe" -b -P "%~dp0heroes\arcane_eye3d.py" > "%~dp0heroes\arcane_eye3d_log.txt" 2>&1
