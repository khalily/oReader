import { test, expect } from '@playwright/test';
import {
  registerUser,
  createTestUser,
  authPost,
  authGet,
  authDelete,
} from './fixtures/auth';

test.describe('Feed API', () => {
  test('POST /api/v1/feeds - create feed subscription (use OpenAI RSS)', async ({ request }) => {
    const testUser = createTestUser();
    const { auth } = await registerUser(request, testUser);

    const response = await authPost(request, '/api/v1/feeds', auth, {
      feed_url: 'https://openai.com/news/rss.xml',
    });

    expect(response.status()).toBe(201);

    const body = await response.json();
    expect(body).toHaveProperty('feed');
    expect(body.feed).toHaveProperty('id');
    expect(body.feed).toHaveProperty('title');
    expect(body.feed).toHaveProperty('link', 'https://openai.com/news/rss.xml');
    expect(body).toHaveProperty('new_item_count');
  });

  test('GET /api/v1/feeds - list feeds', async ({ request }) => {
    const testUser = createTestUser();
    const { auth } = await registerUser(request, testUser);

    // Create a feed first
    await authPost(request, '/api/v1/feeds', auth, {
      feed_url: 'https://openai.com/news/rss.xml',
    });

    const response = await authGet(request, '/api/v1/feeds', auth);
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body).toHaveProperty('feeds');
    expect(Array.isArray(body.feeds)).toBe(true);
    expect(body.feeds.length).toBeGreaterThan(0);
    expect(body).toHaveProperty('total');
  });

  test('GET /api/v1/feeds/:id - get single feed', async ({ request }) => {
    const testUser = createTestUser();
    const { auth } = await registerUser(request, testUser);

    // Create a feed first
    const createResponse = await authPost(request, '/api/v1/feeds', auth, {
      feed_url: 'https://openai.com/news/rss.xml',
    });
    const createBody = await createResponse.json();
    const feedId = createBody.feed.id;

    const response = await authGet(request, `/api/v1/feeds/${feedId}`, auth);
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body).toHaveProperty('feed');
    expect(body.feed).toHaveProperty('id', feedId);
    expect(body).toHaveProperty('item_count');
  });

  test('POST /api/v1/feeds/:id/refresh - refresh feed', async ({ request }) => {
    const testUser = createTestUser();
    const { auth } = await registerUser(request, testUser);

    // Create a feed first
    const createResponse = await authPost(request, '/api/v1/feeds', auth, {
      feed_url: 'https://openai.com/news/rss.xml',
    });
    const createBody = await createResponse.json();
    const feedId = createBody.feed.id;

    const response = await authPost(request, `/api/v1/feeds/${feedId}/refresh`, auth);
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body).toHaveProperty('new_item_count');
  });

  test('POST /api/v1/feeds/:id/mark-all-read - mark all items read', async ({ request }) => {
    const testUser = createTestUser();
    const { auth } = await registerUser(request, testUser);

    // Create a feed first
    const createResponse = await authPost(request, '/api/v1/feeds', auth, {
      feed_url: 'https://openai.com/news/rss.xml',
    });
    const createBody = await createResponse.json();
    const feedId = createBody.feed.id;

    const response = await authPost(request, `/api/v1/feeds/${feedId}/mark-all-read`, auth);
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body).toHaveProperty('count');
  });

  test('DELETE /api/v1/feeds/:id - delete feed', async ({ request }) => {
    const testUser = createTestUser();
    const { auth } = await registerUser(request, testUser);

    // Create a feed first
    const createResponse = await authPost(request, '/api/v1/feeds', auth, {
      feed_url: 'https://openai.com/news/rss.xml',
    });
    const createBody = await createResponse.json();
    const feedId = createBody.feed.id;

    const response = await authDelete(request, `/api/v1/feeds/${feedId}`, auth);
    expect(response.status()).toBe(204);

    // Verify feed is deleted
    const getResponse = await authGet(request, `/api/v1/feeds/${feedId}`, auth);
    expect(getResponse.status()).toBe(404);
  });

  test('Error cases - invalid URL, duplicate feed, not found', async ({ request }) => {
    const testUser = createTestUser();
    const { auth } = await registerUser(request, testUser);

    // Test invalid URL
    const invalidUrlResponse = await authPost(request, '/api/v1/feeds', auth, {
      feed_url: 'not-a-valid-url',
    });
    expect(invalidUrlResponse.status()).toBeGreaterThanOrEqual(400);

    // Create a feed
    await authPost(request, '/api/v1/feeds', auth, {
      feed_url: 'https://openai.com/news/rss.xml',
    });

    // Test duplicate feed
    const duplicateResponse = await authPost(request, '/api/v1/feeds', auth, {
      feed_url: 'https://openai.com/news/rss.xml',
    });
    expect(duplicateResponse.status()).toBe(409);

    // Test not found
    const notFoundResponse = await authGet(request, '/api/v1/feeds/nonexistent-id', auth);
    expect(notFoundResponse.status()).toBe(404);
  });
});
