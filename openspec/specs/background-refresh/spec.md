# Background Refresh Specification

## ADDED Requirements

### Requirement: Automatic periodic feed refresh
The system SHALL automatically refresh all subscribed feeds at configurable intervals.

#### Scenario: Periodic refresh trigger
- **WHEN** refresh interval elapses (default 15 minutes)
- **THEN** system iterates through all feeds
- **AND** system fetches and parses each feed
- **AND** system creates new items not in database
- **AND** system updates last_updated timestamp

#### Scenario: Refresh continues on individual feed error
- **WHEN** one feed fails to fetch or parse during periodic refresh
- **THEN** system logs the error
- **AND** system continues refreshing remaining feeds
- **AND** system does not stop the refresh process

### Requirement: Concurrent feed refresh
The system SHALL refresh multiple feeds concurrently for efficiency.

#### Scenario: Parallel processing
- **WHEN** periodic refresh runs
- **THEN** system processes multiple feeds in parallel (up to configurable limit)
- **AND** system uses goroutines with context timeout

#### Scenario: Timeout handling
- **WHEN** feed fetch exceeds timeout (default 30 seconds)
- **THEN** system cancels the request
- **AND** system logs timeout error
- **AND** system proceeds to next feed

### Requirement: Refresh on startup
The system SHALL refresh feeds on application startup.

#### Scenario: Startup refresh
- **WHEN** application starts
- **THEN** system waits for database connection
- **AND** system triggers initial refresh of all feeds
- **AND** system starts periodic ticker after initial refresh

### Requirement: Configurable refresh interval
The system SHALL allow configuration of refresh interval.

#### Scenario: Default interval
- **WHEN** no interval is configured
- **THEN** system uses 15 minute default

#### Scenario: Custom interval
- **WHEN** REFRESH_INTERVAL environment variable is set
- **THEN** system uses configured interval
- **AND** invalid values fall back to default

### Requirement: Graceful shutdown
The system SHALL handle graceful shutdown during refresh operations.

#### Scenario: Shutdown during refresh
- **WHEN** application receives SIGTERM or SIGINT
- **THEN** system stops refresh ticker
- **AND** system waits for in-progress refreshes to complete (with timeout)
- **AND** system exits cleanly
