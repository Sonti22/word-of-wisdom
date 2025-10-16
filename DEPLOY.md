# Deployment Guide

## 🚀 Quick Deploy to GitHub

### Option 1: Automated Script (Recommended)

```powershell
# Run the setup script
.\scripts\git_setup.ps1
```

This will:
1. Initialize Git repository
2. Configure commit template
3. Check/set user credentials
4. Create initial commit
5. Show next steps

---

### Option 2: Manual Setup

```powershell
# 1. Initialize Git
git init

# 2. Configure commit template
git config commit.template .gitmessage

# 3. Set user credentials (if not set globally)
git config user.name "YOUR_USERNAME"
git config user.email "your-email@example.com"

# 4. Add all files
git add .

# 5. Create initial commit
git commit -m "feat(all): initial implementation - PoW TCP server"

# 6. Create GitHub repository at https://github.com/new
#    - Name: word-of-wisdom
#    - Description: TCP server with PoW-based DDoS protection (Hashcash + Rate Limiting)
#    - Public/Private: your choice
#    - Don't add README, .gitignore, license

# 7. Add remote (replace YOUR_USERNAME)
git remote add origin https://github.com/YOUR_USERNAME/word-of-wisdom.git

# 8. Push to GitHub
git branch -M main
git push -u origin main
```

---

## 📦 GitHub Repository Settings

### Description
```
TCP server with PoW-based DDoS protection (Hashcash + Rate Limiting + Adaptive Difficulty)
```

### Topics (tags)
```
go golang tcp proof-of-work hashcash ddos-protection rate-limiting 
adaptive-difficulty docker docker-compose tcp-server security
```

### About
```
🛡️ Production-ready TCP server protected by Proof-of-Work (Hashcash)
- SHA-256 with configurable difficulty
- IP-based rate limiting (token bucket)
- Adaptive difficulty scaling
- Anti-DDoS/replay protection
- Full test coverage (31 unit + 5 e2e)
- Docker ready (7MB images)
```

---

## 🔒 Production Deployment

### Docker Compose (Simple)

```bash
# Clone repository
git clone https://github.com/YOUR_USERNAME/word-of-wisdom.git
cd word-of-wisdom

# Run with rate limiting + adaptive difficulty
WOW_RATE_LIMIT=10 WOW_ADAPTIVE_BITS=true docker-compose up -d

# Check logs
docker-compose logs -f
```

### Docker (Custom)

```bash
# Build images
docker build -f Dockerfile.server -t wow-server:latest .
docker build -f Dockerfile.client -t wow-client:latest .

# Run server
docker run -d \
  --name wow-server \
  -p 8080:8080 \
  -e WOW_BITS=22 \
  -e WOW_RATE_LIMIT=10 \
  -e WOW_ADAPTIVE_BITS=true \
  wow-server:latest

# Run client
docker run --rm \
  --name wow-client \
  --link wow-server:server \
  -e WOW_ADDR=server:8080 \
  wow-client:latest
```

### Kubernetes (Advanced)

```bash
# Coming soon: Helm charts
helm install wow-server ./charts/wow-server
```

---

## 🔑 Environment Variables (Production)

```bash
# Server Configuration
export WOW_ADDR=:8080              # Listen address
export WOW_BITS=22                 # Base PoW difficulty (production)
export WOW_EXPIRES=60              # Challenge TTL (seconds)
export WOW_RATE_LIMIT=10           # Max requests/sec per IP
export WOW_ADAPTIVE_BITS=true      # Enable adaptive difficulty

# Client Configuration
export WOW_ADDR=server:8080        # Server address
```

---

## 📊 Monitoring

### Prometheus Metrics (Future)

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'wow-server'
    static_configs:
      - targets: ['localhost:8080']
```

### Grafana Dashboard (Future)

- Request rate per IP
- PoW solve time distribution
- Rate limit hits
- Adaptive difficulty events

---

## 🔐 Security Checklist

- [x] PoW enabled (SHA-256 hashcash)
- [x] Rate limiting configured (per-IP)
- [x] Adaptive difficulty enabled
- [x] Connection timeouts set (30s)
- [x] Challenge TTL configured
- [x] Salt randomness (crypto/rand)
- [ ] TLS/SSL enabled (add nginx reverse proxy)
- [ ] IP whitelist for trusted clients
- [ ] Metrics endpoint secured
- [ ] DDoS mitigation (CloudFlare/AWS Shield)

---

## 🧪 Testing in Production

```bash
# Simple health check
echo "" | nc localhost 8080

# Load test (requires hey/vegeta)
hey -n 1000 -c 10 -m GET http://localhost:8080

# Rate limit test
for i in {1..20}; do
  echo "Request $i"
  nc localhost 8080 < /dev/null
  sleep 0.1
done
```

---

## 📚 Further Reading

- [README.md](README.md) - Full documentation
- [BRANCHES.md](BRANCHES.md) - Git workflow
- [.gitmessage](.gitmessage) - Commit template
- [Hashcash (Wikipedia)](https://en.wikipedia.org/wiki/Hashcash)
- [Token Bucket Algorithm](https://en.wikipedia.org/wiki/Token_bucket)

---

## 🤝 Contributing

See [BRANCHES.md](BRANCHES.md) for branch strategy and commit conventions.

```bash
# Create feature branch
git checkout -b feat/your-feature

# Make changes and commit
git commit -m "feat(scope): description"

# Push and create PR
git push origin feat/your-feature
```

---

## 📄 License

MIT

