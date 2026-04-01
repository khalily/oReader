import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: '.',
  testMatch: '*.spec.ts',
  fullyParallel: false,
  timeout: 120_000,
  expect: {
    timeout: 15_000,
  },
  retries: 0,
  reporter: [['list'], ['html', { open: 'never', outputFolder: 'test-results/html' }]],
  use: {
    baseURL: process.env.BASE_URL || 'http://localhost:5173',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  projects: [
    {
      name: 'api',
      testMatch: 'paper-api.spec.ts',
      use: {
        // API tests go directly to backend (bypass Vite proxy for speed)
        baseURL: process.env.API_URL || 'http://localhost:8080',
      },
    },
    {
      name: 'ui',
      testMatch: 'paper-rendering.spec.ts',
      use: {
        // UI tests use the full frontend dev server
        baseURL: process.env.BASE_URL || 'http://localhost:5173',
      },
    },
  ],
})
