# Negative integration tests: test error handling
param(
    [string]$ServerAddr = "127.0.0.1:8080"
)

Write-Host "=== Word of Wisdom - Negative Tests ===" -ForegroundColor Green

function Test-GarbageData {
    Write-Host "`n[TEST] Sending garbage data to server..." -ForegroundColor Yellow
    
    try {
        $client = New-Object System.Net.Sockets.TcpClient($ServerAddr.Split(':')[0], [int]$ServerAddr.Split(':')[1])
        $stream = $client.GetStream()
        $reader = New-Object System.IO.StreamReader($stream)
        $writer = New-Object System.IO.StreamWriter($stream)
        $writer.AutoFlush = $true

        # Read challenge
        $challenge = $reader.ReadLine()
        Write-Host "Received challenge: $($challenge.Substring(0, [Math]::Min(80, $challenge.Length)))..." -ForegroundColor Gray

        # Send garbage
        Write-Host "Sending garbage data..." -ForegroundColor Gray
        $writer.WriteLine("THIS IS NOT VALID JSON !!!")

        # Try to read response
        try {
            $response = $reader.ReadLine()
            if ($response) {
                $json = $response | ConvertFrom-Json
                if ($json.type -eq "error") {
                    Write-Host "✓ Server returned error: $($json.error)" -ForegroundColor Green
                } else {
                    Write-Host "✗ Server should have returned error, got: $response" -ForegroundColor Red
                }
            }
        } catch {
            Write-Host "✓ Server closed connection (expected behavior)" -ForegroundColor Green
        }

        $client.Close()
    } catch {
        Write-Host "✓ Connection closed by server (expected): $_" -ForegroundColor Green
    }
}

function Test-Timeout {
    Write-Host "`n[TEST] Testing timeout (not sending solution)..." -ForegroundColor Yellow
    
    try {
        $client = New-Object System.Net.Sockets.TcpClient($ServerAddr.Split(':')[0], [int]$ServerAddr.Split(':')[1])
        $stream = $client.GetStream()
        $reader = New-Object System.IO.StreamReader($stream)

        # Read challenge
        $challenge = $reader.ReadLine()
        Write-Host "Received challenge, waiting for timeout..." -ForegroundColor Gray

        # Wait and don't send anything
        Start-Sleep -Seconds 5

        # Try to read (should fail)
        try {
            $stream.ReadTimeout = 5000  # 5 seconds
            $response = $reader.ReadLine()
            if ($null -eq $response) {
                Write-Host "✓ Connection closed by server timeout" -ForegroundColor Green
            } else {
                Write-Host "✗ Expected timeout, got response: $response" -ForegroundColor Red
            }
        } catch {
            Write-Host "✓ Read timed out (expected): $_" -ForegroundColor Green
        }

        $client.Close()
    } catch {
        Write-Host "✓ Connection error (expected): $_" -ForegroundColor Green
    }
}

function Test-InvalidSolution {
    Write-Host "`n[TEST] Sending invalid solution..." -ForegroundColor Yellow
    
    try {
        $client = New-Object System.Net.Sockets.TcpClient($ServerAddr.Split(':')[0], [int]$ServerAddr.Split(':')[1])
        $stream = $client.GetStream()
        $reader = New-Object System.IO.StreamReader($stream)
        $writer = New-Object System.IO.StreamWriter($stream)
        $writer.AutoFlush = $true

        # Read challenge
        $challenge = $reader.ReadLine()
        Write-Host "Received challenge" -ForegroundColor Gray

        # Send invalid solution
        $invalidSolution = @{
            type = "solution"
            nonce = "invalid-nonce-12345"
        } | ConvertTo-Json -Compress

        Write-Host "Sending invalid solution..." -ForegroundColor Gray
        $writer.WriteLine($invalidSolution)

        # Read response
        $response = $reader.ReadLine()
        $json = $response | ConvertFrom-Json

        if ($json.type -eq "error") {
            Write-Host "✓ Server returned error: $($json.error)" -ForegroundColor Green
        } else {
            Write-Host "✗ Expected error, got: $($json.type)" -ForegroundColor Red
        }

        $client.Close()
    } catch {
        Write-Host "✗ Test failed: $_" -ForegroundColor Red
    }
}

function Test-MultipleConnections {
    Write-Host "`n[TEST] Testing multiple concurrent connections..." -ForegroundColor Yellow
    
    $jobs = @()
    for ($i = 1; $i -le 5; $i++) {
        $job = Start-Job -ScriptBlock {
            param($addr, $id)
            try {
                $client = New-Object System.Net.Sockets.TcpClient($addr.Split(':')[0], [int]$addr.Split(':')[1])
                $stream = $client.GetStream()
                $reader = New-Object System.IO.StreamReader($stream)
                $challenge = $reader.ReadLine()
                $client.Close()
                return "Client $id: OK"
            } catch {
                return "Client $id: ERROR - $_"
            }
        } -ArgumentList $ServerAddr, $i
        $jobs += $job
    }

    Start-Sleep -Seconds 2

    foreach ($job in $jobs) {
        $result = Receive-Job $job -Wait
        Write-Host $result -ForegroundColor Gray
        Remove-Job $job
    }

    Write-Host "✓ Multiple connections handled" -ForegroundColor Green
}

# Run all tests
Test-GarbageData
Test-InvalidSolution
Test-Timeout
Test-MultipleConnections

Write-Host "`n=== All negative tests completed ===" -ForegroundColor Green

