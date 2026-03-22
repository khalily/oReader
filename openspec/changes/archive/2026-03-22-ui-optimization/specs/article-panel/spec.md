# Article Panel Specification

## ADDED Requirements

### Requirement: ArticlePanel displays article content

The ArticlePanel component SHALL display the selected article's title, metadata, and content.

#### Scenario: Article content renders

- **GIVEN** article with title, author, pub_date, and content
- **WHEN** ArticlePanel receives article id
- **THEN** title is displayed in header
- **AND** metadata (feed name, date, author) is shown
- **AND** content is rendered with prose styling

### Requirement: ArticlePanel shows loading state

When fetching article data, ArticlePanel SHALL display a loading indicator.

#### Scenario: Loading spinner shown

- **WHEN** ArticlePanel is fetching article
- **THEN** loading spinner is displayed
- **AND** no content is shown

### Requirement: ArticlePanel shows empty state

When no article is selected (`itemId=null`), ArticlePanel SHALL display an empty state message.

#### Scenario: No article selected

- **GIVEN** selectedItemId is null
- **WHEN** ArticlePanel renders
- **THEN** "Select an article to read" message is displayed
- **AND** no loading spinner

### Requirement: ArticlePanel shows error state

When article fetch fails, ArticlePanel SHALL display an error message.

#### Scenario: Fetch error

- **WHEN** article fetch returns error
- **THEN** error message is displayed
- **AND** retry button is shown

### Requirement: ArticlePanel supports actions

ArticlePanel SHALL provide Read/Unread, Star/Unstar, and Open External Link actions.

#### Scenario: Toggle read status

- **GIVEN** article is unread
- **WHEN** user clicks Read button
- **THEN** article is marked as read
- **AND** button changes to "Unread" state

#### Scenario: Toggle star status

- **GIVEN** article is not starred
- **WHEN** user clicks Star button
- **THEN** article is starred
- **AND** star icon fills yellow

#### Scenario: Open external link

- **WHEN** user clicks external link button
- **THEN** article URL opens in new tab (`target="_blank"`)

### Requirement: ArticlePanel shows summary section

ArticlePanel SHALL display a collapsible summary section based on article description.

#### Scenario: Summary shown by default

- **GIVEN** article has description
- **WHEN** ArticlePanel renders
- **THEN** summary section is visible below title

#### Scenario: Summary can be collapsed

- **GIVEN** summary is expanded
- **WHEN** user clicks collapse button
- **THEN** summary section is hidden
- **AND** expand button is shown

### Requirement: Mobile back button

On mobile viewport (<768px), ArticlePanel SHALL display a back button.

#### Scenario: Mobile back button

- **GIVEN** viewport width < 768px
- **WHEN** ArticlePanel renders
- **THEN** back button is visible in header

#### Scenario: Back button closes panel

- **WHEN** user clicks back button on mobile
- **THEN** ArticlePanel is hidden
- **AND** ItemList is shown
- **AND** URL `?id=` parameter is removed
