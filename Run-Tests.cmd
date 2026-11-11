@echo off
rem Builds and tests Arcane Orb, writing everything to test-results.txt
setlocal enabledelayedexpansion
cd /d "%~dp0Source"
set OUT=%~dp0test-results.txt
echo === %DATE% %TIME% === > "%OUT%"
set GO=
for %%P in ("%ProgramFiles%\Go\bin\go.exe" "%LOCALAPPDATA%\Programs\Go\bin\go.exe" "C:\Go\bin\go.exe" "%USERPROFILE%\scoop\apps\go\current\bin\go.exe") do if exist %%P if not defined GO set GO=%%~P
if not defined GO for /f "delims=" %%F in ('dir /b /s "%USERPROFILE%\go\pkg\mod\golang.org\go.exe" 2^>nul') do if not defined GO set GO=%%F
if not defined GO for /f "delims=" %%F in ('dir /b /s "%USERPROFILE%\sdk\go.exe" 2^>nul') do if not defined GO set GO=%%F
if not defined GO if not defined GO if exist "%~dp0..\go-sdk\go\bin\go.exe" set GO=%~dp0..\go-sdk\go\bin\go.exe
if not defined GO (
  echo Go not found: downloading the official Go 1.26.8 for Windows into RoguelikeBreaker\go-sdk ... >> "%OUT%"
  echo Downloading Go 1.26.8 from go.dev, about 75 MB. This window closes by itself when everything is done.
  if not exist "%~dp0..\go-sdk" mkdir "%~dp0..\go-sdk"
  curl.exe -L --fail -o "%~dp0..\go-sdk\go.zip" https://go.dev/dl/go1.26.8.windows-amd64.zip >> "%OUT%" 2>&1
  tar -xf "%~dp0..\go-sdk\go.zip" -C "%~dp0..\go-sdk" >> "%OUT%" 2>&1
  del "%~dp0..\go-sdk\go.zip" 2>nul
  if exist "%~dp0..\go-sdk\go\bin\go.exe" set GO=%~dp0..\go-sdk\go\bin\go.exe
)
if not defined GO (echo NO GO FOUND >> "%OUT%" & echo Searched Program Files, LocalAppData, C:\Go, scoop, %%USERPROFILE%%\go\pkg\mod, %%USERPROFILE%%\sdk, RoguelikeBreaker >> "%OUT%" & goto :end)
echo Using "!GO!" >> "%OUT%"
for %%D in ("!GO!") do set PATH=%%~dpD;%PATH%
"!GO!" version >> "%OUT%" 2>&1
echo === go vet ./... === >> "%OUT%"
"!GO!" vet ./... >> "%OUT%" 2>&1
echo vet exit !ERRORLEVEL! >> "%OUT%"
echo === go test ./... === >> "%OUT%"
"!GO!" test ./... >> "%OUT%" 2>&1
echo test exit !ERRORLEVEL! >> "%OUT%"
echo === go build === >> "%OUT%"
"!GO!" build -o ArcaneOrb-test.exe . >> "%OUT%" 2>&1
echo build exit !ERRORLEVEL! >> "%OUT%"
:end
echo === DONE === >> "%OUT%"
