# oReader API Documentation

Version: v2.0.0
Base URL: `http://localhost:8080`

## Table of Contents

- [Authentication](#authentication)
- [Errors](#errors)
- [Auth Endpoints](#auth-endpoints)
- [Feed Endpoints](#feed-endpoints)
- [Item Endpoints](#item-endpoints)
- [Import/Export Endpoints](#importexport-endpoints)
- [OAuth Endpoints](#oauth-endpoints)
- [Health Check](#health-check)

## Authentication

oReader uses JWT (JSON Web Tokens) for authentication with a dual-token system:

- **Access Token**: Short-lived (1 hour) token for API access
- **Refresh Token**: Long-lived (7 days) token for obtaining new access tokens

### Authentication Flow

1. Register or login to receive access and refresh tokens
2. Include the access token in the `Authorization` header for authenticated requests
3. When the access token expires, use the refresh token to obtain a new access token
4. Refresh tokens are stored in HttpOnly cookies for security

### CSRF Protection

State-changing requests (POST, PUT, DELETE, PATCH) require a CSRF token:

1. CSRF token is automatically included in responses as a cookie
2. Include the CSRF token in the `X-CSRF-Token` header for state-changing requests
3. GET, HEAD, and OPTIONS requests are exempt from CSRF protection

### Example Request

```http
GET /api/v1/me
Authorization: Bearer <access_token>
X-CSRF-Token: <csrf_token>
```

## Errors

All error responses follow this format:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {}
  }
}
```

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_REQUEST` | 400 | Invalid request parameters |
| `UNAUTHORIZED` | 401 | Authentication required |
| `TOKEN_EXPIRED` | 401 | Access token expired |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests |
| `INTERNAL_ERROR` | 500 | Internal server error |

## Auth Endpoints

### Register

Creates a new user account.

**Endpoint**: `POST /auth/register`
**Auth Required**: No
**CSRF Required**: Yes

**Request Body**:
```json
{
  "email": "user@example.com",
  "password": "securepassword",
  "nickname": "John Doe"
}
```

**Response** (200 OK):
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "refresh_token_here",
  "user": {
    "id": "user-id",
    "email": "user@example.com",
    "nickname": "John Doe",
    "avatar_url": null
  }
}
```

### Login

Authenticates a user and returns tokens.

**Endpoint**: `POST /auth/login`
**Auth Required**: No
**CSRF Required**: Yes

**Request Body**:
```json
{
  "email": "user@example.com",
  "password": "securepassword"
}
```

**Response** (200 OK):
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "refresh_token_here",
  "user": {
    "id": "user-id",
    "email": "user@example.com",
    "nickname": "John Doe",
    "avatar_url": "https://example.com/avatar.jpg"
  }
}
```

### Logout

Invalidates the refresh token and clears cookies.

**Endpoint**: `POST /auth/logout`
**Auth Required**: Yes
**CSRF Required**: Yes

**Response** (200 OK):
```json
{
  "message": "Successfully logged out"
}
```

### Refresh Token

Obtains a new access token using a refresh token.

**Endpoint**: `POST /auth/refresh`
**Auth Required**: No (refresh token in cookie)
**CSRF Required**: Yes

**Response** (200 OK):
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "user-id",
    "email": "user@example.com",
    "nickname": "John Doe"
  }
}
```

### Get Current User

Returns information about the authenticated user.

**Endpoint**: `GET /auth/me`
**Auth Required**: Yes
**CSRF Required**: No

**Response** (200 OK):
```json
{
  "id": "user-id",
  "email": "user@example.com",
  "nickname": "John Doe",
  "avatar_url": "https://example.com/avatar.jpg",
  "created_at": "2024-01-01T00:00:00Z"
}
```

## Feed Endpoints

### Subscribe to Feed

Subscribes the authenticated user to a feed.

**Endpoint**: `POST /api/v1/feeds`
**Auth Required**: Yes
**CSRF Required**: Yes

**Request Body**:
```json
{
  "feed_url": "https://example.com/feed.xml"
}
```

**Response** (201 Created):
```json
{
  "id": "feed-id",
  "feed_url": "https://example.com/feed.xml",
  "title": "Example Feed",
  "description": "Feed description",
  "image_url": "https://example.com/icon.png",
  "created_at": "2024-01-01T00:00:00Z",
  "new_items": 5
}
```

### List Feeds

Returns all feeds subscribed by the authenticated user.

**Endpoint**: `GET /api/v1/feeds`
**Auth Required**: Yes
**CSRF Required**: No

**Query Parameters**:
- `limit` (optional): Number of feeds to return (default: 100)
- `offset` (optional): Number of feeds to skip (default: 0)

**Response** (200 OK):
```json
{
  "feeds": [
    {
      "id": "feed-id",
      "feed_url": "https://example.com/feed.xml",
      "title": "Example Feed",
      "description": "Feed description",
      "image_url": "https://example.com/icon.png",
      "last_fetched_at": "2024-01-01T12:00:00Z",
      "item_count": 42
    }
  ],
  "total": 1
}
```

### Get Feed

Returns details of a specific feed.

**Endpoint**: `GET /api/v1/feeds/:feed_id`
**Auth Required**: Yes
**CSRF Required**: No

**Response** (200 OK):
```json
{
  "id": "feed-id",
  "feed_url": "https://example.com/feed.xml",
  "title": "Example Feed",
  "description": "Feed description",
  "image_url": "https://example.com/icon.png",
  "last_fetched_at": "2024-01-01T12:00:00Z",
  "created_at": "2024-01-01T00:00:00Z"
}
```

### Delete Feed

Unsubscribes the authenticated user from a feed.

**Endpoint**: `DELETE /api/v1/feeds/:feed_id`
**Auth Required**: Yes
**CSRF Required**: Yes

**Response** (200 OK):
```json
{
  "message": "Feed deleted successfully"
}
```

### Refresh Feed

Manually triggers a refresh of a specific feed.

**Endpoint**: `POST /api/v1/feeds/:feed_id/refresh`
**Auth Required**: Yes
**CSRF Required**: Yes

**Response** (200 OK):
```json
{
  "message": "Feed refresh triggered",
  "new_items": 3
}
```

## Item Endpoints

### List Items

Returns items for the authenticated user with optional filtering.

**Endpoint**: `GET /api/v1/items`
**Auth Required**: Yes
**CSRF Required**: No

**Query Parameters**:
- `limit` (optional): Number of items to return (default: 20)
- `cursor` (optional): Pagination cursor
- `feed_id` (optional): Filter by feed ID
- `starred` (optional): Filter by starred status (true/false)
- `read` (optional): Filter by read status (true/false)

**Response** (200 OK):
```json
{
  "items": [
    {
      "id": "item-id",
      "feed_id": "feed-id",
      "title": "Article Title",
      "link": "https://example.com/article",
      "description": "Article description",
      "content": "<p>Full article content</p>",
      "pub_date": "2024-01-01T12:00:00Z",
      "creator": "Author Name",
      "is_starred": false,
      "is_read": false,
      "read_at": null
    }
  ],
  "cursor": "next-cursor-token",
  "has_more": true
}
```

### Get Item

Returns details of a specific item.

**Endpoint**: `GET /api/v1/items/:item_id`
**Auth Required**: Yes
**CSRF Required**: No

**Response** (200 OK):
```json
{
  "id": "item-id",
  "feed_id": "feed-id",
  "feed": {
    "id": "feed-id",
    "title": "Example Feed"
  },
  "title": "Article Title",
  "link": "https://example.com/article",
  "description": "Article description",
  "content": "<p>Full article content</p>",
  "pub_date": "2024-01-01T12:00:00Z",
  "creator": "Author Name",
  "is_starred": false,
  "is_read": false,
  "read_at": null,
  "created_at": "2024-01-01T00:00:00Z"
}
```

### Toggle Star

Toggles the starred status of an item.

**Endpoint**: `POST /api/v1/items/:item_id/star`
**Auth Required**: Yes
**CSRF Required**: Yes

**Response** (200 OK):
```json
{
  "id": "item-id",
  "is_starred": true
}
```

### Toggle Read

Toggles the read status of an item.

**Endpoint**: `POST /api/v1/items/:item_id/read`
**Auth Required**: Yes
**CSRF Required**: Yes

**Response** (200 OK):
```json
{
  "id": "item-id",
  "is_read": true,
  "read_at": "2024-01-01T12:00:00Z"
}
```

### Mark All as Read

Marks all items in a feed as read.

**Endpoint**: `POST /api/v1/feeds/:feed_id/mark-all-read`
**Auth Required**: Yes
**CSRF Required**: Yes

**Response** (200 OK):
```json
{
  "message": "All items marked as read",
  "count": 42
}
```

## Import/Export Endpoints

### Export OPML

Exports the user's feeds as an OPML file.

**Endpoint**: `GET /api/v1/opml/export`
**Auth Required**: Yes
**CSRF Required**: No

**Response** (200 OK):
```
Content-Type: application/xml; charset=utf-8
Content-Disposition: attachment; filename="feeds.opml"

<?xml version="1.0" encoding="UTF-8"?>
<opml version="1.0">
  <head>
    <title>My Feeds</title>
  </head>
  <body>
    <outline text="Feed Title" xmlUrl="https://example.com/feed.xml"/>
  </body>
</opml>
```

### Import OPML

Imports feeds from an OPML file.

**Endpoint**: `POST /api/v1/opml/import`
**Auth Required**: Yes
**CSRF Required**: Yes

**Request Body**:
```
Content-Type: application/xml

<?xml version="1.0" encoding="UTF-8"?>
<opml version="1.0">
  <head>
    <title>My Feeds</title>
  </head>
  <body>
    <outline text="Feed Title" xmlUrl="https://example.com/feed.xml"/>
  </body>
</opml>
```

**Response** (202 Accepted):
```json
{
  "job_id": "job-id",
  "status": "pending",
  "total_feeds": 5,
  "message": "Import job started"
}
```

### Get Import Job Status

Returns the status of an import job.

**Endpoint**: `GET /api/v1/opml/import/:job_id`
**Auth Required**: Yes
**CSRF Required**: No

**Response** (200 OK):
```json
{
  "id": "job-id",
  "user_id": "user-id",
  "status": "completed",
  "total_feeds": 5,
  "processed": 5,
  "failed": 0,
  "error": null,
  "started_at": "2024-01-01T00:00:00Z",
  "ended_at": "2024-01-01T00:01:00Z"
}
```

## OAuth Endpoints

### Initiate GitHub OAuth

Initiates the GitHub OAuth flow.

**Endpoint**: `GET /auth/github`
**Auth Required**: No
**CSRF Required**: No

**Response** (302 Found):
Redirects to GitHub authorization page.

### GitHub OAuth Callback

Handles the GitHub OAuth callback.

**Endpoint**: `GET /auth/github/callback`
**Auth Required**: No
**CSRF Required**: No

**Query Parameters**:
- `code`: Authorization code from GitHub
- `state`: CSRF protection token

**Response** (302 Found):
Redirects to frontend with authentication tokens.

## Health Check

### Health Check

Returns the health status of the application.

**Endpoint**: `GET /health`
**Auth Required**: No
**CSRF Required**: No

**Response** (200 OK):
```json
{
  "status": "ok",
  "version": "2.0.0"
}
```

## Rate Limiting

API endpoints are rate-limited to prevent abuse:

| Endpoint Type | Limit |
|---------------|-------|
| Authentication | 10 requests/minute |
| Feed Operations | 20 requests/minute |
| API Requests | 100 requests/minute |

Rate limit headers are included in responses:
- `X-RateLimit-Limit`: Request limit
- `X-RateLimit-Remaining`: Remaining requests
- `X-RateLimit-Reset`: Unix timestamp when limit resets

## Data Models

### User
```json
{
  "id": "string (UUID)",
  "email": "string",
  "nickname": "string",
  "avatar_url": "string (optional)",
  "created_at": "datetime (ISO 8601)"
}
```

### Feed
```json
{
  "id": "string (UUID)",
  "feed_url": "string",
  "title": "string",
  "description": "string (optional)",
  "image_url": "string (optional)",
  "last_fetched_at": "datetime (ISO 8601, optional)",
  "created_at": "datetime (ISO 8601)"
}
```

### Item
```json
{
  "id": "string (UUID)",
  "feed_id": "string (UUID)",
  "guid": "string",
  "title": "string",
  "link": "string",
  "description": "string",
  "content": "string (HTML)",
  "pub_date": "datetime (ISO 8601, optional)",
  "creator": "string (optional)",
  "created_at": "datetime (ISO 8601)"
}
```

### UserItemState
```json
{
  "id": "string (UUID)",
  "user_id": "string (UUID)",
  "item_id": "string (UUID)",
  "is_starred": "boolean",
  "is_read": "boolean",
  "read_at": "datetime (ISO 8601, optional)",
  "created_at": "datetime (ISO 8601)"
}
```

## WebSocket API (Future)

WebSocket support for real-time updates is planned for future releases.

## Changelog

### v2.0.0 (Current)
- Initial release with full REST API
- JWT authentication with dual-token system
- CSRF protection
- Rate limiting
- OPML import/export
- GitHub OAuth integration

## Support

For API support and questions:
- Documentation: [docs/](docs/)
- Issues: [GitHub Issues](https://github.com/yourusername/oreader/issues)
- Email: api-support@example.com

---

**Last Updated**: 2026-03-16
