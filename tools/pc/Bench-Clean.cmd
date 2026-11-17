@echo off
rem Clean benchmark (no screenshots) on the Go-Check build, fullscreen like the Nov 13 run.
rem Writes Source\bench_report_nov17b.txt plus CPU and allocation profiles.
cd /d "%~dp0Source"
if exist "%~dp0bench_done.txt" del "%~dp0bench_done.txt"
if not exist ArcaneOrb-test.exe (echo no build: run Go-Check first> "%~dp0bench_done.txt" & goto :eof)
ArcaneOrb-test.exe -bench -benchtag=nov17b
echo ok> "%~dp0bench_done.txt"
