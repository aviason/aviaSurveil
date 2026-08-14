import { defineConfig } from "@playwright/test";

const e2eProfile = process.env.AVIA_E2E_PROFILE;
const profile =
  e2eProfile === "local-demo"
    ? "local-demo"
    : e2eProfile === "http"
      ? "http"
      : e2eProfile === "canonical-quick-tunnel"
        ? "canonical-quick-tunnel"
        : e2eProfile === "offline"
          ? "offline"
          : e2eProfile === "visual-parity"
            ? "visual-parity"
            : "mock";
const command =
  profile === "http"
    ? "AVIA_HTTP_TEST_PROFILE=canonical npm run dev:http -- --host 127.0.0.1 --port 4174 --strictPort"
    : profile === "visual-parity"
      ? "VITE_AVIA_VISUAL_FIXTURES=1 npm run dev:demo -- --host 127.0.0.1 --port 4174 --strictPort"
    : "npm run dev:demo -- --host 127.0.0.1 --port 4174 --strictPort";
const shouldStartWebServer =
  profile !== "offline" &&
  profile !== "canonical-quick-tunnel" &&
  profile !== "local-demo" &&
  process.env.AVIA_UPDATE_LEGACY_BASELINES !== "1";
const visualUse = {
  browserName: "chromium" as const,
  colorScheme: "light" as const,
  deviceScaleFactor: 1,
  headless: true,
  locale: "en-GB",
  reducedMotion: "reduce" as const,
  serviceWorkers: "block" as const,
  timezoneId: "UTC",
  viewport: { width: 1440, height: 900 },
};

export default defineConfig({
  testDir: "./tests",
  outputDir:
    process.env.AVIA_PLAYWRIGHT_OUTPUT_DIR ??
    "/private/tmp/aviasurveil360-react-playwright-results",
  fullyParallel: false,
  workers: 1,
  forbidOnly: true,
  retries: 0,
  maxFailures: 0,
  reporter: [["line"]],
  use: {
    baseURL: process.env.AVIA_E2E_BASE_URL ?? "http://127.0.0.1:4174",
    browserName: "chromium",
    headless: true,
    ignoreHTTPSErrors: process.env.AVIA_E2E_IGNORE_HTTPS_ERRORS === "1",
    viewport: { width: 1440, height: 900 },
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "off",
  },
  webServer:
    !shouldStartWebServer
      ? undefined
      : {
          command,
          url: "http://127.0.0.1:4174",
          reuseExistingServer: false,
          timeout: 30_000,
          stdout: "pipe",
          stderr: "pipe",
        },
  projects: [
    {
      name: "local-demo",
      testMatch: ["e2e/local-demo-platform.spec.ts"],
    },
    {
      name: "mock",
      testMatch: [
        "e2e/canonical-scenario.spec.ts",
        "e2e/first-production-routes.spec.ts",
        "e2e/release-candidate-gates.spec.ts",
        "e2e/full-route-accessibility.spec.ts",
        "e2e/full-platform-scenarios.spec.ts",
        "e2e/visible-action-contract.spec.ts",
        "e2e/manager-responsive-contract.spec.ts",
        "e2e/manager-intelligence-responsive-contract.spec.ts",
        "e2e/executive-responsive-contract.spec.ts",
        "e2e/auditee-responsive-contract.spec.ts",
        "e2e/admin-responsive-contract.spec.ts",
        "e2e/regulatory-checklist-governance.spec.ts",
        "e2e/governed-checklist-intake.spec.ts",
      ],
    },
    {
      name: "http",
      testMatch: [
        "e2e/canonical-scenario.spec.ts",
        "e2e/first-production-routes.spec.ts",
        "e2e/full-platform-scenarios.spec.ts",
        "e2e/offline-sync.http.spec.ts",
        "e2e/release-candidate-gates.spec.ts",
        "e2e/visible-action-contract.spec.ts",
        "e2e/generated-document.http.spec.ts",
        "e2e/notification-delivery.http.spec.ts",
        "e2e/local-service-failures.http.spec.ts",
        "e2e/user-lifecycle.http.spec.ts",
        "e2e/regulatory-checklist-governance.http.spec.ts",
        "e2e/regulatory-source-refresh.http.spec.ts",
        "e2e/governed-checklist-intake.http.spec.ts",
      ],
    },
    {
      name: "canonical-quick-tunnel",
      testMatch: [
        "e2e/canonical-quick-tunnel-panels.spec.ts",
        "e2e/canonical-quick-tunnel-lifecycle.spec.ts",
      ],
      use: {
        actionTimeout: 30_000,
        navigationTimeout: 30_000,
        serviceWorkers: "allow",
        launchOptions: {
          args: process.env.AVIA_E2E_IGNORE_HTTPS_ERRORS === "1"
            ? ["--ignore-certificate-errors", "--allow-insecure-localhost"]
            : [],
        },
        trace: "off",
        screenshot: "off",
        video: "off",
      },
    },
    {
      name: "offline",
      testMatch: [
        "e2e/brand-app-shell-restart.spec.ts",
        "e2e/offline-*.spec.ts",
        "e2e/attachment-restart-recovery.spec.ts",
        "offline/restart-recovery.spec.ts",
      ],
      testIgnore: ["e2e/offline-sync.http.spec.ts"],
    },
    {
      name: "legacy-baseline-update",
      testMatch: ["e2e/legacy-baseline-update.spec.ts"],
      use: visualUse,
    },
    {
      name: "legacy-parity",
      testMatch: ["e2e/legacy-visual-parity.spec.ts"],
      use: visualUse,
    },
  ],
});
