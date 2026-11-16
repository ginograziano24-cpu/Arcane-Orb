@echo off
rem Free check only: balance, docs and a dry run. No credits are spent.
cd /d "%~dp0"
set R=%~dp0heroes\intro_chars\fumbleton_ref
set OUT=%~dp0tripo-fumbleton-check.txt
echo === %DATE% %TIME% === > "%OUT%"
echo ===== balance ===== >> "%OUT%"
call tripo balance --json >> "%OUT%" 2>&1
echo ===== docs generate ===== >> "%OUT%"
call tripo docs --topic commands/generate >> "%OUT%" 2>&1
echo ===== docs make ===== >> "%OUT%"
call tripo docs --topic commands/make >> "%OUT%" 2>&1
echo ===== docs rig ===== >> "%OUT%"
call tripo docs --topic commands/anim >> "%OUT%" 2>&1
echo ===== dry run multiview ===== >> "%OUT%"
call tripo make "%R%\fumbleton_front.png" "%R%\fumbleton_side_right.png" "%R%\fumbleton_back.png" --for anim --then rig --dry-run --json >> "%OUT%" 2>&1
echo === DONE === >> "%OUT%"
