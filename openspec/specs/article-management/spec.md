# Article Management Specification

> **Note**: `is_starred` and `is_read` are stored in `UserItemState` table, not on `Item` directly.
> This allows multiple users to have independent read/starred states for the same article.
> See `specs/data-model/spec.md` for details.

## ADDED Requirements

### Requirement: List all articles across subscriptions
The system SHALL allow authenticated users to list all articles from their subscriptions.

#### Scenario: List all articles
- **WHEN** authenticated user calls GET /api/v1/items
- **THEN** system returns paginated array of items from user's subscribed feeds
- **AND** items are sorted by published_at descending
- **AND** each item includes id, title, link, description, feed_id, feed_title, published_at
- **AND** each item includes is_starred, is_read from UserItemState (defaults to false if no state exists)

#### Scenario: Filter by read status
- **WHEN** user calls GET /api/v1/items?read=false
- **THEN** system returns only unread items

#### Scenario: Filter by starred status
- **WHEN** user calls GET /api/v1/items?starred=true
- **THEN** system returns only starred items

### Requirement: List articles for specific feed
The system SHALL allow authenticated users to list articles from a specific subscription.

#### Scenario: List feed articles
- **WHEN** authenticated user calls GET /api/v1/feeds/:feed_id/items
- **THEN** system returns paginated array of items for that feed
- **AND** feed belongs to requesting user

### Requirement: Get article details
The system SHALL allow authenticated users to view full article content.

#### Scenario: Get article
- **WHEN** authenticated user calls GET /api/v1/items/:id
- **THEN** system returns full item details including content field
- **AND** item belongs to user's feed

#### Scenario: Article not found
- **WHEN** user requests non-existent item ID
- **THEN** system returns 404 Not Found

#### Scenario: Article belongs to another user
- **WHEN** user requests item from another user's feed
- **THEN** system returns 404 Not Found

### Requirement: Star/unstar articles
The system SHALL allow authenticated users to mark articles as favorites.

#### Scenario: Star article
- **WHEN** authenticated user calls PUT /api/v1/items/:id/star with { starred: true }
- **THEN** system sets is_starred = true
- **AND** system returns updated item

#### Scenario: Unstar article
- **WHEN** authenticated user calls PUT /api/v1/items/:id/star with { starred: false }
- **THEN** system sets is_starred = false
- **AND** system returns updated item

### Requirement: Mark articles as read/unread
The system SHALL allow authenticated users to mark articles as read.

#### Scenario: Mark as read
- **WHEN** authenticated user calls PUT /api/v1/items/:id/read with { read: true }
- **THEN** system sets is_read = true
- **AND** system returns updated item

#### Scenario: Mark as unread
- **WHEN** authenticated user calls PUT /api/v1/items/:id/read with { read: false }
- **THEN** system sets is_read = false
- **AND** system returns updated item

### Requirement: Mark all feed articles as read
The system SHALL allow authenticated users to mark all articles in a feed as read.

#### Scenario: Mark all read
- **WHEN** authenticated user calls POST /api/v1/feeds/:id/read-all
- **THEN** system sets is_read = true for all items in feed
- **AND** system returns count of updated items
