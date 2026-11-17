@echo off
rem Downloads Blender Studio's free Human Base Meshes v1.2 (CC0, ~49 MB) from blender.org's mirror.
cd /d "%~dp0"
set D=%~dp0heroes\basemesh
if not exist "%D%" mkdir "%D%"
curl -L --fail -o "%D%\human-base-meshes-bundle-v1.2.0.zip" "https://ftp.nluug.nl/pub/graphics/blender/demo/asset-bundles/human-base-meshes/human-base-meshes-bundle-v1.2.0.zip" > "%D%\download-log.txt" 2>&1
echo curl exit %ERRORLEVEL% >> "%D%\download-log.txt"
tar -xf "%D%\human-base-meshes-bundle-v1.2.0.zip" -C "%D%" >> "%D%\download-log.txt" 2>&1
echo tar exit %ERRORLEVEL% >> "%D%\download-log.txt"
dir /s /b "%D%" >> "%D%\download-log.txt"
