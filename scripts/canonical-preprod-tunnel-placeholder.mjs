#!/usr/bin/env node

import http from "node:http";

const port = Number(process.argv[2] ?? 8085);
if (!Number.isInteger(port) || port < 1024 || port > 65535) {
  throw new Error("placeholder port must be a user-space TCP port");
}

const server = http.createServer((request, response) => {
  if (request.url === "/__canonical_preprod_tunnel_placeholder_ready") {
    response.writeHead(200, { "content-type": "text/plain; charset=utf-8" });
    response.end("ready\n");
    return;
  }
  response.writeHead(503, { "content-type": "text/plain; charset=utf-8" });
  response.end("canonical local preprod is starting\n");
});

server.listen(port, "127.0.0.1", () => {
  process.stdout.write(`canonical-preprod tunnel placeholder listening on 127.0.0.1:${port}\n`);
});

function close() {
  server.close(() => process.exit(0));
}
process.on("SIGTERM", close);
process.on("SIGINT", close);
