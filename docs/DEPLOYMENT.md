# Deployment Guide - oReader v2.0.0

This guide covers deploying oReader in various environments, from development to production.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Environment Variables](#environment-variables)
- [Development Deployment](#development-deployment)
- [Production Deployment](#production-deployment)
- [Docker Deployment](#docker-deployment)
- [Database Setup](#database-setup)
- [SSL/TLS Configuration](#ssltls-configuration)
- [Monitoring & Logging](#monitoring--logging)
- [Scaling Considerations](#scaling-considerations)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### System Requirements

- **Go**: 1.25 or higher
- **Node.js**: 18+ (for building frontend only)
- **Database**: SQLite 3.x (development) or MySQL 8.x (production)
- **Memory**: Minimum 512MB RAM, 1GB+ recommended
- **Disk**: 100MB minimum, 1GB+ recommended with growth
- **OS**: Linux (Ubuntu 20.04+, Debian 11+, CentOS 8+)

### Network Requirements

- **HTTP Port**: 8080 (configurable)
- **HTTPS Port**: 443 (if using reverse proxy)
- **Database Access**: Network access to MySQL server (if remote)

## Environment Variables

### Required Variables

| Variable | Description | Example | Default |
|----------|-------------|---------|---------|
| `OREADER_CONFIG` | Configuration mode | `production` | `devlopment` |
| `DATABASE_URL` | Database connection string | `user:pass@tcp(db.example.com:3306)/oreader` | `oreader.db` |
| `JWT_SECRET_KEY` | JWT signing secret (min 32 chars) | `your-secret-key-min-32-chars-long` | - |

### Optional Variables

| Variable | Description | Example | Default |
|----------|-------------|---------|---------|
| `SERVER_PORT` | HTTP server port | `8080` | `8080` |
| `SERVER_HOST` | HTTP server host | `0.0.0.0` | `0.0.0.0` |
| `REFRESH_TOKEN_DAYS` | Refresh token expiry (days) | `7` | `7` |
| `REFRESH_INTERVAL` | Feed refresh interval | `1h` | `1h` |
| `MAX_CONCURRENT_REFRESH` | Max concurrent feed refreshes | `10` | `10` |
| `LOG_LEVEL` | Logging level | `info` | `info` |
| `LOG_FORMAT` | Log output format | `json` | `console` |
| `CORS_ALLOWED_ORIGINS` | CORS allowed origins | `https://example.com` | `*` |
| `GITHUB_CLIENT_ID` | GitHub OAuth client ID | `github-client-id` | - |
| `GITHUB_CLIENT_SECRET` | GitHub OAuth client secret | `github-client-secret` | - |

### Environment File Example

Create a `.env` file in the project root:

```bash
# Configuration
OREADER_CONFIG=production

# Database
DATABASE_URL=user:password@tcp(db.example.com:3306)/oreader

# Security
JWT_SECRET_KEY=your-very-secure-secret-key-at-least-32-characters-long

# Server
SERVER_PORT=8080
SERVER_HOST=0.0.0.0

# Refresh Tokens
REFRESH_TOKEN_DAYS=7

# Feed Refresh
REFRESH_INTERVAL=1h
MAX_CONCURRENT_REFRESH=10

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# CORS
CORS_ALLOWED_ORIGINS=https://example.com,https://www.example.com

# OAuth (Optional)
GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret
```

## Development Deployment

### Local Development Setup

1. **Clone the repository**:
```bash
git clone https://github.com/yourusername/oreader.git
cd oreader
```

2. **Install dependencies**:
```bash
go mod download
cd web && npm install && cd ..
```

3. **Set environment variables**:
```bash
export OREADER_CONFIG=devlopment
export DATABASE_URL=oreader.db
export JWT_SECRET_KEY=dev-secret-key-min-32-chars
```

4. **Run migrations**:
```bash
make migrate-up
```

5. **Build and run**:
```bash
make run
```

The application will be available at `http://localhost:8080`

## Production Deployment

### Building for Production

1. **Build frontend**:
```bash
cd web
npm run build
cd ..
```

2. **Build backend**:
```bash
make build
```

3. **The binary will be created at**:
```
./build/oreader
```

### Systemd Service

Create a systemd service file `/etc/systemd/system/oreader.service`:

```ini
[Unit]
Description=oReader RSS Reader
After=network.target mysql.service

[Service]
Type=simple
User=oreader
Group=oreader
WorkingDirectory=/opt/oreader
Environment="OREADER_CONFIG=production"
EnvironmentFile=/opt/oreader/.env
ExecStart=/opt/oreader/oreader
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable oreader
sudo systemctl start oreader
sudo systemctl status oreader
```

### Nginx Reverse Proxy

Configure Nginx as a reverse proxy with SSL:

```nginx
server {
    listen 80;
    server_name reader.example.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name reader.example.com;

    ssl_certificate /etc/ssl/certs/reader.example.com.crt;
    ssl_certificate_key /etc/ssl/private/reader.example.com.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    client_max_body_size 10M;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_redirect off;
    }
}
```

## Docker Deployment

### Docker Compose (Development)

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - OREADER_CONFIG=devlopment
      - DATABASE_URL=/data/oreader.db
      - JWT_SECRET_KEY=your-secret-key-min-32-chars-long
    volumes:
      - ./data:/data
    restart: unless-stopped
```

Run with:

```bash
docker-compose up -d
```

### Docker Compose (Production)

Create `docker-compose.prod.yml`:

```yaml
version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - OREADER_CONFIG=production
      - DATABASE_URL=user:password@tcp(mysql:3306)/oreader
      - JWT_SECRET_KEY=${JWT_SECRET_KEY}
      - REFRESH_TOKEN_DAYS=7
      - REFRESH_INTERVAL=1h
      - LOG_LEVEL=info
      - LOG_FORMAT=json
    depends_on:
      - mysql
    restart: unless-stopped

  mysql:
    image: mysql:8.0
    environment:
      - MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASSWORD}
      - MYSQL_DATABASE=oreader
      - MYSQL_USER=oreader
      - MYSQL_PASSWORD=${MYSQL_PASSWORD}
    volumes:
      - mysql-data:/var/lib/mysql
    restart: unless-stopped

volumes:
  mysql-data:
```

Run with:

```bash
docker-compose -f docker-compose.prod.yml up -d
```

## Database Setup

### MySQL Production Setup

1. **Create database and user**:

```sql
CREATE DATABASE oreader CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'oreader'@'localhost' IDENTIFIED BY 'strong-password';
GRANT ALL PRIVILEGES ON oreader.* TO 'oreader'@'localhost';
FLUSH PRIVILEGES;
```

2. **Configure connection string**:

```bash
DATABASE_URL=oreader:strong-password@tcp(localhost:3306)/oreader
```

3. **Run migrations**:

```bash
make migrate-up
```

### Database Backup

**Backup**:
```bash
mysqldump -u oader -p oreader > oreader-backup-$(date +%Y%m%d).sql
```

**Restore**:
```bash
mysql -u oreader -p oreader < oreader-backup-20240316.sql
```

### Automated Backups

Create a cron job for daily backups:

```bash
# Edit crontab
crontab -e

# Add daily backup at 2 AM
0 2 * * * /usr/bin/mysqldump -u oreader -p'password' oreader | gzip > /backups/oreader-$(date +\%Y\%m\%d).sql.gz
```

## SSL/TLS Configuration

### Let's Encrypt with Certbot

1. **Install Certbot**:
```bash
sudo apt-get update
sudo apt-get install certbot python3-certbot-nginx
```

2. **Obtain certificate**:
```bash
sudo certbot --nginx -d reader.example.com
```

3. **Auto-renewal** (configured by default):
```bash
sudo certbot renew --dry-run
```

### Manual SSL Configuration

If you have SSL certificates:

```nginx
ssl_certificate /etc/ssl/certs/reader.example.com.crt;
ssl_certificate_key /etc/ssl/private/reader.example.com.key;
ssl_protocols TLSv1.2 TLSv1.3;
ssl_ciphers 'ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384';
ssl_prefer_server_ciphers on;
ssl_session_cache shared:SSL:10m;
ssl_session_timeout 10m;
```

## Monitoring & Logging

### Application Logging

Logs are written to stdout/stderr in JSON format:

```bash
# View logs
journalctl -u oreader -f

# View logs for last hour
journalctl -u oreader --since "1 hour ago"

# Export logs
journalctl -u oreader > oreader-logs.txt
```

### Log Aggregation

For centralized logging, configure your logging driver:

```bash
# systemd service override
[Service]
StandardOutput=journal
StandardError=journal
SyslogIdentifier=oreader
```

### Health Checks

Monitor application health:

```bash
# Simple health check
curl http://localhost:8080/health

# With monitoring tools
# Add to your monitoring service (e.g., Pingdom, UptimeRobot)
# Monitor: http://your-domain.com/health
```

### Metrics (Future)

Consider integrating with:
- Prometheus + Grafana
- Datadog
- New Relic
- CloudWatch (AWS)

## Scaling Considerations

### Horizontal Scaling

For multiple instances:

1. **Use shared MySQL database** (not SQLite)
2. **Configure session storage** (already using JWT tokens)
3. **Load balancer configuration**:
```nginx
upstream oreader_backend {
    server 10.0.1.10:8080;
    server 10.0.1.11:8080;
    server 10.0.1.12:8080;
}

server {
    location / {
        proxy_pass http://oreader_backend;
    }
}
```

### Caching

Consider adding Redis for:
- Rate limiting (already has interface support)
- Session storage
- Feed content caching

### CDN for Frontend

Serve static assets via CDN:
- Build frontend: `cd web && npm run build`
- Upload `web/dist/` to CDN
- Configure CDN to proxy API requests to backend

## Troubleshooting

### Common Issues

**Issue**: Database connection fails
```bash
# Check database is running
sudo systemctl status mysql

# Test connection
mysql -u oreader -p -h localhost oreader

# Check credentials
echo $DATABASE_URL
```

**Issue**: Port already in use
```bash
# Find process using port 8080
sudo lsof -i :8080

# Kill process
sudo kill -9 <PID>

# Or change port
export SERVER_PORT=8081
```

**Issue**: JWT secret key error
```bash
# Ensure secret is at least 32 characters
export JWT_SECRET_KEY="new-secret-key-at-least-32-characters-long"
```

**Issue**: Migrations fail
```bash
# Check migration status
make migrate-version

# Rollback if needed
make migrate-down

# Re-run migrations
make migrate-up
```

### Performance Tuning

**Database optimization**:
```sql
-- Add indexes for better performance
CREATE INDEX idx_items_pub_date ON items(pub_date DESC);
CREATE INDEX idx_user_items_created_at ON user_item_states(created_at DESC);
```

**Go runtime optimization**:
```bash
# Set GOMAXPROCS
export GOMAXPROCS=4

# Enable GC optimizations
export GOGC=100
```

### Security Hardening

1. **Update dependencies regularly**:
```bash
go get -u ./...
go mod tidy
npm update
```

2. **Run vulnerability scans**:
```bash
govulncheck ./...
npm audit
```

3. **Configure firewall**:
```bash
# Allow only necessary ports
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

4. **Enable fail2ban** for brute-force protection

## Backup & Disaster Recovery

### Backup Strategy

1. **Database backups** (daily)
2. **Application backups** (on updates)
3. **Configuration backups** (.env files)

### Recovery Procedure

1. **Restore database**:
```bash
mysql -u oreader -p oreader < backup.sql
```

2. **Re-deploy application**:
```bash
git pull
make build
sudo systemctl restart oreader
```

3. **Verify service**:
```bash
curl http://localhost:8080/health
```

## Support

For deployment assistance:
- Documentation: [docs/](docs/)
- Issues: [GitHub Issues](https://github.com/yourusername/oreader/issues)
- Email: support@example.com

---

**Last Updated**: 2026-03-16
**Version**: 2.0.0
