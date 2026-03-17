# OAuth Integration Specification

## ADDED Requirements

### Requirement: GitHub OAuth login initiation
The system SHALL provide an endpoint to initiate GitHub OAuth flow.

#### Scenario: Initiate GitHub OAuth
- **WHEN** user calls GET /api/v1/auth/github
- **THEN** system generates secure state parameter
- **AND** system stores state in session or cookie
- **AND** system redirects to GitHub authorization URL with client_id, redirect_uri, scope, state

### Requirement: GitHub OAuth callback handling
The system SHALL handle GitHub OAuth callback and create/link user account.

#### Scenario: Successful GitHub OAuth
- **WHEN** GitHub redirects to GET /api/v1/auth/github/callback with valid code and state
- **THEN** system verifies state matches stored value
- **AND** system exchanges code for GitHub access token
- **AND** system fetches GitHub user profile
- **AND** system finds or creates local user by github_id
- **AND** system sets access_token and refresh_token cookies
- **AND** system redirects to frontend with user profile

#### Scenario: Invalid state parameter
- **WHEN** callback has state that doesn't match stored value
- **THEN** system returns 400 Bad Request
- **AND** system does not proceed with OAuth flow

#### Scenario: GitHub API error
- **WHEN** GitHub returns error or fails to respond
- **THEN** system returns appropriate error
- **AND** system redirects to frontend with error message

### Requirement: OAuth user account creation
The system SHALL create user accounts for new OAuth users.

#### Scenario: New GitHub user
- **WHEN** GitHub OAuth succeeds for user without existing account
- **THEN** system creates new User with:
  - id: UUID
  - email: GitHub email (if available and public)
  - nickname: GitHub login or name
  - avatar_url: GitHub avatar URL
  - auth_provider: "github"
  - github_id: GitHub user ID
  - password_hash: null

#### Scenario: Existing user with same email
- **WHEN** GitHub OAuth returns email matching existing local account
- **THEN** system links GitHub account to existing user
- **AND** system sets github_id on existing user
- **AND** system does not create duplicate account

### Requirement: OAuth user profile mapping
The system SHALL map GitHub profile fields to local user fields.

#### Scenario: Profile field mapping
- **WHEN** user authenticates via GitHub
- **THEN** system maps:
  - GitHub id → github_id
  - GitHub login → nickname (if name not available)
  - GitHub name → nickname (preferred)
  - GitHub avatar_url → avatar_url
  - GitHub email → email (if public)

### Requirement: Reserved OAuth endpoints for future providers
The system SHALL reserve endpoint patterns for additional OAuth providers.

#### Scenario: Future provider endpoints
- **WHEN** system is deployed
- **THEN** following endpoint patterns are reserved:
  - GET /api/v1/auth/google
  - GET /api/v1/auth/google/callback
- **AND** endpoints return 501 Not Implemented until implemented
