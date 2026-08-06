import { createServer, type Server } from "node:http";
import { readFile, stat } from "node:fs/promises";
import { extname, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

const buildRoot = resolve(fileURLToPath(new URL("../../../dist/demo/", import.meta.url)));

const contentTypes: Record<string, string> = {
  ".css": "text/css; charset=utf-8",
  ".html": "text/html; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".json": "application/json; charset=utf-8",
  ".map": "application/json; charset=utf-8",
};

export class OfflineStaticServer {
  private server: Server | null = null;
  private servedShellVersion = 1;

  constructor(readonly port: number) {}

  get origin(): string {
    return `http://127.0.0.1:${this.port}`;
  }

  setShellVersion(version: number): void {
    this.servedShellVersion = version;
  }

  async start(): Promise<void> {
    if (this.server) return;
    this.server = createServer(async (request, response) => {
      try {
        const requestURL = new URL(request.url ?? "/", this.origin);
        const relativePath = decodeURIComponent(requestURL.pathname).replace(/^\/+/, "");
        let filePath = resolve(buildRoot, relativePath || "index.html");
        if (filePath !== buildRoot && !filePath.startsWith(`${buildRoot}${sep}`)) {
          response.writeHead(403).end("Forbidden");
          return;
        }
        try {
          if (!(await stat(filePath)).isFile()) throw new Error("not a file");
        } catch {
          filePath = resolve(buildRoot, "index.html");
        }
        let body = await readFile(filePath);
        if (requestURL.pathname === "/sw.js") {
          const marker = "AVIA_APP_SHELL_VERSION:000002";
          const replacement = `AVIA_APP_SHELL_VERSION:${String(this.servedShellVersion).padStart(6, "0")}`;
          const original = body.toString("utf8");
          if (!original.includes(marker)) throw new Error("Built Service Worker version marker is missing");
          const source = original.replaceAll(marker, replacement);
          body = Buffer.from(`${source}\n// offline-test-shell-${this.servedShellVersion}\n`);
        } else if (requestURL.pathname === "/app-shell-assets.json") {
          const manifest = JSON.parse(body.toString("utf8")) as { appShellVersion: number };
          manifest.appShellVersion = this.servedShellVersion;
          body = Buffer.from(`${JSON.stringify(manifest, null, 2)}\n`);
        }
        response.setHeader("Content-Type", contentTypes[extname(filePath)] ?? "application/octet-stream");
        response.setHeader(
          "Cache-Control",
          requestURL.pathname === "/sw.js" ? "no-store" : "no-cache",
        );
        response.setHeader("Service-Worker-Allowed", "/");
        response.writeHead(200).end(body);
      } catch (error) {
        response.writeHead(500).end(error instanceof Error ? error.message : "Static server failure");
      }
    });
    await new Promise<void>((resolveStart, reject) => {
      this.server?.once("error", reject);
      this.server?.listen(this.port, "127.0.0.1", () => resolveStart());
    });
  }

  async stop(): Promise<void> {
    const active = this.server;
    this.server = null;
    if (!active) return;
    active.closeAllConnections();
    await new Promise<void>((resolveStop, reject) => {
      active.close((error) => (error ? reject(error) : resolveStop()));
    });
  }
}
