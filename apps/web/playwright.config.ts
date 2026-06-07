import { defineConfig } from "@playwright/test";

const frontendPort = 3001;
const backendPort = 4100;

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: false,
  retries: 0,
  workers: 1,
  use: {
    baseURL: `http://127.0.0.1:${frontendPort}`,
    trace: "retain-on-failure",
  },
  webServer: [
    {
      command: `node e2e/mock-review-backend.mjs`,
      url: `http://127.0.0.1:${backendPort}/__playwright__/health`,
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
      env: {
        MOCK_REVIEW_BACKEND_HOST: "127.0.0.1",
        MOCK_REVIEW_BACKEND_PORT: String(backendPort),
      },
    },
    {
      command: `pnpm exec next dev --hostname 127.0.0.1 --port ${frontendPort}`,
      url: `http://127.0.0.1:${frontendPort}/projects/prj-001/reviews`,
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
      env: {
        BACKEND_BASE_URL: `http://127.0.0.1:${backendPort}`,
      },
    },
  ],
});
