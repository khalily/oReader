import { test, expect } from '@playwright/test';
import {
  registerUser,
  createTestUser,
  authPost,
  authGet,
} from './fixtures/auth';

test.describe('Error Handling', () => {
  test('401 Unauthorized - missing access token', async ({ request }) => {
    // Try to access protected endpoint without authentication
    const response = await request.get('/api/v1/feeds');
    expect(response.status()).toBe(401);

    const body = await response.json();
    expect(body).toHaveProperty('error');
  });

  test('403 Forbidden - missing CSRF token on POST', async ({ request }) => {
    const testUser = createTestUser();
    const { auth } = await registerUser(request, testUser);

    // Try to POST without CSRF token header
    const response = await request.post('/api/v1/feeds', {
      data: {
        feed_url: 'https://openai.com/news/rss.xml',
      },
    });

    expect(response.status()).toBe(403);

    const body = await response.json();
    expect(body).toHaveProperty('error');
  });

  test('403 Forbidden - CSRF token mismatch', async ({ request }) => {
    const testUser = createTestUser();
    const { auth } = await registerUser(request, testUser);

    // Try to POST with invalid CSRF token
    const response = await request.post('/api/v1/feeds', {
      headers: {
        'X-CSRF-Token': 'invalid-csrf-token',
        'Content-Type': 'application/json',
      },
      data: {
        feed_url: 'https://openai.com/news/rss.xml',
      },
    });

    expect(response.status()).toBe(403);

    const body = await response.json();
    expect(body).toHaveProperty('error');
  });

  test('404 Not Found - non-existent resources', async ({ request }) => {
    const testUser = createTestUser();
    const { auth } = await registerUser(request, testUser);

    // Test non-existent feed
    const feedResponse = await authGet(request, '/api/v1/feeds/nonexistent-id', auth);
    expect(feedResponse.status()).toBe(404);

    // Test non-existent item
    const itemResponse = await authGet(request, '/api/v1/items/nonexistent-id', auth);
    expect(itemResponse.status()).toBe(404);

    // Test non-existent route
    const routeResponse = await authGet(request, '/api/v1/nonexistent', auth);
    expect(routeResponse.status()).toBe(404);
  });

  test('Rate limiting behavior (optional, if testing with rate limits enabled)', async ({ request }) => {
    // This test is optional and should only run if rate limiting is enabled
    // For most test runs, rate limiting is disabled via RATE_LIMIT_ENABLED=false
    // This is a placeholder for future rate limiting tests

    const testUser = createTestUser();
    const { auth } = await registerUser(request, testUser);

    // If rate limiting is enabled, make multiple requests rapidly
    // and verify 429 Too Many Requests response

    // For now, just verify normal behavior
    const response = await authGet(request, '/api/v1/feeds', auth);
    expect(response.status()).toBeLessThan(500);
  });
});
