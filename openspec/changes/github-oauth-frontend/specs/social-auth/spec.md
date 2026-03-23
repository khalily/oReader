## ADDED Requirements

### Requirement: GitHub login button on login page
The system SHALL display a "Sign in with GitHub" button on the login page, positioned above the email/password form.

#### Scenario: User sees GitHub login option
- **WHEN** user navigates to the login page
- **THEN** a "Sign in with GitHub" button is displayed above the email input field

#### Scenario: User clicks GitHub login button
- **WHEN** user clicks the "Sign in with GitHub" button
- **THEN** the browser navigates to `/api/v1/auth/github`

### Requirement: GitHub login button on register page
The system SHALL display a "Sign in with GitHub" button on the register page, positioned above the email/password form.

#### Scenario: User sees GitHub registration option
- **WHEN** user navigates to the register page
- **THEN** a "Sign in with GitHub" button is displayed above the email input field

#### Scenario: User clicks GitHub register button
- **WHEN** user clicks the "Sign in with GitHub" button
- **THEN** the browser navigates to `/api/v1/auth/github`

### Requirement: OAuth error handling
The system SHALL display appropriate error messages when OAuth fails.

#### Scenario: User cancels GitHub authorization
- **WHEN** user denies authorization on GitHub's consent screen
- **THEN** user is redirected to login page with error message "GitHub 登录已取消"

#### Scenario: OAuth state expired
- **WHEN** OAuth state parameter has expired
- **THEN** user is redirected to login page with error message "登录已过期，请重试"

#### Scenario: GitHub service unavailable
- **WHEN** GitHub OAuth service returns an error
- **THEN** user is redirected to login page with error message "GitHub 服务暂时不可用"

### Requirement: Social login button styling
The system SHALL render the social login button with consistent styling.

#### Scenario: Button visual appearance
- **WHEN** the social login button is rendered
- **THEN** it displays with GitHub's official branding (black background, white text, GitHub icon)

#### Scenario: Button is accessible
- **WHEN** the social login button is rendered
- **THEN** it has appropriate ARIA labels and keyboard focus indicators
