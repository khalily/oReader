# oReader v2.0.0

A modern RSS reader built with Go (backend) and React (frontend), featuring a clean interface and powerful feed management capabilities.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)
![React Version](https://img.shields.io/badge/React-18-61DAFB?logo=react)

## Features

- 🔐 **Secure Authentication**: JWT-based auth with refresh tokens and CSRF protection
- 📰 **RSS/Atom Feed Support**: Subscribe to any RSS or Atom feed
- 🌍 **Multi-User Support**: Each user has their own feed subscriptions and reading state
- ⭐ **Star & Save**: Mark articles as starred to read later
- 📖 **Read/Unread Tracking**: Automatic read tracking with manual mark as read options
- 📥 **OPML Import/Export**: Easily migrate your feeds from other readers
- 🔄 **Auto Refresh**: Background refresh keeps your feeds up to date
- 🚀 **Fast & Responsive**: Built with Go and React for optimal performance
- 🎨 **Modern UI**: Clean, responsive interface with dark mode support
- 🔒 **Security First**: Comprehensive security measures including SSRF protection, input sanitization, and rate limiting

## Tech Stack

### Backend
- **Language**: Go 1.25+
- **Framework**: Gin Web Framework
- **Database**: SQLite (development), MySQL (production)
- **ORM**: GORM
- **Authentication**: JWT (HS256) with dual-token system
- **Security**: bcrypt password hashing, CSRF protection, rate limiting
- **Logging**: zerolog structured logging

### Frontend
- **Framework**: React 18 with TypeScript
- **Build Tool**: Vite
- **Styling**: Tailwind CSS
- **State Management**: Zustand
- **HTTP Client**: Axios with interceptors
- **UI Components**: shadcn/ui
- **Routing**: React Router

## Quick Start

### Prerequisites

- Go 1.25 or higher
- Node.js 18+ and npm
- SQLite (for development) or MySQL (for production)

### Installation

1. **Clone the repository**:
```bash
git clone https://github.com/yourusername/oreader.git
cd oreader
```

2. **Install backend dependencies**:
```bash
go mod download
```

3. **Install frontend dependencies**:
```bash
cd web
npm install
```

4. **Configure environment variables**:
```bash
cp .env.example .env
# Edit .env with your configuration
```

5. **Run database migrations**:
```bash
make migrate-up
```

6. **Build the frontend**:
```bash
make frontend-build
```

7. **Run the application**:
```bash
# Option 1: Using the run script (recommended)
chmod +x run.sh
./run.sh

# Option 2: Setting environment variables manually
DATABASE_URL=oreader.db JWT_SECRET_KEY=dev-secret-key-min-32-chars make run

# Option 3: Export and run
export DATABASE_URL=oreader.db
export JWT_SECRET_KEY=dev-secret-key-min-32-chars
make run
```

The application will be available at `http://localhost:8080`

**Required Environment Variables**:
- `DATABASE_URL`: Database connection string (e.g., `oreader.db` for SQLite)
- `JWT_SECRET_KEY`: Secret key for JWT signing (minimum 32 characters)

**Optional Environment Variables**:
- `ENV`: Environment mode (`development` or `production`, default: `development`)
- `PORT`: Server port (default: `8080`)
- `LOG_LEVEL`: Logging level (`debug`, `info`, `warn`, `error`, default: `info`)

## Development

### Development Server

For development with hot-reload:

1. **Start backend** (in one terminal):
```bash
make run
```

2. **Start frontend** (in another terminal):
```bash
cd web
npm run dev
```

The frontend will be available at `http://localhost:5173` and proxy API requests to the backend.

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
go test -cover ./internal/...

# Run specific package tests
go test ./internal/service

# Run tests without race detection
make test-short
```

### Building for Production

```bash
# Build the complete binary with embedded frontend
make build

# The binary will be at ./build/oreader
./build/oreader
```

### Database Migrations

```bash
# Run migrations up
make migrate-up

# Run migrations down
make migrate-down

# Create a new migration
make migrate-create name=add_new_field
```

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `OREADER_CONFIG` | Config mode (devlopment/testing/production) | `devlopment` | Yes |
| `DATABASE_URL` | Database connection string | `oreader.db` | Yes |
| `JWT_SECRET_KEY` | JWT signing secret (min 32 chars) | - | Yes |
| `REFRESH_TOKEN_DAYS` | Refresh token expiry in days | `7` | No |
| `REFRESH_INTERVAL` | Feed refresh interval | `1h` | No |
| `MAX_CONCURRENT_REFRESH` | Max concurrent feed refreshes | `10` | No |
| `SERVER_PORT` | Server port | `8080` | No |
| `SERVER_HOST` | Server host | `0.0.0.0` | No |
| `LOG_LEVEL` | Log level (debug/info/warn/error) | `info` | No |
| `GITHUB_CLIENT_ID` | GitHub OAuth client ID | - | No |
| `GITHUB_CLIENT_SECRET` | GitHub OAuth client secret | - | No |

See `.env.example` for all available options.

### Production Deployment

For production deployment, ensure:

1. Set `OREADER_CONFIG=production`
2. Use MySQL database with `DATABASE_URL` in format: `user:password@tcp(host:port)/dbname`
3. Set a strong `JWT_SECRET_KEY` (min 32 characters)
4. Enable HTTPS/TLS
5. Configure proper CORS origins
6. Set up proper logging and monitoring

## Docker Deployment

### Using Docker Compose (Development)

```bash
docker-compose up
```

### Using Docker Compose (Production)

```bash
docker-compose -f docker-compose.prod.yml up -d
```

### Building Docker Image

```bash
make docker-build
```

### Running Docker Container

```bash
make docker-run
```

## API Documentation

See [API.md](docs/API.md) for detailed API documentation.

## Security

See [SECURITY_REVIEW.md](docs/SECURITY_REVIEW.md) for comprehensive security review.

### Key Security Features

- **Authentication**: JWT with dual-token system (access + refresh tokens)
- **CSRF Protection**: Double-submit cookie pattern for state-changing requests
- **SSRF Protection**: Blocks private IPs and validates URL schemes
- **Input Sanitization**: XSS protection with bluemonday
- **Rate Limiting**: Token bucket algorithm with configurable limits
- **Password Security**: bcrypt hashing with cost factor 12
- **Security Headers**: X-Frame-Options, X-XSS-Protection, HSTS, CSP
- **Cookie Security**: HttpOnly, Secure, SameSite=Strict

## Project Structure

```
oreader/
├── cmd/
│   └── server/          # Application entry point
├── internal/
│   ├── config/          # Configuration management
│   ├── handler/         # HTTP request handlers
│   ├── infra/           # Infrastructure code (JWT, CSRF, etc.)
│   ├── middleware/      # HTTP middleware
│   ├── model/           # Data models
│   ├── repository/      # Database repositories
│   ├── service/         # Business logic
│   └── worker/          # Background workers
├── web/                 # React frontend
├── migrations/          # Database migrations
├── docs/                # Documentation
├── Makefile             # Build commands
├── go.mod               # Go module definition
└── .env.example         # Environment variables template
```

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow Test-Driven Development (TDD): write tests first, then implement
- Write meaningful commit messages
- Add tests for new features
- Ensure all tests pass before submitting PR
- Follow Go and React best practices
- Update documentation as needed

## Troubleshooting

### Common Issues

**Issue**: Database migration fails
- **Solution**: Ensure `DATABASE_URL` is correctly set and database exists

**Issue**: Frontend build fails
- **Solution**: Run `npm install` in the `web/` directory

**Issue**: JWT secret key error
- **Solution**: Set `JWT_SECRET_KEY` to a value with at least 32 characters

**Issue**: Go version mismatch
- **Solution**: Ensure you're using Go 1.25 or higher (`go version`)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [Gin](https://gin-gonic.com/) web framework
- Frontend built with [React](https://react.dev/) and [Vite](https://vitejs.dev/)
- UI components from [shadcn/ui](https://ui.shadcn.com/)
- RSS parsing powered by [gofeed](https://github.com/mmcdole/gofeed)

## Support

- 📖 Documentation: [docs/](docs/)
- 🐛 Bug Reports: [GitHub Issues](https://github.com/yourusername/oreader/issues)
- 💡 Feature Requests: [GitHub Issues](https://github.com/yourusername/oreader/issues)
- 📧 Email: support@example.com

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history and changes.

---

**Made with ❤️ by the oReader team**
