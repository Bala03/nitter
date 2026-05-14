# Nitter

[![License](https://img.shields.io/github/license/zedeus/nitter?style=flat)](#license)

A free and open source alternative Twitter front-end focused on privacy and
performance, rebuilt in **Go** for high performance and easy deployment.

Inspired by the [Invidious](https://github.com/iv-org/invidious) project.

- No JavaScript or ads
- All requests go through the backend, client never talks to Twitter
- Prevents Twitter from tracking your IP or JavaScript fingerprint
- Uses Twitter's unofficial API (no rate limits or developer account required)
- Lightweight pages
- **Enhanced RSS feeds** with full media information and direct download links
- Themes
- Mobile support (responsive design)
- AGPLv3 licensed, no proprietary instances permitted

## RSS Feed Features

The RSS feeds include comprehensive media information for each tweet:

- **Media type detection**: Photos, videos, GIFs, and stickers are all identified
- **Direct download links**: Every media attachment includes a direct download URL
  - Images: Original quality (`?name=orig`) download links
  - Videos: Best quality MP4 direct download, plus all available quality variants
  - GIFs: Direct MP4 download link
- **Media metadata**: Dimensions (width/height), duration, bitrate
- **RSS enclosures**: Standard `<enclosure>` elements for feed reader media playback
- **MRSS support**: `<media:content>` and `<media:group>` elements with full metadata
- **Quoted tweet media**: Media from quoted tweets is also included with download links

### RSS Endpoints

| Endpoint | Description |
|----------|-------------|
| `/@username/rss` | User's timeline feed |
| `/@username/media/rss` | User's media tweets feed |
| `/@username/with_replies/rss` | User's tweets and replies feed |
| `/search/rss?q=query` | Search results feed |

## Installation

### Dependencies

- Go 1.22+
- Redis

### Building

```bash
go build -o nitter .
cp nitter.example.conf nitter.conf
# Edit nitter.conf with your settings
./nitter
```

### Docker

```bash
docker-compose up -d
```

Or build and run manually:

```bash
docker build -t nitter:latest .
docker run -v $(pwd)/nitter.conf:/src/nitter.conf -d --network host nitter:latest
```

### Configuration

Copy `nitter.example.conf` to `nitter.conf` and edit the settings:

- Set your `hostname` for generating correct links
- Set `hmacKey` to a random string for cryptographic signing
- Configure Redis connection if not using defaults
- Set `enableRSS = true` to enable RSS feeds (enabled by default)

## Development

### Running locally

```bash
# Start Redis
redis-server --daemonize yes

# Build and run
go build -o nitter . && ./nitter
```

### Running tests

```bash
go test ./... -v
```

### Project structure

```
├── main.go              # Entry point
├── config/              # Configuration parsing
├── twitter/             # Twitter API client, types, and parser
├── cache/               # Redis caching layer
├── handlers/            # HTTP request handlers
├── rss/                 # RSS feed generation (with full media support)
├── templates/           # HTML templates
├── public/              # Static assets (CSS, fonts, images)
└── nitter.example.conf  # Example configuration
```

## Contact

Feel free to join our [Matrix channel](https://matrix.to/#/#nitter:matrix.org).
You can email me at zedeus@pm.me if you wish to contact me personally.

## License

AGPLv3
