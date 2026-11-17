@echo off
rem Profiles World 7 on its own (fullscreen, 20 s), for finding frame-time wins.
rem Writes Source\bench_report_w7.txt and the CPU / allocation profiles.
cd /d "%~dp0Source"
if exist "%~dp0w7_done.txt" del "%~dp0w7_done.txt"
if not exist ArcaneOrb-test.exe (echo no build: run Go-Check first> "%~dp0w7_done.txt" & goto :eof)
ArcaneOrb-test.exe -bench -benchscenes=world7 -benchduration=20s -benchtag=w7
echo ok> "%~dp0w7_done.txt"
