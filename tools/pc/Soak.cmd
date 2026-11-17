@echo off
rem Memory soak: 30 whole runs back to back on the Go-Check build. Report: Source\bench_soak_nov17.txt
cd /d "%~dp0Source"
if exist "%~dp0soak_done.txt" del "%~dp0soak_done.txt"
if not exist ArcaneOrb-test.exe (echo no build: run Go-Check first> "%~dp0soak_done.txt" & goto :eof)
ArcaneOrb-test.exe -soak -soakruns=30 -benchtag=nov17 -benchwin=1280x720
echo ok> "%~dp0soak_done.txt"
