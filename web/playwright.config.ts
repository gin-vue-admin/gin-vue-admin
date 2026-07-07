import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './test/smoke',
  timeout: 60000,
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: process.env.CI ? [['github'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL: process.env.SMOKE_BASE_URL || 'http://localhost:5173/',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure'
  },
  // webServer：起 dev server（base ''，匹配 smoke 测试设计；VITE_ENABLE_MOCK=true 启用 MSW）。
  // CI 自动起 dev；本地若已起 dev（同端口）则复用，或设 SMOKE_BASE_URL 指向你的 dev 端口。
  webServer: {
    command: 'pnpm dev',
    url: process.env.SMOKE_BASE_URL || 'http://localhost:5173/',
    reuseExistingServer: !process.env.CI,
    timeout: 120000
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }]
})
