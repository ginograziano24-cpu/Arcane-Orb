@echo off
rem Arcanist: rebuild the face under the hair lock, then render both versions (lock kept / lock trimmed).
cd /d "%~dp0"
set B="C:\Program Files\Blender Foundation\Blender 5.2\blender.exe"
if exist "%~dp0heroes\arcane_layers2\both_done.txt" del "%~dp0heroes\arcane_layers2\both_done.txt"
set LOCK_MODE=keep
set SHOW_TAG=_keep
%B% -b -P "%~dp0heroes\arcane_eyes_bake.py" > "%~dp0heroes\arcane_eyes_bake_log.txt" 2>&1
copy /y "%~dp0heroes\arcane_layers2\eyes_bake_log.txt" "%~dp0heroes\arcane_layers2\eyes_bake_log_keep.txt" >nul
%B% -b -P "%~dp0heroes\arcane_showcase.py" > "%~dp0heroes\arcane_showcase_log_keep.txt" 2>&1
set LOCK_MODE=trim
set SHOW_TAG=_trim
%B% -b -P "%~dp0heroes\arcane_eyes_bake.py" > "%~dp0heroes\arcane_eyes_bake_log.txt" 2>&1
copy /y "%~dp0heroes\arcane_layers2\eyes_bake_log.txt" "%~dp0heroes\arcane_layers2\eyes_bake_log_trim.txt" >nul
%B% -b -P "%~dp0heroes\arcane_showcase.py" > "%~dp0heroes\arcane_showcase_log_trim.txt" 2>&1
echo ok> "%~dp0heroes\arcane_layers2\both_done.txt"
