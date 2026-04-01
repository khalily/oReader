import type { APIRequestContext } from '@playwright/test'

/**
 * Test user credentials for E2E auth.
 * In CI/dev environment, a test user is pre-seeded or registered on the fly.
 */
const TEST_USER = {
  email: `e2e-paper-${Date.now()}@test.oreader.dev`,
  password: 'TestPass123!e2e',
  nickname: 'E2E Paper Tester',
}

/** Stored auth state: JWT cookies + CSRF token from register/login response. */
export interface AuthState {
  cookies: string
  csrfToken: string
  userId: string
}

/**
 * Register a new test user via the API and return the auth state.
 * Uses Playwright's APIRequestContext to call the backend directly.
 */
export async function registerTestUser(
  request: APIRequestContext,
  user?: { email?: string; password?: string; nickname?: string },
): Promise<AuthState> {
  const email = user?.email || TEST_USER.email
  const password = user?.password || TEST_USER.password
  const nickname = user?.nickname || TEST_USER.nickname

  const response = await request.post('/api/v1/auth/register', {
    data: { email, password, nickname },
  })

  // Accept both 200 (new user) and 409 (already exists -- reuse it)
  if (response.status() !== 200 && response.status() !== 409) {
    throw new Error(`Register failed: ${response.status()} ${await response.text()}`)
  }

  // If 409 (conflict), log in with the existing user
  if (response.status() === 409) {
    return loginTestUser(request, email, password)
  }

  const body = await response.json()
  const csrfToken = body.csrf_token || ''

  // Extract cookies from response headers
  const setCookieHeaders = response.headersArray().filter((h) => h.name.toLowerCase() === 'set-cookie')
  const cookies = setCookieHeaders.map((h) => h.value).join('; ')

  return {
    cookies,
    csrfToken,
    userId: body.user?.id || '',
  }
}

/**
 * Log in an existing test user and return the auth state.
 */
export async function loginTestUser(
  request: APIRequestContext,
  email: string,
  password: string,
): Promise<AuthState> {
  const response = await request.post('/api/v1/auth/login', {
    data: { email, password },
  })

  if (response.status() !== 200) {
    throw new Error(`Login failed: ${response.status()} ${await response.text()}`)
  }

  const body = await response.json()
  const csrfToken = body.csrf_token || ''

  const setCookieHeaders = response.headersArray().filter((h) => h.name.toLowerCase() === 'set-cookie')
  const cookies = setCookieHeaders.map((h) => h.value).join('; ')

  return {
    cookies,
    csrfToken,
    userId: body.user?.id || '',
  }
}

/**
 * Register or log in a test user, then return an authenticated API context.
 * Uses the registration endpoint first; falls back to login on conflict.
 */
export async function ensureAuthState(request: APIRequestContext): Promise<AuthState> {
  return registerTestUser(request)
}
