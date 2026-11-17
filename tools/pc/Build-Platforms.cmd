@echo off
rem Cross-builds Arcane Orb for other platforms from this PC (no Cgo needed
rem since Ebitengine 2.10). Results and errors: build-platforms.txt.
rem Output goes to builds\<platform>\ beside this file.
setlocal
cd /d "%~dp0Source"
set GOEXE=%~dp0..\go-sdk\go\bin\go.exe
set OUT=%~dp0build-platforms.txt
if exist "%~dp0platforms_done.txt" del "%~dp0platforms_done.txt"
echo === %DATE% %TIME% === > "%OUT%"
set CGO_ENABLED=0
call :one linux amd64 ArcaneOrb
call :one linux arm64 ArcaneOrb
call :one darwin arm64 ArcaneOrb
call :one darwin amd64 ArcaneOrb
call :one js wasm ArcaneOrb.wasm
echo === DONE === >> "%OUT%"
echo ok> "%~dp0platforms_done.txt"
goto :eof

:one
set GOOS=%1
set GOARCH=%2
if not exist "%~dp0builds\%1-%2" mkdir "%~dp0builds\%1-%2"
echo === %1/%2 === >> "%OUT%"
"%GOEXE%" build -trimpath -ldflags "-s -w" -o "%~dp0builds\%1-%2\%3" . >> "%OUT%" 2>&1
echo exit %ERRORLEVEL% >> "%OUT%"
goto :eof
