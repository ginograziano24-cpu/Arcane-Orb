@echo off
rem Pre-generates the Roblox port's brick layouts with the game's own generator.
rem Output: roblox-levels\Act1-3.luau and Arenas.luau beside this file.
cd /d "%~dp0Source"
if exist "%~dp0levelbank_done.txt" del "%~dp0levelbank_done.txt"
"%~dp0..\go-sdk\go\bin\go.exe" run ./tools/levelbank -out "%~dp0roblox-levels" > "%~dp0levelbank-log.txt" 2>&1
echo ok> "%~dp0levelbank_done.txt"
