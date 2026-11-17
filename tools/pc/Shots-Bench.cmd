@echo off
rem Runs the screenshot tour (also the benchmark tour) on the build made by Go-Check.
rem Pictures go to Source\shots, the frame-time report to bench_report_shots.txt.
cd /d "%~dp0Source"
if exist "%~dp0shots_done.txt" del "%~dp0shots_done.txt"
if not exist ArcaneOrb-test.exe (echo no build: run Go-Check first> "%~dp0shots_done.txt" & goto :eof)
ArcaneOrb-test.exe -shots -benchwin=1600x900
echo ok> "%~dp0shots_done.txt"
