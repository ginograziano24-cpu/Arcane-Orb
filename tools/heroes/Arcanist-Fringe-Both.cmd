@echo off
rem Arcanist: new strand fringe, two versions (swept aside / over her right eye) - test renders only.
cd /d "%~dp0"
set B="C:\Program Files\Blender Foundation\Blender 5.2\blender.exe"
if exist "%~dp0heroes\arcane_layers2\fringe_done.txt" del "%~dp0heroes\arcane_layers2\fringe_done.txt"
for %%m in (swept over) do (
  set FRINGE=%%m
  set SHOW_TAG=_%%m
  call :one %%m
)
echo ok> "%~dp0heroes\arcane_layers2\fringe_done.txt"
goto :eof
:one
set FRINGE=%1
set SHOW_TAG=_%1
%B% -b -P "%~dp0heroes\arcane_eyes_bake.py" > "%~dp0heroes\arcane_eyes_bake_log_%1.txt" 2>&1
copy /y "%~dp0heroes\arcane_layers2\eyes_bake_log.txt" "%~dp0heroes\arcane_layers2\eyes_bake_log_%1.txt" >nul
%B% -b -P "%~dp0heroes\arcane_showcase.py" > "%~dp0heroes\arcane_showcase_log_%1.txt" 2>&1
goto :eof
