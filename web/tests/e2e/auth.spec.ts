import { test, expect } from '@playwright/test';
import {
  registerUser,
  loginUser,
  createTestUser,
  authPost,
  authGet,
} from './fixtures/auth';

test.describe('Authentication API', () => {
  test('POST /api/v1/auth/register - successful registration', async ({ request }) => {
    const testUser = createTestUser();
    const { response, auth } = await registerUser(request, testUser);

    expect(response.status()).toBe(201);

    const body = await response.json();
    expect(body).toHaveProperty('user');
    expect(body.user).toHaveProperty('id');
    expect(body.user).toHaveProperty('email', testUser.email);
    expect(body).toHaveProperty('csrf_token');
    expect(auth.csrfToken).toBeTruthy();
  });

  test('POST /api/v1/auth/register - duplicate email error', async ({ request }) => {
    const testUser = createTestUser();

    // First registration should succeed
    const { response: firstResponse } = await registerUser(request, testUser);
    expect(firstResponse.status()).toBe(201);

    // Second registration with same email should fail
    const { response: secondResponse } = await registerUser(request, testUser);
    expect(secondResponse.status()).toBe(409);

    const body = await secondResponse.json();
    expect(body).toHaveProperty('error');
  });

  test('POST /api/v1/auth/register - invalid email format', async ({ request }) => {
    const response = await request.post('/api/v1/auth/register', {
      data: {
        email: 'invalid-email',
        password: 'TestPassword123!',
      },
    });

    expect(response.status()).toBe(400);

    const body = await response.json();
    expect(body).toHaveProperty('error');
  });

  test('POST /api/v1/auth/login - successful login with csrf_token in response', async ({ request }) => {
    const testUser = createTestUser();

    // Register user first
    const { response: registerResponse } = await registerUser(request, testUser);
    expect(registerResponse.status()).toBe(201);

    // Login with same credentials
    const { response: loginResponse, auth } = await loginUser(request, testUser);
    expect(loginResponse.status()).toBe(200);

    const body = await loginResponse.json();
    expect(body).toHaveProperty('user');
    expect(body.user).toHaveProperty('email', testUser.email);
    expect(body).toHaveProperty('csrf_token');
    expect(auth.csrfToken).toBeTruthy();
  });

  test('POST /api/v1/auth/login - invalid credentials', async ({ request }) => {
    const response = await request.post('/api/v1/auth/login', {
      data: {
        email: 'nonexistent@example.com',
        password: 'WrongPassword123!',
      },
    });

    expect(response.status()).toBe(401);

    const body = await response.json();
    expect(body).toHaveProperty('error');
  });

  test('GET /api/v1/auth/me - get current user', async ({ request }) => {
    const testUser = createTestUser();
    const { auth } = await registerUser(request, testUser);

    const response = await authGet(request, '/api/v1/auth/me', auth);
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body).toHaveProperty('user');
    expect(body.user).toHaveProperty('email', testUser.email);
  });

  test('POST /api/v1/auth/refresh - token refresh', async ({ request }) => {
    const testUser = createTestUser();
    const { auth } = await registerUser(request, testUser);

    const response = await authPost(request, '/api/v1/auth/refresh', auth);
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body).toHaveProperty('user');
    expect(body).toHaveProperty('csrf_token');
  });

  test('POST /api/v1/auth/logout - logout clears cookies', async ({ request }) => {
    const testUser = createTestUser();
    const { auth } = await registerUser(request, testUser);

    const response = await authPost(request, '/api/v1/auth/logout', auth);
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body).toHaveProperty('message');

    // Verify we can't access protected endpoints after logout
    const meResponse = await authGet(request, '/api/v1/auth/me', auth);
    expect(meResponse.status()).toBe(401);
  });
});
