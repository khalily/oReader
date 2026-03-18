import { test as setup } from '@playwright/test';
import { execSync } from 'child_process';
import { existsSync, unlinkSync } from 'fs';

setup.beforeAll(() => {
  // Remove old test database if it exists
  const dbPath = 'oreader_test.db';
  if (existsSync(dbPath)) {
    unlinkSync(dbPath);
    console.log('Deleted old test database');
  }

  // The Go server will auto-migrate the database when it starts
  console.log('Test database will be initialized by the server');
});
