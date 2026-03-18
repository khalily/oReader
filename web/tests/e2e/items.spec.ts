import { test, expect } from '@playwright/test';
import {
  registerUser,
  createTestUser,
  authPost,
  authGet,
} from './fixtures/auth';

test.describe('Item API', () => {
  test('GET /api/v1/items - list items with pagination', async ({ request }) => {
    const testUser = createTestUser();
    const { auth } = await registerUser(request, testUser);

    // Create a feed first to have items
    await authPost(request, '/api/v1/feeds', auth, {
      feed_url: 'https://openai.com/news/rss.xml',
    });

    const response = await authGet(request, '/api/v1/items?limit=10', auth);
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body).toHaveProperty('items');
    expect(Array.isArray(body.items)).toBe(true);
    expect(body).toHaveProperty('total');
    expect(body).toHaveProperty('has_more');
    expect(body).toHaveProperty('next_cursor');

    // Validate item structure - ensure feed is included
    if (body.items.length > 0) {
      const item = body.items[0];
      expect(item).toHaveProperty('id');
      expect(item).toHaveProperty('title');
      expect(item).toHaveProperty('feed_id');

      // Critical: feed object must be present for frontend
      expect(item).toHaveProperty('feed');
      expect(item.feed).toHaveProperty('id');
      expect(item.feed).toHaveProperty('title');
      expect(typeof item.feed.title).toBe('string');
      expect(item.feed.title.length).toBeGreaterThan(0);

      // user_state should be present (can be null)
      expect(item).toHaveProperty('user_state');
    }
  });

  test('GET /api/v1/items?feed_id=X - filter by feed', async ({ request }) => {
    const testUser = createTestUser();
    const { auth } = await registerUser(request, testUser);

    // Create a feed
    const createResponse = await authPost(request, '/api/v1/feeds', auth, {
      feed_url: 'https://openai.com/news/rss.xml',
    });
    const createBody = await createResponse.json();
    const feedId = createBody.feed.id;

    const response = await authGet(request, `/api/v1/items?feed_id=${feedId}`, auth);
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body).toHaveProperty('items');
    expect(Array.isArray(body.items)).toBe(true);

    // Validate feed object is included in items
    if (body.items.length > 0) {
      expect(body.items[0]).toHaveProperty('feed');
      expect(body.items[0].feed).toHaveProperty('title');
    }
  });

  test('GET /api/v1/items?starred=true - filter by starred', async ({ request }) => {
    const testUser = createTestUser();
    const { auth } = await registerUser(request, testUser);

    // Create a feed first
    await authPost(request, '/api/v1/feeds', auth, {
      feed_url: 'https://openai.com/news/rss.xml',
    });

    const response = await authGet(request, '/api/v1/items?starred=true', auth);
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body).toHaveProperty('items');
    expect(Array.isArray(body.items)).toBe(true);

    // If there are starred items, validate their structure
    if (body.items.length > 0) {
      expect(body.items[0]).toHaveProperty('feed');
      expect(body.items[0].feed).toHaveProperty('title');
    }
  });

  test('GET /api/v1/items/:id - get single item', async ({ request }) => {
    const testUser = createTestUser();
    const { auth } = await registerUser(request, testUser);

    // Create a feed first
    await authPost(request, '/api/v1/feeds', auth, {
      feed_url: 'https://openai.com/news/rss.xml',
    });

    // Get items list
    const listResponse = await authGet(request, '/api/v1/items?limit=1', auth);
    const listBody = await listResponse.json();

    if (listBody.items.length > 0) {
      const itemId = listBody.items[0].id;

      const response = await authGet(request, `/api/v1/items/${itemId}`, auth);
      expect(response.status()).toBe(200);

      const body = await response.json();
      expect(body).toHaveProperty('item');
      expect(body.item).toHaveProperty('id', itemId);

      // Validate feed object is included
      expect(body.item).toHaveProperty('feed');
      expect(body.item.feed).toHaveProperty('title');
    }
  });

  test('POST /api/v1/items/:id/star - toggle star status', async ({ request }) => {
    const testUser = createTestUser();
    const { auth } = await registerUser(request, testUser);

    // Create a feed first
    await authPost(request, '/api/v1/feeds', auth, {
      feed_url: 'https://openai.com/news/rss.xml',
    });

    // Get items list
    const listResponse = await authGet(request, '/api/v1/items?limit=1', auth);
    const listBody = await listResponse.json();

    if (listBody.items.length > 0) {
      const itemId = listBody.items[0].id;

      const response = await authPost(request, `/api/v1/items/${itemId}/star`, auth, {
        starred: true,
      });
      expect(response.status()).toBe(200);

      const body = await response.json();
      expect(body).toHaveProperty('item');
      expect(body.item).toHaveProperty('is_starred', true);

      // Validate feed object is included
      expect(body.item).toHaveProperty('feed');
      expect(body.item.feed).toHaveProperty('title');
    }
  });

  test('POST /api/v1/items/:id/read - toggle read status', async ({ request }) => {
    const testUser = createTestUser();
    const { auth } = await registerUser(request, testUser);

    // Create a feed first
    await authPost(request, '/api/v1/feeds', auth, {
      feed_url: 'https://openai.com/news/rss.xml',
    });

    // Get items list
    const listResponse = await authGet(request, '/api/v1/items?limit=1', auth);
    const listBody = await listResponse.json();

    if (listBody.items.length > 0) {
      const itemId = listBody.items[0].id;

      const response = await authPost(request, `/api/v1/items/${itemId}/read`, auth, {
        read: true,
      });
      expect(response.status()).toBe(200);

      const body = await response.json();
      expect(body).toHaveProperty('item');
      expect(body.item).toHaveProperty('is_read', true);

      // Validate feed object is included
      expect(body.item).toHaveProperty('feed');
      expect(body.item.feed).toHaveProperty('title');
    }
  });

  test('Error cases - item not found, unauthorized access', async ({ request }) => {
    const testUser = createTestUser();
    const { auth } = await registerUser(request, testUser);

    // Test item not found
    const notFoundResponse = await authGet(request, '/api/v1/items/nonexistent-id', auth);
    expect(notFoundResponse.status()).toBe(404);

    // Test unauthorized access (no auth)
    const unauthorizedResponse = await request.get('/api/v1/items');
    expect(unauthorizedResponse.status()).toBe(401);
  });
});
