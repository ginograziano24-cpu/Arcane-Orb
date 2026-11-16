@echo off
cd /d "%~dp0"
"C:\Program Files\Blender Foundation\Blender 5.2\blender.exe" -b -P "%~dp0intro\fumb_check.py" > "%~dp0heroes\intro_chars\fumb_check_log.txt" 2>&1
