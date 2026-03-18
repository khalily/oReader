import { APIRequestContext, APIResponse } from '@playwright/test';

export interface TestUser {
  email: string;
  password: string;
}

export interface AuthContext {
  cookies: Record<string, string>;
  csrfToken: string;
  accessToken: string;
  refreshToken: string;
}

let userCounter = 0;

/**
 * Generate a unique test user
 */
export function createTestUser(): TestUser {
  userCounter++;
  const timestamp = Date.now();
  return {
    email: `test-user-${timestamp}-${userCounter}@example.com`,
    password: 'TestPassword123!',
  };
}

/**
 * Register a new user and extract authentication tokens
 */
export async function registerUser(
  request: APIRequestContext,
  user?: TestUser
): Promise<{ response: APIResponse; user: TestUser; auth: AuthContext }> {
  const testUser = user || createTestUser();

  const response = await request.post('/api/v1/auth/register', {
    data: {
      email: testUser.email,
      password: testUser.password,
    },
  });

  const auth = await extractAuthContext(request, response);
  return { response, user: testUser, auth };
}

/**
 * Login an existing user and extract authentication tokens
 */
export async function loginUser(
  request: APIRequestContext,
  user: TestUser
): Promise<{ response: APIResponse; auth: AuthContext }> {
  const response = await request.post('/api/v1/auth/login', {
    data: {
      email: user.email,
      password: user.password,
    },
  });

  const auth = await extractAuthContext(request, response);
  return { response, auth };
}

/**
 * Extract authentication context from login/register response
 */
export async function extractAuthContext(
  request: APIRequestContext,
  response: APIResponse
): Promise<AuthContext> {
  // Extract CSRF token from response body
  const body = await response.json();
  const csrfToken = body.csrf_token || '';

  // Extract cookies from storage state
  const storageState = await request.storageState();
  const cookies: Record<string, string> = {};

  for (const cookie of storageState.cookies) {
    cookies[cookie.name] = cookie.value;
  }

  return {
    cookies,
    csrfToken: csrfToken || cookies['csrf_token'] || '',
    accessToken: cookies['access_token'] || '',
    refreshToken: cookies['refresh_token'] || '',
  };
}

/**
 * Build Cookie header from auth context
 */
function buildCookieHeader(auth: AuthContext): string {
  const parts: string[] = [];
  for (const [name, value] of Object.entries(auth.cookies)) {
    if (value) {
      parts.push(`${name}=${value}`);
    }
  }
  return parts.join('; ');
}

/**
 * Make authenticated POST request with CSRF token
 */
export async function authPost(
  request: APIRequestContext,
  url: string,
  auth: AuthContext,
  data?: any
): Promise<APIResponse> {
  const headers: Record<string, string> = {
    'X-CSRF-Token': auth.csrfToken,
    'Content-Type': 'application/json',
    'Cookie': buildCookieHeader(auth),
  };

  return request.post(url, {
    headers,
    data,
  });
}

/**
 * Make authenticated DELETE request with CSRF token
 */
export async function authDelete(
  request: APIRequestContext,
  url: string,
  auth: AuthContext
): Promise<APIResponse> {
  const headers: Record<string, string> = {
    'X-CSRF-Token': auth.csrfToken,
    'Cookie': buildCookieHeader(auth),
  };

  return request.delete(url, {
    headers,
  });
}

/**
 * Make authenticated GET request
 */
export async function authGet(
  request: APIRequestContext,
  url: string,
  auth: AuthContext
): Promise<APIResponse> {
  const headers: Record<string, string> = {
    'Cookie': buildCookieHeader(auth),
  };

  return request.get(url, { headers });
}
