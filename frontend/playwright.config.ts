import { defineConfig } from "@playwright/test";

const baseURL = "http://localhost:3100";

export default defineConfig({
  testDir: "./tests",
  workers: 1,
  use: {
    baseURL,
    browserName: "chromium",
    launchOptions: process.env.TEST_CHROMIUM_PATH
      ? { executablePath: process.env.TEST_CHROMIUM_PATH }
      : {},
  },
  webServer: {
    command: "npm run dev -- --port 3100",
    url: baseURL,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
    env: {
      NEXT_PUBLIC_API_URL: `${baseURL}/mock-api`,
      NEXT_PUBLIC_SUPABASE_URL: `${baseURL}/mock-auth`,
      NEXT_PUBLIC_SUPABASE_ANON_KEY: "fixture-anon-key",
    },
  },
});
