import { test, expect } from '@playwright/test';
import {
  registerUser,
  createTestUser,
  authPost,
  authGet,
} from './fixtures/auth';

test.describe('OPML API', () => {
  test('GET /api/v1/opml/export - export feeds as OPML', async ({ request }) => {
    const testUser = createTestUser();
    const { auth } = await registerUser(request, testUser);

    // Create a feed first
    await authPost(request, '/api/v1/feeds', auth, {
      feed_url: 'https://openai.com/news/rss.xml',
    });

    const response = await authGet(request, '/api/v1/opml/export', auth);
    expect(response.status()).toBe(200);

    // Check Content-Type header
    const contentType = response.headers()['content-type'];
    expect(contentType).toContain('application/xml');

    // Verify it's valid XML/OPML
    const body = await response.text();
    expect(body).toContain('<?xml');
    expect(body).toContain('<opml');
    expect(body).toContain('</opml>');
  });

  test('POST /api/v1/opml/import - import OPML file', async ({ request }) => {
    const testUser = createTestUser();
    const { auth } = await registerUser(request, testUser);

    // Create a simple OPML file
    const opmlContent = `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head>
    <title>Test Feeds</title>
  </head>
  <body>
    <outline type="rss" text="OpenAI News" xmlUrl="https://openai.com/news/rss.xml"/>
  </body>
</opml>`;

    const response = await authPost(request, '/api/v1/opml/import', auth, {
      opml: opmlContent,
    });

    expect(response.status()).toBe(202);

    const body = await response.json();
    expect(body).toHaveProperty('job_id');
    expect(body).toHaveProperty('total_feeds');
  });

  test('GET /api/v1/opml/import/:job_id - get import status', async ({ request }) => {
    const testUser = createTestUser();
    const { auth } = await registerUser(request, testUser);

    // Import an OPML file first
    const opmlContent = `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head>
    <title>Test Feeds</title>
  </head>
  <body>
    <outline type="rss" text="OpenAI News" xmlUrl="https://openai.com/news/rss.xml"/>
  </body>
</opml>`;

    const importResponse = await authPost(request, '/api/v1/opml/import', auth, {
      opml: opmlContent,
    });
    const importBody = await importResponse.json();
    const jobId = importBody.job_id;

    const response = await authGet(request, `/api/v1/opml/import/${jobId}`, auth);
    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body).toHaveProperty('status');
    expect(body).toHaveProperty('progress');
    expect(body).toHaveProperty('total');
    expect(body).toHaveProperty('processed');
  });
});
