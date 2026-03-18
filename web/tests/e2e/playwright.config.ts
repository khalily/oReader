import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1, // Run tests sequentially to avoid database conflicts
  reporter: 'html',
  use: {
    baseURL: 'http://localhost:8080',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'setup',
      testMatch: /setup\.ts/,
    },
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
      dependencies: ['setup'],
    },
  ],
  webServer: {
    command: 'cd ../.. && ENV=development DATABASE_URL=oreader_test.db RATE_LIMIT_ENABLED=false go run cmd/server/main.go',
    url: 'http://localhost:8080/health',
    reuseExistingServer: !process.env.CI,
    timeout: 120000,
  },
});
