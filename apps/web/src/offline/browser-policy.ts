export const OFFICIAL_BROWSER_FAMILIES = [
  "ios-ipados-safari",
  "ios-ipados-chrome",
  "macos-safari",
  "macos-chrome",
  "android-chrome",
  "windows-chrome",
] as const;

export type OfficialBrowserFamily = (typeof OFFICIAL_BROWSER_FAMILIES)[number];
export type BrowserEngineLane = "webkit" | "chromium";
export type BrowserVersionLane = "stable-n" | "stable-n-1" | "unsupported" | "policy-unavailable";
export type BrowserAdmissionReasonCode =
  | "OFFICIAL_BROWSER"
  | "UNSUPPORTED_BROWSER"
  | "UNSUPPORTED_CHANNEL"
  | "BROWSER_POLICY_UNAVAILABLE"
  | "BROWSER_VERSION_UNSUPPORTED"
  | "OS_VERSION_UNSUPPORTED";

export interface BrowserVersionPolicy {
  safariStableMajor: number | null;
  chromeStableMajor: number | null;
  minimumOsVersionByFamily: Partial<Record<OfficialBrowserFamily, string>>;
}

export const DEFAULT_BROWSER_VERSION_POLICY: Readonly<BrowserVersionPolicy> = Object.freeze({
  safariStableMajor: null,
  chromeStableMajor: null,
  minimumOsVersionByFamily: {},
});

export interface OfficialBrowserClassification {
  official: boolean;
  family: OfficialBrowserFamily | null;
  engineLane: BrowserEngineLane | null;
  majorVersion: number | null;
  osVersion: string | null;
  versionLane: BrowserVersionLane;
  reasonCode: BrowserAdmissionReasonCode;
}

interface BrowserMatch {
  family: OfficialBrowserFamily;
  engineLane: BrowserEngineLane;
  majorVersion: number | null;
  osVersion: string | null;
  stableMajor: number | null;
}

function matchMajor(userAgent: string, expression: RegExp): number | null {
  const match = userAgent.match(expression);
  const major = match?.[1] ? Number.parseInt(match[1], 10) : Number.NaN;
  return Number.isSafeInteger(major) && major > 0 ? major : null;
}

function matchOsVersion(userAgent: string, expression: RegExp, separator = "."): string | null {
  const match = userAgent.match(expression);
  if (!match?.[1]) return null;
  return match[1].split(separator).join(".");
}

function hasUnsupportedChannelMarker(userAgent: string): boolean {
  return /(?:HeadlessChrome|Chromium|\b(?:Beta|Dev|Canary|Nightly|Technology Preview|Preview)\b)/i.test(
    userAgent,
  );
}

function isIosOrIpados(userAgent: string, maxTouchPoints: number): boolean {
  return /iPhone|iPad|iPod/i.test(userAgent) ||
    (/Macintosh/i.test(userAgent) && maxTouchPoints > 1 && /Mobile\//i.test(userAgent));
}

function classifyUserAgent(userAgent: string, maxTouchPoints: number): BrowserMatch | null {
  const ios = isIosOrIpados(userAgent, maxTouchPoints);
  if (ios) {
    const iosVersion = matchOsVersion(userAgent, /(?:CPU (?:iPhone )?OS|iPhone OS) ([0-9_]+)/i, "_");
    const chromeMajor = matchMajor(userAgent, /CriOS\/([0-9]+)/i);
    if (chromeMajor !== null && !hasUnsupportedChannelMarker(userAgent)) {
      return {
        family: "ios-ipados-chrome",
        engineLane: "webkit",
        majorVersion: chromeMajor,
        osVersion: iosVersion,
        stableMajor: null,
      };
    }
    const safariMajor = matchMajor(userAgent, /Version\/([0-9]+)/i);
    if (
      safariMajor !== null &&
      /Safari\//i.test(userAgent) &&
      !/(?:CriOS|FxiOS|EdgiOS|OPiOS|GSA)\//i.test(userAgent) &&
      !hasUnsupportedChannelMarker(userAgent)
    ) {
      return {
        family: "ios-ipados-safari",
        engineLane: "webkit",
        majorVersion: safariMajor,
        osVersion: iosVersion,
        stableMajor: null,
      };
    }
    return null;
  }

  if (/Android/i.test(userAgent)) {
    const chromeMajor = matchMajor(userAgent, /Chrome\/([0-9]+)/i);
    if (
      chromeMajor !== null &&
      /Mobile/i.test(userAgent) &&
      !/(?:EdgA|OPR|SamsungBrowser|UCBrowser|YaBrowser|CriOS)\//i.test(userAgent) &&
      !hasUnsupportedChannelMarker(userAgent)
    ) {
      return {
        family: "android-chrome",
        engineLane: "chromium",
        majorVersion: chromeMajor,
        osVersion: matchOsVersion(userAgent, /Android ([0-9.]+)/i),
        stableMajor: null,
      };
    }
    return null;
  }

  if (/Windows NT/i.test(userAgent)) {
    const chromeMajor = matchMajor(userAgent, /Chrome\/([0-9]+)/i);
    if (
      chromeMajor !== null &&
      !/(?:Edg|OPR|Brave|UCBrowser|YaBrowser)\//i.test(userAgent) &&
      !/Version\//i.test(userAgent) &&
      !hasUnsupportedChannelMarker(userAgent)
    ) {
      return {
        family: "windows-chrome",
        engineLane: "chromium",
        majorVersion: chromeMajor,
        osVersion: matchOsVersion(userAgent, /Windows NT ([0-9.]+)/i),
        stableMajor: null,
      };
    }
    return null;
  }

  if (/Macintosh/i.test(userAgent)) {
    const chromeMajor = matchMajor(userAgent, /Chrome\/([0-9]+)/i);
    if (
      chromeMajor !== null &&
      !/(?:Edg|OPR|Brave|UCBrowser|YaBrowser)\//i.test(userAgent) &&
      !/Version\//i.test(userAgent) &&
      !hasUnsupportedChannelMarker(userAgent)
    ) {
      return {
        family: "macos-chrome",
        engineLane: "chromium",
        majorVersion: chromeMajor,
        osVersion: matchOsVersion(userAgent, /Mac OS X ([0-9_]+)/i, "_"),
        stableMajor: null,
      };
    }
    const safariMajor = matchMajor(userAgent, /Version\/([0-9]+)/i);
    if (
      safariMajor !== null &&
      /Safari\//i.test(userAgent) &&
      !/(?:CriOS|FxiOS|EdgiOS|OPiOS|Chrome|Chromium)\//i.test(userAgent) &&
      !hasUnsupportedChannelMarker(userAgent)
    ) {
      return {
        family: "macos-safari",
        engineLane: "webkit",
        majorVersion: safariMajor,
        osVersion: matchOsVersion(userAgent, /Mac OS X ([0-9_]+)/i, "_"),
        stableMajor: null,
      };
    }
  }
  return null;
}

function compareVersions(left: string, right: string): number {
  const leftParts = left.split(".").map((part) => Number.parseInt(part, 10) || 0);
  const rightParts = right.split(".").map((part) => Number.parseInt(part, 10) || 0);
  for (let index = 0; index < Math.max(leftParts.length, rightParts.length); index += 1) {
    const difference = (leftParts[index] ?? 0) - (rightParts[index] ?? 0);
    if (difference !== 0) return difference;
  }
  return 0;
}

function unsuccessful(
  match: BrowserMatch | null,
  reasonCode: Exclude<BrowserAdmissionReasonCode, "OFFICIAL_BROWSER">,
  versionLane: BrowserVersionLane = "unsupported",
): OfficialBrowserClassification {
  return {
    official: false,
    family: match?.family ?? null,
    engineLane: match?.engineLane ?? null,
    majorVersion: match?.majorVersion ?? null,
    osVersion: match?.osVersion ?? null,
    versionLane,
    reasonCode,
  };
}

export function classifyOfficialBrowser(
  userAgent: string,
  options: { policy?: BrowserVersionPolicy; maxTouchPoints?: number } = {},
): OfficialBrowserClassification {
  const match = classifyUserAgent(userAgent, options.maxTouchPoints ?? 0);
  if (!match) return unsuccessful(null, "UNSUPPORTED_BROWSER");
  const policy = options.policy ?? DEFAULT_BROWSER_VERSION_POLICY;
  const stableMajor = match.family.includes("safari")
    ? policy.safariStableMajor
    : policy.chromeStableMajor;
  if (stableMajor === null) return unsuccessful(match, "BROWSER_POLICY_UNAVAILABLE", "policy-unavailable");
  if (match.majorVersion === null) return unsuccessful(match, "BROWSER_VERSION_UNSUPPORTED");
  const minimumOsVersion = policy.minimumOsVersionByFamily[match.family];
  if (minimumOsVersion && (!match.osVersion || compareVersions(match.osVersion, minimumOsVersion) < 0)) {
    return unsuccessful(match, "OS_VERSION_UNSUPPORTED");
  }
  const versionLane =
    match.majorVersion === stableMajor
      ? "stable-n"
      : match.majorVersion === stableMajor - 1
        ? "stable-n-1"
        : "unsupported";
  if (versionLane === "unsupported") return unsuccessful(match, "BROWSER_VERSION_UNSUPPORTED");
  return {
    official: true,
    family: match.family,
    engineLane: match.engineLane,
    majorVersion: match.majorVersion,
    osVersion: match.osVersion,
    versionLane,
    reasonCode: "OFFICIAL_BROWSER",
  };
}
