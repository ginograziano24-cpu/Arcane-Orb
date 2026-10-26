@echo off
setlocal
cd /d "%~dp0"
if not exist out mkdir out
set "BL="
rem If auto-detection ever fails, put the full path here, e.g.
rem set "BL=C:\Program Files\Blender Foundation\Blender 4.5\blender.exe"
echo search %date% %time% > out\find_log.txt
rem 0) the standard blender.org install (preferred)
for /d %%D in ("%ProgramFiles%\Blender Foundation\Blender*") do if exist "%%D\blender.exe" set "BL=%%D\blender.exe"
echo after where: [%BL%] >> out\find_log.txt
echo after store: [%BL%] >> out\find_log.txt
rem 3) full search (registry, shortcuts, Steam, Downloads)
if not defined BL for /f "usebackq delims=" %%P in (`powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\find_blender.ps1" 2^>^>out\find_log.txt`) do set "BL=%%P"
echo after search: [%BL%] >> out\find_log.txt
powershell -NoProfile -Command "Get-AppxPackage -Name '*Blender*' | Format-List Name,Version,InstallLocation" >> out\find_log.txt 2>&1
if not defined BL (
  echo Could not find Blender automatically. Claude can read out\find_log.txt to fix this.
  pause
  exit /b 1
)
set "TASK=%*"
if "%TASK%"=="" set "TASK=all"
echo ============================================================
echo  Arcane Orb Art Kit  -  task: %TASK%
echo  Blender: %BL%
echo  Rendering... this can take a few minutes. Leave this window open.
echo ============================================================
"%BL%" -b --factory-startup -P "%~dp0scripts\run.py" -- %TASK% > "%~dp0out\log.txt" 2>&1
echo exit code %ERRORLEVEL% >> "%~dp0out\log.txt"
findstr /L /C:"[kit]" "%~dp0out\log.txt"
echo.
echo DONE. Results are in the "out" folder.
pause
