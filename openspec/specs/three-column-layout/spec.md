# Three Column Layout Specification

## ADDED Requirements

### Requirement: Desktop displays three-column layout

On screens >= 1024px (lg breakpoint), the ItemsPage SHALL display three columns: Sidebar | ItemList | ArticlePanel.

#### Scenario: Desktop viewport shows three columns

- **GIVEN** viewport width >= 1024px
- **WHEN** user navigates to `/items`
- **THEN** Sidebar (256px) is visible on left
- **AND** ItemList (320px) is visible in center
- **AND** ArticlePanel (flex-1) is visible on right

### Requirement: Tablet displays two-column layout

On screens 768-1023px (md breakpoint), the ItemsPage SHALL display Sidebar + (ItemList OR ArticlePanel).

#### Scenario: Tablet viewport shows two columns

- **GIVEN** viewport width 768-1023px
- **WHEN** user navigates to `/items`
- **THEN** Sidebar is visible
- **AND** ItemList is visible
- **AND** ArticlePanel is hidden (until article selected)

#### Scenario: Tablet article selection

- **GIVEN** viewport width 768-1023px
- **AND** user is viewing ItemList
- **WHEN** user selects an article
- **THEN** ItemList is hidden
- **AND** ArticlePanel is visible
- **AND** back button is shown in ArticlePanel

### Requirement: Mobile displays single column

On screens < 768px (sm breakpoint), the ItemsPage SHALL display single column with drawer menu.

#### Scenario: Mobile viewport shows single column

- **GIVEN** viewport width < 768px
- **WHEN** user navigates to `/items`
- **THEN** Sidebar is hidden
- **AND** mobile menu button is visible
- **AND** ItemList fills the screen

#### Scenario: Mobile opens sidebar drawer

- **GIVEN** viewport width < 768px
- **WHEN** user clicks mobile menu button
- **THEN** Sidebar appears in drawer overlay
- **AND** backdrop is shown behind drawer

### Requirement: URL reflects selected article

When an article is selected, the URL SHALL include `?id=<article-id>` query parameter.

#### Scenario: Article selection updates URL

- **WHEN** user clicks article with id `abc123`
- **THEN** URL updates to `/items?filter=all&id=abc123`

#### Scenario: Direct URL navigation

- **GIVEN** URL is `/items?id=abc123`
- **WHEN** page loads
- **THEN** article `abc123` is selected
- **AND** ArticlePanel displays the article content

### Requirement: Browser history navigation works

Browser back/forward SHALL navigate between article selections.

#### Scenario: Back button returns to previous state

- **GIVEN** user selected article A, then article B
- **WHEN** user clicks browser back
- **THEN** article A is selected
- **AND** URL shows `?id=article-a`
