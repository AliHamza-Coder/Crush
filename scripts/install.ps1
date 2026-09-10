# CRUSH Installer - One command install
# Usage: irm https://raw.githubusercontent.com/AliHamza-Coder/Crush/main/scripts/install.ps1 | iex

$ErrorActionPreference = "Stop"
$repo = "AliHamza-Coder/Crush"

Write-Host ""
Write-Host "  Installing CRUSH..." -ForegroundColor Cyan
Write-Host ""

# Get latest version from GitHub
$release = Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest"
$version = $release.tag_name -replace '^v', ''

# Find the installer exe
$installer = $release.assets | Where-Object { $_.name -like '*setup*' -and $_.name -like '*.exe' } | Select-Object -First 1

if ($installer) {
    # Download and run NSIS installer
    $tempExe = "$env:TEMP\crush-setup.exe"
    Write-Host "  Downloading v$version installer..." -ForegroundColor Yellow
    Invoke-WebRequest -Uri $installer.browser_download_url -OutFile $tempExe
    Write-Host "  Running installer..." -ForegroundColor Yellow
    Start-Process -FilePath $tempExe -Wait
    Remove-Item $tempExe -Force
} else {
    # Fallback: download portable zip
    $asset = $release.assets | Where-Object { $_.name -like '*windows*' -and $_.name -like '*.zip' } | Select-Object -First 1
    if (-not $asset) {
        Write-Host "  Error: No Windows build found" -ForegroundColor Red
        exit 1
    }
    
    $installDir = "$env:USERPROFILE\Crush"
    $tempZip = "$env:TEMP\crush.zip"
    
    Write-Host "  Downloading v$version portable..." -ForegroundColor Yellow
    Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $tempZip
    
    Write-Host "  Installing to $installDir..." -ForegroundColor Yellow
    if (Test-Path $installDir) { Remove-Item $installDir -Recurse -Force }
    Expand-Archive -Path $tempZip -DestinationPath $env:TEMP
    $extracted = Get-ChildItem "$env:TEMP\crush-*" -Directory | Select-Object -First 1
    Move-Item $extracted.FullName $installDir
    
    # Add to PATH
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($currentPath -notlike "*$installDir*") {
        [Environment]::SetEnvironmentVariable("Path", "$currentPath;$installDir", "User")
    }
    
    Remove-Item $tempZip -Force
}

Write-Host ""
Write-Host "  Done! Restart your terminal and run: crush" -ForegroundColor Green
Write-Host ""
