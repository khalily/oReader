import type { APIRequestContext } from '@playwright/test'
import type { AuthState } from './auth'
import path from 'path'

/**
 * Paper API response types matching the backend model.
 */
export interface Paper {
  id: string
  user_id: string
  title: string
  authors: string | null
  abstract: string | null
  keywords: string | null
  published_year: string | null
  doi: string | null
  pdf_size: number
  markdown_content: string | null
  cover_image: string | null
  original_filename: string
  status: 'pending' | 'processing' | 'completed' | 'failed'
  error: string | null
  created_at: string
  updated_at: string
}

export interface PaperStatusResponse {
  id: string
  status: 'pending' | 'processing' | 'completed' | 'failed'
  progress: number
  error?: string
  created_at: string
  updated_at: string
}

export interface ListPapersResponse {
  papers: Paper[]
  total: number
}

/**
 * Upload a PDF file via the paper upload endpoint.
 * Uses the real rdma.pdf test file from the uploads directory.
 */
export async function uploadPaper(
  request: APIRequestContext,
  auth: AuthState,
  filePath?: string,
): Promise<{ response: import('@playwright/test').APIResponse; body: Paper }> {
  const pdfPath = filePath || getDefaultTestPdfPath()
  const response = await request.post('/api/v1/papers/upload', {
    multipart: {
      file: {
        name: path.basename(pdfPath),
        mimeType: 'application/pdf',
        buffer: Buffer.from(await import('fs').then((fs) => fs.promises.readFile(pdfPath))),
      },
    },
    headers: {
      Cookie: auth.cookies,
      'X-CSRF-Token': auth.csrfToken,
    },
  })

  const body: Paper = await response.json()
  return { response, body }
}

/**
 * Get the default test PDF path (project-root relative).
 */
export function getDefaultTestPdfPath(): string {
  // Resolve relative to project root: frontend/tests/e2e -> ../../../uploads/...
  return path.resolve(__dirname, '../../../../uploads/papers/019d2fae-69a3-73f4-90ea-ce15ee1db999/2026-03-28/rdma.pdf')
}

/**
 * Poll the paper status endpoint until conversion completes or fails.
 * Times out after maxWaitMs.
 */
export async function waitForConversion(
  request: APIRequestContext,
  auth: AuthState,
  paperId: string,
  maxWaitMs: number = 90_000,
  pollIntervalMs: number = 3_000,
): Promise<PaperStatusResponse> {
  const deadline = Date.now() + maxWaitMs

  while (Date.now() < deadline) {
    const response = await request.get(`/api/v1/papers/${paperId}/status`, {
      headers: {
        Cookie: auth.cookies,
        'X-CSRF-Token': auth.csrfToken,
      },
    })

    if (response.status() !== 200) {
      throw new Error(`Status check failed: ${response.status()} ${await response.text()}`)
    }

    const status: PaperStatusResponse = await response.json()

    if (status.status === 'completed') {
      return status
    }

    if (status.status === 'failed') {
      throw new Error(`Paper conversion failed: ${status.error || 'unknown error'}`)
    }

    // Still pending or processing -- wait and retry
    await new Promise((resolve) => setTimeout(resolve, pollIntervalMs))
  }

  throw new Error(`Conversion did not complete within ${maxWaitMs}ms`)
}

/**
 * Get paper details by ID.
 */
export async function getPaper(
  request: APIRequestContext,
  auth: AuthState,
  paperId: string,
): Promise<{ response: import('@playwright/test').APIResponse; body: { paper: Paper } }> {
  const response = await request.get(`/api/v1/papers/${paperId}`, {
    headers: {
      Cookie: auth.cookies,
      'X-CSRF-Token': auth.csrfToken,
    },
  })
  const body = await response.json()
  return { response, body }
}

/**
 * List papers with optional query parameters.
 */
export async function listPapers(
  request: APIRequestContext,
  auth: AuthState,
  params?: Record<string, string>,
): Promise<{ response: import('@playwright/test').APIResponse; body: ListPapersResponse }> {
  const url = new URL('/api/v1/papers', 'http://placeholder')
  if (params) {
    Object.entries(params).forEach(([k, v]) => url.searchParams.set(k, v))
  }

  const response = await request.get(`/api/v1/papers${url.search}`, {
    headers: {
      Cookie: auth.cookies,
      'X-CSRF-Token': auth.csrfToken,
    },
  })
  const body = await response.json()
  return { response, body }
}

/**
 * Update paper metadata.
 */
export async function updatePaper(
  request: APIRequestContext,
  auth: AuthState,
  paperId: string,
  data: Record<string, string>,
): Promise<{ response: import('@playwright/test').APIResponse; body: { paper: Paper } }> {
  const response = await request.put(`/api/v1/papers/${paperId}`, {
    data,
    headers: {
      Cookie: auth.cookies,
      'X-CSRF-Token': auth.csrfToken,
      'Content-Type': 'application/json',
    },
  })
  const body = await response.json()
  return { response, body }
}

/**
 * Delete a paper by ID.
 */
export async function deletePaper(
  request: APIRequestContext,
  auth: AuthState,
  paperId: string,
): Promise<import('@playwright/test').APIResponse> {
  return request.delete(`/api/v1/papers/${paperId}`, {
    headers: {
      Cookie: auth.cookies,
      'X-CSRF-Token': auth.csrfToken,
    },
  })
}

/**
 * Retry a failed paper conversion.
 */
export async function retryPaper(
  request: APIRequestContext,
  auth: AuthState,
  paperId: string,
): Promise<{ response: import('@playwright/test').APIResponse; body: { paper: Paper } }> {
  const response = await request.post(`/api/v1/papers/${paperId}/retry`, {
    headers: {
      Cookie: auth.cookies,
      'X-CSRF-Token': auth.csrfToken,
    },
  })
  const body = await response.json()
  return { response, body }
}

/**
 * Get paper conversion status.
 */
export async function getPaperStatus(
  request: APIRequestContext,
  auth: AuthState,
  paperId: string,
): Promise<{ response: import('@playwright/test').APIResponse; body: PaperStatusResponse }> {
  const response = await request.get(`/api/v1/papers/${paperId}/status`, {
    headers: {
      Cookie: auth.cookies,
      'X-CSRF-Token': auth.csrfToken,
    },
  })
  const body = await response.json()
  return { response, body }
}

/**
 * Update paper tags.
 */
export async function updatePaperTags(
  request: APIRequestContext,
  auth: AuthState,
  paperId: string,
  tags: string[],
): Promise<{ response: import('@playwright/test').APIResponse; body: { tags: string[] } }> {
  const response = await request.put(`/api/v1/papers/${paperId}/tags`, {
    data: { tags },
    headers: {
      Cookie: auth.cookies,
      'X-CSRF-Token': auth.csrfToken,
      'Content-Type': 'application/json',
    },
  })
  const body = await response.json()
  return { response, body }
}

/**
 * List all user tags.
 */
export async function listTags(
  request: APIRequestContext,
  auth: AuthState,
): Promise<{ response: import('@playwright/test').APIResponse; body: { tags: string[] } }> {
  const response = await request.get('/api/v1/papers/tags', {
    headers: {
      Cookie: auth.cookies,
      'X-CSRF-Token': auth.csrfToken,
    },
  })
  const body = await response.json()
  return { response, body }
}

/**
 * Download the original PDF file.
 */
export async function downloadPaper(
  request: APIRequestContext,
  auth: AuthState,
  paperId: string,
): Promise<import('@playwright/test').APIResponse> {
  return request.get(`/api/v1/papers/${paperId}/download`, {
    headers: {
      Cookie: auth.cookies,
      'X-CSRF-Token': auth.csrfToken,
    },
  })
}
