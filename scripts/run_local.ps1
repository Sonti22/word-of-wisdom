# Local integration test: start server and run client
param(
    [string]$ServerAddr = ":8080",
    [int]$Bits = 20,
    [int]$Expires = 60
)

Write-Host "=== Word of Wisdom - Local Integration Test ===" -ForegroundColor Green

# Build binaries
Write-Host "`nBuilding binaries..." -ForegroundColor Yellow
go build -o bin\server.exe .\cmd\server
go build -o bin\client.exe .\cmd\client

if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed" -ForegroundColor Red
    exit 1
}

# Start server in background
Write-Host "`nStarting server (addr=$ServerAddr, bits=$Bits, expires=$Expires)..." -ForegroundColor Yellow
$env:WOW_ADDR = $ServerAddr
$env:WOW_BITS = $Bits
$env:WOW_EXPIRES = $Expires

$serverJob = Start-Job -ScriptBlock {
    param($addr, $bits, $expires, $path)
    $env:WOW_ADDR = $addr
    $env:WOW_BITS = $bits
    $env:WOW_EXPIRES = $expires
    Set-Location $path
    & .\bin\server.exe
} -ArgumentList $ServerAddr, $Bits, $Expires, (Get-Location).Path

# Wait for server to start
Write-Host "Waiting for server to start..." -ForegroundColor Yellow
Start-Sleep -Seconds 3

# Check if server is running
if ($serverJob.State -eq "Failed") {
    Write-Host "Server failed to start" -ForegroundColor Red
    Receive-Job $serverJob
    Remove-Job $serverJob
    exit 1
}

Write-Host "Server started (Job ID: $($serverJob.Id))" -ForegroundColor Green

# Run client
Write-Host "`nRunning client..." -ForegroundColor Yellow
$clientAddr = $ServerAddr -replace "^:", "127.0.0.1:"
$env:WOW_ADDR = $clientAddr

try {
    & .\bin\client.exe
    if ($LASTEXITCODE -eq 0) {
        Write-Host "`n✓ Client succeeded" -ForegroundColor Green
        $exitCode = 0
    } else {
        Write-Host "`n✗ Client failed (exit code: $LASTEXITCODE)" -ForegroundColor Red
        $exitCode = 1
    }
} catch {
    Write-Host "`n✗ Client failed: $_" -ForegroundColor Red
    $exitCode = 1
}

# Cleanup: stop server
Write-Host "`nStopping server..." -ForegroundColor Yellow
Stop-Job $serverJob -ErrorAction SilentlyContinue
Remove-Job $serverJob -Force -ErrorAction SilentlyContinue

Write-Host "`n=== Test completed ===" -ForegroundColor Green

exit $exitCode

