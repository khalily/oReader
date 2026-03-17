# Data Model Specification

## ADDED Requirements

### Requirement: Multi-tenant feed sharing
The system SHALL allow multiple users to subscribe to the same RSS feed URL without duplicating content.

#### Scenario: Shared feed subscription
- **WHEN** User A and User B both subscribe to `https://example.com/feed.xml`
- **THEN** only one Feed record exists with `feed_url = https://example.com/feed.xml`
- **AND** only one set of Item records exists for this feed
- **AND** User A and User B each have their own UserFeed record pointing to this shared Feed

#### Scenario: Independent user states
- **WHEN** User A marks an item as read
- **THEN** User B's view of the same item remains unread
- **AND** UserItemState records are independent per user

### Requirement: User-specific item states
The system SHALL store `is_starred` and `is_read` states per user, not per item.

#### Scenario: Star item
- **WHEN** authenticated user marks an item as starred
- **THEN** system creates or updates UserItemState record for (user_id, item_id)
- **AND** other users' states for this item are unaffected

#### Scenario: Read item
- **WHEN** authenticated user marks an item as read
- **THEN** system creates or updates UserItemState record with is_read=true, read_at=now
- **AND** other users' states for this item are unaffected

### Requirement: Item deduplication by GUID
The system SHALL use RSS item GUID to prevent duplicate items.

#### Scenario: New item with new GUID
- **WHEN** RSS feed contains item with GUID not in database
- **THEN** system creates new Item record

#### Scenario: Existing item GUID
- **WHEN** RSS feed contains item with GUID already in database
- **THEN** system updates existing Item record (if content changed)
- **AND** does not create duplicate Item

## Data Model Definition

### Tables

#### users
| Column | Type | Description |
|--------|------|-------------|
| id | UUID v7 | Primary key |
| email | VARCHAR(255) | Unique, not null |
| password_hash | VARCHAR(255) | Nullable for OAuth users |
| nickname | VARCHAR(100) | Display name |
| avatar_url | VARCHAR(500) | Nullable |
| auth_provider | VARCHAR(20) | 'local' or 'github' |
| github_id | VARCHAR(50) | Nullable, unique |
| created_at | TIMESTAMP | |
| updated_at | TIMESTAMP | |

#### feeds (Shared RSS feed metadata)
| Column | Type | Description |
|--------|------|-------------|
| id | UUID v7 | Primary key |
| feed_url | VARCHAR(500) | Unique, the RSS/Atom URL |
| title | VARCHAR(255) | Feed title |
| description | TEXT | Nullable |
| image_url | VARCHAR(500) | Nullable, feed image/favicon |
| site_url | VARCHAR(500) | Nullable, link to website |
| last_fetched_at | TIMESTAMP | Nullable |
| last_fetch_status | VARCHAR(20) | 'success', 'error', 'timeout' |
| last_fetch_error | TEXT | Nullable, error message |
| consecutive_failures | INTEGER | Default 0 |
| created_at | TIMESTAMP | |
| updated_at | TIMESTAMP | |

#### user_feeds (User subscription relationship)
| Column | Type | Description |
|--------|------|-------------|
| id | UUID v7 | Primary key |
| user_id | UUID v7 | FK -> users.id |
| feed_id | UUID v7 | FK -> feeds.id |
| position | INTEGER | Default 0, for ordering |
| created_at | TIMESTAMP | |

**Unique Constraint**: (user_id, feed_id)

#### items (Shared article content)
| Column | Type | Description |
|--------|------|-------------|
| id | UUID v7 | Primary key |
| feed_id | UUID v7 | FK -> feeds.id |
| guid | VARCHAR(500) | Unique identifier from RSS |
| title | VARCHAR(255) | |
| link | VARCHAR(500) | Article URL |
| description | TEXT | Nullable, short summary |
| content | TEXT | Nullable, full content |
| creator | VARCHAR(255) | Nullable, author name |
| pub_date | TIMESTAMP | Nullable |
| created_at | TIMESTAMP | |
| updated_at | TIMESTAMP | |

**Unique Constraint**: (feed_id, guid)
**Index**: feed_id, pub_date

#### user_item_states (Per-user item states)
| Column | Type | Description |
|--------|------|-------------|
| id | UUID v7 | Primary key |
| user_id | UUID v7 | FK -> users.id |
| item_id | UUID v7 | FK -> items.id |
| is_starred | BOOLEAN | Default false |
| is_read | BOOLEAN | Default false |
| read_at | TIMESTAMP | Nullable |
| created_at | TIMESTAMP | |
| updated_at | TIMESTAMP | |

**Unique Constraint**: (user_id, item_id)
**Index**: user_id, is_starred, is_read

## Relationships

```
User ──1:N──▶ UserFeed ──N:1──▶ Feed ──1:N──▶ Item ──1:1──▶ UserItemState
  │                                                      ▲
  └──────────────────────────────────────────────────────┘
```

## Migration Considerations

### From Flask (Current) to Go (New)

The current Flask implementation has:
- Feed.user_id (each user has their own Feed copy)
- Item.feed_id (items belong to user-specific feeds)
- Item.star (stored on Item directly)

This means:
1. Same RSS URL is stored multiple times (one per user)
2. Same article content is stored multiple times
3. User states are isolated but at cost of data duplication

### Migration Path

1. **Export**: Use OPML export from Flask version
2. **Import**: Import into new Go version
3. **States**: Starred items cannot be migrated (different ID scheme)

### Benefits of New Model

| Aspect | Current (Flask) | New (Go) |
|--------|-----------------|----------|
| Storage | Duplicated per user | Shared, minimal duplication |
| User States | Per-item (works but inefficient) | Explicit join table |
| Feed Updates | Each user's feed refreshed separately | Single refresh updates all subscribers |
| Scalability | Linear growth with users | Sublinear growth (shared feeds) |
