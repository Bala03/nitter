# AGENTS.md

## Cursor Cloud specific instructions

### Overview

Nitter is a Go-based alternative Twitter/X front-end focused on privacy. It proxies requests to Twitter's unofficial API and serves content without JavaScript or ads. The RSS feeds are enhanced to include full media data and direct download links.

### Required services

| Service | How to start |
|---------|-------------|
| Redis | `redis-server --daemonize yes` |
| Nitter | `./nitter` (runs on port 8080) |

### Build commands

```bash
go build -o nitter .    # Compile the binary
go test ./... -v        # Run tests
go vet ./...            # Lint
```

### Running

1. Ensure Redis is running: `redis-cli ping` should return `PONG`
2. Copy config if needed: `cp nitter.example.conf nitter.conf`
3. Build: `go build -o nitter .`
4. Run: `./nitter`

The server listens on `http://localhost:8080` by default.

### Testing

```bash
go test ./... -v
```

Tests cover RSS feed generation with photos, videos, GIFs, quoted tweets, and the config parser. No external services needed for tests.

### Key gotchas

- **Go version**: Uses Go 1.22+ (system Go on the VM).
- **Redis required**: The app exits immediately if it cannot connect to Redis.
- **Twitter API tokens**: Nitter relies on Twitter guest tokens which Twitter/X has been restricting. Token fetch failures are expected and not indicative of a local environment issue.
- **Config**: `nitter.conf` is gitignored. Always regenerate from `nitter.example.conf` if missing.
- **Templates**: HTML templates are in `templates/` and loaded at startup. Changes require rebuild.
- **Static assets**: CSS/fonts from `public/` are served directly. The existing Nim-generated CSS still works.
