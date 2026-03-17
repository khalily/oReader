# Feed Import/Export Specification

## ADDED Requirements

### Requirement: Export feeds to OPML format
The system SHALL allow authenticated users to export their subscriptions as OPML 2.0 file.

#### Scenario: Successful export
- **WHEN** authenticated user calls GET /api/v1/feeds/export
- **THEN** system generates OPML 2.0 formatted XML
- **AND** response has Content-Type: application/xml
- **AND** response has Content-Disposition: attachment; filename="oreader-subscriptions.xml"
- **AND** OPML includes all user's feed subscriptions

#### Scenario: Empty subscriptions export
- **WHEN** user with no subscriptions calls export endpoint
- **THEN** system returns valid OPML with empty body
- **AND** response status is 200 OK

### Requirement: OPML export format compliance
The system SHALL generate OPML 2.0 compliant XML.

#### Scenario: OPML structure
- **WHEN** system generates OPML export
- **THEN** XML includes:
  - XML declaration with UTF-8 encoding
  - opml element with version="2.0"
  - head element with title and dateCreated
  - body element with outline elements for each feed
- **AND** each outline includes:
  - type="rss"
  - text: feed title
  - xmlUrl: feed URL
  - htmlUrl: feed website link (if available)

### Requirement: Import feeds from OPML file
The system SHALL allow authenticated users to import subscriptions from OPML file.

#### Scenario: Successful import
- **WHEN** authenticated user uploads valid OPML file to POST /api/v1/feeds/import
- **THEN** system parses OPML and extracts feed URLs
- **AND** system creates subscriptions for feeds not already subscribed
- **AND** system returns import summary with counts (added, skipped, failed)

#### Scenario: Import with duplicate feeds
- **WHEN** OPML contains feed URL user already subscribes to
- **THEN** system skips that feed
- **AND** system includes in skipped count in response

#### Scenario: Import with invalid feed URLs
- **WHEN** OPML contains invalid or unreachable feed URLs
- **THEN** system attempts to parse each URL
- **AND** system adds valid feeds only
- **AND** system includes failed URLs in response with error reasons

### Requirement: OPML import validation
The system SHALL validate OPML file structure and content.

#### Scenario: Invalid OPML format
- **WHEN** user uploads malformed XML
- **THEN** system returns 400 Bad Request
- **AND** response includes parse error details

#### Scenario: Missing required elements
- **WHEN** OPML is valid XML but missing required structure
- **THEN** system returns 400 Bad Request
- **AND** response indicates missing elements

#### Scenario: File size limit
- **WHEN** uploaded file exceeds 1MB
- **THEN** system returns 413 Payload Too Large

### Requirement: Bulk import with rate awareness
The system SHALL handle large imports without overwhelming external feed servers.

#### Scenario: Large import processing
- **WHEN** user imports OPML with many feeds (>50)
- **THEN** system processes feeds in batches
- **AND** system returns immediate response with job ID
- **AND** system processes feeds asynchronously
- **AND** system provides status endpoint to check progress

#### Scenario: Check import progress
- **WHEN** user calls GET /api/v1/feeds/import/:job_id/status
- **THEN** system returns import progress (total, processed, added, failed)
- **AND** system indicates if import is complete or in progress

### Requirement: OPML version compatibility
The system SHALL support common OPML versions for import.

#### Scenario: OPML 1.0 import
- **WHEN** user uploads OPML 1.0 format
- **THEN** system parses successfully
- **AND** system extracts feed information

#### Scenario: OPML 2.0 import
- **WHEN** user uploads OPML 2.0 format
- **THEN** system parses successfully
- **AND** system extracts all available metadata

#### Scenario: No version specified
- **WHEN** OPML has no version attribute
- **THEN** system attempts to parse as best effort
- **AND** system extracts feeds if structure is valid
