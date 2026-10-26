# Prints the full path of blender.exe, searching the usual places.
$ErrorActionPreference = 'SilentlyContinue'
$diag = Join-Path (Split-Path -Parent $PSScriptRoot) 'out\find_blender_report.txt'
New-Item -ItemType Directory -Force -Path (Split-Path -Parent $diag) | Out-Null
Set-Content -Path $diag -Value ('search ' + (Get-Date))
$cands = @()
# 1) .blend file association (set by the installer)
foreach ($k in @('Registry::HKEY_CLASSES_ROOT\blendfile\shell\open\command',
                 'Registry::HKEY_CLASSES_ROOT\Applications\blender.exe\shell\open\command',
                 'Registry::HKEY_CURRENT_USER\Software\Classes\blendfile\shell\open\command')) {
  $v = (Get-ItemProperty -Path $k).'(default)'
  if ($v -and $v -match '"?([^"]*blender[^"]*\.exe)') { $cands += $Matches[1] }
}
# 2) uninstall entries
foreach ($root in @('HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
                    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*',
                    'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*')) {
  Get-ItemProperty $root | Where-Object { $_.DisplayName -like '*Blender*' } | ForEach-Object {
    if ($_.InstallLocation) { $cands += (Join-Path $_.InstallLocation 'blender.exe') }
    if ($_.DisplayIcon -and $_.DisplayIcon -match '([^",]*blender\.exe)') { $cands += $Matches[1] }
  }
}
# 3) common folders, Steam libraries on every drive, Store alias, Downloads
$dirs = @("$env:ProgramFiles\Blender Foundation", "${env:ProgramFiles(x86)}\Blender Foundation",
          "$env:LOCALAPPDATA\Programs", "$env:USERPROFILE\Downloads", "$env:USERPROFILE\Desktop")
foreach ($d in (Get-PSDrive -PSProvider FileSystem)) {
  $dirs += "$($d.Root)SteamLibrary\steamapps\common\Blender"
  $dirs += "$($d.Root)Program Files (x86)\Steam\steamapps\common\Blender"
  $dirs += "$($d.Root)Program Files\Steam\steamapps\common\Blender"
  $dirs += "$($d.Root)Blender"
}
foreach ($d in $dirs) {
  if (Test-Path $d) {
    Get-ChildItem -Path $d -Filter blender.exe -Recurse -Depth 3 | ForEach-Object { $cands += $_.FullName }
  }
}
# 4) Start-menu / desktop shortcuts
$sh = New-Object -ComObject WScript.Shell
$lnkDirs = @("$env:ProgramData\Microsoft\Windows\Start Menu\Programs", "$env:APPDATA\Microsoft\Windows\Start Menu\Programs",
             "$env:USERPROFILE\Desktop", "$env:PUBLIC\Desktop", "$env:USERPROFILE\OneDrive\Desktop")
foreach ($d in $lnkDirs) {
  if (Test-Path $d) {
    Get-ChildItem -Path $d -Filter *.lnk -Recurse | Where-Object { $_.Name -like '*lender*' } | ForEach-Object {
      $t = $sh.CreateShortcut($_.FullName).TargetPath
      Add-Content -Path $diag -Value ("shortcut: " + $_.FullName + " -> " + $t)
      if ($t -like '*blender*.exe') { $cands += ($t -replace 'blender-launcher\.exe$', 'blender.exe') }
    }
  }
}
# 5) Microsoft Store install
Get-AppxPackage -Name '*Blender*' | ForEach-Object {
  Add-Content -Path $diag -Value ("store: " + $_.InstallLocation)
  foreach ($sub in @('blender.exe', 'Blender\blender.exe')) { $cands += (Join-Path $_.InstallLocation $sub) }
}
$alias = "$env:LOCALAPPDATA\Microsoft\WindowsApps\blender.exe"
if (Test-Path $alias) { $cands += $alias }
$w = (Get-Command blender.exe).Source
if ($w) { $cands += $w }
$cands | ForEach-Object { Add-Content -Path $diag -Value ("candidate: " + $_ + "  exists=" + (Test-Path $_)) }
$found = $cands | Where-Object { $_ -and ($_ -notlike '*WindowsApps*') -and (Test-Path $_) } | Select-Object -Unique
# prefer the newest version
$best = $found | Sort-Object { (Get-Item $_).VersionInfo.FileVersion } -Descending | Select-Object -First 1
if ($best) { Write-Output $best }
