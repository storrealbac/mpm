#!/usr/bin/env pwsh
# mpm installer

Write-Host "mpm: Minecraft Plugin Manager Installer" -ForegroundColor Cyan

# Create directory
New-Item -ItemType Directory -Force -Path "$HOME\mpm" | Out-Null

# Download
Write-Host "Downloading mpm..." -ForegroundColor Yellow
Invoke-WebRequest -Uri "https://github.com/storrealbac/mpm/releases/latest/download/mpm-windows-latest-amd64.zip" -OutFile "$HOME\mpm\mpm.zip"

# Extract
Write-Host "Installing..." -ForegroundColor Yellow
Expand-Archive -Path "$HOME\mpm\mpm.zip" -DestinationPath "$HOME\mpm" -Force

# Add to PATH
$path = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($path -notlike "*$HOME\mpm*") {
    $newPath = if ($path) { "$path;$HOME\mpm" } else { "$HOME\mpm" }
    [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
}

# Cleanup
Remove-Item "$HOME\mpm\mpm.zip" -Force

Write-Host "mpm installed to $HOME\mpm" -ForegroundColor Green
Write-Host "Run 'mpm init' to get started" -ForegroundColor Yellow
Write-Host "Restart your terminal to use mpm" -ForegroundColor Yellow