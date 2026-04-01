# GoShort

A lightweight, self-hosted URL shortener built with Go.

## Features

- 🔗 Short URL generation with base62 encoding
- 📊 Click tracking and statistics
- 🌐 Simple web UI
- 📱 RESTful API
- ⚡ In-memory storage (no external database required)
- 🎨 Clean, modern design

## Quick Start

```bash
# Clone and enter directory
cd goshort

# Build
go build -o goshort ./cmd/server

# Run
./goshort
```

Visit http://localhost:8080 to use the web interface.

## API Usage

### Create a short URL
```bash
curl -X POST http://localhost:8080/api/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://github.com"}'
```

### List all URLs
```bash
curl http://localhost:8080/api/urls
```

### Get stats for a URL
```bash
curl http://localhost:8080/api/stats/abc123
```

### Delete a URL
```bash
curl -X DELETE http://localhost:8080/api/urls/abc123
```

## Configuration

Edit `config.json`:

```json
{
  "host": "0.0.0.0",
  "port": 8080,
  "base_url": "http://localhost:8080",
  "expiry_hours": 720
}
```

## Project Structure

```
goshort/
├── cmd/server/main.go     # Entry point
├── internal/
│   ├── handler/          # HTTP handlers
│   ├── model/            # Data models
│   ├── service/          # Business logic
│   └── store/            # Storage
├── templates/            # HTML templates
├── static/css/           # Stylesheets
├── config/               # Configuration
└── config.json           # Config file
```

## License

MIT
