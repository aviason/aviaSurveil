import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "node",
    include: [
      "tests/contract/http-backend-live.test.ts",
      "src/backend/governed-checklist-http-parity.test.ts",
    ],
    testTimeout: 15_000,
    hookTimeout: 15_000,
    restoreMocks: true,
  },
});
