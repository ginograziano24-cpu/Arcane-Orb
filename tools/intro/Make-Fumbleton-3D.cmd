@echo off
rem Makes Fumbleton from his 3-view turnaround (approved by Gino, Nov 16). ~70 credits, then a rig attempt.
cd /d "%~dp0"
set R=%~dp0heroes\intro_chars\fumbleton_ref
set O=%~dp0heroes\intro_chars\fumbleton_tripo
set OUT=%~dp0tripo-fumbleton.txt
echo === %DATE% %TIME% === > "%OUT%"
call tripo balance --json >> "%OUT%" 2>&1
echo ===== dry run ===== >> "%OUT%"
call tripo make "%R%\fumbleton_front.png" "%R%\fumbleton_back.png" --model tripo-v3.1 -p texture_quality=detailed -p geometry_quality=detailed --dry-run --json >> "%OUT%" 2>&1
echo ===== make ===== >> "%OUT%"
call tripo make "%R%\fumbleton_front.png" "%R%\fumbleton_back.png" --model tripo-v3.1 -p texture_quality=detailed -p geometry_quality=detailed --name fumbleton -o "%O%" --json --yes --no-open >> "%OUT%" 2>&1
echo make exit %ERRORLEVEL% >> "%OUT%"
call tripo balance --json >> "%OUT%" 2>&1
echo ===== rig ===== >> "%OUT%"
call tripo anim rig @fumbleton --rig-type biped --spec mixamo --out-format glb -o "%O%\rigged" --json --yes --no-open >> "%OUT%" 2>&1
echo rig exit %ERRORLEVEL% >> "%OUT%"
call tripo balance --json >> "%OUT%" 2>&1
echo === DONE === >> "%OUT%"
