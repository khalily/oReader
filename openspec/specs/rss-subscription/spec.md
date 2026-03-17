# RSS Subscription Specification

## ADDED Requirements

### Requirement: Add RSS feed subscription
The system SHALL allow authenticated users to subscribe to RSS/Atom feeds by URL.

#### Scenario: Successful subscription
- **WHEN** authenticated user submits valid RSS URL to POST /api/v1/feeds
- **THEN** system fetches and parses the feed
- **AND** system creates Feed record linked to user
- **AND** system creates Item records for all feed entries
- **AND** system returns feed details with items

#### Scenario: Invalid URL
- **WHEN** user submits malformed or non-URL string
- **THEN** system returns 400 Bad Request with validation error

#### Scenario: Unreachable URL
- **WHEN** user submits URL that cannot be fetched
- **THEN** system returns 400 Bad Request with fetch error

#### Scenario: Invalid RSS/Atom feed
- **WHEN** user submits URL that is not valid RSS/Atom
- **THEN** system returns 400 Bad Request with parse error

#### Scenario: Duplicate subscription
- **WHEN** user submits URL already subscribed
- **THEN** system returns 409 Conflict
- **AND** system does not create duplicate feed

### Requirement: List user subscriptions
The system SHALL allow authenticated users to list their subscriptions.

#### Scenario: List all subscriptions
- **WHEN** authenticated user calls GET /api/v1/feeds
- **THEN** system returns array of user's feeds
- **AND** each feed includes id, title, link, description, image_url, last_updated, item_count

### Requirement: Get subscription details
The system SHALL allow authenticated users to view a specific subscription.

#### Scenario: Get feed details
- **WHEN** authenticated user calls GET /api/v1/feeds/:id
- **THEN** system returns feed details
- **AND** feed belongs to requesting user

#### Scenario: Feed not found
- **WHEN** user requests non-existent feed ID
- **THEN** system returns 404 Not Found

#### Scenario: Feed belongs to another user
- **WHEN** user requests feed ID belonging to another user
- **THEN** system returns 404 Not Found (not 403 to prevent enumeration)

### Requirement: Delete subscription
The system SHALL allow authenticated users to unsubscribe from feeds.

#### Scenario: Successful deletion
- **WHEN** authenticated user calls DELETE /api/v1/feeds/:id for their own feed
- **THEN** system deletes feed and all associated items
- **AND** system returns 204 No Content

#### Scenario: Delete another user's feed
- **WHEN** user attempts to delete feed belonging to another user
- **THEN** system returns 404 Not Found

### Requirement: Manual feed refresh
The system SHALL allow authenticated users to manually refresh a subscription.

#### Scenario: Successful refresh
- **WHEN** authenticated user calls POST /api/v1/feeds/:id/refresh for their own feed
- **THEN** system fetches latest feed content
- **AND** system creates new items not already in database
- **AND** system updates last_updated field
- **AND** system returns updated feed with new item count

#### Scenario: No new items
- **WHEN** feed has no new items since last refresh
- **THEN** system returns feed with new_item_count: 0
