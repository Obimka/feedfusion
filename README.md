# FeedFusion

![Go](https://img.shields.io/badge/go-1.21-blue.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/obimka/feedfusion)](https://goreportcard.com/report/github.com/obimka/feedfusion)

**FeedFusion** is a self-hosted RSS/Atom feed aggregator with WebSub support for real-time updates.

## Features

- **Feed Aggregation**: Combine multiple RSS/Atom feeds into a single stream
- **WebSub Support**: Real-time updates via WebSub (PubSubHubbub) for supported publishers
- **SQLite Storage**: Lightweight, file-based database (no server required)
- **Web Interface**: Simple HTTP server to view and manage feeds
- **YAML Configuration**: Easy configuration of feeds and weather cities
- **Feed Filtering**: Include/exclude specific feeds from fusion
- **Read/Unread Tracking**: Mark items as read
- **JWT Authentication**: Secure API access with JSON Web Tokens
- **User Management**: Login with predefined users

## Installation

### From Source

```bash
git clone https://github.com/obimka/feedfusion.git
cd feedfusion
go build -o feedfusion
```

### Using Make

```bash
git clone https://github.com/obimka/feedfusion.git
cd feedfusion
make build
```

The binary will be created in the `bin/` directory.

## Quick Start

1. **Run the server**:
   ```bash
   ./bin/feedfusion
   ```
   Or with Make:
   ```bash
   make run
   ```

2. **Access the web interface**: Open http://localhost:8081 in your browser

3. **Add feeds**: Send a POST request to `/api/feeds` with the feed URL:
   ```bash
   curl -X POST http://localhost:8081/api/feeds -d '{"url":"https://example.com/feed.xml"}'
   ```

## Configuration

Create a `data/config.yaml` file to configure default feeds, weather cities, and server settings:

```yaml
server_port: 8081

feeds:
  - url: https://example.com/feed.xml
    title: Example Feed
    include: true
  - url: https://blog.example.com/rss
    title: Example Blog
    include: true

weather_cities:
  - name: Paris
    lat: 48.8566
    lon: 2.3522
    timezone: Europe/Paris
  - name: London
    lat: 51.5074
    lon: -0.1278
    timezone: Europe/London
```

The application will automatically create the `data/` directory if it doesn't exist.

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/` | Web interface |
| GET | `/api/feeds` | List all feeds |
| POST | `/api/feeds` | Add a new feed |
| GET | `/api/items` | List all items (with pagination) |
| GET | `/api/items/unread` | List unread items |
| POST | `/api/items/{id}/read` | Mark item as read |
| POST | `/api/items/{id}/unread` | Mark item as unread |
| DELETE | `/api/feeds/{id}` | Delete a feed |
| POST | `/api/feeds/{id}/toggle` | Toggle feed include status |
| GET | `/websub/callback` | WebSub callback endpoint |

## Project Structure

```
feedfusion/
├── bin/               # Compiled binary
├── data/              # Configuration and data files (gitignored)
│   └── config.yaml    # YAML configuration
├── internal/
│   ├── config/        # Configuration loading
│   ├── handlers/      # HTTP route handlers
│   ├── models/        # Data models (Feed, Item)
│   ├── parser/        # RSS/Atom feed parsing
│   ├── storage/       # Database operations
│   ├── websub/        # WebSub subscription management
│   └── worker/        # Background feed fetching
├── Makefile           # Build targets
├── go.mod             # Go module definition
├── go.sum             # Go dependencies checksum
└── main.go            # Application entry point
```

## Development

### Build

```bash
make build
```

### Test

```bash
make test
```

Or run all tests:
```bash
go test ./...
```

### Run with hot reload

```bash
make watch
```
(Requires [modd](https://github.com/canthefason/go-modd) for file watching)

## Authentication

FeedFusion uses JWT (JSON Web Tokens) for secure authentication.

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | random | Secret key for JWT signing (REQUIRED in production) |
| `ADMIN_USERNAME` | admin | Default admin username |
| `ADMIN_PASSWORD` | admin123 | Default admin password (CHANGE THIS!) |

**IMPORTANT**: On first run, if no users exist, a default admin user is created. If `ADMIN_USERNAME` and `ADMIN_PASSWORD` are not set, the credentials will be **admin/admin123**. Change this immediately!

### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/login` | Login and get JWT token |
| POST | `/api/refresh` | Refresh access token with refresh token |
| POST | `/api/logout` | Logout (clears cookie) |

### Login

```bash
curl -X POST http://localhost:8081/api/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "admin123"}'
```

Response:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "username": "admin",
  "is_admin": "true"
}
```

### Use the token

Include the token in the Authorization header:

```bash
curl http://localhost:8081/api/feeds \
  -H "Authorization: Bearer <your-token>"
```

Or use the cookie (set automatically on login for web interface).

### Refresh token

When your access token expires, use the refresh token to get a new one:

```bash
curl -X POST http://localhost:8081/api/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "<your-refresh-token>"}'
```

## Configuration via Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8081 | HTTP server port |
| `DB_PATH` | rss.db | SQLite database path |
| `CONFIG_PATH` | data/config.yaml | Configuration file path |

## WebSub Support

FeedFusion automatically discovers and subscribes to WebSub hubs for supported publishers:

- lemonde.fr
- lefigaro.fr
- reddit.com
- bbc.com
- nytimes.com
- washingtonpost.com
- theguardian.com
- wordpress.com
- blogspot.com
- medium.com
- dev.to

When a feed supports WebSub, updates are received in real-time instead of waiting for the next fetch cycle.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Acknowledgments

- [gofeed](https://github.com/mmcdole/gofeed) - RSS/Atom feed parsing
- [gorilla/mux](https://github.com/gorilla/mux) - HTTP router
- [gorm](https://gorm.io/) - ORM for SQLite
