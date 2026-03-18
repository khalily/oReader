import { test, expect } from '@playwright/test';

test.describe('Debug Auth', () => {
  test('debug cookie extraction', async ({ request }) => {
    // Register a new user
    const timestamp = Date.now();
    const email = `debug-user-${timestamp}@example.com`;
    const password = 'TestPassword123!';

    const response = await request.post('/api/v1/auth/register', {
      data: { email, password },
    });

    console.log('Register response status:', response.status());
    const body = await response.json();
    console.log('Register response body:', JSON.stringify(body, null, 2));
    
    // Check response headers for Set-Cookie
    const headers = response.headers();
    console.log('Response headers:', JSON.stringify(headers, null, 2));
    
    // Check storage state after registration
    const storageState = await request.storageState();
    console.log('Storage state after register:', JSON.stringify(storageState, null, 2));
    
    expect(response.status()).toBe(201);
    
    // Now try to access /me with the cookies
    const cookies = storageState.cookies;
    const cookieHeader = cookies.map(c => `${c.name}=${c.value}`).join('; ');
    console.log('Cookie header:', cookieHeader);
    
    const meResponse = await request.get('/api/v1/auth/me', {
      headers: {
        'Cookie': cookieHeader,
      },
    });
    
    console.log('Me response status:', meResponse.status());
    const meBody = await meResponse.json();
    console.log('Me response body:', JSON.stringify(meBody, null, 2));
  });
});
