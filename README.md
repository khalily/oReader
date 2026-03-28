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
- 📄 **Paper Import**: Upload PDF academic papers, auto-convert to Markdown with MinerU + LLM refinement, extract metadata (title, authors, DOI, keywords)
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
- **Paper Converter**: Python gRPC service (MinerU for PDF parsing, OpenAI-compatible LLM for metadata extraction)

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
- Python 3.10+ and pip (optional, for Paper converter service)

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

4. **Install converter dependencies** (optional, for Paper Import):
```bash
pip install -r converter/requirements.txt
```

5. **Run the application**:
```bash
# One-click dev environment (backend + frontend + converter)
chmod +x dev.sh
./dev.sh

# Or API-only mode
chmod +x run.sh
./run.sh
```

The application will be available at `http://localhost:8080`

**Required Environment Variables**:
- `DATABASE_URL`: Database connection string (e.g., `oreader.db` for SQLite)
- `JWT_SECRET_KEY`: Secret key for JWT signing (minimum 32 characters)

**Note**: The application automatically runs database migrations on startup.

**Optional Environment Variables**:
- `ENV`: Environment mode (`development` or `production`, default: `development`)
- `PORT`: Server port (default: `8080`)
- `LOG_LEVEL`: Logging level (`debug`, `info`, `warn`, `error`, default: `info`)

## Development

### Development Server

**One-click full development environment** (backend + frontend + converter):
```bash
chmod +x dev.sh
./dev.sh
```

Or start services individually:

1. **Start backend** (in one terminal):
```bash
# API-only mode
./run.sh

# Or set environment variables and run
DATABASE_URL=oreader.db JWT_SECRET_KEY=dev-secret-key-min-32-chars make run
```

2. **Start frontend** (in another terminal):
```bash
cd web
npm run dev
```

3. **Start converter service** (optional, for Paper Import — in another terminal):
```bash
cd converter
python server.py
```

The converter service runs on `localhost:50051` by default. Without it, paper upload will return an error.

The frontend will be available at `http://localhost:5173` and proxy API requests to the backend at `http://localhost:8080`.

**Required Environment Variables**:
- `DATABASE_URL`: Database file path (e.g., `oreader.db`)
- `JWT_SECRET_KEY`: Secret key for JWT signing (minimum 32 characters)

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

### Running Converter Tests

```bash
cd converter
python -m pytest tests/ -v
```

### Running All Tests

```bash
# Run Go + Frontend + Converter tests together
make test-all

# Or use the convenience script
./test-all.sh
```

### Running E2E Tests

End-to-end API tests use Playwright to validate the full request/response cycle:

```bash
# Install Playwright browsers (first time only)
cd web
npx playwright install chromium

# Run all E2E tests
npm run test:e2e

# Run tests with UI mode
npm run test:e2e:ui

# Run specific test file
npx playwright test --config=tests/e2e/playwright.config.ts tests/e2e/auth.spec.ts
```

See [web/tests/e2e/README.md](web/tests/e2e/README.md) for detailed E2E testing documentation.


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
| `DATABASE_URL` | Database connection string | `oreader.db` | Yes |
| `JWT_SECRET_KEY` | JWT signing secret (min 32 chars) | - | Yes |
| `ENV` | Environment mode (development/production) | `development` | No |
| `PORT` | Server port | `8080` | No |
| `LOG_LEVEL` | Log level (debug/info/warn/error) | `info` | No |
| `JWT_ACCESS_TTL` | Access token lifetime | `15m` | No |
| `JWT_REFRESH_TTL` | Refresh token lifetime | `168h` (7 days) | No |
| `REFRESH_INTERVAL` | Feed refresh interval | `15m` | No |
| `RATE_LIMIT_ENABLED` | Enable rate limiting | `true` | No |
| `GITHUB_CLIENT_ID` | GitHub OAuth client ID | - | No |
| `GITHUB_CLIENT_SECRET` | GitHub OAuth client secret | - | No |
| `PAPER_GRPC_ADDR` | Paper converter gRPC service address | `localhost:50051` | No |
| `PAPER_UPLOAD_DIR` | Directory for uploaded PDF files | `uploads/papers` | No |
| `PAPER_MAX_UPLOAD_SIZE` | Maximum PDF upload size in bytes | `52428800` (50MB) | No |
| `PAPER_GRPC_TIMEOUT` | gRPC client timeout | `5m` | No |

See `.env.example` for all available options.

**Converter Service Environment Variables** (set in the converter's environment):

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `LLM_API_KEY` | API key for metadata extraction | - | No* |
| `LLM_BASE_URL` | LLM API base URL | `https://api.openai.com/v1` | No |
| `LLM_MODEL` | LLM model name | `gpt-4o-mini` | No |
| `GRPC_PORT` | gRPC server listen port | `50051` | No |

\* Without `LLM_API_KEY`, the converter skips metadata extraction and LLM refinement, but PDF-to-Markdown conversion still works via MinerU.

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
├── converter/           # Python gRPC Paper converter service
│   ├── server.py        # gRPC server entry point
│   ├── converter.py     # PDF conversion & LLM metadata extraction
│   ├── requirements.txt # Python dependencies
│   ├── proto/           # Protobuf definitions & generated code
│   └── tests/           # Converter unit tests
├── internal/
│   ├── config/          # Configuration management
│   ├── handler/         # HTTP request handlers (feeds, items, papers)
│   ├── infra/
│   │   ├── grpc/        # gRPC client for paper converter
│   │   └── markdown/    # Markdown conversion infrastructure
│   ├── middleware/       # Auth, CSRF, logging middleware
│   ├── model/           # Domain models (User, Feed, Item, Paper)
│   ├── repository/      # Database repositories (including paper_repository)
│   ├── service/         # Business logic (including paper_service)
│   └── worker/          # Background feed refresh worker
├── web/                 # React frontend
│   └── src/
│       ├── components/papers/  # Paper UI components
│       ├── pages/papers/       # Paper pages
│       └── hooks/              # Custom hooks (useAuth, useFeeds, useItems, usePapers)
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

**Issue**: Paper upload returns "converter service not available"
- **Solution**: Start the converter service: `cd converter && python server.py`

**Issue**: Paper conversion fails with "MinerU conversion failed"
- **Solution**: Ensure MinerU is installed: `pip install -r converter/requirements.txt`

**Issue**: Paper metadata is empty after conversion
- **Solution**: Set `LLM_API_KEY` in the converter's environment. Without it, metadata extraction is skipped.

**Issue**: Converter gRPC connection refused
- **Solution**: Verify the converter service is running and `PAPER_GRPC_ADDR` matches (default: `localhost:50051`)

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
