import { defineConfig, devices } from '@playwright/test';
import { assertSafeE2EEnvironment, e2eBaseURL } from './support/env';

assertSafeE2EEnvironment();

export default defineConfig({
  testDir: './tests',
  timeout: 30_000,
  expect: {
    timeout: 5_000,
  },
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: process.env.CI ? [['html', { open: 'never' }], ['list']] : 'list',
  use: {
    baseURL: e2eBaseURL(),
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: process.env.E2E_SKIP_WEB_SERVER === 'true' ? undefined : {
    command: [
      'VITE_FIREBASE_API_KEY=e2e_dummy',
      'VITE_FIREBASE_AUTH_DOMAIN=e2e.local',
      'VITE_FIREBASE_PROJECT_ID=e2e-project',
      'VITE_FIREBASE_MESSAGING_SENDER_ID=0',
      'VITE_FIREBASE_APP_ID=1:0:web:e2e',
      'npm --prefix ../frontend run dev -- --host 127.0.0.1',
    ].join(' '),
    url: e2eBaseURL(),
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
});
