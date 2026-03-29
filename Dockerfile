# =============================================================================
# Stage 1: Build React Frontend
# =============================================================================
FROM node:22-alpine AS frontend-builder

WORKDIR /web

# Copy package files first for better caching
COPY web/package.json web/package-lock.json ./

# Install dependencies
RUN npm ci --only=production=false

# Copy frontend source
COPY web/ ./

# Build frontend
RUN npm run build

# =============================================================================
# Stage 2: Build Go Backend with Embedded Frontend
# =============================================================================
FROM golang:1.25-alpine AS backend-builder

# Install build dependencies
# No CGO needed - MySQL driver is pure Go
RUN apk add --no-cache git ca-certificates tzdata

# Pure Go build (no CGO)
ENV CGO_ENABLED=0

# Set Go proxy for reliable downloads
ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /src

# Copy go mod files for dependency caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Copy frontend build from previous stage
COPY --from=frontend-builder /web/dist ./cmd/server/dist

# Build the binary with embedded frontend (static, no CGO)
RUN go build \
    -ldflags="-w -s -X main.Version=1.0.0" \
    -o /oreader \
    ./cmd/server

# =============================================================================
# Stage 3: Final Minimal Image
# =============================================================================
FROM alpine:latest

# Install runtime dependencies
# - ca-certificates for HTTPS requests
# - wget for health check
# - tzdata for timezone support
# No libc6-compat needed - pure Go binary, no CGO
RUN apk add --no-cache ca-certificates wget tzdata

# Copy the binary
COPY --from=backend-builder /oreader /oreader

# Create non-root user
RUN addgroup -g 1000 oreader && \
    adduser -D -u 1000 -G oreader oreader

# Expose port
EXPOSE 8080

# Create data directory before switching users
RUN mkdir -p /data && chown -R oreader:oreader /data

# Health check using wget
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Use non-root user
USER oreader

# Run the binary
ENTRYPOINT ["/oreader"]
