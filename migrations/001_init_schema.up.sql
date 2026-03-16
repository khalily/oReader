-- 001_init_schema.up.sql
-- Initial schema for oReader

-- Enable foreign keys for SQLite
PRAGMA foreign_keys = ON;

-- Users table
CREATE TABLE users (
    id VARCHAR(36) PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    nickname VARCHAR(100),
    avatar_url VARCHAR(500),
    auth_provider VARCHAR(50) DEFAULT 'email',
    github_id VARCHAR(100),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_github_id ON users(github_id);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);

-- Feeds table (shared among users)
CREATE TABLE feeds (
    id VARCHAR(36) PRIMARY KEY,
    feed_url VARCHAR(500) NOT NULL UNIQUE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    image_url VARCHAR(500),
    last_fetched_at DATETIME,
    last_fetch_status VARCHAR(20),
    consecutive_failures INTEGER DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

CREATE INDEX idx_feeds_url ON feeds(feed_url);
CREATE INDEX idx_feeds_deleted_at ON feeds(deleted_at);

-- User feeds table (subscription relationship)
CREATE TABLE user_feeds (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    feed_id VARCHAR(36) NOT NULL,
    position INTEGER DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (feed_id) REFERENCES feeds(id) ON DELETE CASCADE,
    UNIQUE(user_id, feed_id)
);

CREATE INDEX idx_user_feeds_user_id ON user_feeds(user_id);
CREATE INDEX idx_user_feeds_feed_id ON user_feeds(feed_id);
CREATE INDEX idx_user_feeds_deleted_at ON user_feeds(deleted_at);

-- Items table (RSS feed items, shared)
CREATE TABLE items (
    id VARCHAR(36) PRIMARY KEY,
    feed_id VARCHAR(36) NOT NULL,
    guid VARCHAR(500) NOT NULL,
    title VARCHAR(500) NOT NULL,
    link VARCHAR(500) NOT NULL,
    description TEXT,
    content LONGTEXT,
    pub_date DATETIME,
    creator VARCHAR(255),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    FOREIGN KEY (feed_id) REFERENCES feeds(id) ON DELETE CASCADE,
    UNIQUE(feed_id, guid)
);

CREATE INDEX idx_items_feed_id ON items(feed_id);
CREATE INDEX idx_items_pub_date ON items(pub_date);
CREATE INDEX idx_items_deleted_at ON items(deleted_at);

-- User item states (per-user read/star status)
CREATE TABLE user_item_states (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    item_id VARCHAR(36) NOT NULL,
    is_starred BOOLEAN DEFAULT FALSE,
    is_read BOOLEAN DEFAULT FALSE,
    read_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (item_id) REFERENCES items(id) ON DELETE CASCADE,
    UNIQUE(user_id, item_id)
);

CREATE INDEX idx_user_item_states_user_id ON user_item_states(user_id);
CREATE INDEX idx_user_item_states_item_id ON user_item_states(item_id);
CREATE INDEX idx_user_item_states_starred ON user_item_states(user_id, is_starred);
CREATE INDEX idx_user_item_states_deleted_at ON user_item_states(deleted_at);

-- Refresh tokens table
CREATE TABLE refresh_tokens (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    revoked BOOLEAN DEFAULT FALSE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_deleted_at ON refresh_tokens(deleted_at);

-- Import jobs table
CREATE TABLE import_jobs (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    total_feeds INTEGER DEFAULT 0,
    processed INTEGER DEFAULT 0,
    failed INTEGER DEFAULT 0,
    error TEXT,
    started_at DATETIME,
    ended_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_import_jobs_user_id ON import_jobs(user_id);
CREATE INDEX idx_import_jobs_status ON import_jobs(status);
CREATE INDEX idx_import_jobs_deleted_at ON import_jobs(deleted_at);

-- OAuth states table
CREATE TABLE oauth_states (
    id VARCHAR(36) PRIMARY KEY,
    state VARCHAR(64) NOT NULL UNIQUE,
    provider VARCHAR(50) NOT NULL,
    user_id VARCHAR(36),
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

CREATE INDEX idx_oauth_states_state ON oauth_states(state);
CREATE INDEX idx_oauth_states_deleted_at ON oauth_states(deleted_at);
