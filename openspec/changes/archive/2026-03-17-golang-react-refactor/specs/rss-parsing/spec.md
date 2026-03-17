# RSS Parsing Specification

## ADDED Requirements

### Requirement: Parse RSS 2.0 feeds
The system SHALL parse RSS 2.0 format feeds and extract feed and item data.

#### Scenario: Parse valid RSS 2.0 feed
- **WHEN** system parses a valid RSS 2.0 feed
- **THEN** system extracts feed title, link, description, lastBuildDate
- **AND** system extracts items with title, link, description, pubDate, author, content:encoded

### Requirement: Parse Atom 1.0 feeds
The system SHALL parse Atom 1.0 format feeds and extract feed and entry data.

#### Scenario: Parse valid Atom feed
- **WHEN** system parses a valid Atom feed
- **THEN** system extracts feed title, link, subtitle, updated
- **AND** system extracts entries with title, link, summary, content, published, updated, author

### Requirement: Handle malformed feeds gracefully
The system SHALL handle malformed or incomplete feeds without crashing.

#### Scenario: Missing optional fields
- **WHEN** feed is missing optional fields (description, author, etc.)
- **THEN** system parses successfully with empty/null values for missing fields

#### Scenario: Invalid XML
- **WHEN** feed contains invalid XML
- **THEN** system returns parse error without crashing

### Requirement: Deduplicate feed items
The system SHALL not create duplicate items for the same feed entry.

#### Scenario: Item with same GUID
- **WHEN** feed contains item with GUID that already exists for this feed
- **THEN** system skips creating duplicate item
- **AND** system does not update existing item

#### Scenario: Item without GUID uses link
- **WHEN** feed item has no GUID but has link
- **THEN** system uses link as unique identifier
- **AND** system does not create duplicate for same link

### Requirement: Extract feed favicon
The system SHALL attempt to extract favicon URL for feed display.

#### Scenario: Favicon available
- **WHEN** feed source website has favicon at /favicon.ico
- **THEN** system stores favicon URL as image_url

#### Scenario: Favicon not available
- **WHEN** feed source website has no favicon
- **THEN** system stores null for image_url
- **AND** parsing still succeeds

### Requirement: Sanitize item content
The system SHALL store raw content but provide sanitized excerpts for display.

#### Scenario: Store raw content
- **WHEN** item has HTML content
- **THEN** system stores raw HTML in content field

#### Scenario: Generate description excerpt
- **WHEN** item is created
- **THEN** system generates plain text description (stripped HTML, truncated to 200 chars)
