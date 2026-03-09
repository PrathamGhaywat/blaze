#!/usr/bin/env pwsh
# Build script for Blaze cross-platform package manager
# Builds binaries for Windows and Linux (amd64 and arm64)

param(
    [string]$OutputDir = "dist",
    [switch]$Clean,
    [switch]$Verbose
)

# Color output
function Write-Success { Write-Host $args -ForegroundColor Green }
function Write-Error-Custom { Write-Host $args -ForegroundColor Red }
function Write-Info { Write-Host $args -ForegroundColor Cyan }

# Clean previous builds
if ($Clean -or !(Test-Path $OutputDir)) {
    Write-Info "Cleaning output directory..."
    if (Test-Path $OutputDir) {
        Remove-Item -Recurse -Force $OutputDir
    }
    New-Item -ItemType Directory -Path $OutputDir | Out-Null
}

Write-Info "Building Blaze binaries..."
Write-Info "Output directory: $OutputDir"

# Build targets
$targets = @(
    @{
        OS     = "windows"
        Arch   = "amd64"
        Output = "blaze.exe"
    }
    @{
        OS     = "linux"
        Arch   = "amd64"
        Output = "blaze"
    }
    @{
        OS     = "linux"
        Arch   = "arm64"
        Output = "blaze"
    }
    @{
        OS     = "windows"
        Arch   = "arm64"
        Output = "blaze.exe"
    }
    @{
        OS     = "darwin"
        Arch   = "amd64"
        Output = "blaze"
    }
    @{
        OS     = "darwin"
        Arch   = "arm64"
        Output = "blaze"
    }
)

$successful = 0
$failed = 0

foreach ($target in $targets) {
    $outputFile = "$OutputDir/blaze-$($target.OS)-$($target.Arch)/$($target.Output)"
    $outputFolder = Split-Path -Parent $outputFile
    
    Write-Info "Building for $($target.OS)/$($target.Arch)..."
    
    # Create output directory
    if (!(Test-Path $outputFolder)) {
        New-Item -ItemType Directory -Path $outputFolder -Force | Out-Null
    }
    
    # Set environment variables
    $env:GOOS = $target.OS
    $env:GOARCH = $target.Arch
    
    # Build
    $buildArgs = @("-o", $outputFile, "./src")
    if ($Verbose) {
        Write-Info "  Command: go build $($buildArgs -join ' ')"
    }
    
    $output = go build $buildArgs 2>&1
    
    if ($LASTEXITCODE -eq 0) {
        $fileInfo = Get-Item $outputFile
        $fileSize = [math]::Round($fileInfo.Length / 1MB, 2)
        Write-Success "  OK - $(Split-Path -Leaf $outputFile) ($fileSize MB)"
        $successful++
    } else {
        Write-Error-Custom "  FAILED - $output"
        $failed++
    }
    
    # Clear environment variables
    $env:GOOS = ""
    $env:GOARCH = ""
}

Write-Info ""
Write-Info "Build Summary:"
Write-Success "  Successful: $successful"
if ($failed -gt 0) {
    Write-Error-Custom "  Failed: $failed"
} else {
    Write-Success "  Failed: $failed"
}

if ($failed -eq 0) {
    Write-Success ""
    Write-Success "All builds completed successfully!"
    Write-Info "Binaries available in: $OutputDir"
    Write-Info ""
    Write-Info "Next steps:"
    Write-Info "  - Test: ./dist/blaze-windows-amd64/blaze.exe list"
    Write-Info "  - Package and distribute binaries"
} else {
    exit 1
}
