# Simple PowerShell script to test the TCP server
# Usage: .\test-server.ps1

param(
    [string]$Host = "localhost",
    [int]$Port = 8080
)

Write-Host "Connecting to $Host:$Port..." -ForegroundColor Green

try {
    $client = New-Object System.Net.Sockets.TcpClient($Host, $Port)
    $stream = $client.GetStream()
    $reader = New-Object System.IO.StreamReader($stream)
    $writer = New-Object System.IO.StreamWriter($stream)
    $writer.AutoFlush = $true

    Write-Host "`nReceived challenge:" -ForegroundColor Yellow
    $challenge = $reader.ReadLine()
    Write-Host $challenge

    # Parse JSON to pretty-print (optional)
    try {
        $json = $challenge | ConvertFrom-Json
        Write-Host "`nParsed challenge:" -ForegroundColor Cyan
        $json | Format-List
    } catch {
        # If JSON parsing fails, just show raw
    }

    $client.Close()
    Write-Host "`nConnection closed." -ForegroundColor Green
} catch {
    Write-Host "Error: $_" -ForegroundColor Red
}

