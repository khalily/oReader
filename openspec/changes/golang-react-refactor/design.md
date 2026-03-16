## Context

The current oReader is a Flask + AngularJS 1.x single-page application for RSS reading. This design document outlines the architecture for a complete rewrite using Go (backend) and React (frontend).

### Current State
- Flask backend with Flask-RESTful for API
- SQLAlchemy ORM with SQLite/PostgreSQL
- Custom lxml-based RSS parser
- AngularJS 1.x frontend with angular-route, $resource
- HTTP Basic Auth with itsdangerous token
- Heroku deployment with gunicorn

### Constraints
- Team proficiency: Go > Python, learning React
- Single binary deployment preferred
- Production uses MySQL, development uses SQLite
- Must support multi-tenant (multiple users)

## Goals / Non-Goals

**Goals:**
- Modern, maintainable technology stack
- Production-grade security (dual-token, HttpOnly cookies, CSRF protection)
- Single binary deployment with embedded frontend
- Scalable architecture for future features
- Automatic RSS refresh with background worker
- GitHub OAuth integration (reserved interface)

**Non-Goals:**
- Data migration from old system (fresh start with OPML migration path)
- Real-time updates (WebSocket) - can be added later
- Mobile app - web-first approach
- Full-text search - can be added later
- Social features (sharing, comments)

**Development Methodology:**
- Test-Driven Development (TDD): Write failing tests first, then implement to pass
- Git commits: Commit after each completed phase with descriptive messages

## Decisions

### 1. Backend Framework: Gin

**Choice:** Gin over Echo, Fiber, stdlib

**Rationale:**
- Mature ecosystem with extensive middleware
- Performance comparable to alternatives
- Familiar Flask-like routing patterns
- Good documentation and community

**Alternatives Considered:**
- Echo: Slightly less middleware ecosystem
- Fiber: Express-like but uses fasthttp (compatibility concerns)
- stdlib: Too much boilerplate for this project size

### 2. ORM: GORM

**Choice:** GORM over sqlx, ent

**Rationale:**
- Similar patterns to SQLAlchemy (familiar transition)
- Auto-migration support for development
- Good MySQL and SQLite support
- Active community

**Alternatives Considered:**
- sqlx: More control but more boilerplate
- ent: Facebook's ORM, steeper learning curve

### 3. Authentication: Dual-Token with HttpOnly Cookies + CSRF Protection

**Choice:** Access Token (15min) + Refresh Token (7day) in HttpOnly + SameSite=Strict cookies, with explicit CSRF token protection

**Rationale:**
- Short-lived access tokens limit attack window
- Refresh tokens stored in database can be revoked
- HttpOnly cookies prevent XSS token theft
- SameSite=Strict provides strong CSRF protection
- **Explicit CSRF tokens** as defense-in-depth (not all browsers support SameSite)

**Flow:**
```
Login → Set-Cookie: access_token (15min, Path=/, HttpOnly, SameSite=Strict)
       → Set-Cookie: refresh_token (7day, Path=/api/v1/auth/refresh, HttpOnly, SameSite=Strict)
       → Set-Cookie: csrf_token (15min, Path=/, HttpOnly=false) [for JS to read]
       → Response body includes csrf_token for frontend to store

API Request → Browser sends access_token cookie automatically
            → Frontend includes X-CSRF-Token header
            → Middleware validates JWT + CSRF token match, injects user_id

Token Expired → 401 with code=TOKEN_EXPIRED → Frontend calls /auth/refresh
                                        → Server validates refresh_token from cookie
                                        → Issues new access_token + csrf_token via Set-Cookie
```

**Implementation:**
```go
// CSRF token stored in access token claims
type AccessTokenClaims struct {
    UserID   string `json:"user_id"`
    CSRFToken string `json:"csrf_token"`
    jwt.RegisteredClaims
}

// Middleware validates CSRF token
func CSRFMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        cookieToken, err := c.Cookie("csrf_token")
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"error": "csrf token missing"})
            return
        }
        headerToken := c.GetHeader("X-CSRF-Token")
        if cookieToken != headerToken {
            c.AbortWithStatusJSON(403, gin.H{"error": "csrf token mismatch"})
            return
        }
        c.Next()
    }
}
```

**Alternatives Considered:**
- Single long-lived token: Less secure, cannot revoke
- Authorization header: Requires JS access to token (XSS risk)
- localStorage: XSS accessible, not recommended
- SameSite only: Not all browsers support it properly

### 4. User ID: UUID v7

**Choice:** UUID v7 over auto-increment integer

**Rationale:**
- Prevents user enumeration attacks
- Time-sortable (v7 feature)
- JWT payload doesn't reveal user count
- No business information leakage

### 5. RSS Parsing: gofeed

**Choice:** gofeed over custom parser

**Rationale:**
- Battle-tested library
- Supports RSS 0.90, 0.91, 0.92, 0.93, 1.0, 2.0 and Atom 0.3, 1.0
- Handles edge cases and malformed feeds
- Active maintenance

### 6. Background Refresh: Goroutine + Ticker

**Choice:** In-process goroutine over external scheduler (cron)

**Rationale:**
- Single binary deployment
- No external dependencies
- Simple implementation for this use case
- Can be extracted to separate service later if needed

**Implementation:**
```go
func StartRefreshWorker(service *FeedService, interval time.Duration) {
    ticker := time.NewTicker(interval)
    go func() {
        for range ticker.C {
            service.RefreshAllFeeds()
        }
    }()
}
```

### 7. Frontend Build: Vite with Embed

**Choice:** Vite + embed package over separate deployment

**Rationale:**
- Single binary simplifies deployment
- Vite's fast HMR for development
- embed package for production static file serving
- Clean separation of dev/prod builds

**Implementation:**
```go
//go:embed all:web/dist
var staticFS embed.FS

// Serve embedded files in production
router.NoRoute(func(c *gin.Context) {
    c.FileFromFS(c.Request.URL.Path, http.FS(staticFS))
})
```

### 8. Frontend State: React Query + Zustand

**Choice:** React Query for server state, Zustand for client state

**Rationale:**
- React Query handles caching, refetching, background updates
- Zustand minimal boilerplate for auth state
- Clear separation of concerns
- Both are lightweight and well-maintained

### 9. UI Components: shadcn/ui + Tailwind

**Choice:** shadcn/ui over Ant Design, Material UI

**Rationale:**
- Full control over components (copy-paste, not dependency)
- Tailwind integration out of box
- Modern, accessible defaults
- Good learning opportunity for React patterns

### 10. Rate Limiting: Token Bucket with Redis (optional)

**Choice:** In-memory token bucket for single instance, Redis-based for distributed

**Rationale:**
- Protects API from abuse and DoS attacks
- Sliding window algorithm for smoother rate limiting
- Configurable limits per endpoint type
- Redis optional for horizontal scaling

**Implementation:**
```go
// Rate limits configuration
var rateLimits = map[string]RateLimit{
    "/api/v1/auth/login":    {Requests: 10, Window: time.Minute},      // Strict for auth
    "/api/v1/auth/register": {Requests: 5, Window: time.Minute},
    "/api/v1/feeds":         {Requests: 100, Window: time.Minute},     // Relaxed for read
    "default":               {Requests: 60, Window: time.Minute},
}

// Middleware using token bucket
func RateLimitMiddleware(limiter *Limiter) gin.HandlerFunc {
    return func(c *gin.Context) {
        key := getUserKey(c) // IP or user_id
        if !limiter.Allow(key, c.FullPath()) {
            c.JSON(429, gin.H{"error": "rate limit exceeded"})
            c.Abort()
            return
        }
        c.Next()
    }
}
```

**Alternatives Considered:**
- Fixed window: Can allow burst at window boundaries
- External service (Cloudflare): Adds dependency and cost

### 11. Feed Import/Export: OPML 2.0

**Choice:** OPML 2.0 format for feed subscriptions

**Rationale:**
- Standard format supported by most RSS readers
- Simple XML structure, easy to parse
- Supports nested categories (future use)
- Human-readable

**Implementation:**
```xml
<!-- Export format -->
<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head>
    <title>oReader Subscriptions</title>
    <dateCreated>Sat, 15 Mar 2025 00:00:00 GMT</dateCreated>
  </head>
  <body>
    <outline type="rss" text="Feed Title" xmlUrl="https://example.com/feed.xml"/>
  </body>
</opml>
```

**API Endpoints:**
- `GET /api/v1/feeds/export` - Download OPML file
- `POST /api/v1/feeds/import` - Upload and parse OPML file

### 12. Security Headers Middleware

**Choice:** Comprehensive security headers on all responses

**Rationale:**
- Defense-in-depth against common web vulnerabilities
- Prevents clickjacking, MIME sniffing, XSS
- Modern browser security features

**Implementation:**
```go
func SecurityHeadersMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-Frame-Options", "DENY")
        c.Header("X-XSS-Protection", "1; mode=block")
        c.Header("Content-Security-Policy", "default-src 'self'; img-src * data:; style-src 'self' 'unsafe-inline'")
        c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
        c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
        c.Next()
    }
}
```

### 13. Content Sanitization: bluemonday

**Choice:** bluemonday for HTML sanitization of RSS content

**Rationale:**
- RSS feeds may contain malicious HTML/JavaScript
- XSS prevention for user-generated content display
- Whitelist-based approach is more secure than blacklist

**Implementation:**
```go
import "github.com/microcosm-cc/bluemonday"

var (
    // UGC policy for article content - allows safe HTML
    articlePolicy = bluemonday.UGCPolicy()
    // Strict policy for feed titles/descriptions
    strictPolicy = bluemonday.StrictPolicy()
)

func SanitizeArticleContent(html string) string {
    return articlePolicy.Sanitize(html)
}

func SanitizeFeedTitle(title string) string {
    return strictPolicy.Sanitize(title)
}
```

### 14. SSRF Protection: URL Validation

**Choice:** Block private IP ranges and validate URL schemes

**Rationale:**
- User-provided feed URLs could target internal services
- Prevents access to metadata endpoints (169.254.169.254)
- Limits attack surface for server-side request forgery

**Implementation:**
```go
var blockedIPRanges = []string{
    "10.0.0.0/8",
    "172.16.0.0/12",
    "192.168.0.0/16",
    "127.0.0.0/8",
    "169.254.0.0/16",  // AWS metadata
    "::1/128",          // IPv6 loopback
    "fc00::/7",         // IPv6 private
}

func ValidateFeedURL(rawURL string) error {
    u, err := url.Parse(rawURL)
    if err != nil {
        return err
    }

    // Only allow http/https
    if u.Scheme != "http" && u.Scheme != "https" {
        return errors.New("only http/https URLs allowed")
    }

    // Resolve and check IP
    host := u.Hostname()
    ips, err := net.LookupIP(host)
    if err != nil {
        return err
    }

    for _, ip := range ips {
        if isBlockedIP(ip) {
            return errors.New("URL resolves to blocked IP range")
        }
    }

    return nil
}
```

### 15. Logging: zerolog with Structured Output

**Choice:** zerolog for structured, performant logging

**Rationale:**
- Zero-allocation JSON logging for performance
- Structured logs for searchability and analysis
- Configurable log levels
- Context-aware logging with fields

**Implementation:**
```go
import "github.com/rs/zerolog/log"

// Initialize logger
func InitLogger(level string) {
    l, _ := zerolog.ParseLevel(level)
    zerolog.SetGlobalLevel(l)

    if os.Getenv("ENV") == "development" {
        log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
    }
}

// Usage in handlers
func (h *FeedHandler) CreateFeed(c *gin.Context) {
    log.Info().
        Str("user_id", userID).
        Str("url", feedURL).
        Msg("creating feed subscription")
    // ...
}
```

### 16. Database Migration: golang-migrate

**Choice:** golang-migrate over GORM AutoMigrate for production

**Rationale:**
- Version-controlled migrations with up/down
- Rollback capability for production incidents
- SQL-based migrations for fine control
- Migration history tracking

**Implementation:**
```
migrations/
├── 001_create_users.up.sql
├── 001_create_users.down.sql
├── 002_create_feeds.up.sql
├── 002_create_feeds.down.sql
├── 003_create_items.up.sql
├── 003_create_items.down.sql
├── 004_create_refresh_tokens.up.sql
└── 004_create_refresh_tokens.down.sql
```

```go
import "github.com/golang-migrate/migrate/v4"

func RunMigrations(dbURL string) error {
    m, err := migrate.New(
        "file://migrations",
        dbURL)
    if err != nil {
        return err
    }
    return m.Up()
}
```

### 17. API Error Response Format

**Choice:** Consistent structured error responses

**Rationale:**
- Predictable error structure for frontend handling
- Error codes for programmatic handling
- Human-readable messages
- Optional details for validation errors

**Implementation:**
```go
type ErrorResponse struct {
    Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
    Code    string                 `json:"code"`
    Message string                 `json:"message"`
    Details map[string]interface{} `json:"details,omitempty"`
}

// Error codes
const (
    ErrValidation     = "VALIDATION_ERROR"
    ErrUnauthorized   = "UNAUTHORIZED"
    ErrTokenExpired   = "TOKEN_EXPIRED"
    ErrForbidden      = "FORBIDDEN"
    ErrNotFound       = "NOT_FOUND"
    ErrConflict       = "CONFLICT"
    ErrRateLimit      = "RATE_LIMIT_EXCEEDED"
    ErrInternal       = "INTERNAL_ERROR"
)

// Helper function
func SendError(c *gin.Context, status int, code, message string, details map[string]interface{}) {
    c.JSON(status, ErrorResponse{
        Error: ErrorDetail{
            Code:    code,
            Message: message,
            Details: details,
        },
    })
}
```

**Example Responses:**
```json
// Validation error
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input",
    "details": {
      "field": "email",
      "value": "invalid-email"
    }
  }
}

// Token expired
{
  "error": {
    "code": "TOKEN_EXPIRED",
    "message": "Access token has expired"
  }
}
```

### 18. Layered Architecture with Interfaces

**Choice:** Explicit interface definitions between layers

**Rationale:**
- Enables proper mocking for unit tests
- Clear contracts between layers
- Dependency inversion principle
- Easier to swap implementations

**Implementation:**
```go
// internal/service/interfaces.go
package service

import "context"

type UserRepository interface {
    Create(ctx context.Context, user *model.User) error
    GetByEmail(ctx context.Context, email string) (*model.User, error)
    GetByID(ctx context.Context, id string) (*model.User, error)
    Update(ctx context.Context, user *model.User) error
}

type FeedRepository interface {
    Create(ctx context.Context, feed *model.Feed) error
    GetByID(ctx context.Context, id string) (*model.Feed, error)
    ListByUserID(ctx context.Context, userID string, opts ListOptions) ([]*model.Feed, int64, error)
    Delete(ctx context.Context, id string) error
}

type ItemRepository interface {
    Create(ctx context.Context, item *model.Item) error
    GetByID(ctx context.Context, id string) (*model.Item, error)
    ListByFeedID(ctx context.Context, feedID string, opts ListOptions) ([]*model.Item, int64, error)
    Update(ctx context.Context, item *model.Item) error
}

type RefreshTokenRepository interface {
    Create(ctx context.Context, token *model.RefreshToken) error
    GetByToken(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
    Revoke(ctx context.Context, tokenHash string) error
    RevokeAllByUser(ctx context.Context, userID string) error
}
```

## Architecture

### Backend Structure
```
cmd/
└── server/main.go              # Entry point
internal/
├── config/                     # Configuration (viper)
├── domain/                     # Domain entities and business rules
│   └── entity/                 # User, Feed, Item entities
├── handler/                    # HTTP handlers (Gin)
├── middleware/                 # Auth, CORS, logging, CSRF, security headers
├── service/                    # Business logic
│   └── interfaces.go           # Repository interfaces
├── repository/                 # Data access implementations (GORM)
├── model/                      # Database models (GORM structs)
└── infra/                      # Infrastructure utilities
    ├── jwt/                    # JWT generation/validation
    ├── rss/                    # gofeed wrapper + URL validation
    ├── cookie/                 # Cookie utilities
    ├── sanitize/               # HTML sanitization (bluemonday)
    └── ratelimit/              # Rate limiting (in-memory + Redis interface)
migrations/                     # Database migrations (golang-migrate)
web/                            # React frontend (embedded)
```

### Frontend Structure
```
web/src/
├── components/
│   └── ui/                     # shadcn components
├── features/
│   ├── auth/                   # Login, Register, authStore
│   ├── feeds/                  # FeedList, AddFeed
│   └── items/                  # ItemList, ItemView
├── hooks/                      # Custom hooks
├── lib/
│   ├── api/                    # Typed API client
│   │   ├── client.ts           # Axios instance
│   │   ├── feeds.ts            # Feed API methods
│   │   └── items.ts            # Item API methods
│   └── utils.ts                # Utilities
├── stores/                     # Zustand stores
│   ├── authStore.ts            # Auth state (isAuthenticated, user)
│   └── uiStore.ts              # UI state (sidebarOpen, theme)
├── types/                      # TypeScript types
│   └── api.ts                  # API response types
└── App.tsx                     # Router setup
```

### Data Model

> **Important**: `is_starred` and `is_read` are per-user states, not per-item states.
> If stored on Item directly, User A's read status would affect User B.
> Solution: Use `UserItemState` join table for per-user item states.

```
User (id: UUID v7) ──1:N──▶ UserFeed ──N:1──▶ Feed
       │                              │
       ├──1:N──▶ RefreshToken        └──1:N──▶ Item ──1:1──▶ UserItemState
       │
       └── fields: email, password_hash, nickname, avatar_url, auth_provider, github_id

UserFeed (用户订阅关系)
  └── fields: user_id, feed_id, position (排序), created_at

Feed (RSS源 - 共享)
  └── fields: feed_url (唯一), title, description, image_url, last_fetched_at, last_fetch_status, consecutive_failures

Item (文章内容 - 共享)
  └── fields: feed_id, guid (去重), title, link, description, content, pub_date, creator

UserItemState (用户文章状态 - 私有)
  └── fields: user_id, item_id, is_starred, is_read, read_at, created_at
```

### Key Design Decisions

1. **Feed 共享**: 多个用户订阅同一个 RSS URL 时，只存储一份 Feed 和 Item 记录
2. **UserFeed 关联**: 用户通过 UserFeed 表关联到 Feed，可存储用户特定的订阅元数据（如排序）
3. **UserItemState 隔离**: 每个用户对文章的阅读/收藏状态独立存储，互不影响
4. **GUID 去重**: 使用 RSS item 的 guid 字段防止重复抓取

### API Implications

```go
// 获取用户的订阅列表
GET /api/v1/feeds
→ 返回 UserFeed 列表，包含 Feed 详情

// 获取文章列表
GET /api/v1/items?feed_id=xxx
→ 返回 Item 列表，通过 UserItemState JOIN 获取 is_starred/is_read

// 标记已读
POST /api/v1/items/:id/read
→ 创建/更新 UserItemState 记录

// 收藏文章
POST /api/v1/items/:id/star
→ 更新 UserItemState.is_starred
```

### State Management Split

| State Type | Tool | Examples |
|------------|------|----------|
| Server State | React Query | feeds, items, user profile |
| Client State | Zustand | isAuthenticated, sidebarOpen, theme |
| Form State | React Hook Form | login form, add feed form |

## Risks / Trade-offs

### Risk 1: Cookie-based Auth and CORS
**Risk:** SameSite=Strict cookies may not work with separate frontend domain
**Mitigation:**
- Deploy as single origin (primary approach)
- If separate domains needed, use SameSite=Lax with explicit CSRF tokens
- Document fallback configuration in deployment guide

### Risk 2: Background Worker in Same Process
**Risk:** Long-running feed refresh may block API responses
**Mitigation:**
- Use goroutines with context timeout (30s per feed)
- Bounded concurrency with semaphore (max 10 concurrent fetches)
- Plan for extraction to separate worker service in v2.1:
  - v2.0: In-process goroutine (acceptable for MVP)
  - v2.1: Redis-based distributed lock + job queue
  - v3.0: Separate worker service with message queue

### Risk 3: Single Binary Size
**Risk:** Embedding frontend increases binary size
**Mitigation:** Acceptable trade-off for deployment simplicity; ~10-20MB typical

### Risk 4: No Data Migration
**Risk:** Existing users lose their subscriptions
**Mitigation:**
- Add OPML export to Flask v1 before switching
- Document migration guide for users to export/import subscriptions
- Consider 30-day sunset period with read-only Flask version

### Risk 5: GitHub OAuth Implementation Complexity
**Risk:** OAuth flow adds complexity
**Mitigation:** Reserve interface only for v1; implement in future iteration

### Risk 6: RSS Feed Malicious Content (NEW)
**Risk:** Feeds may contain XSS payloads in HTML content
**Mitigation:**
- Sanitize all HTML content with bluemonday before serving
- Use UGC policy for article content
- Use strict policy for feed titles/descriptions

### Risk 7: Feed Fetch SSRF (NEW)
**Risk:** User-provided URLs could target internal services
**Mitigation:**
- Block private IP ranges (10.x, 172.16-31.x, 192.168.x, 127.x, 169.254.x)
- Only allow http/https schemes
- Validate URL before fetching

### Risk 8: Large Feed Attack (NEW)
**Risk:** Malicious feeds with thousands of items
**Mitigation:**
- Limit items per feed to 1000
- Limit individual item content size to 1MB
- Timeout fetch requests at 30 seconds

### Risk 9: Rate Limiting Bypass (NEW)
**Risk:** In-memory rate limiting doesn't work with multiple instances
**Mitigation:**
- Design rate limiter with interface from day one
- Support both in-memory and Redis backends
- Use IP + user_id combination for authenticated rate limiting

### Risk 10: Dependency Vulnerabilities (NEW)
**Risk:** Third-party packages may have security vulnerabilities
**Mitigation:**
- Enable Dependabot for automated dependency scanning
- Run `govulncheck` in CI pipeline
- Regular dependency updates in sprint planning

## Migration Plan

### Pre-Migration (Flask v1)
1. Add OPML export endpoint to Flask version
2. Document user migration guide
3. Announce sunset timeline (30 days)

### Development Phase
1. Set up Go project structure
2. Implement core models and migrations
3. Build authentication system
4. Implement RSS parsing and refresh
5. Build React frontend
6. Integrate and test

### Deployment Phase
1. Build Docker image with multi-stage Dockerfile
2. Run database migrations (golang-migrate)
3. Deploy container
4. Verify health checks
5. Monitor logs and metrics

### Rollback Strategy
- Stateless application - just redeploy previous image
- Database migrations use golang-migrate with down scripts
- No breaking schema changes in v1
- Keep Flask version in read-only mode for 30 days as fallback

## Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | - | Database connection string |
| `JWT_SECRET_KEY` | Yes | - | Secret key for JWT signing (256-bit) |
| `JWT_ACCESS_TTL` | No | 15m | Access token lifetime |
| `JWT_REFRESH_TTL` | No | 168h | Refresh token lifetime (7 days) |
| `REFRESH_INTERVAL` | No | 15m | Feed refresh interval |
| `RATE_LIMIT_ENABLED` | No | true | Enable rate limiting |
| `RATE_LIMIT_REDIS_URL` | No | - | Redis URL for distributed rate limiting |
| `LOG_LEVEL` | No | info | Log level (debug, info, warn, error) |
| `ENV` | No | development | Environment (development, production) |
| `GITHUB_CLIENT_ID` | No | - | GitHub OAuth client ID |
| `GITHUB_CLIENT_SECRET` | No | - | GitHub OAuth client secret |

### Configuration Loading
```go
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Auth     AuthConfig
    Refresh  RefreshConfig
    RateLimit RateLimitConfig
    Logging  LoggingConfig
    OAuth    OAuthConfig
}

func LoadConfig() (*Config, error) {
    v := viper.New()
    v.AutomaticEnv()
    v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

    // Set defaults
    v.SetDefault("JWT_ACCESS_TTL", "15m")
    v.SetDefault("JWT_REFRESH_TTL", "168h")
    v.SetDefault("REFRESH_INTERVAL", "15m")
    v.SetDefault("RATE_LIMIT_ENABLED", true)
    v.SetDefault("LOG_LEVEL", "info")

    // ... bind and validate
}
```

## Open Questions

1. **Refresh interval:** Default 15 minutes - should this be configurable per feed?
   - **Resolution:** Start with global config; per-feed can be added in v2.1

2. **Email verification:** Required for registration or optional?
   - **Resolution:** Optional for v2.0; can be added later if spam is an issue

3. **Rate limit storage:** In-memory sufficient or need Redis for production?
   - **Resolution:** Design with interface; start in-memory, add Redis config option

4. **Password complexity:** What requirements should be enforced?
   - **Resolution:** Minimum 8 characters, bcrypt cost 12

5. **API versioning:** How will v2 be introduced?
   - **Resolution:** URL path versioning (/api/v1, /api/v2); maintain v1 for 6 months after v2 launch
