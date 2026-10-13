import process from 'node:process'
import { defineConfig, devices } from '@playwright/test'

// Runs against an already running stack (`make up`); override with E2E_BASE_URL.
export default defineConfig({
  testDir: '.',
  testMatch: ['e2e/**/*.e2e.spec.ts', 'src/**/*.e2e.spec.ts'],
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? 'github' : 'list',
  use: {
    baseURL: process.env.E2E_BASE_URL ?? 'http://localhost:8080',
    trace: 'retain-on-failure',
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
  ],
})
