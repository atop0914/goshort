# GoShort — Lightweight URL Shortener

A lightweight, self-hosted URL shortener built with Go. No external dependencies required — just a single binary and you're ready to go.

## Features

| Feature | Description |
|---------|-------------|
| 🔗 **URL Shortening** | Generate compact short codes from long URLs using base62 encoding |
| 📊 **Click Tracking** | Track and display click statistics for each short URL |
| 🌐 **Web UI** | Simple, modern web interface for creating and managing short URLs |
| 📱 **RESTful API** | Full JSON API for programmatic access |
| ⚡ **In-Memory Storage** | No external database required |
| 🎨 **Custom Codes** | Use your own custom short codes (alphanumeric only) |
| ⏰ **Expiration** | Set expiration times for temporary short URLs |
| 🔒 **Rate Limiting** | Built-in rate limiting to prevent abuse |
| 🛡️ **Input Sanitization** | XSS protection and URL validation |
| 🐳 **Docker Ready** | Easy containerized deployment |

## Quick Start

### Download Pre-built Binary

```bash
# Download the latest release for your platform
# https://github.com/yourusername/goshort/releases

# Run directly
./goshort
```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/goshort.git
cd goshort

# Build
go build -o goshort ./cmd/server

# Run
./goshort
```

Visit **http://localhost:8080** to use the web interface.

## Configuration

Create a `config.json` (or `config.yaml`) file:

```json
{
  "host": "0.0.0.0",
  "port": 8080,
  "base_url": "http://localhost:8080",
  "expiry_hours": 720,
  "rate_limit_rate": 10,
  "rate_limit_cap": 20
}
```

### Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `host` | Bind address | `0.0.0.0` |
| `port` | HTTP port | `8080` |
| `base_url` | Base URL for short links | `http://localhost:8080` |
| `expiry_hours` | Default URL expiration (0 = never) | `720` (30 days) |
| `rate_limit_rate` | Requests per second per client | `10` |
| `rate_limit_cap` | Burst capacity for rate limiter | `20` |

### Using Custom Config

```bash
./goshort -config /path/to/config.yaml
```

## API Reference

### Create Short URL

```bash
curl -X POST http://localhost:8080/api/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/very/long/url/path"}'
```

**Response:**
```json
{
  "short_url": "http://localhost:8080/r/abc123",
  "code": "abc123",
  "original_url": "https://example.com/very/long/url/path",
  "created_at": "2026-04-02T01:00:00Z",
  "expires_at": null
}
```

### With Custom Code

```bash
curl -X POST http://localhost:8080/api/shorten \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "custom_code": "my-link"
  }'
```

### With Expiration (hours)

```bash
curl -X POST http://localhost:8080/api/shorten \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "expiry_hours": 24
  }'
```

### List All URLs

```bash
curl http://localhost:8080/api/urls
```

**Response:**
```json
{
  "urls": [
    {
      "code": "abc123",
      "original_url": "https://example.com/very/long/url/path",
      "short_url": "http://localhost:8080/r/abc123",
      "clicks": 42,
      "created_at": "2026-04-02T01:00:00Z",
      "expires_at": null
    }
  ],
  "total": 1
}
```

### Get URL Statistics

```bash
curl http://localhost:8080/api/stats/abc123
```

**Response:**
```json
{
  "code": "abc123",
  "original_url": "https://example.com/very/long/url/path",
  "short_url": "http://localhost:8080/r/abc123",
  "clicks": 42,
  "created_at": "2026-04-02T01:00:00Z",
  "expires_at": null
}
```

### Delete a URL

```bash
curl -X DELETE http://localhost:8080/api/urls/abc123
```

### Redirect

```
GET /r/{code} → 302 redirect to original URL
```

### Health Check

```bash
curl http://localhost:8080/health
# Returns: OK
```

## Error Responses

All API errors follow this format:

```json
{
  "error": "ERROR_CODE",
  "message": "Human-readable error message"
}
```

| Status | Error Code | Description |
|--------|------------|-------------|
| 400 | `INVALID_URL` | URL format is invalid |
| 400 | `MISSING_URL` | URL is required |
| 400 | `INVALID_CODE` | Custom code is invalid |
| 404 | `NOT_FOUND` | Short URL not found |
| 409 | `CODE_EXISTS` | Custom code already in use |
| 429 | `RATE_LIMITED` | Too many requests |
| 410 | `EXPIRED` | Short URL has expired |

## Web UI

The web interface provides:

- **Create short URLs** with optional custom codes
- **View all URLs** with click counts
- **Copy to clipboard** functionality
- **Delete URLs** with confirmation
- **Statistics dashboard** with top URLs

Navigate to:
- `/` — Main page (create and manage URLs)
- `/stats` — Statistics page (view all URLs sorted by clicks)

## Deployment Guide

### Standalone Binary

```bash
# Build
go build -o goshort ./cmd/server

# Create config
cat > config.json << 'EOF'
{
  "host": "0.0.0.0",
  "port": 8080,
  "base_url": "https://short.yourdomain.com",
  "expiry_hours": 720
}
EOF

# Run
./goshort
```

### Systemd Service

```ini
# /etc/systemd/system/goshort.service
[Unit]
Description=GoShort URL Shortener
After=network.target

[Service]
Type=simple
User=goshort
Group=goshort
WorkingDirectory=/opt/goshort
ExecStart=/opt/goshort/goshort -config /opt/goshort/config.json
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable goshort
sudo systemctl start goshort
```

### Docker

#### Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o goshort ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/goshort .
COPY config.json .
EXPOSE 8080
CMD ["./goshort"]
```

#### docker-compose.yml

```yaml
version: '3.8'
services:
  goshort:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./config.json:/root/config.json:ro
    restart: unless-stopped
```

```bash
docker-compose up -d
```

### Reverse Proxy (nginx)

```nginx
server {
    listen 80;
    server_name short.yourdomain.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

For HTTPS, wrap with Certbot or use a managed certificate.

## Project Structure

```
goshort/
├── cmd/
│   └── server/
│       └── main.go           # Entry point
├── internal/
│   ├── handler/
│   │   ├── api.go            # REST API handlers
│   │   └── web.go            # Web UI handlers
│   ├── model/
│   │   └── url.go            # Data models
│   ├── service/
│   │   ├── shortener.go      # Short code generation
│   │   └── shortener_test.go
│   └── store/
│       ├── memory.go         # In-memory storage
│       └── memory_test.go
├── static/
│   └── css/
│       └── style.css         # Web UI styles
├── templates/
│   ├── index.html            # Main page
│   └── stats.html            # Statistics page
├── config/
│   └── config.go             # Configuration loader
├── config.json               # Default configuration
├── go.mod
├── go.sum
├── README.md
└── LICENSE
```

## Technical Details

### Short Code Generation

- Uses base62 encoding (a-z, A-Z, 0-9)
- Default code length: 7 characters
- Collision detection with retry logic
- Thread-safe generation

### Rate Limiting

- Token bucket algorithm
- Per-client IP tracking
- Configurable rate and burst capacity
- Automatic cleanup of stale entries

### Storage

- In-memory map with mutex protection
- Optional Redis interface (interface-based design)
- Automatic expiration check on access

### Security

- URL validation (HTTP/HTTPS only)
- XSS prevention via HTML escaping
- SQL injection prevention (no SQL used)
- Rate limiting for API abuse prevention

## Development

```bash
# Run tests
go test ./...

# Run with hot reload (using air)
air

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o goshort-linux-amd64 ./cmd/server
GOOS=darwin GOARCH=amd64 go build -o goshort-darwin-amd64 ./cmd/server
GOOS=windows GOARCH=amd64 go build -o goshort.exe ./cmd/server
```

## License

MIT License - See [LICENSE](LICENSE) file for details.
