# AGENTS.md

## Cursor Cloud specific instructions

### Overview

Nitter is a Nim-language alternative Twitter/X front-end. It uses the Jester web framework, Redis for caching, and proxies requests to Twitter's unofficial API.

### Required services

| Service | How to start |
|---------|-------------|
| Redis | `redis-server --daemonize yes` |
| Nitter | `./nitter` (runs on port 8080) |

### Build commands

```bash
nimble build -d:release -Y   # Compile the binary
nimble scss -Y                # Generate CSS from SCSS
nimble md -Y                  # Render markdown to HTML
```

### Running

1. Ensure Redis is running: `redis-cli ping` should return `PONG`
2. Copy config if needed: `cp nitter.example.conf nitter.conf`
3. Run: `./nitter`

The server listens on `http://localhost:8080` by default.

### Testing

Tests use Python seleniumbase + pytest with Chrome/ChromeDriver in headless mode:

```bash
python3 -m pytest tests/ -n4 --headless
```

**Important**: Tests will fail if Twitter's guest token API is unreachable (returns 404). This is an external dependency issue, not a code bug. The test infrastructure (pytest, seleniumbase, chromedriver) works correctly regardless.

### Key gotchas

- **Nim version**: Use Nim 1.6.x (the CI uses `1.x`). Nim 2.x has package resolution issues with this project's nimble dependencies.
- **PATH**: Nimble binaries are at `/home/ubuntu/.nimble/bin` — already added to `~/.bashrc`.
- **Twitter API tokens**: Nitter relies on Twitter guest tokens which Twitter/X has been restricting. Token fetch failures (`[tokens] fetching token failed: 404`) are expected and not indicative of a local environment issue.
- **Config**: `nitter.conf` is gitignored. Always regenerate from `nitter.example.conf` if missing.
- **libpcre**: Nim needs `libpcre3` (v1), not just `libpcre2`. Install `libpcre3-dev`.
