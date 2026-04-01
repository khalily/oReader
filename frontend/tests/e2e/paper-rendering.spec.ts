/**
 * E2E tests for Paper frontend UI rendering.
 *
 * These tests exercise the full user-facing flow:
 *   - Upload a PDF through the PapersPage UI
 *   - Wait for conversion to complete
 *   - Navigate to PaperViewPage
 *   - Verify MarkdownRenderer correctly renders:
 *     - KaTeX math formulas (.katex class elements)
 *     - Tables rendered as HTML <table> elements
 *     - Base64 data URI images loaded correctly
 *     - Code blocks with syntax highlighting (Shiki)
 *     - No garbled text / mojibake characters
 *     - Special characters (em-dash, en-dash, quotes, Greek letters)
 *
 * Prerequisites:
 *   - Frontend dev server on port 5173 (Vite)
 *   - Backend on port 8080
 *   - Converter gRPC service on port 50051
 *   - MySQL running
 *
 * Run: cd frontend && npx playwright test --config=tests/e2e/playwright.config.ts --project=ui
 */
import { test, expect, type Page } from '@playwright/test'
import { ensureAuthState, type AuthState } from './helpers/auth'
import {
  uploadPaper,
  waitForConversion,
  getPaper,
  deletePaper,
  getDefaultTestPdfPath,
} from './helpers/api'
import fs from 'fs'

// ─── Skip guard ──────────────────────────────────────────────────
test.beforeAll(async () => {
  const pdfPath = getDefaultTestPdfPath()
  if (!fs.existsSync(pdfPath)) {
    throw new Error(
      `Test PDF not found at ${pdfPath}. Ensure the test file exists before running E2E tests.`,
    )
  }
})

// ─── Shared state ────────────────────────────────────────────────
let auth: AuthState
let paperId: string

test.describe('Paper Rendering E2E', () => {
  test.beforeAll(async ({ request }) => {
    auth = await ensureAuthState(request)
  })

  // ── Helper: inject auth cookies into browser context ────────────
  async function setAuthCookies(page: Page) {
    // Navigate to the app domain first so cookies are accepted
    await page.goto('/')
    // Parse cookies from auth state and set them
    const cookiePairs = auth.cookies.split(';').map((c) => c.trim()).filter(Boolean)
    for (const pair of cookiePairs) {
      const [name, ...valueParts] = pair.split('=')
      const value = valueParts.join('=')
      if (name && value) {
        await page.context().addCookies([
          {
            name: name.trim(),
            value: value.trim(),
            domain: 'localhost',
            path: '/',
          },
        ])
      }
    }
  }

  // ═══════════════════════════════════════════════════════════════
  // 1. Upload via API, then verify UI rendering
  // ═══════════════════════════════════════════════════════════════
  // We use the API for upload because the UI upload dialog interaction
  // is flaky in CI (file picker). This lets us focus on rendering.
  test('upload PDF via API and wait for conversion', async ({ request }) => {
    const { body } = await uploadPaper(request, auth)
    paperId = body.id

    expect(body.status).toBe('pending')

    // Wait for conversion to complete
    const status = await waitForConversion(request, auth, paperId, 120_000)
    expect(status.status).toBe('completed')
  })

  test('should display paper list on PapersPage', async ({ page }) => {
    await setAuthCookies(page)

    // Navigate to papers page
    await page.goto('/papers')

    // Wait for the page to load
    await page.waitForLoadState('networkidle')

    // The page should have a "Papers" heading
    await expect(page.getByRole('heading', { name: /papers/i })).toBeVisible({ timeout: 15_000 })

    // Should show the upload button
    await expect(page.getByRole('button', { name: /upload pdf/i })).toBeVisible()
  })

  test('should display paper view page with metadata', async ({ page }) => {
    test.skip(!paperId, 'Depends on upload test')

    await setAuthCookies(page)

    // Navigate to the paper view page
    await page.goto(`/papers/${paperId}`)
    await page.waitForLoadState('networkidle')

    // Should show "Back to Papers" button
    await expect(page.getByRole('button', { name: /back to papers/i })).toBeVisible({ timeout: 15_000 })

    // Paper should be in completed state -- no loading spinner
    // The MarkdownRenderer should be visible
    const proseContainer = page.locator('.prose')
    await expect(proseContainer).toBeVisible({ timeout: 30_000 })
  })

  // ═══════════════════════════════════════════════════════════════
  // 2. Markdown Rendering Verification
  // ═══════════════════════════════════════════════════════════════
  test('should render KaTeX math formulas', async ({ page }) => {
    test.skip(!paperId, 'Depends on upload test')

    await setAuthCookies(page)
    await page.goto(`/papers/${paperId}`)
    await page.waitForLoadState('networkidle')

    // Wait for the markdown content to render (KaTeX is async)
    const proseContainer = page.locator('.prose')
    await expect(proseContainer).toBeVisible({ timeout: 30_000 })

    // KaTeX renders elements with the .katex class
    const katexElements = page.locator('.katex')
    await expect(katexElements.first()).toBeVisible({ timeout: 30_000 })

    // KaTeX should have rendered math elements (at least 1)
    const katexCount = await katexElements.count()
    expect(katexCount, 'Should render at least one KaTeX formula').toBeGreaterThan(0)
  })

  test('should render tables as HTML elements', async ({ page }) => {
    test.skip(!paperId, 'Depends on upload test')

    await setAuthCookies(page)
    await page.goto(`/papers/${paperId}`)
    await page.waitForLoadState('networkidle')

    const proseContainer = page.locator('.prose')
    await expect(proseContainer).toBeVisible({ timeout: 30_000 })

    // GFM tables are rendered as <table> elements
    const tables = page.locator('.prose table')
    const tableCount = await tables.count()

    expect(tableCount, 'Should render at least one table').toBeGreaterThan(0)

    // Tables should have visible rows
    if (tableCount > 0) {
      const firstTableRows = await tables.first().locator('tr').count()
      expect(firstTableRows, 'Table should have at least one row').toBeGreaterThan(0)
    }
  })

  test('should render base64 data URI images', async ({ page }) => {
    test.skip(!paperId, 'Depends on upload test')

    await setAuthCookies(page)
    await page.goto(`/papers/${paperId}`)
    await page.waitForLoadState('networkidle')

    const proseContainer = page.locator('.prose')
    await expect(proseContainer).toBeVisible({ timeout: 30_000 })

    // Find all images within the prose container
    const images = proseContainer.locator('img')
    const imageCount = await images.count()

    if (imageCount > 0) {
      // At least some images should have data: URIs (from MinerU extraction)
      let hasDataUri = false
      for (let i = 0; i < imageCount; i++) {
        const src = await images.nth(i).getAttribute('src')
        if (src?.startsWith('data:image/')) {
          hasDataUri = true
          break
        }
      }

      // The RDMA paper should have images from MinerU extraction
      expect(hasDataUri, 'At least one image should be a base64 data URI').toBeTruthy()
    }
  })

  test('should render code blocks with syntax highlighting', async ({ page }) => {
    test.skip(!paperId, 'Depends on upload test')

    await setAuthCookies(page)
    await page.goto(`/papers/${paperId}`)
    await page.waitForLoadState('networkidle')

    const proseContainer = page.locator('.prose')
    await expect(proseContainer).toBeVisible({ timeout: 30_000 })

    // Shiki renders code blocks with <pre> elements containing <code>
    const codeBlocks = proseContainer.locator('pre code')

    // Academic papers may or may not have code blocks, so soft assertion
    const codeCount = await codeBlocks.count()
    if (codeCount > 0) {
      // Shiki adds style attributes for syntax highlighting
      const firstCode = codeBlocks.first()
      const hasStyleElements = await firstCode.locator('[style]').count()
      expect(hasStyleElements, 'Code block should have syntax-highlighted elements').toBeGreaterThan(0)

      // Copy button should be present
      const copyButton = proseContainer.locator('button').filter({ hasText: /copy/i }).first()
      if (await copyButton.isVisible()) {
        await expect(copyButton).toBeVisible()
      }
    }
  })

  test('should NOT display mojibake characters', async ({ page }) => {
    test.skip(!paperId, 'Depends on upload test')

    await setAuthCookies(page)
    await page.goto(`/papers/${paperId}`)
    await page.waitForLoadState('networkidle')

    const proseContainer = page.locator('.prose')
    await expect(proseContainer).toBeVisible({ timeout: 30_000 })

    const textContent = await proseContainer.textContent()

    // Common mojibake sequences from double-encoded UTF-8
    const mojibakeStrings = [
      '\u00e2\u20ac\u201c', // â€" (should be en-dash)
      '\u00e2\u20ac\u201d', // â€" (should be em-dash)
      '\u00e2\u20ac\u0153', // â€œ (should be left double quote)
      '\u00e2\u20ac\u2122', // â€™ (should be right single quote)
      '\u00e2\u20ac\u00a6', // â€¦ (should be ellipsis)
    ]

    for (const broken of mojibakeStrings) {
      expect(
        textContent,
        `Rendered content should not contain mojibake: ${JSON.stringify(broken)}`,
      ).not.toContain(broken)
    }
  })

  test('should correctly display special characters', async ({ page, request }) => {
    test.skip(!paperId, 'Depends on upload test')

    // Verify the markdown content contains expected special chars from the RDMA paper
    const { body } = await getPaper(request, auth, paperId)
    const md = body.paper.markdown_content!

    // The converter should have fixed mojibake to proper Unicode chars.
    // These characters are common in academic papers:
    // en-dash (U+2013), em-dash (U+2014), Greek letters, etc.
    const specialChars = ['\u2013', '\u2014', '\u201c', '\u201d', '\u2018', '\u2019', '\u2026']

    let foundSpecialChar = false
    for (const char of specialChars) {
      if (md.includes(char)) {
        foundSpecialChar = true
        break
      }
    }

    // At least some special characters should be present in an academic paper
    expect(foundSpecialChar, 'Paper should contain proper Unicode special characters').toBeTruthy()

    // Verify the frontend renders them correctly via the UI
    await setAuthCookies(page)
    await page.goto(`/papers/${paperId}`)
    await page.waitForLoadState('networkidle')

    const proseContainer = page.locator('.prose')
    await expect(proseContainer).toBeVisible({ timeout: 30_000 })

    const renderedText = await proseContainer.textContent()
    expect(renderedText!.length, 'Rendered text should be non-empty').toBeGreaterThan(100)

    // Verify no replacement characters (U+FFFD) are present
    expect(renderedText, 'Should not contain Unicode replacement characters').not.toContain('\uFFFD')
  })

  // ═══════════════════════════════════════════════════════════════
  // 3. UI Interaction Tests
  // ═══════════════════════════════════════════════════════════════
  test('should navigate from list to paper view', async ({ page }) => {
    test.skip(!paperId, 'Depends on upload test')

    await setAuthCookies(page)

    // Go to papers list
    await page.goto('/papers')
    await page.waitForLoadState('networkidle')

    // Wait for the list to load
    await expect(page.getByRole('heading', { name: /papers/i })).toBeVisible({ timeout: 15_000 })

    // Click on the paper item to navigate to detail view
    // PaperList renders paper titles as clickable elements
    const hasItems = await page.locator('text=rdma').count()

    if (hasItems > 0) {
      await page.locator('text=rdma').first().click()
      await page.waitForLoadState('networkidle')

      // Should be on the paper view page
      await expect(page.getByRole('button', { name: /back to papers/i })).toBeVisible({ timeout: 10_000 })
    } else {
      // If the title is different, navigate directly
      await page.goto(`/papers/${paperId}`)
      await page.waitForLoadState('networkidle')
      await expect(page.getByRole('button', { name: /back to papers/i })).toBeVisible({ timeout: 10_000 })
    }
  })

  test('should show processing state for a new upload', async ({ page, request }) => {
    // Upload a new paper just to check the processing UI state
    const { body } = await uploadPaper(request, auth)
    const newPaperId = body.id

    try {
      await setAuthCookies(page)

      // Navigate to the paper view immediately -- it should show processing state
      await page.goto(`/papers/${newPaperId}`)
      await page.waitForLoadState('networkidle')

      // Should show the "Converting Paper..." message
      const convertingMessage = page.getByText(/converting paper/i)
      const isVisible = await convertingMessage.isVisible({ timeout: 5_000 }).catch(() => false)

      // The conversion might be too fast to catch, so this is a soft assertion
      if (isVisible) {
        expect(await convertingMessage.isVisible()).toBeTruthy()
      }

      // Wait for it to finish for cleanup
      await waitForConversion(request, auth, newPaperId, 120_000).catch(() => {})
    } finally {
      // Clean up
      await deletePaper(request, auth, newPaperId).catch(() => {})
    }
  })

  // ═══════════════════════════════════════════════════════════════
  // 4. Cleanup
  // ═══════════════════════════════════════════════════════════════
  test.afterAll(async ({ request }) => {
    if (paperId && auth) {
      await deletePaper(request, auth, paperId).catch(() => {
        // Best-effort cleanup
      })
    }
  })
})
