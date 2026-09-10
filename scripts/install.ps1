# CRUSH Installer - One command install
# Usage: irm https://raw.githubusercontent.com/AliHamza-Coder/Crush/main/scripts/install.ps1 | iex

$ErrorActionPreference = "Stop"

Write-Host ""
Write-Host "  ╔════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "  ║      CRUSH v3.0.0 Installer        ║" -ForegroundColor Cyan
Write-Host "  ╚════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

$installDir = "$env:USERPROFILE\Crush"
$tempDir = "$env:TEMP\crush-build"

Write-Host "  [1/4] Cloning repository..." -ForegroundColor Yellow
if (Test-Path $tempDir) { Remove-Item $tempDir -Recurse -Force }
git clone --depth 1 https://github.com/AliHamza-Coder/Crush.git $tempDir 2>&1 | Out-Null

Write-Host "  [2/4] Building release binary..." -ForegroundColor Yellow
Push-Location $tempDir
cargo build --release 2>&1 | Out-Null
Pop-Location

Write-Host "  [3/4] Installing to $installDir..." -ForegroundColor Yellow
if (Test-Path $installDir) { Remove-Item $installDir -Recurse -Force }
New-Item -ItemType Directory -Path $installDir -Force | Out-Null
Copy-Item "$tempDir\target\release\crush.exe" "$installDir\crush.exe"

# Create batch wrapper
$batContent = "@echo off`r`n`"$installDir\crush.exe`" %*"
Set-Content -Path "$installDir\crush.bat" -Value $batContent

Write-Host "  [4/4] Adding to PATH..." -ForegroundColor Yellow
$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($currentPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$currentPath;$installDir", "User")
    $env:Path = "$env:Path;$installDir"
}

# Cleanup
Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "  ✓ CRUSH installed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "  Run: crush" -ForegroundColor Cyan
Write-Host "  Or: crush analyse ." -ForegroundColor Cyan
Write-Host ""
Write-Host "  Restart your terminal to refresh PATH." -ForegroundColor DarkGray
Write-Host ""
