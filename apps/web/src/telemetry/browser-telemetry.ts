import {
  browserTelemetryContract,
  validateBrowserTelemetryContract,
  type BrowserTelemetrySignal,
} from "./telemetry-contract";

export interface BrowserTelemetryRecord {
  name: string;
  kind: BrowserTelemetrySignal["kind"];
  unit: string;
  timestamp: string;
  attributes: Record<string, string | number>;
  value?: number;
  traceId?: string;
  spanId?: string;
}

export interface BrowserTelemetryBatch {
  resource: {
    "service.name": "aviasurveil360-web";
    "service.version": string;
    "deployment.environment.name": string;
    "build.profile": "demo" | "http";
  };
  records: BrowserTelemetryRecord[];
}

export interface BrowserTelemetryTransport {
  send(batch: BrowserTelemetryBatch): Promise<void>;
}

export interface BrowserTelemetry {
  recordNavigation(routeID: string, navigationType: string): void;
  recordWebVital(name: string, value: number, rating: string, routeID: string): void;
  recordAPIOutcome(operationClass: string, outcome: string, routeID: string): void;
  recordHandledError(error: unknown, routeID: string): void;
  requestHeaders(): Record<string, string>;
  flush(): Promise<{ delivered: boolean; count: number }>;
  shutdown(): Promise<void>;
}

interface BrowserTelemetryOptions {
  buildProfile: "demo" | "http";
  serviceVersion: string;
  transport?: BrowserTelemetryTransport;
  now?: () => Date;
  autoFlush?: boolean;
  correlationID?: string;
}

const safeRouteID = /^[a-z][a-z0-9-]{0,63}$/;
const safeCorrelationID = /^[A-Za-z0-9._-]{8,128}$/;
const webVitals = new Set(["LCP", "CLS", "INP", "FCP", "TTFB"]);
const ratings = new Set(["good", "needs-improvement", "poor"]);
const navigationTypes = new Set(["load", "push", "replace", "popstate"]);
const operationClasses = new Set(["read", "command"]);
const outcomes = new Set(["succeeded", "failed", "canceled"]);
let activeTelemetry: BrowserTelemetry | null = null;
let activeRouteID: (() => string) | null = null;

export function activateBrowserTelemetry(
  telemetry: BrowserTelemetry,
  routeID: () => string,
): () => void {
  activeTelemetry = telemetry;
  activeRouteID = routeID;
  return () => {
    if (activeTelemetry === telemetry) {
      activeTelemetry = null;
      activeRouteID = null;
    }
  };
}

export function recordActiveBrowserAPIOutcome(
  operationClass: string,
  outcome: string,
): void {
  activeTelemetry?.recordAPIOutcome(
    operationClass,
    outcome,
    activeRouteID?.() ?? "unknown",
  );
}

export function activeBrowserRequestHeaders(): Record<string, string> {
  return activeTelemetry?.requestHeaders() ?? {};
}

export function createBrowserTelemetry(
  options: BrowserTelemetryOptions,
): BrowserTelemetry {
  const contract = browserTelemetryContract();
  const contractErrors = validateBrowserTelemetryContract(contract);
  if (contractErrors.length > 0) {
    throw new Error(`invalid browser telemetry contract: ${contractErrors.join("; ")}`);
  }
  const signalByName = new Map(contract.signals.map((signal) => [signal.name, signal]));
  const transport = options.transport ?? fetchTransport();
  const now = options.now ?? (() => new Date());
  const correlationID = safeCorrelationID.test(options.correlationID ?? "")
    ? options.correlationID!
    : randomHex(16);
  const queue: BrowserTelemetryRecord[] = [];
  let currentTraceID = randomHex(16);
  let currentSpanID = randomHex(8);
  let stopped = false;
  let flushScheduled = false;

  const enqueue = (
    name: string,
    attributes: Record<string, string | number>,
    value?: number,
    spanContext?: { traceId: string; spanId: string },
  ): void => {
    if (stopped) return;
    const signal = signalByName.get(name);
    if (!signal) return;
    queue.push({
      name,
      kind: signal.kind,
      unit: signal.unit,
      timestamp: now().toISOString(),
      attributes: allowlistedAttributes(signal, attributes),
      ...(value === undefined ? {} : { value }),
      ...(spanContext ?? {}),
    });
    if (options.autoFlush && !flushScheduled) {
      flushScheduled = true;
      queueMicrotask(() => {
        flushScheduled = false;
        void flush();
      });
    }
  };

  const flush = async (): Promise<{ delivered: boolean; count: number }> => {
    const records = queue.splice(0, queue.length);
    if (records.length === 0) return { delivered: true, count: 0 };
    const batch: BrowserTelemetryBatch = {
      resource: {
        "service.name": "aviasurveil360-web",
        "service.version": options.serviceVersion,
        "deployment.environment.name": options.buildProfile === "http" ? "production" : "demo",
        "build.profile": options.buildProfile,
      },
      records,
    };
    try {
      await transport.send(batch);
      return { delivered: true, count: records.length };
    } catch {
      return { delivered: false, count: records.length };
    }
  };

  return {
    recordNavigation(routeID, navigationType) {
      currentTraceID = randomHex(16);
      currentSpanID = randomHex(8);
      enqueue("browser.route.navigation", {
        "route.id": boundedRouteID(routeID),
        "build.profile": options.buildProfile,
        "navigation.type": boundedValue(navigationType, navigationTypes),
        "outcome.class": "succeeded",
        "correlation.id": correlationID,
      }, undefined, {
        traceId: currentTraceID,
        spanId: currentSpanID,
      });
    },
    recordWebVital(name, value, rating, routeID) {
      const metricValue = Number.isFinite(value) ? Math.max(0, value) : 0;
      enqueue("browser.web_vital", {
        "route.id": boundedRouteID(routeID),
        "build.profile": options.buildProfile,
        "web_vital.name": boundedValue(name, webVitals),
        rating: boundedValue(rating, ratings),
      }, metricValue);
    },
    recordAPIOutcome(operationClass, outcome, routeID) {
      enqueue("browser.api.outcome", {
        "route.id": boundedRouteID(routeID),
        "build.profile": options.buildProfile,
        "operation.class": boundedValue(operationClass, operationClasses),
        "outcome.class": boundedValue(outcome, outcomes),
      }, 1);
    },
    recordHandledError(error, routeID) {
      enqueue("browser.error.handled", {
        "route.id": boundedRouteID(routeID),
        "build.profile": options.buildProfile,
        "error.class": boundedErrorClass(error),
        "outcome.class": "handled",
        "correlation.id": correlationID,
      }, undefined, {
        traceId: currentTraceID,
        spanId: currentSpanID,
      });
    },
    requestHeaders() {
      return {
        traceparent: `00-${currentTraceID}-${currentSpanID}-01`,
        "X-Correlation-ID": correlationID,
      };
    },
    flush,
    async shutdown() {
      await flush();
      stopped = true;
    },
  };
}

function allowlistedAttributes(
  signal: BrowserTelemetrySignal,
  attributes: Record<string, string | number>,
): Record<string, string | number> {
  const allowed = new Set(signal.allowedAttributes);
  return Object.fromEntries(
    Object.entries(attributes)
      .filter(([name]) => allowed.has(name))
      .sort(([left], [right]) => left.localeCompare(right)),
  );
}

function boundedRouteID(routeID: string): string {
  return safeRouteID.test(routeID) ? routeID : "unknown";
}

function boundedValue(value: string, allowed: Set<string>): string {
  return allowed.has(value) ? value : "other";
}

function boundedErrorClass(error: unknown): string {
  if (!(error instanceof Error)) return "unknown";
  return /^[A-Za-z][A-Za-z0-9]{0,63}$/.test(error.name) ? error.name : "Error";
}

function fetchTransport(): BrowserTelemetryTransport {
  return {
    async send(batch) {
      const requests: Promise<Response>[] = [];
      const spans = batch.records.filter((record) => record.kind === "span");
      const metrics = batch.records.filter((record) => record.kind === "metric");
      const logs = batch.records.filter((record) => record.kind === "log");
      if (spans.length > 0) {
        requests.push(postOTLP("/otel/v1/traces", toOTLPTraces(batch, spans)));
      }
      if (metrics.length > 0) {
        requests.push(postOTLP("/otel/v1/metrics", toOTLPMetrics(batch, metrics)));
      }
      if (logs.length > 0) {
        requests.push(postOTLP("/otel/v1/logs", toOTLPLogs(batch, logs)));
      }
      const responses = await Promise.all(requests);
      if (responses.some((response) => !response.ok)) {
        throw new Error("telemetry collector unavailable");
      }
    },
  };
}

function postOTLP(path: string, body: unknown): Promise<Response> {
  return fetch(path, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(body),
    credentials: "same-origin",
    keepalive: true,
  });
}

function toOTLPTraces(
  batch: BrowserTelemetryBatch,
  records: BrowserTelemetryRecord[],
) {
  return {
    resourceSpans: [
      {
        resource: { attributes: resourceAttributes(batch) },
        scopeSpans: [
          {
            scope: { name: "aviasurveil360-browser" },
            spans: records.map((record) => {
              const timestamp = unixNano(record.timestamp);
              return {
                traceId: record.traceId ?? randomHex(16),
                spanId: record.spanId ?? randomHex(8),
                name: record.name,
                kind: 1,
                startTimeUnixNano: timestamp,
                endTimeUnixNano: timestamp,
                attributes: otlpAttributes(record.attributes),
                status: { code: 1 },
              };
            }),
          },
        ],
      },
    ],
  };
}

function toOTLPMetrics(
  batch: BrowserTelemetryBatch,
  records: BrowserTelemetryRecord[],
) {
  const grouped = new Map<string, BrowserTelemetryRecord[]>();
  for (const record of records) {
    grouped.set(record.name, [...(grouped.get(record.name) ?? []), record]);
  }
  return {
    resourceMetrics: [
      {
        resource: { attributes: resourceAttributes(batch) },
        scopeMetrics: [
          {
            scope: { name: "aviasurveil360-browser" },
            metrics: [...grouped.entries()].map(([name, points]) => ({
              name,
              unit: points[0]?.unit ?? "1",
              ...(name === "browser.api.outcome"
                ? {
                    sum: {
                      aggregationTemporality: 2,
                      isMonotonic: true,
                      dataPoints: points.map((record) => ({
                        timeUnixNano: unixNano(record.timestamp),
                        asInt: "1",
                        attributes: otlpAttributes(record.attributes),
                      })),
                    },
                  }
                : {
                    gauge: {
                      dataPoints: points.map((record) => ({
                        timeUnixNano: unixNano(record.timestamp),
                        asDouble: record.value ?? 0,
                        attributes: otlpAttributes(record.attributes),
                      })),
                    },
                  }),
            })),
          },
        ],
      },
    ],
  };
}

function toOTLPLogs(
  batch: BrowserTelemetryBatch,
  records: BrowserTelemetryRecord[],
) {
  return {
    resourceLogs: [
      {
        resource: { attributes: resourceAttributes(batch) },
        scopeLogs: [
          {
            scope: { name: "aviasurveil360-browser" },
            logRecords: records.map((record) => ({
              timeUnixNano: unixNano(record.timestamp),
              severityText: "INFO",
              body: { stringValue: record.name },
              attributes: otlpAttributes(record.attributes),
              ...(record.traceId ? { traceId: record.traceId } : {}),
              ...(record.spanId ? { spanId: record.spanId } : {}),
            })),
          },
        ],
      },
    ],
  };
}

function resourceAttributes(batch: BrowserTelemetryBatch) {
  return Object.entries(batch.resource).map(([key, value]) => ({
    key,
    value: { stringValue: value },
  }));
}

function otlpAttributes(attributes: Record<string, string | number>) {
  return Object.entries(attributes).map(([key, value]) => ({
    key,
    value:
      typeof value === "number"
        ? { doubleValue: value }
        : { stringValue: value },
  }));
}

function unixNano(timestamp: string): string {
  return String(BigInt(new Date(timestamp).getTime()) * 1_000_000n);
}

function randomHex(byteLength: number): string {
  const bytes = new Uint8Array(byteLength);
  globalThis.crypto.getRandomValues(bytes);
  return Array.from(bytes, (value) => value.toString(16).padStart(2, "0")).join("");
}
