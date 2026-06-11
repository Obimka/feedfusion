# FeedFusion

![Go](https://img.shields.io/badge/go-1.25-blue.svg)
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

## Text-to-Speech (TTS)

FeedFusion integrates with Mistral AI's TTS API to convert text to speech.

### Enable TTS

**Option 1: Environment Variable**
```bash
export MISTRAL_API_KEY=your-mistral-api-key
./bin/feedfusion
```

**Option 2: Configuration File**
Add to `data/config.yaml`:
```yaml
server_port: 8081
mistral_api_key: "your-mistral-api-key"
feeds:
  - url: https://...
```

**Option 3: Docker**
```bash
docker run -e MISTRAL_API_KEY=your-key -p 8081:8081 feedfusion
```

### TTS API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/tts/status` | Check if TTS is configured and enabled |
| POST | `/api/tts/speak` | Generate speech from text |
| GET | `/api/tts/voices` | List available voices |
| GET | `/api/tts/models` | List available models |

**Note**: TTS endpoints do not require authentication. They only need the Mistral API key configured in `data/config.yaml` or via `MISTRAL_API_KEY` environment variable.

### Generate Speech

```bash
curl -X POST http://localhost:8081/api/tts/speak \
  -H "Content-Type: application/json" \
  -d '{"text": "Hello world", "voice": "fr_FR", "model": "mistral-small"}' \
  --output speech.wav
```

**Request Body:**
```json
{
  "text": "Text to convert to speech",
  "voice": "fr_FR",  // Optional, default: fr_FR
  "model": "mistral-small"  // Optional, default: mistral-small
}
```

**Available Voices:** `fr_FR`, `en_US`, `en_GB`, `de_DE`, `es_ES`, `it_IT`, `pt_PT`, `nl_NL`, `pl_PL`

**Available Models:** `mistral-small`, `mistral-medium`

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

## User Management

FeedFusion now supports multi-user management. Each user has their own feeds and items.

### Features
- Each user can only see and manage their own feeds
- Admin users can manage all users
- Public registration can be enabled or disabled
- Password change functionality

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ALLOW_REGISTRATION` | false | Enable public user registration |

### Configuration via YAML

Add to `data/config.yaml`:
```yaml
allow_registration: true  # Enable public registration
```

### API Endpoints

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/api/register` | Register a new user | Public (if enabled) |
| GET | `/api/users` | List all users | Admin |
| POST | `/api/users` | Create a new user | Admin |
| GET | `/api/users/{id}` | Get user details | Admin or Self |
| PUT | `/api/users/{id}` | Update user | Admin or Self |
| DELETE | `/api/users/{id}` | Delete user | Admin |
| POST | `/api/users/{id}/password` | Change password | Admin or Self |

### Register a new user

```bash
curl -X POST http://localhost:8081/api/register \
  -H "Content-Type: application/json" \
  -d '{"username": "newuser", "password": "mypassword"}'
```

### Create a user (admin only)

```bash
curl -X POST http://localhost:8081/api/users \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"username": "newuser", "password": "mypassword", "is_admin": false}'
```

### List all users (admin only)

```bash
curl -X GET http://localhost:8081/api/users \
  -H "Authorization: Bearer <admin-token>"
```

### Get user details

```bash
curl -X GET http://localhost:8081/api/users/1 \
  -H "Authorization: Bearer <token>"
```

### Update user

```bash
curl -X PUT http://localhost:8081/api/users/1 \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"username": "updatedname", "is_admin": true}'
```

### Change password

```bash
curl -X POST http://localhost:8081/api/users/1/password \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"current_password": "oldpassword", "new_password": "newpassword"}'
```

**Note**: When changing your own password, you must provide `current_password`. Admin users can change other users' passwords without providing the current password.

### Delete user (admin only)

```bash
curl -X DELETE http://localhost:8081/api/users/1 \
  -H "Authorization: Bearer <admin-token>"
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
