import { defineConfig, devices } from '@playwright/test';

/**
 * Tillsyn a11y suite — runs against Wails dev server started by Playwright's
 * webServer config. Tests in tests/ verify WCAG 2 AA via axe-core
 * across the three planned viewports (375, 768, 1280).
 *
 * webServer.cwd is '../../..' (3 levels from ui/frontend/a11y/) so `mage uiDev`
 * resolves via PATH to the magefile.go at the repo root, which launches
 * `wails dev` and serves the Astro+SolidJS bundle on 127.0.0.1:34115.
 * This is the Wails AssetServer with window.go IPC bindings — NOT the bare
 * Astro dev server on 51428, which would produce false PASS empty-state coverage.
 */
export default defineConfig({
  testDir: './tests',
  timeout: 30_000,
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: 1,
  reporter: process.env.CI ? 'list' : 'html',
  webServer: {
    command: 'mage uiDev',
    cwd: '../../..',
    url: 'http://127.0.0.1:34115/',
    timeout: 120_000,
    reuseExistingServer: !process.env.CI,
  },
  use: {
    baseURL: 'http://127.0.0.1:34115',
  },
  projects: [
    {
      name: 'mobile-375',
      use: { ...devices['Desktop Chrome'], viewport: { width: 375, height: 667 } },
    },
    {
      name: 'tablet-768',
      use: { ...devices['Desktop Chrome'], viewport: { width: 768, height: 1024 } },
    },
    {
      name: 'desktop-1280',
      use: { ...devices['Desktop Chrome'], viewport: { width: 1280, height: 800 } },
    },
  ],
});
