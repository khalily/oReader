# Today Filter Specification

## ADDED Requirements

### Requirement: User can filter articles published today

The system SHALL allow users to filter articles to show only those published today.

#### Scenario: Filter by published_today parameter

- **WHEN** user sends `GET /api/v1/items?published_today=true`
- **THEN** system returns only articles with `pub_date >= today 00:00:00 UTC`

#### Scenario: Published_today combined with other filters

- **WHEN** user sends `GET /api/v1/items?published_today=true&read=false`
- **THEN** system returns today's unread articles

### Requirement: Today filter uses UTC timezone

The "today" boundary SHALL be calculated using UTC midnight.

#### Scenario: Article at UTC boundary

- **GIVEN** current time is `2026-03-22 15:00:00 UTC`
- **AND** article A has `pub_date = 2026-03-22 00:00:00 UTC`
- **AND** article B has `pub_date = 2026-03-21 23:59:59 UTC`
- **WHEN** user filters by `published_today=true`
- **THEN** article A is included
- **AND** article B is NOT included

### Requirement: Frontend Today filter button

The Sidebar SHALL display a "Today" filter button that filters articles by today's date.

#### Scenario: Click Today filter

- **WHEN** user clicks "Today" filter button in Sidebar
- **THEN** URL updates to `?filter=today`
- **AND** ItemList shows only today's articles
- **AND** Today filter shows article count

#### Scenario: Today filter shows count

- **GIVEN** 5 articles were published today
- **WHEN** Sidebar renders
- **THEN** Today filter displays badge with "5"
