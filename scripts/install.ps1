<#
.SYNOPSIS
    CRUSH — One-command installer for Windows
.DESCRIPTION
    Downloads crush.exe and sets up PATH automatically.
    Run: iex "& {$(iwr -Uri https://raw.githubusercontent.com/AliHamza-Coder/crush/main/scripts/install.ps1)}"
.PARAMETER Portable
    Skip PATH setup, download to current directory
#>

param([switch]$Portable)

$repo = "AliHamza-Coder/crush"
$url = "https://github.com/$repo/releases/latest/download/crush.exe"

if ($Portable) {
    $exePath = Join-Path (Get-Location) "crush.exe"
} else {
    $installDir = Join-Path $env:LOCALAPPDATA "crush"
    $exePath = Join-Path $installDir "crush.exe"
    New-Item -ItemType Directory -Force -Path $installDir | Out-Null
}

Write-Host "╔═══════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║     CRUSH Installer                   ║" -ForegroundColor Cyan
Write-Host "╚═══════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

try {
    Write-Host "Downloading CRUSH..." -NoNewline
    # Force TLS 1.2 — required by GitHub on older PowerShell
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
    Invoke-WebRequest -Uri $url -OutFile $exePath -UseBasicParsing -ErrorAction Stop
    Write-Host " ✓" -ForegroundColor Green
} catch {
    Write-Host " ✗ Failed: $_" -ForegroundColor Red
    exit 1
}

if (-not $Portable) {
    $path = [Environment]::GetEnvironmentVariable('Path', 'User')
    if ($path -notlike "*$installDir*") {
        [Environment]::SetEnvironmentVariable('Path', "$path;$installDir", 'User')
        Write-Host "Added to PATH: $installDir" -ForegroundColor Green
    }
    Write-Host ""
    Write-Host "Install FFmpeg? (required)" -ForegroundColor Yellow
    Write-Host "  Run: winget install -e --id Gyan.FFmpeg" -ForegroundColor Cyan
}

Write-Host ""
Write-Host "╔═══════════════════════════════════════╗" -ForegroundColor Green
Write-Host "║  ✓ READY                              ║" -ForegroundColor Green
if ($Portable) {
    Write-Host "║     crush.exe is in current folder    ║" -ForegroundColor Green
} else {
    Write-Host "║     Run 'crush' from any terminal      ║" -ForegroundColor Green
}
Write-Host "║     Developed by Ali Hamza Coder       ║" -ForegroundColor Green
Write-Host "╚═══════════════════════════════════════╝" -ForegroundColor Green
