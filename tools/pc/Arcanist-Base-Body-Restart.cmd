@echo off
rem Arcanist: closes a stuck background body build (only the Blender running arcane_basebody.py), then runs it again.
cd /d "%~dp0"
powershell -NoProfile -Command "Get-CimInstance Win32_Process -Filter \"name='blender.exe'\" | Where-Object { $_.CommandLine -like '*arcane_basebody.py*' } | ForEach-Object { 'closing ' + $_.ProcessId; Stop-Process -Id $_.ProcessId -Force }" > "%~dp0heroes\restart_log.txt" 2>&1
timeout /t 3 /nobreak > nul
set B="C:\Program Files\Blender Foundation\Blender 5.2\blender.exe"
if exist "%~dp0heroes\arcane_layers2\basebody_done.txt" del "%~dp0heroes\arcane_layers2\basebody_done.txt"
%B% -b -P "%~dp0heroes\arcane_basebody.py" > "%~dp0heroes\arcane_basebody_out2.txt" 2>&1
