# Stats API Specification

## ADDED Requirements

### Requirement: User can retrieve article statistics

The system SHALL provide an API endpoint that returns the user's article statistics including total count, unread count, starred count, and today's count.

#### Scenario: Authenticated user retrieves stats

- **WHEN** authenticated user sends `GET /api/v1/stats`
- **THEN** system returns JSON with `total`, `unread`, `starred`, `today` counts

#### Scenario: Unauthenticated user is rejected

- **WHEN** unauthenticated user sends `GET /api/v1/stats`
- **THEN** system returns `401 Unauthorized`

### Requirement: Stats response format

The stats response SHALL include the following fields:

| Field | Type | Description |
|-------|------|-------------|
| total | int64 | Total articles across all subscribed feeds |
| unread | int64 | Articles not marked as read |
| starred | int64 | Articles marked as starred |
| today | int64 | Articles published today (UTC) |

#### Scenario: Response structure validation

- **WHEN** user retrieves stats successfully
- **THEN** response body matches `{"total": <int>, "unread": <int>, "starred": <int>, "today": <int>}`

### Requirement: Today calculation uses UTC

The "today" count SHALL be calculated based on UTC midnight (`pub_date >= start of today UTC`).

#### Scenario: Articles from today are counted

- **GIVEN** article with `pub_date` >= today 00:00:00 UTC
- **WHEN** user retrieves stats
- **THEN** article is included in `today` count

#### Scenario: Articles from yesterday are not counted

- **GIVEN** article with `pub_date` < today 00:00:00 UTC
- **WHEN** user retrieves stats
- **THEN** article is NOT included in `today` count

### Requirement: Stats are user-scoped

Statistics SHALL only include articles from feeds the user is subscribed to.

#### Scenario: Stats reflect user's subscriptions only

- **GIVEN** user A subscribed to feed X and user B subscribed to feed Y
- **WHEN** user A retrieves stats
- **THEN** only articles from feed X are counted
