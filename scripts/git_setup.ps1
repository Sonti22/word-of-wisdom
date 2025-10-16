# Git Setup Script for Word of Wisdom
# Run this to initialize git and prepare for GitHub push

Write-Host "=== Git Setup for Word of Wisdom ===" -ForegroundColor Cyan

# Check if git is installed
try {
    $gitVersion = git --version
    Write-Host "✓ Git found: $gitVersion" -ForegroundColor Green
} catch {
    Write-Host "✗ Git not found. Please install Git first:" -ForegroundColor Red
    Write-Host "  https://git-scm.com/download/win" -ForegroundColor Yellow
    exit 1
}

# Initialize git if not already
if (-Not (Test-Path ".git")) {
    Write-Host "`nInitializing Git repository..." -ForegroundColor Yellow
    git init
    Write-Host "✓ Git initialized" -ForegroundColor Green
} else {
    Write-Host "`n✓ Git already initialized" -ForegroundColor Green
}

# Set commit template
Write-Host "`nSetting commit template..." -ForegroundColor Yellow
git config commit.template .gitmessage
Write-Host "✓ Commit template configured" -ForegroundColor Green

# Check git config
$userName = git config user.name
$userEmail = git config user.email

if (-Not $userName) {
    Write-Host "`n⚠ Git user.name not set" -ForegroundColor Yellow
    $name = Read-Host "Enter your Git username (e.g., mjmln)"
    git config user.name $name
    Write-Host "✓ User name set to: $name" -ForegroundColor Green
} else {
    Write-Host "`n✓ Git user.name: $userName" -ForegroundColor Green
}

if (-Not $userEmail) {
    Write-Host "`n⚠ Git user.email not set" -ForegroundColor Yellow
    $email = Read-Host "Enter your Git email"
    git config user.email $email
    Write-Host "✓ User email set to: $email" -ForegroundColor Green
} else {
    Write-Host "✓ Git user.email: $userEmail" -ForegroundColor Green
}

# Show status
Write-Host "`n=== Current Status ===" -ForegroundColor Cyan
git status --short

# Prompt for commit
Write-Host "`n=== Ready to Commit ===" -ForegroundColor Cyan
$commit = Read-Host "Create initial commit? (y/n)"

if ($commit -eq "y" -or $commit -eq "Y") {
    Write-Host "`nAdding files..." -ForegroundColor Yellow
    git add .
    
    Write-Host "Creating initial commit..." -ForegroundColor Yellow
    git commit -m "feat(all): initial implementation - PoW TCP server with rate limiting

Complete implementation of Word of Wisdom TCP server with:
- Hashcash PoW (SHA-256 with configurable difficulty)
- Rate limiting (token bucket per IP)
- Adaptive difficulty (dynamic bits scaling)
- Anti-DDoS/replay protection (salt, timestamp, TTL)
- Docker + Compose deployment
- Comprehensive test suite (31 unit + 5 e2e)
- Full documentation (README, BRANCHES, templates)

Components:
- internal/pow: Challenge generation and verification
- internal/quotes: Random wisdom quotes
- internal/ratelimit: Token bucket + adaptive difficulty
- internal/server: TCP protocol + JSON handlers
- cmd/server: Server with graceful shutdown
- cmd/client: Parallel PoW solver (8 goroutines)
- tests: Integration tests (success/error/timeout/CLI/garbage)
- scripts: Local runners + test helpers
- Docker: Multi-stage builds (scratch/alpine, 7MB)

Testing:
- go test ./...: 31/31 PASS
- go vet ./...: clean
- gofmt: formatted

Refs: #1-12"
    
    Write-Host "✓ Initial commit created" -ForegroundColor Green
    
    # Show log
    Write-Host "`n=== Commit Log ===" -ForegroundColor Cyan
    git log --oneline -1
}

Write-Host "`n=== Next Steps ===" -ForegroundColor Cyan
Write-Host "1. Create repository on GitHub:" -ForegroundColor Yellow
Write-Host "   https://github.com/new" -ForegroundColor White
Write-Host "   - Name: word-of-wisdom" -ForegroundColor White
Write-Host "   - Description: TCP server with PoW-based DDoS protection" -ForegroundColor White
Write-Host "   - Public/Private: your choice" -ForegroundColor White
Write-Host "   - Don't add README, .gitignore, license (already exist)" -ForegroundColor White
Write-Host ""
Write-Host "2. After creating, run:" -ForegroundColor Yellow
Write-Host "   git remote add origin https://github.com/YOUR_USERNAME/word-of-wisdom.git" -ForegroundColor White
Write-Host "   git branch -M main" -ForegroundColor White
Write-Host "   git push -u origin main" -ForegroundColor White

Write-Host "`n✓ Git setup complete!" -ForegroundColor Green

