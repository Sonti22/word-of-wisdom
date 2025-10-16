# PowerShell Makefile alternative for Windows
# Usage: .\make.ps1 <target>

param(
    [Parameter(Position=0)]
    [string]$Target = "help"
)

function Show-Help {
    Write-Host "`n=== Word of Wisdom - Available Targets ===" -ForegroundColor Green
    Write-Host ""
    Write-Host "Build:"
    Write-Host "  build            - Build server and client binaries"
    Write-Host "  build-windows    - Build Windows binaries (.exe)"
    Write-Host "  install-deps     - Download Go dependencies"
    Write-Host ""
    Write-Host "Run:"
    Write-Host "  run-server       - Run the server"
    Write-Host "  run-client       - Run the client"
    Write-Host ""
    Write-Host "Test:"
    Write-Host "  test             - Run all tests with race detector"
    Write-Host "  test-short       - Run only unit tests"
    Write-Host "  test-integration - Run only integration tests"
    Write-Host "  test-coverage    - Generate coverage report"
    Write-Host ""
    Write-Host "Quality:"
    Write-Host "  fmt              - Format Go code"
    Write-Host "  lint             - Run linters (gofmt, go vet)"
    Write-Host "  check            - Run fmt + lint + test"
    Write-Host ""
    Write-Host "Docker:"
    Write-Host "  docker-build     - Build Docker images"
    Write-Host "  up               - Start services via docker-compose"
    Write-Host "  down             - Stop services"
    Write-Host "  logs             - Show docker-compose logs"
    Write-Host ""
    Write-Host "Cleanup:"
    Write-Host "  clean            - Remove build artifacts"
    Write-Host "  clean-docker     - Remove Docker resources"
    Write-Host ""
}

function Build {
    Write-Host "Building binaries..." -ForegroundColor Yellow
    New-Item -ItemType Directory -Force -Path bin | Out-Null
    go build -ldflags="-w -s" -o bin\server.exe .\cmd\server
    go build -ldflags="-w -s" -o bin\client.exe .\cmd\client
    Write-Host "✓ Build complete: bin\server.exe, bin\client.exe" -ForegroundColor Green
}

function Install-Deps {
    Write-Host "Downloading dependencies..." -ForegroundColor Yellow
    go mod download
    go mod tidy
    Write-Host "✓ Dependencies installed" -ForegroundColor Green
}

function Run-Server {
    Write-Host "Starting server..." -ForegroundColor Yellow
    go run .\cmd\server
}

function Run-Client {
    Write-Host "Starting client..." -ForegroundColor Yellow
    go run .\cmd\client
}

function Test {
    Write-Host "Running tests..." -ForegroundColor Yellow
    go test -race -v ./...
}

function Test-Short {
    Write-Host "Running unit tests..." -ForegroundColor Yellow
    go test -short -race -v ./...
}

function Test-Integration {
    Write-Host "Running integration tests..." -ForegroundColor Yellow
    go test -v .\tests\
}

function Test-Coverage {
    Write-Host "Generating coverage report..." -ForegroundColor Yellow
    go test -race -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    Write-Host "✓ Coverage report: coverage.html" -ForegroundColor Green
}

function Format {
    Write-Host "Formatting code..." -ForegroundColor Yellow
    gofmt -l -s -w .
    Write-Host "✓ Code formatted" -ForegroundColor Green
}

function Lint {
    Write-Host "Running linters..." -ForegroundColor Yellow
    Write-Host "  - gofmt..."
    $unformatted = gofmt -l -s .
    if ($unformatted) {
        Write-Host "Code not formatted. Run '.\make.ps1 fmt'" -ForegroundColor Red
        exit 1
    }
    Write-Host "  - go vet..."
    go vet ./...
    Write-Host "✓ Lint passed" -ForegroundColor Green
}

function Check {
    Format
    Lint
    Test
}

function Docker-Build {
    Write-Host "Building Docker images..." -ForegroundColor Yellow
    docker build -f Dockerfile.server -t word-of-wisdom-server:latest .
    docker build -f Dockerfile.client -t word-of-wisdom-client:latest .
    Write-Host "✓ Docker images built" -ForegroundColor Green
}

function Docker-Up {
    Write-Host "Starting services via docker-compose..." -ForegroundColor Yellow
    docker-compose up --build
}

function Docker-Up-D {
    Write-Host "Starting services in background..." -ForegroundColor Yellow
    docker-compose up -d --build
}

function Docker-Down {
    Write-Host "Stopping services..." -ForegroundColor Yellow
    docker-compose down
    Write-Host "✓ Services stopped" -ForegroundColor Green
}

function Show-Logs {
    docker-compose logs -f
}

function Clean {
    Write-Host "Cleaning up..." -ForegroundColor Yellow
    Remove-Item -Recurse -Force -ErrorAction SilentlyContinue bin, coverage.out, coverage.html, server.log
    Write-Host "✓ Clean complete" -ForegroundColor Green
}

function Clean-Docker {
    Write-Host "Cleaning Docker resources..." -ForegroundColor Yellow
    docker-compose down -v --rmi all --remove-orphans
    Write-Host "✓ Docker resources cleaned" -ForegroundColor Green
}

# Execute target
switch ($Target.ToLower()) {
    "help"             { Show-Help }
    "build"            { Build }
    "build-windows"    { Build }
    "install-deps"     { Install-Deps }
    "run-server"       { Run-Server }
    "run-client"       { Run-Client }
    "test"             { Test }
    "test-short"       { Test-Short }
    "test-integration" { Test-Integration }
    "test-coverage"    { Test-Coverage }
    "fmt"              { Format }
    "lint"             { Lint }
    "check"            { Check }
    "docker-build"     { Docker-Build }
    "up"               { Docker-Up }
    "docker-up"        { Docker-Up }
    "docker-up-d"      { Docker-Up-D }
    "down"             { Docker-Down }
    "docker-down"      { Docker-Down }
    "logs"             { Show-Logs }
    "clean"            { Clean }
    "clean-docker"     { Clean-Docker }
    default {
        Write-Host "Unknown target: $Target" -ForegroundColor Red
        Write-Host "Run '.\make.ps1 help' for available targets" -ForegroundColor Yellow
        exit 1
    }
}

