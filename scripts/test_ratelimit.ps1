# Test rate limiting and adaptive difficulty

$ErrorActionPreference = "Stop"

Write-Host "Building binaries..."
go build -o bin\server_test.exe .\cmd\server
go build -o bin\client_test.exe .\cmd\client

Write-Host "Starting server with rate limit (5 req/sec) and adaptive bits..."
$env:WOW_ADDR = ":8081"
$env:WOW_BITS = "18"
$env:WOW_RATE_LIMIT = "5"
$env:WOW_ADAPTIVE_BITS = "true"
$env:WOW_EXPIRES = "300"

$server = Start-Process -FilePath "bin\server_test.exe" -NoNewWindow -PassThru

# Wait for server to start
Start-Sleep -Seconds 2

Write-Host "Testing rapid requests (should trigger rate limit and adaptive difficulty)..."
$env:WOW_ADDR = "127.0.0.1:8081"

for ($i = 1; $i -le 10; $i++) {
    Write-Host "Request #$i:"
    & bin\client_test.exe
    Start-Sleep -Milliseconds 100
}

Write-Host "Killing server..."
Stop-Process -Id $server.Id -Force

Write-Host "Done! Check logs above for:"
Write-Host "  - 'rate limited' messages after 5-10 requests"
Write-Host "  - 'adaptive difficulty increased' with rising bits"

