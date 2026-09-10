# Build script for TinyProxyAPI
# Uses -ldflags="-H windowsgui" to hide console window on Windows

$ErrorActionPreference = "Stop"

Write-Host "Building TinyProxyAPI..." -ForegroundColor Cyan

go build -ldflags="-H windowsgui" -o TinyProxyAPI.exe

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful: TinyProxyAPI.exe" -ForegroundColor Green
} else {
    Write-Host "Build failed with exit code: $LASTEXITCODE" -ForegroundColor Red
    exit $LASTEXITCODE
}
