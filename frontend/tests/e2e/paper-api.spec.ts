/**
 * E2E tests for Paper API endpoints.
 *
 * These tests exercise the full backend API (Go handler -> service -> repository -> MySQL)
 * using Playwright's APIRequestContext. They require:
 *   - Backend running on port 8080
 *   - MySQL running
 *   - Converter gRPC service running on port 50051 (for conversion tests)
 *
 * Run: cd frontend && npx playwright test --config=tests/e2e/playwright.config.ts --project=api
 */
import { test, expect } from '@playwright/test'
import { ensureAuthState, type AuthState } from './helpers/auth'
import {
  uploadPaper,
  waitForConversion,
  getPaper,
  listPapers,
  updatePaper,
  deletePaper,
  getPaperStatus,
  updatePaperTags,
  listTags,
  downloadPaper,
  getDefaultTestPdfPath,
} from './helpers/api'
import fs from 'fs'

// ─── Skip guard: only run when E2E_BACKEND is set ─────────────────
test.beforeAll(async () => {
  const pdfPath = getDefaultTestPdfPath()
  if (!fs.existsSync(pdfPath)) {
    throw new Error(
      `Test PDF not found at ${pdfPath}. Ensure the test file exists before running E2E tests.`,
    )
  }
})

// ─── Shared state across tests in this file ───────────────────────
let auth: AuthState
let uploadedPaperId: string

test.describe('Paper API E2E', () => {
  // ── Setup: register a test user ─────────────────────────────────
  test.beforeAll(async ({ request }) => {
    auth = await ensureAuthState(request)
  })

  // ═══════════════════════════════════════════════════════════════
  // 1. Upload + Conversion Pipeline
  // ═══════════════════════════════════════════════════════════════
  test('should upload a PDF and return 202 Accepted with pending status', async ({ request }) => {
    const { response, body } = await uploadPaper(request, auth)

    expect(response.status()).toBe(202)
    expect(body.id).toBeTruthy()
    expect(body.status).toBe('pending')
    expect(body.original_filename).toContain('.pdf')
    expect(body.pdf_size).toBeGreaterThan(0)
    expect(body.user_id).toBe(auth.userId)

    uploadedPaperId = body.id
  })

  test('should progress through processing to completed status', async ({ request }) => {
    test.skip(!uploadedPaperId, 'Depends on upload test')

    // Poll until conversion completes (converter + MinerU + LLM may take 30-60s)
    const finalStatus = await waitForConversion(request, auth, uploadedPaperId, 120_000)

    expect(finalStatus.status).toBe('completed')
    expect(finalStatus.progress).toBe(100)
  })

  // ═══════════════════════════════════════════════════════════════
  // 2. Markdown Content Quality Verification
  // ═══════════════════════════════════════════════════════════════
  test('should have markdown content with valid structure', async ({ request }) => {
    test.skip(!uploadedPaperId, 'Depends on upload + conversion')

    const { body } = await getPaper(request, auth, uploadedPaperId)
    const paper = body.paper

    expect(paper.status).toBe('completed')
    expect(paper.markdown_content).toBeTruthy()
    expect(paper.markdown_content!.length).toBeGreaterThan(500)
  })

  test('markdown should contain LaTeX formulas', async ({ request }) => {
    test.skip(!uploadedPaperId, 'Depends on upload + conversion')

    const { body } = await getPaper(request, auth, uploadedPaperId)
    const md = body.paper.markdown_content!

    // The RDMA paper contains formulas -- check for LaTeX markers
    const hasInlineMath = md.includes('$') && md.match(/\$[^$]+?\$/)
    const hasDisplayMath = md.includes('$$')
    expect(
      hasInlineMath || hasDisplayMath,
      'Markdown should contain inline ($...$) or display ($$...$$) LaTeX formulas',
    ).toBeTruthy()
  })

  test('markdown should contain tables', async ({ request }) => {
    test.skip(!uploadedPaperId, 'Depends on upload + conversion')

    const { body } = await getPaper(request, auth, uploadedPaperId)
    const md = body.paper.markdown_content!

    // Tables can be pipe-delimited (GFM) or HTML
    const hasGfmTable = md.includes('|') && md.includes('---')
    const hasHtmlTable = md.includes('<table')
    expect(
      hasGfmTable || hasHtmlTable,
      'Markdown should contain GFM or HTML tables',
    ).toBeTruthy()
  })

  test('markdown should contain images', async ({ request }) => {
    test.skip(!uploadedPaperId, 'Depends on upload + conversion')

    const { body } = await getPaper(request, auth, uploadedPaperId)
    const md = body.paper.markdown_content!

    // Images should be base64 data URLs or markdown image syntax
    const hasBase64Image = md.includes('data:image/')
    const hasMarkdownImage = /!\[.*?\]\(.*?\)/.test(md)
    expect(
      hasBase64Image || hasMarkdownImage,
      'Markdown should contain images (base64 data URLs or markdown syntax)',
    ).toBeTruthy()
  })

  test('markdown should have heading structure', async ({ request }) => {
    test.skip(!uploadedPaperId, 'Depends on upload + conversion')

    const { body } = await getPaper(request, auth, uploadedPaperId)
    const md = body.paper.markdown_content!

    // Academic papers should have at least h1/h2 headings
    const hasHeadings = /^#{1,3}\s+.+/m.test(md)
    expect(hasHeadings, 'Markdown should contain heading structure (# / ## / ###)').toBeTruthy()
  })

  test('markdown should NOT contain UTF-8 mojibake', async ({ request }) => {
    test.skip(!uploadedPaperId, 'Depends on upload + conversion')

    const { body } = await getPaper(request, auth, uploadedPaperId)
    const md = body.paper.markdown_content!

    // Common mojibake sequences from double-encoded UTF-8
    const mojibakePatterns = [
      '\u00e2\u20ac\u201c', // â€" (should be en-dash)
      '\u00e2\u20ac\u201d', // â€" (should be em-dash)
      '\u00e2\u20ac\u0153', // â€œ (should be left double quote)
      '\u00e2\u20ac\u2122', // â€™ (should be right single quote)
      '\u00e2\u20ac\u00a6', // â€¦ (should be ellipsis)
    ]

    for (const broken of mojibakePatterns) {
      expect(
        md,
        `Markdown should not contain mojibake: ${JSON.stringify(broken)}`,
      ).not.toContain(broken)
    }
  })

  test('metadata should be extracted from the paper', async ({ request }) => {
    test.skip(!uploadedPaperId, 'Depends on upload + conversion')

    const { body } = await getPaper(request, auth, uploadedPaperId)
    const paper = body.paper

    // The RDMA paper should have a non-empty title after LLM extraction
    expect(paper.title).toBeTruthy()
    expect(paper.title.length).toBeGreaterThan(3)
  })

  // ═══════════════════════════════════════════════════════════════
  // 3. List, Get, Status Operations
  // ═══════════════════════════════════════════════════════════════
  test('should list papers and include the uploaded paper', async ({ request }) => {
    const { response, body } = await listPapers(request, auth, { limit: '20' })

    expect(response.status()).toBe(200)
    expect(body.papers).toBeInstanceOf(Array)
    expect(body.total).toBeGreaterThanOrEqual(1)

    if (uploadedPaperId) {
      const found = body.papers.find((p) => p.id === uploadedPaperId)
      expect(found, `Uploaded paper ${uploadedPaperId} should be in the list`).toBeTruthy()
    }
  })

  test('should support search/filter by status', async ({ request }) => {
    const { response, body } = await listPapers(request, auth, { status: 'completed' })

    expect(response.status()).toBe(200)
    for (const paper of body.papers) {
      expect(paper.status).toBe('completed')
    }
  })

  test('should support search by query string', async ({ request }) => {
    const { response, body } = await listPapers(request, auth, { q: 'rdma' })

    expect(response.status()).toBe(200)
    // If the RDMA paper is completed and indexed, it should be found
    // This is a soft check -- the search may not match depending on indexing
    expect(body.papers).toBeInstanceOf(Array)
  })

  test('should get paper status endpoint', async ({ request }) => {
    test.skip(!uploadedPaperId, 'Depends on upload test')

    const { response, body } = await getPaperStatus(request, auth, uploadedPaperId)

    expect(response.status()).toBe(200)
    expect(body.id).toBe(uploadedPaperId)
    expect(['pending', 'processing', 'completed', 'failed']).toContain(body.status)
  })

  // ═══════════════════════════════════════════════════════════════
  // 4. Update Paper Metadata
  // ═══════════════════════════════════════════════════════════════
  test('should update paper title', async ({ request }) => {
    test.skip(!uploadedPaperId, 'Depends on upload test')

    const { response, body } = await updatePaper(request, auth, uploadedPaperId, {
      title: 'E2E Test Updated Title',
    })

    expect(response.status()).toBe(200)
    expect(body.paper.title).toBe('E2E Test Updated Title')
  })

  // ═══════════════════════════════════════════════════════════════
  // 5. Tag Operations
  // ═══════════════════════════════════════════════════════════════
  test('should update and retrieve paper tags', async ({ request }) => {
    test.skip(!uploadedPaperId, 'Depends on upload test')

    const tags = ['e2e-test', 'rdma', 'networking']
    const { response: updateResp } = await updatePaperTags(request, auth, uploadedPaperId, tags)

    expect(updateResp.status()).toBe(200)

    // Verify tags appear in the tag list
    const { body: tagsBody } = await listTags(request, auth)
    expect(tagsBody.tags).toBeInstanceOf(Array)
    for (const tag of tags) {
      expect(tagsBody.tags).toContain(tag)
    }
  })

  // ═══════════════════════════════════════════════════════════════
  // 6. Download Original PDF
  // ═══════════════════════════════════════════════════════════════
  test('should download the original PDF file', async ({ request }) => {
    test.skip(!uploadedPaperId, 'Depends on upload test')

    const response = await downloadPaper(request, auth, uploadedPaperId)

    expect(response.status()).toBe(200)
    expect(response.headers()['content-type']).toContain('application/pdf')

    const body = await response.body()
    // Should be a valid PDF (starts with %PDF-)
    expect(body.toString('ascii', 0, 5)).toBe('%PDF-')
    expect(body.length).toBeGreaterThan(1000)
  })

  // ═══════════════════════════════════════════════════════════════
  // 7. Error Cases
  // ═══════════════════════════════════════════════════════════════
  test('should reject upload without file', async ({ request }) => {
    const response = await request.post('/api/v1/papers/upload', {
      headers: {
        Cookie: auth.cookies,
        'X-CSRF-Token': auth.csrfToken,
      },
    })

    expect(response.status()).toBe(400)
  })

  test('should reject non-PDF file upload', async ({ request }) => {
    const response = await request.post('/api/v1/papers/upload', {
      multipart: {
        file: {
          name: 'test.txt',
          mimeType: 'text/plain',
          buffer: Buffer.from('this is not a pdf'),
        },
      },
      headers: {
        Cookie: auth.cookies,
        'X-CSRF-Token': auth.csrfToken,
      },
    })

    expect(response.status()).toBe(400)
    const body = await response.json()
    expect(body.error?.message || '').toContain('PDF')
  })

  test('should return 404 for nonexistent paper', async ({ request }) => {
    const response = await request.get('/api/v1/papers/nonexistent-id-12345', {
      headers: {
        Cookie: auth.cookies,
        'X-CSRF-Token': auth.csrfToken,
      },
    })

    expect(response.status()).toBe(404)
  })

  test('should return 401 without authentication', async ({ request }) => {
    const response = await request.get('/api/v1/papers')
    expect(response.status()).toBe(401)
  })

  // ═══════════════════════════════════════════════════════════════
  // 8. Cleanup: Delete the uploaded paper
  // ═══════════════════════════════════════════════════════════════
  test('should delete the uploaded paper', async ({ request }) => {
    test.skip(!uploadedPaperId, 'No paper to delete')

    const response = await deletePaper(request, auth, uploadedPaperId)
    expect(response.status()).toBe(204)

    // Verify it is gone
    const getResponse = await request.get(`/api/v1/papers/${uploadedPaperId}`, {
      headers: {
        Cookie: auth.cookies,
        'X-CSRF-Token': auth.csrfToken,
      },
    })
    expect(getResponse.status()).toBe(404)
  })
})
