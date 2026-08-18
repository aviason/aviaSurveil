import { describe, expect, it } from "vitest";

import {
  classifyOfficialBrowser,
  type BrowserVersionPolicy,
} from "./browser-policy";

const policy: BrowserVersionPolicy = {
  safariStableMajor: 18,
  chromeStableMajor: 140,
  minimumOsVersionByFamily: {},
};

describe("official browser admission", () => {
  it("accepts Safari and Chrome on iOS/iPadOS through one WebKit lane", () => {
    const safari = classifyOfficialBrowser(
      "Mozilla/5.0 (iPhone; CPU iPhone OS 17_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
      { policy },
    );
    const chrome = classifyOfficialBrowser(
      "Mozilla/5.0 (iPhone; CPU iPhone OS 17_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/139.0.7258.76 Mobile/15E148 Safari/604.1",
      { policy },
    );

    expect(safari).toMatchObject({
      official: true,
      family: "ios-ipados-safari",
      engineLane: "webkit",
      versionLane: "stable-n",
    });
    expect(chrome).toMatchObject({
      official: true,
      family: "ios-ipados-chrome",
      engineLane: "webkit",
      versionLane: "stable-n-1",
    });
  });

  it("accepts only the six official platform/browser families", () => {
    const unsupported = [
      "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:141.0) Gecko/20100101 Firefox/141.0",
      "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/28.0 Chrome/140.0.0.0 Mobile Safari/537.36",
      "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_6) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Safari/605.1.15 Chrome/140.0.0.0",
    ];

    for (const userAgent of unsupported) {
      expect(classifyOfficialBrowser(userAgent, { policy })).toMatchObject({
        official: false,
        reasonCode: "UNSUPPORTED_BROWSER",
      });
    }
  });

  it("rejects stale browser versions and missing release policy", () => {
    const stale = classifyOfficialBrowser(
      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36",
      { policy },
    );
    const missingPolicy = classifyOfficialBrowser(
      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36",
    );

    expect(stale).toMatchObject({
      official: false,
      reasonCode: "BROWSER_VERSION_UNSUPPORTED",
      family: "windows-chrome",
    });
    expect(missingPolicy).toMatchObject({
      official: false,
      reasonCode: "BROWSER_POLICY_UNAVAILABLE",
    });
  });
});
