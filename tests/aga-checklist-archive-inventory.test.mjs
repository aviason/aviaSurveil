import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { createReadStream } from "node:fs";
import { open, stat } from "node:fs/promises";
import { basename, normalize } from "node:path";
import test from "node:test";
import { pipeline } from "node:stream/promises";
import { createInflateRaw } from "node:zlib";

const EXPECTED = {
  archiveHash: "dd819cfa6a670760e0cfceed94496e2e466dc53bac13e6fd792b1128314d6e32",
  archiveBytes: 12_227_415,
  registerHash: "29ed8384693b615926fc42a0ca4654be2ea9a36b0946f217975571ca0ad9564f",
  form048Hash: "495aa7b0a1edca1ac5e874e6a63f50b47c6d207aa264cc390970a7db1acdc6e3",
};

function unsafePath(name) {
  return !name || name.includes("\0") || name.includes("\\") || name.startsWith("/") || /^[A-Za-z]:/.test(name)
    || normalize(name).split(/[\\/]/).includes("..") || name.split("/").includes("..");
}

async function sha256Stream(path, start, end, inflater) {
  const hash = createHash("sha256");
  const stream = createReadStream(path, start === undefined ? {} : { start, end });
  await pipeline(stream, ...(inflater ? [inflater] : []), hash);
  return hash.digest("hex");
}

async function readExactly(handle, position, length) {
  const buffer = Buffer.alloc(length);
  const { bytesRead } = await handle.read(buffer, 0, length, position);
  assert.equal(bytesRead, length, "ZIP metadata is truncated");
  return buffer;
}

async function readZipEntries(path, size) {
  const handle = await open(path, "r");
  try {
    const tailLength = Math.min(size, 65_557);
    const tailStart = size - tailLength;
    const tail = await readExactly(handle, tailStart, tailLength);
    let eocd = -1;
    for (let offset = tail.length - 22; offset >= 0; offset -= 1) {
      if (tail.readUInt32LE(offset) === 0x06054b50) { eocd = offset; break; }
    }
    assert.notEqual(eocd, -1, "ZIP end-of-central-directory is required");
    const entryCount = tail.readUInt16LE(eocd + 10);
    const directorySize = tail.readUInt32LE(eocd + 12);
    const directoryOffset = tail.readUInt32LE(eocd + 16);
    assert.notEqual(entryCount, 0xffff, "ZIP64 is not accepted by AGA_ZIP_PDF_V1");
    assert.notEqual(directorySize, 0xffffffff, "ZIP64 is not accepted by AGA_ZIP_PDF_V1");
    assert.ok(directoryOffset + directorySize <= size, "central directory must be in archive bounds");
    const directory = await readExactly(handle, directoryOffset, directorySize);
    const entries = [];
    let offset = 0;
    while (offset < directory.length) {
      assert.equal(directory.readUInt32LE(offset), 0x02014b50, "central directory entry signature");
      const flags = directory.readUInt16LE(offset + 8);
      const method = directory.readUInt16LE(offset + 10);
      const compressedBytes = directory.readUInt32LE(offset + 20);
      const uncompressedBytes = directory.readUInt32LE(offset + 24);
      const nameBytes = directory.readUInt16LE(offset + 28);
      const extraBytes = directory.readUInt16LE(offset + 30);
      const commentBytes = directory.readUInt16LE(offset + 32);
      const localOffset = directory.readUInt32LE(offset + 42);
      const name = directory.subarray(offset + 46, offset + 46 + nameBytes).toString("utf8");
      assert.ok(!unsafePath(name), `unsafe ZIP entry path: ${JSON.stringify(name)}`);
      assert.equal(flags & 1, 0, `encrypted ZIP entry: ${name}`);
      assert.ok(method === 0 || method === 8, `unsupported ZIP compression method for ${name}`);
      entries.push({ name, method, compressedBytes, uncompressedBytes, localOffset });
      offset += 46 + nameBytes + extraBytes + commentBytes;
    }
    assert.equal(offset, directory.length, "central directory must parse exactly");
    assert.equal(entries.length, entryCount, "central-directory entry count");
    return entries;
  } finally {
    await handle.close();
  }
}

async function entryHash(path, entry) {
  const handle = await open(path, "r");
  try {
    const header = await readExactly(handle, entry.localOffset, 30);
    assert.equal(header.readUInt32LE(0), 0x04034b50, `local header signature for ${entry.name}`);
    const nameBytes = header.readUInt16LE(26);
    const extraBytes = header.readUInt16LE(28);
    const start = entry.localOffset + 30 + nameBytes + extraBytes;
    return sha256Stream(path, start, start + entry.compressedBytes - 1, entry.method === 8 ? createInflateRaw() : undefined);
  } finally {
    await handle.close();
  }
}

test("AGA_CHECKLIST_ARCHIVE is read-only, bounded, and matches the frozen inventory", async () => {
  assert.equal(unsafePath("..\\escape.pdf"), true, "single-backslash traversal is unsafe");
  assert.equal(unsafePath("folder\\file.pdf"), true, "backslash paths are unsafe");
  assert.equal(unsafePath("folder/file.pdf"), false, "normal archive paths remain valid");
  assert.equal(unsafePath("nul\0entry.pdf"), true, "NUL paths are unsafe");
  const archive = process.env.AGA_CHECKLIST_ARCHIVE;
  assert.ok(archive, "AGA_CHECKLIST_ARCHIVE must name the supplied AGA ZIP");
  const metadata = await stat(archive);
  assert.equal(metadata.size, EXPECTED.archiveBytes);
  assert.equal(await sha256Stream(archive), EXPECTED.archiveHash);

  const entries = await readZipEntries(archive, metadata.size);
  assert.equal(entries.length, 53);
  assert.ok(entries.length <= 128, "AGA_ZIP_PDF_V1 entry limit");
  assert.ok(entries.every((entry) => entry.name.toLowerCase().endsWith(".pdf")), "only PDF entries are accepted");
  const compressedBytes = entries.reduce((total, entry) => total + entry.compressedBytes, 0);
  const uncompressedBytes = entries.reduce((total, entry) => total + entry.uncompressedBytes, 0);
  assert.ok(compressedBytes > 0, "compressed archive entries must have non-zero bytes");
  assert.ok(uncompressedBytes / compressedBytes <= 20, "AGA_ZIP_PDF_V1 whole-archive expansion limit");
  assert.ok(entries.every((entry) => entry.uncompressedBytes > 0 && entry.uncompressedBytes / entry.compressedBytes <= 20), "AGA_ZIP_PDF_V1 expansion limit");

  const names = new Set();
  const hashes = new Map();
  for (const entry of entries) {
    assert.ok(!names.has(entry.name), `duplicate ZIP entry: ${entry.name}`);
    names.add(entry.name);
    hashes.set(entry.name, await entryHash(archive, entry));
  }
  const register = [...hashes.entries()].filter(([, hash]) => hash === EXPECTED.registerHash);
  const form048 = [...hashes.entries()].filter(([, hash]) => hash === EXPECTED.form048Hash);
  assert.equal(register.length, 1, "exactly one register hash");
  assert.equal(form048.length, 1, "exactly one Form 048 hash");
  assert.equal(basename(form048[0][0]), "FSS-AGA-FORM-048.pdf");

  const formCodes = [...names].map((name) => basename(name).match(/^FSS-AGA-FORM-([0-9]{3}A?)\.pdf$/i)?.[1]).filter(Boolean);
  assert.equal(formCodes.length, 52);
  assert.equal(new Set(formCodes).size, 52);
  assert.ok(formCodes.includes("035A"));
  assert.ok(!formCodes.includes("049"));
});
