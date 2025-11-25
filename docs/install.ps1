#!/usr/bin/env pwsh
# mpm installer

Write-Host "mpm: Minecraft Plugin Manager Installer" -ForegroundColor DarkYellow

# Create directory
New-Item -ItemType Directory -Force -Path "$HOME\.mpm" | Out-Null

# Remove old version if exists
if (Test-Path "$HOME\.mpm\mpm.exe") {
    Write-Host "Removing old version..." -ForegroundColor Yellow
    Remove-Item "$HOME\.mpm\mpm.exe" -Force -ErrorAction SilentlyContinue
}
if (Test-Path "$HOME\.mpm\mpm.cmd") {
    Remove-Item "$HOME\.mpm\mpm.cmd" -Force -ErrorAction SilentlyContinue
}

# Download
Write-Host "Downloading mpm..." -ForegroundColor Yellow
try {
    Invoke-WebRequest -Uri "https://github.com/storrealbac/mpm/releases/latest/download/mpm-windows-latest-amd64.zip" -OutFile "$HOME\.mpm\mpm.zip"
} catch {
    Write-Host "Failed to download. Please check your internet connection." -ForegroundColor Red
    exit 1
}

# Extract
Write-Host "Installing..." -ForegroundColor Yellow
try {
    Expand-Archive -Path "$HOME\.mpm\mpm.zip" -DestinationPath "$HOME\.mpm" -Force
} catch {
    Write-Host "Failed to extract archive." -ForegroundColor Red
    exit 1
}

# Create batch wrapper for 'mpm' command (without .exe)
$batchContent = @"
@echo off
"%~dp0mpm.exe" %*
"@
Set-Content -Path "$HOME\.mpm\mpm.cmd" -Value $batchContent -Encoding ASCII

# Add to PATH
$path = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($path -notlike "*$HOME\.mpm*") {
    $newPath = if ($path) { "$path;$HOME\.mpm" } else { "$HOME\.mpm" }
    [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
}

# Cleanup
Remove-Item "$HOME\.mpm\mpm.zip" -Force

Write-Host "mpm installed to $HOME\.mpm" -ForegroundColor Green
Write-Host "Run 'mpm init' to get started" -ForegroundColor Yellow
Write-Host "Restart your terminal to use mpm" -ForegroundColor Yellow