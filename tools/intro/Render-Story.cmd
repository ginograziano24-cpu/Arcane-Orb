@echo off
rem Renders the full story opening (about 2,030 frames; this takes a while).
rem The old film's frames and audio are kept as *-before-story.
cd /d "%~dp0"
copy /y "%~dp0Source\assets\bg\dream_meadow_0.jpg" "%~dp0intro\sky.jpg" >nul
if exist "%~dp0intro\done.txt" del "%~dp0intro\done.txt"
if exist "%~dp0intro\frames" if not exist "%~dp0intro\frames-before-story" move "%~dp0intro\frames" "%~dp0intro\frames-before-story" >nul
if exist "%~dp0intro\voice.wav" if not exist "%~dp0intro\voice-before-story.wav" copy /y "%~dp0intro\voice.wav" "%~dp0intro\voice-before-story.wav" >nul
if exist "%~dp0intro\lines.txt" if not exist "%~dp0intro\lines-before-story.txt" copy /y "%~dp0intro\lines.txt" "%~dp0intro\lines-before-story.txt" >nul
"C:\Program Files\Blender Foundation\Blender 5.2\blender.exe" -b -P "%~dp0intro\build_story.py" > "%~dp0intro\story-blender-log.txt" 2>&1
copy /y "%~dp0intro\voice_story.wav" "%~dp0intro\voice.wav" >nul
copy /y "%~dp0intro\lines_story.txt" "%~dp0intro\lines.txt" >nul
