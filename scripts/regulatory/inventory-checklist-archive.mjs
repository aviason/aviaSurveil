#!/usr/bin/env node
// Read-only metadata verifier. It streams the configured archive and central
// directory; it never extracts, copies, or writes source content.
import { createHash } from "node:crypto";
import { createReadStream } from "node:fs";
import { open, stat } from "node:fs/promises";
import { basename } from "node:path";

const archivePath = process.env.AGA_CHECKLIST_ARCHIVE;
if (!archivePath) {
  console.error("AGA_CHECKLIST_ARCHIVE is required");
  process.exit(2);
}
const metadata = await stat(archivePath);
const hash = createHash("sha256");
for await (const chunk of createReadStream(archivePath)) hash.update(chunk);
const handle = await open(archivePath, "r");
try {
  const tailSize = Math.min(metadata.size, 65_557);
  const tail = Buffer.alloc(tailSize);
  await handle.read(tail, 0, tailSize, metadata.size - tailSize);
  let eocd = -1;
  for (let offset = tail.length - 22; offset >= 0; offset -= 1) {
    if (tail.readUInt32LE(offset) === 0x06054b50) { eocd = offset; break; }
  }
  if (eocd < 0) throw new Error("ZIP end-of-central-directory is required");
  const count = tail.readUInt16LE(eocd + 10);
  const directorySize = tail.readUInt32LE(eocd + 12);
  const directoryOffset = tail.readUInt32LE(eocd + 16);
  if (count === 0xffff || directorySize === 0xffffffff || directoryOffset + directorySize > metadata.size) throw new Error("ZIP64 or out-of-bounds central directory");
  const directory = Buffer.alloc(directorySize);
  await handle.read(directory, 0, directorySize, directoryOffset);
  const names = [];
  let offset = 0;
  while (offset < directory.length) {
    if (directory.readUInt32LE(offset) !== 0x02014b50) throw new Error("invalid central directory entry");
    const flags = directory.readUInt16LE(offset + 8);
    const method = directory.readUInt16LE(offset + 10);
    const nameBytes = directory.readUInt16LE(offset + 28);
    const extraBytes = directory.readUInt16LE(offset + 30);
    const commentBytes = directory.readUInt16LE(offset + 32);
    const name = directory.subarray(offset + 46, offset + 46 + nameBytes).toString("utf8");
    if (!name || name.includes("\0") || name.includes("\\") || name.startsWith("/") || /^[A-Za-z]:/.test(name) || name.split("/").includes("..")) throw new Error(`unsafe ZIP path: ${name}`);
    if ((flags & 1) !== 0 || (method !== 0 && method !== 8)) throw new Error(`unsupported ZIP entry: ${name}`);
    names.push(name);
    offset += 46 + nameBytes + extraBytes + commentBytes;
  }
  if (offset !== directory.length || names.length !== count || names.some((name) => !name.toLowerCase().endsWith(".pdf"))) throw new Error("AGA archive inventory is not PDF-only");
  console.log(JSON.stringify({ archive: basename(archivePath), archiveBytes: metadata.size, archiveSha256: hash.digest("hex"), entryCount: names.length, pdfCount: names.length }, null, 2));
} finally {
  await handle.close();
}
