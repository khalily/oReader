# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

oReader is an online RSS reader built with Flask (backend) and AngularJS (frontend). It allows users to subscribe to RSS feeds, read articles, and manage their subscriptions.

## Development Commands

```bash
# Install dependencies
pip install -r requirements.txt

# Run development server
python manage.py runserver

# Run with specific config (devlopment, testing, production)
OREADER_CONFIG=testing python manage.py runserver

# Access Flask shell with app context (includes db, User, Feed, Item models)
python manage.py shell
```

## Architecture

### Backend (Flask)

The application uses the **application factory pattern** with Flask blueprints:

- **`app/__init__.py`**: Factory function `create_app(config_name)` initializes extensions and registers blueprints
- **`config.py`**: Three environments - `devlopment` (SQLite), `testing` (SQLite), `production` (PostgreSQL/MySQL via env vars)

#### Blueprints

| Blueprint | URL Prefix | Purpose |
|-----------|------------|---------|
| `main` | `/` | Serves the SPA index.html |
| `api` | `/api/v1` | RESTful API endpoints |
| `auth` | `/auth` | Traditional web authentication |

#### API Layer (`app/api/`)

- Uses **Flask-RESTful** for resource-based routing
- **Marshmallow** schemas for serialization/deserialization and validation
- **HTTP Basic Auth** with token support (email+password for initial auth, then token for subsequent requests)
- Authentication flow: `GET /api/v1/get_token` (email:password) → returns token → use token for subsequent requests

#### Data Models (`app/models.py`)

- **User**: email, password_hash, generates auth tokens (1-hour expiry)
- **Feed**: RSS feed data (title, link, description, last_build_date, img) - belongs to User
- **Item**: Individual RSS items (title, link, description, content, pub_date, creator, star) - belongs to Feed

#### RSS Parsing (`app/feedparser/`)

Custom feedparser module parses RSS feeds when users add subscriptions. The `FeedSchema.make_object()` method fetches the URL, parses it, creates Feed and Item records, and commits to database.

### Frontend (AngularJS)

Single-page application in `app/static/`:

- **`js/app.js`**: Route configuration with `loginRequired` guards
- **`js/services.js`**: Auth service, token injection interceptor, $resource factories for API
- **`js/controllers.js`**: View controllers
- **`partials/`**: HTML templates

Auth tokens are stored via `angular-store` and injected via `tokenInjector` interceptor.

## Key Patterns

- **Schema-driven API**: Marshmallow schemas handle both validation and object creation (`make_object` method)
- **Token auth**: JWT-style tokens via itsdangerous, passed as HTTP Basic Auth username with empty password
- **SPA with API**: Flask serves static index.html, AngularJS handles routing and API consumption

## Deployment

Heroku deployment via Procfile using gunicorn:
```
web: gunicorn app:create_app\(\"production\"\) -e OREADER_CONFIG='production' -e SECRET_KEY='...' -e DATABASE_URI='...'
```

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `OREADER_CONFIG` | Config name: `devlopment`, `testing`, or `production` |
| `SECRET_KEY` | App secret key (production) |
| `DATABASE_URI` | Database URL (production) |
