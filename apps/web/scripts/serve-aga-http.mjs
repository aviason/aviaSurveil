import { createReadStream, promises as fs } from "node:fs";
import { createServer } from "node:http";
import { extname, join, relative, resolve } from "node:path";
import { URL } from "node:url";

const root = resolve(process.env.AVIA_AGA_WEB_ROOT ?? "/app/dist/http");
const apiTarget = new URL(process.env.AVIA_HTTP_API_TARGET ?? "http://127.0.0.1:58081");
const port = Number.parseInt(process.env.AVIA_AGA_WEB_PORT ?? "4174", 10);
const proxyPrefixes = ["/api", "/v1", "/auth", "/health"];
const hopByHopHeaders = new Set([
  "connection",
  "keep-alive",
  "proxy-authenticate",
  "proxy-authorization",
  "te",
  "trailer",
  "transfer-encoding",
  "upgrade",
]);
const contentTypes = {
  ".css": "text/css; charset=utf-8",
  ".gif": "image/gif",
  ".html": "text/html; charset=utf-8",
  ".ico": "image/x-icon",
  ".jpeg": "image/jpeg",
  ".jpg": "image/jpeg",
  ".js": "text/javascript; charset=utf-8",
  ".json": "application/json; charset=utf-8",
  ".png": "image/png",
  ".svg": "image/svg+xml",
  ".ttf": "font/ttf",
  ".webp": "image/webp",
  ".woff": "font/woff",
  ".woff2": "font/woff2",
};

function isProxyRequest(pathname) {
  return proxyPrefixes.some((prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`));
}

function proxyPath(pathname, search) {
  return pathname.startsWith("/api") ? `${pathname.slice(4) || "/"}${search}` : `${pathname}${search}`;
}

function copyHeaders(headers) {
  return Object.fromEntries(
    Object.entries(headers).filter(([name]) => !hopByHopHeaders.has(name.toLowerCase())),
  );
}

function proxy(request, response, requestUrl) {
  const targetPath = proxyPath(requestUrl.pathname, requestUrl.search);
  const headers = copyHeaders(request.headers);
  headers.host = apiTarget.host;
  const client = apiTarget.protocol === "https:" ? import("node:https") : import("node:http");
  void client.then(({ request: requestUpstream }) => {
    const upstream = requestUpstream(
      {
        hostname: apiTarget.hostname,
        port: apiTarget.port || undefined,
        method: request.method,
        path: targetPath,
        headers,
      },
      (upstreamResponse) => {
        response.writeHead(upstreamResponse.statusCode ?? 502, copyHeaders(upstreamResponse.headers));
        upstreamResponse.pipe(response);
      },
    );
    upstream.on("error", (error) => {
      if (!response.headersSent) response.writeHead(502, { "content-type": "text/plain; charset=utf-8" });
      response.end(`AGA demo API proxy failed: ${error.message}\n`);
    });
    request.pipe(upstream);
  });
}

async function serveStatic(request, response, requestUrl) {
  const requestedPath = decodeURIComponent(requestUrl.pathname);
  const candidate = resolve(root, `.${requestedPath}`);
  const relativePath = relative(root, candidate);
  const safeCandidate = relativePath === "" || (!relativePath.startsWith("..") && !relativePath.includes("../"));
  if (!safeCandidate) {
    response.writeHead(404);
    response.end();
    return;
  }
  let filePath = candidate;
  try {
    const stat = await fs.stat(filePath);
    if (!stat.isFile()) throw new Error("not a file");
  } catch {
    filePath = join(root, "index.html");
  }
  try {
    const stat = await fs.stat(filePath);
    response.writeHead(200, {
      "cache-control": filePath.endsWith("index.html") ? "no-store" : "public, max-age=31536000, immutable",
      "content-length": stat.size,
      "content-type": contentTypes[extname(filePath).toLowerCase()] ?? "application/octet-stream",
    });
    if (request.method !== "HEAD") createReadStream(filePath).pipe(response);
    else response.end();
  } catch {
    response.writeHead(404, { "content-type": "text/plain; charset=utf-8" });
    response.end("Not found\n");
  }
}

const server = createServer((request, response) => {
  let requestUrl;
  try {
    requestUrl = new URL(request.url ?? "/", `http://${request.headers.host ?? "localhost"}`);
  } catch {
    response.writeHead(400);
    response.end("Bad request\n");
    return;
  }
  if (isProxyRequest(requestUrl.pathname)) proxy(request, response, requestUrl);
  else void serveStatic(request, response, requestUrl);
});

server.listen(port, "0.0.0.0", () => {
  console.log(`AGA demo HTTP server listening on 0.0.0.0:${port}`);
});
