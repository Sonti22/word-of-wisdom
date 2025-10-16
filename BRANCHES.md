# Branch Strategy

## Feature Branches

- `feat/pow` — PoW implementation (internal/pow)
- `feat/server` — TCP server (internal/server, cmd/server)
- `feat/client` — CLI client (cmd/client)
- `feat/quotes` — Quotes module (internal/quotes)
- `feat/ratelimit` — Rate limiting and adaptive difficulty (internal/ratelimit)
- `chore/docker` — Docker images and compose
- `chore/ci` — CI/CD pipeline
- `docs/readme` — Documentation updates
- `test/integration` — Integration tests

## Commit Message Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Type
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Code style (formatting, no logic change)
- `refactor`: Code refactoring
- `test`: Add or update tests
- `chore`: Maintenance tasks (dependencies, config)

### Scope
- `pow`: PoW logic
- `server`: Server implementation
- `client`: Client implementation
- `quotes`: Quotes module
- `ratelimit`: Rate limiting
- `docker`: Docker/compose
- `ci`: CI/CD
- (empty): global changes

### Examples

```
feat(pow): verify() with leading-zero-bits check

Implement SHA-256 hashcash verification with configurable
difficulty (leading zero bits).

Refs: #1

---

feat(server): TCP handler with deadlines and JSON protocol

Add TCP listener with:
- Connection deadlines (30s total)
- JSON newline-delimited messages
- Challenge generation and solution verification

Refs: #2

---

feat(client): parallel nonce search + solution submit

Implement parallel PoW solver using 8 goroutines with
context-based cancellation.

Refs: #3

---

feat(ratelimit): IP-based token bucket + adaptive difficulty

Add rate limiting (requests/sec per IP) and adaptive PoW
difficulty that increases with repeated attempts.

Refs: #11

---

chore(docker): server & client Dockerfiles + compose

Multi-stage builds with scratch/alpine base images.
docker-compose orchestration with startup delays.

Refs: #7, #8

---

docs: README with PoW rationale and anti-DDoS measures

Explain hashcash choice, security measures (anti-DDoS,
anti-replay), and 5-minute quickstart.

Refs: #10
```

## Pull Request Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] feat: New feature
- [ ] fix: Bug fix
- [ ] docs: Documentation
- [ ] test: Tests
- [ ] chore: Maintenance

## Checklist
- [ ] Code follows gofmt style
- [ ] go vet passes
- [ ] All tests pass (go test -race ./...)
- [ ] Documentation updated
- [ ] Commit messages follow convention
```

