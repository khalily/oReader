-- 001_init_schema.down.sql
-- Rollback initial schema

DROP TABLE IF EXISTS oauth_states;
DROP TABLE IF EXISTS import_jobs;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS user_item_states;
DROP TABLE IF EXISTS items;
DROP TABLE IF EXISTS user_feeds;
DROP TABLE IF EXISTS feeds;
DROP TABLE IF EXISTS users;
