#!/usr/bin/env node

import { createHash } from "node:crypto";
import { createReadStream, createWriteStream } from "node:fs";
import {
  access,
  mkdir,
  open,
  readFile,
  rename,
  stat,
  unlink,
  writeFile,
} from "node:fs/promises";
import { homedir } from "node:os";
import { basename, dirname, extname, join, relative, resolve } from "node:path";
import { pathToFileURL } from "node:url";
import { pipeline } from "node:stream/promises";
import { execFile, spawn } from "node:child_process";
import { promisify } from "node:util";

const execFileAsync = promisify(execFile);
const repositoryRoot = resolve(dirname(new URL(import.meta.url).pathname), "../..");
const defaultVaultRoot = join(
  repositoryRoot,
  ".local/aviasurveil360/regulatory-sources/ncaa/namcats/all-pages",
);
const defaultManifestPath = join(
  repositoryRoot,
  "docs/regulatory-sources/ncaa-namcats-manifest.json",
);
const sourcePages = [1, 2, 3];
const canonicalPageUrls = new Map(
  sourcePages.map((page) => [
    page,
    `https://www.ncaa.com.na/downloads.php?pagetitle=NAMCATS&page=${page}`,
  ]),
);
const allowedHosts = new Set(["www.ncaa.com.na", "ncaa.com.na"]);
const expectedPdfCounts = new Map([[1, 20], [2, 20], [3, 18]]);
const expectedPdfCount = 57;
const expectedWorkbookCount = 1;

function decodeHtml(value) {
  return value
    .replaceAll("&nbsp;", " ")
    .replaceAll("&amp;", "&")
    .replaceAll("&quot;", "\"")
    .replaceAll("&#039;", "'")
    .replaceAll("&#39;", "'")
    .replaceAll("&lt;", "<")
    .replaceAll("&gt;", ">")
    .replace(/&#(\d+);/gu, (_match, code) => String.fromCodePoint(Number(code)));
}

function visibleText(value) {
  return decodeHtml(
    value
      .replace(/<script\b[\s\S]*?<\/script>/giu, " ")
      .replace(/<style\b[\s\S]*?<\/style>/giu, " ")
      .replace(/<[^>]+>/gu, " "),
  )
    .replace(/\s+/gu, " ")
    .trim();
}

function hrefFromHtml(value) {
  const match = value.match(/\bhref\s*=\s*["']([^"']+)["']/iu);
  return match ? decodeHtml(match[1]) : null;
}

function safeSourceUrl(href, pageUrl) {
  const url = new URL(href, pageUrl);
  if (url.protocol !== "https:" || !allowedHosts.has(url.hostname)) {
    throw new Error(`unapproved source origin: ${url.href}`);
  }
  if (!url.pathname.startsWith("/publications/")) {
    throw new Error(`unapproved source path: ${url.pathname}`);
  }
  if (!/\.(?:pdf|xlsx)$/iu.test(url.pathname)) {
    throw new Error(`unapproved source extension: ${url.pathname}`);
  }
  url.hostname = "www.ncaa.com.na";
  return url.href;
}

function fileNameFromUrl(sourceUrl) {
  const fileName = decodeURIComponent(basename(new URL(sourceUrl).pathname));
  if (
    fileName === "." ||
    fileName === ".." ||
    fileName.includes("/") ||
    fileName.includes("\\") ||
    !/\.(?:pdf|xlsx)$/iu.test(fileName)
  ) {
    throw new Error(`unsafe source filename: ${fileName}`);
  }
  return fileName;
}

export function discoverPageDocuments(
  html,
  page,
  pageUrl = canonicalPageUrls.get(page),
) {
  if (!canonicalPageUrls.has(page) || !pageUrl) {
    throw new Error(`unsupported NAMCATS page: ${page}`);
  }
  const documents = [];
  const seenUrls = new Set();
  if (page === 1) {
    const indexAnchor = [...html.matchAll(/<a\b[^>]*href\s*=\s*["'][^"']+\.xlsx["'][^>]*>[\s\S]*?<\/a>/giu)]
      .find((match) => visibleText(match[0]).toLowerCase().includes("index"));
    if (!indexAnchor) throw new Error("NAMCAR-NAMCATS index workbook link not found");
    const indexUrl = safeSourceUrl(hrefFromHtml(indexAnchor[0]), pageUrl);
    seenUrls.add(indexUrl);
    documents.push({
      title: visibleText(indexAnchor[0]),
      fileName: fileNameFromUrl(indexUrl),
      sourceUrl: indexUrl,
      sourcePage: page,
      mediaType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
      observedUpdatedAt: null,
      observedSizeLabel: null,
      sourceOrderWithinPage: 0,
    });
  }

  for (const rowMatch of html.matchAll(/<tr\b[^>]*>([\s\S]*?)<\/tr>/giu)) {
    const row = rowMatch[1];
    const anchor = row.match(/<a\b[^>]*href\s*=\s*["'][^"']+\.pdf["'][^>]*>[\s\S]*?<\/a>/iu);
    if (!anchor) continue;
    const sourceUrl = safeSourceUrl(hrefFromHtml(anchor[0]), pageUrl);
    if (seenUrls.has(sourceUrl)) continue;
    const cells = [...row.matchAll(/<td\b[^>]*>([\s\S]*?)<\/td>/giu)].map((cell) =>
      visibleText(cell[1])
    );
    seenUrls.add(sourceUrl);
    documents.push({
      title: visibleText(anchor[0]),
      fileName: fileNameFromUrl(sourceUrl),
      sourceUrl,
      sourcePage: page,
      mediaType: "application/pdf",
      observedUpdatedAt: cells[1] || null,
      observedSizeLabel: cells[2] || null,
      sourceOrderWithinPage: documents.length,
    });
  }

  const pdfCount = documents.filter((document) => document.mediaType === "application/pdf").length;
  const workbookCount = documents.length - pdfCount;
  const expectedPagePdfCount = expectedPdfCounts.get(page);
  const expectedPageWorkbookCount = page === 1 ? expectedWorkbookCount : 0;
  if (pdfCount !== expectedPagePdfCount || workbookCount !== expectedPageWorkbookCount) {
    throw new Error(
      `source-set mismatch for page ${page}: expected ${expectedPagePdfCount} PDFs and ${expectedPageWorkbookCount} workbook, found ${pdfCount} PDFs and ${workbookCount} workbook`,
    );
  }
  return documents;
}

export function discoverPageOneDocuments(
  html,
  pageUrl = canonicalPageUrls.get(1),
) {
  return discoverPageDocuments(html, 1, pageUrl);
}

async function fileExists(path) {
  try {
    await access(path);
    return true;
  } catch {
    return false;
  }
}

async function sha256File(path) {
  const digest = createHash("sha256");
  for await (const chunk of createReadStream(path)) digest.update(chunk);
  return `sha256:${digest.digest("hex")}`;
}

async function inspectFile(path, mediaType) {
  const fileStat = await stat(path);
  const handle = await open(path, "r");
  const signature = Buffer.alloc(5);
  try {
    await handle.read(signature, 0, signature.length, 0);
  } finally {
    await handle.close();
  }
  const prefix = signature.toString("binary");
  const validSignature =
    mediaType === "application/pdf"
      ? prefix.startsWith("%PDF-")
      : prefix.startsWith("PK");
  if (!validSignature) throw new Error(`unexpected file signature for ${basename(path)}`);
  return {
    byteSize: fileStat.size,
    sha256: await sha256File(path),
  };
}

async function downloadDocument(document, targetPath) {
  const response = await fetch(document.sourceUrl, {
    headers: {
      Accept:
        document.mediaType === "application/pdf"
          ? "application/pdf,application/octet-stream;q=0.8"
          : "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet,application/octet-stream;q=0.8",
      "User-Agent": "AviaSurveil360 regulatory-source-sync/1.0 (public read-only source intake)",
    },
    redirect: "follow",
  });
  if (!response.ok || !response.body) {
    throw new Error(`download failed (${response.status}) for ${document.sourceUrl}`);
  }
  const finalUrl = new URL(response.url);
  if (finalUrl.protocol !== "https:" || !allowedHosts.has(finalUrl.hostname)) {
    throw new Error(`download redirected to an unapproved origin: ${response.url}`);
  }
  if (
    !finalUrl.pathname.startsWith("/publications/") ||
    !/\.(?:pdf|xlsx)$/iu.test(finalUrl.pathname)
  ) {
    throw new Error(`download redirected to an unapproved path: ${response.url}`);
  }
  const temporaryPath = `${targetPath}.partial-${process.pid}`;
  await mkdir(dirname(targetPath), { recursive: true });
  try {
    await pipeline(response.body, createWriteStream(temporaryPath, { flags: "wx" }));
    await inspectFile(temporaryPath, document.mediaType);
    await rename(temporaryPath, targetPath);
  } catch (error) {
    await unlink(temporaryPath).catch(() => {});
    throw error;
  }
  return {
    etag: response.headers.get("etag"),
    lastModified: response.headers.get("last-modified"),
    responseContentType: response.headers.get("content-type"),
  };
}

function xmlText(value) {
  return decodeHtml(
    value
      .replace(/<br\s*\/?>/giu, "\n")
      .replace(/<[^>]+>/gu, ""),
  );
}

async function unzipEntry(archivePath, entryPath) {
  const { stdout } = await execFileAsync("unzip", ["-p", archivePath, entryPath], {
    encoding: "utf8",
    maxBuffer: 64 * 1024 * 1024,
  });
  return stdout;
}

async function extractWorkbookText(inputPath, outputPath) {
  const sharedStringsXml = await unzipEntry(inputPath, "xl/sharedStrings.xml").catch(() => "");
  const sharedStrings = [...sharedStringsXml.matchAll(/<si\b[^>]*>([\s\S]*?)<\/si>/giu)].map(
    (match) =>
      [...match[1].matchAll(/<t\b[^>]*>([\s\S]*?)<\/t>/giu)]
        .map((part) => xmlText(part[1]))
        .join(""),
  );
  const workbookXml = await unzipEntry(inputPath, "xl/workbook.xml");
  const workbookRelsXml = await unzipEntry(inputPath, "xl/_rels/workbook.xml.rels");
  const relationships = new Map(
    [...workbookRelsXml.matchAll(/<Relationship\b([^>]+)>?/giu)].map((match) => {
      const attributes = match[1];
      const id = attributes.match(/\bId="([^"]+)"/u)?.[1];
      const target = attributes.match(/\bTarget="([^"]+)"/u)?.[1];
      return [id, target];
    }),
  );
  const sheets = [...workbookXml.matchAll(/<sheet\b([^>]+)>/giu)].map((match) => {
    const attributes = match[1];
    return {
      name: decodeHtml(attributes.match(/\bname="([^"]+)"/u)?.[1] ?? "Sheet"),
      relationId: attributes.match(/\br:id="([^"]+)"/u)?.[1] ?? "",
    };
  });
  const sections = [];
  let rowCount = 0;
  for (const sheet of sheets) {
    const target = relationships.get(sheet.relationId);
    if (!target) continue;
    const normalizedTarget = target.replace(/^\/?xl\//u, "");
    const worksheetXml = await unzipEntry(inputPath, `xl/${normalizedTarget}`);
    const rows = [];
    for (const rowMatch of worksheetXml.matchAll(/<row\b[^>]*>([\s\S]*?)<\/row>/giu)) {
      const cells = [];
      for (const cellMatch of rowMatch[1].matchAll(/<c\b([^>]*)>([\s\S]*?)<\/c>/giu)) {
        const attributes = cellMatch[1];
        const body = cellMatch[2];
        const type = attributes.match(/\bt="([^"]+)"/u)?.[1] ?? "";
        const rawValue = body.match(/<v\b[^>]*>([\s\S]*?)<\/v>/iu)?.[1] ?? "";
        const inlineValue = body.match(/<is\b[^>]*>([\s\S]*?)<\/is>/iu)?.[1] ?? "";
        const value =
          type === "s"
            ? sharedStrings[Number(rawValue)] ?? ""
            : type === "inlineStr"
              ? xmlText(inlineValue)
              : xmlText(rawValue);
        cells.push(value.replaceAll("\t", " ").replace(/\r?\n/gu, " "));
      }
      rows.push(cells.join("\t"));
      rowCount += 1;
    }
    sections.push(`[SHEET: ${sheet.name}]\n${rows.join("\n")}`);
  }
  const text = `${sections.join("\n\n")}\n`;
  await mkdir(dirname(outputPath), { recursive: true });
  await writeFile(outputPath, text, "utf8");
  return {
    extractionStrategyVersion: 2,
    sheetCount: sheets.length,
    rowCount,
    characterCount: text.length,
  };
}

function pdfPythonCandidates() {
  return [
    process.env.AVIA_PDF_PYTHON,
    "python3",
    join(
      homedir(),
      ".cache/codex-runtimes/codex-primary-runtime/dependencies/python/bin/python3",
    ),
  ].filter(Boolean);
}

async function extractPdfText(inputPath, outputPath) {
  const extractorPath = join(repositoryRoot, "scripts/regulatory/extract-pdf-text.py");
  let lastError = null;
  for (const python of pdfPythonCandidates()) {
    try {
      const { stdout } = await execFileAsync(python, [extractorPath, inputPath, outputPath], {
        encoding: "utf8",
        maxBuffer: 4 * 1024 * 1024,
      });
      return JSON.parse(stdout);
    } catch (error) {
      lastError = error;
    }
  }
  throw new Error(`PDF text extraction unavailable: ${lastError?.message ?? "unknown error"}`);
}

async function ensureOcrBinary(vaultRoot) {
  if (process.platform !== "darwin") {
    throw new Error("Apple Vision OCR requires macOS");
  }
  const sourcePath = join(repositoryRoot, "scripts/regulatory/ocr-pdf-text.swift");
  const toolsRoot = join(vaultRoot, "tools");
  const binaryPath = join(toolsRoot, "ocr-pdf-text");
  const sourceStat = await stat(sourcePath);
  const binaryStat = await stat(binaryPath).catch(() => null);
  if (binaryStat && binaryStat.mtimeMs >= sourceStat.mtimeMs) return binaryPath;

  await mkdir(toolsRoot, { recursive: true });
  const moduleCache = join(toolsRoot, "swift-module-cache");
  await mkdir(moduleCache, { recursive: true });
  console.log("[OCR] compiling local Apple Vision helper");
  await execFileAsync(
    "swiftc",
    ["-O", sourcePath, "-o", binaryPath],
    {
      env: {
        ...process.env,
        CLANG_MODULE_CACHE_PATH: moduleCache,
        SWIFT_MODULECACHE_PATH: moduleCache,
      },
      maxBuffer: 16 * 1024 * 1024,
    },
  );
  return binaryPath;
}

async function runOcr(
  binaryPath,
  inputPath,
  outputPath,
  checkpointPath,
  requestedPages,
) {
  await mkdir(checkpointPath, { recursive: true });
  return new Promise((resolvePromise, rejectPromise) => {
    const child = spawn(
      binaryPath,
      [inputPath, outputPath, checkpointPath, requestedPages.join(",")],
      { stdio: ["ignore", "pipe", "pipe"] },
    );
    let stdout = "";
    let stderr = "";
    child.stdout.setEncoding("utf8");
    child.stderr.setEncoding("utf8");
    child.stdout.on("data", (chunk) => {
      stdout += chunk;
    });
    child.stderr.on("data", (chunk) => {
      stderr += chunk;
      process.stderr.write(chunk);
    });
    child.on("error", rejectPromise);
    child.on("close", (code) => {
      if (code !== 0) {
        rejectPromise(
          new Error(`OCR failed with exit ${code}: ${stderr.trim() || "no diagnostic"}`),
        );
        return;
      }
      try {
        resolvePromise(JSON.parse(stdout));
      } catch (error) {
        rejectPromise(new Error(`OCR returned invalid JSON: ${error.message}`));
      }
    });
  });
}

function ocrCheckpointKey(document) {
  return createHash("sha256").update(document.sourceUrl).digest("hex").slice(0, 20);
}

function isCompleteExtractionStatus(status) {
  return status === "EXTRACTED" ||
    status === "HYBRID_EXTRACTED" ||
    status === "OCR_EXTRACTED" ||
    status === "OCR_NO_TEXT_DETECTED";
}

function readPageSections(text) {
  const sections = new Map();
  const pattern =
    /(?:^|\n\n)--- PAGE (\d+) \[(SEARCHABLE|OCR)\] ---\n\n([\s\S]*?)(?=\n\n--- PAGE \d+ \[(?:SEARCHABLE|OCR)\] ---\n\n|$)/gu;
  for (const match of text.matchAll(pattern)) {
    sections.set(Number(match[1]), {
      method: match[2],
      text: match[3],
    });
  }
  return sections;
}

export function combinePdfPageText(searchableText, ocrText, pageCount) {
  const searchableSections = readPageSections(searchableText);
  const ocrSections = readPageSections(ocrText);
  let characterCount = 0;
  let pagesWithContextText = 0;
  let text = "";
  for (let pageNumber = 1; pageNumber <= pageCount; pageNumber += 1) {
    const searchable = searchableSections.get(pageNumber);
    const ocr = ocrSections.get(pageNumber);
    const selected =
      searchable?.text.trim() ? searchable :
        ocr?.text.trim() ? ocr :
          searchable ?? ocr ?? { method: "SEARCHABLE", text: "" };
    if (selected.text.trim()) pagesWithContextText += 1;
    characterCount += selected.text.length;
    text += `\n\n--- PAGE ${pageNumber} [${selected.method}] ---\n\n${selected.text}`;
  }
  return { text, characterCount, pagesWithContextText };
}

async function mergePdfText(searchablePath, ocrPath, outputPath, pageCount) {
  const combined = combinePdfPageText(
    await readFile(searchablePath, "utf8"),
    await readFile(ocrPath, "utf8"),
    pageCount,
  );
  await writeFile(outputPath, combined.text, "utf8");
  return combined;
}

async function extractDocument(document, inputPath, outputPath, vaultRoot) {
  if (document.mediaType === "application/pdf") {
    const searchablePath = `${outputPath}.searchable-${process.pid}`;
    const ocrPath = `${outputPath}.ocr-${process.pid}`;
    const extraction = await extractPdfText(inputPath, searchablePath);
    const missingPages = extraction.pagesWithoutExtractedText ?? [];
    if (missingPages.length === 0) {
      await rename(searchablePath, outputPath);
      return {
        status: "EXTRACTED",
        details: {
          ...extraction,
          pagesWithContextText: extraction.pagesWithExtractedText,
        },
      };
    }

    try {
      const ocrBinary = await ensureOcrBinary(vaultRoot);
      const checkpointPath = join(
        vaultRoot,
        "ocr-checkpoints",
        ocrCheckpointKey(document),
      );
      console.log(
        `[OCR] ${missingPages.length}/${extraction.pageCount} pages lack searchable text; processing ${document.fileName}`,
      );
      const ocr = await runOcr(
        ocrBinary,
        inputPath,
        ocrPath,
        checkpointPath,
        missingPages,
      );
      const combined = await mergePdfText(
        searchablePath,
        ocrPath,
        outputPath,
        extraction.pageCount,
      );
      const noSearchableText = extraction.pagesWithExtractedText === 0;
      return {
        status: noSearchableText
          ? ocr.characterCount > 0 ? "OCR_EXTRACTED" : "OCR_NO_TEXT_DETECTED"
          : "HYBRID_EXTRACTED",
        details: {
          extractionStrategyVersion: 2,
          engine: noSearchableText
            ? "apple-vision-ocr"
            : `${extraction.engine}+apple-vision-ocr`,
          pageCount: extraction.pageCount,
          pagesWithExtractedText: extraction.pagesWithExtractedText,
          pagesWithoutExtractedText: missingPages,
          pagesWithOCRText: ocr.pagesWithOCRText,
          pagesWithContextText: combined.pagesWithContextText,
          characterCount: combined.characterCount,
          searchableCharacterCount: extraction.characterCount,
          ocrCharacterCount: ocr.characterCount,
          ocrRequestedPageCount: ocr.requestedPageCount,
          checkpointRelativePath: relative(repositoryRoot, checkpointPath),
        },
      };
    } finally {
      await unlink(searchablePath).catch(() => {});
      await unlink(ocrPath).catch(() => {});
    }
  }
  return {
    status: "EXTRACTED",
    details: await extractWorkbookText(inputPath, outputPath),
  };
}

function manifestDocumentId(document) {
  const kind = document.mediaType === "application/pdf" ? "PDF" : "INDEX";
  return `NCAA-NAMCATS-P${document.sourcePage}-${kind}-${String(document.sourceOrderWithinPage + 1).padStart(2, "0")}`;
}

async function loadManifest(path) {
  try {
    return JSON.parse(await readFile(path, "utf8"));
  } catch (error) {
    if (error?.code === "ENOENT") return null;
    throw error;
  }
}

async function writeManifest(path, manifest) {
  await mkdir(dirname(path), { recursive: true });
  await writeFile(path, `${JSON.stringify(manifest, null, 2)}\n`, "utf8");
}

function parseArguments(argv) {
  const options = {
    verifyOnly: false,
    vaultRoot: defaultVaultRoot,
    manifestPath: defaultManifestPath,
  };
  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index];
    if (argument === "--verify-only") options.verifyOnly = true;
    else if (argument === "--vault") options.vaultRoot = resolve(argv[++index]);
    else if (argument === "--manifest") options.manifestPath = resolve(argv[++index]);
    else if (argument === "--page" || argument === "--pages") {
      const value = argv[++index];
      if (!["all", "1,2,3"].includes(value)) {
        throw new Error("the source collection is intentionally bounded to NAMCATS pages 1,2,3");
      }
    }
    else throw new Error(`unknown argument: ${argument}`);
  }
  return options;
}

async function verifyLocalManifest(manifest, vaultRoot) {
  if (!manifest || manifest.schemaVersion !== 2 || !Array.isArray(manifest.documents)) {
    throw new Error("valid manifest is required for --verify-only");
  }
  if (
    manifest.libraryId !== "NCAA-NAMCATS-ALL-PAGES" ||
    manifest.scope?.selfAssessmentPortalAccessed !== false ||
    manifest.scope?.authenticatedFormsSubmitted !== false
  ) {
    throw new Error("manifest collection identity or access boundary is invalid");
  }
  if (manifest.documents.length !== expectedPdfCount + expectedWorkbookCount) {
    throw new Error(`manifest document count is ${manifest.documents.length}, expected 58`);
  }
  const ids = new Set(manifest.documents.map((document) => document.id));
  const urls = new Set(manifest.documents.map((document) => document.sourceUrl));
  if (ids.size !== manifest.documents.length || urls.size !== manifest.documents.length) {
    throw new Error("manifest contains duplicate document identity");
  }
  for (const [page, count] of expectedPdfCounts) {
    const listedPdfCount = manifest.documents.filter(
      (document) =>
        document.mediaType === "application/pdf" &&
        document.listedOnPages?.includes(page),
    ).length;
    if (listedPdfCount !== count) {
      throw new Error(
        `manifest page ${page} lists ${listedPdfCount} PDFs, expected ${count}`,
      );
    }
  }
  for (const document of manifest.documents) {
    const pageUrl = canonicalPageUrls.get(document.sourcePage);
    if (!pageUrl) throw new Error(`manifest page is invalid for ${document.id}`);
    const approvedUrl = safeSourceUrl(document.sourceUrl, pageUrl);
    if (
      approvedUrl !== document.sourceUrl ||
      fileNameFromUrl(approvedUrl) !== document.fileName
    ) {
      throw new Error(`manifest source identity is invalid for ${document.id}`);
    }
    const filePath = join(vaultRoot, "files", document.fileName);
    const inspection = await inspectFile(filePath, document.mediaType);
    if (inspection.sha256 !== document.sha256 || inspection.byteSize !== document.byteSize) {
      throw new Error(`manifest mismatch for ${document.fileName}`);
    }
    const textPath = join(vaultRoot, "text", `${document.fileName}.txt`);
    if (
      !(await fileExists(textPath)) ||
      document.extraction?.extractionStrategyVersion !== 2 ||
      !isCompleteExtractionStatus(document.extractionStatus)
    ) {
      throw new Error(`extracted context is missing for ${document.fileName}`);
    }
  }
  return {
    documentCount: manifest.documents.length,
    totalBytes: manifest.documents.reduce((sum, document) => sum + document.byteSize, 0),
  };
}

export async function synchronize(options) {
  const previousManifest = await loadManifest(options.manifestPath);
  if (options.verifyOnly) {
    return verifyLocalManifest(previousManifest, options.vaultRoot);
  }

  const discovered = [];
  const seenUrls = new Set();
  for (const page of sourcePages) {
    const pageUrl = canonicalPageUrls.get(page);
    const pageResponse = await fetch(pageUrl, {
      headers: {
        Accept: "text/html",
        "User-Agent": "AviaSurveil360 regulatory-source-sync/1.0 (public read-only source intake)",
      },
    });
    if (!pageResponse.ok) {
      throw new Error(`source page ${page} request failed (${pageResponse.status})`);
    }
    const pageDocuments = discoverPageDocuments(
      await pageResponse.text(),
      page,
      pageUrl,
    );
    for (const document of pageDocuments) {
      if (seenUrls.has(document.sourceUrl)) {
        const existing = discovered.find(
          (candidate) => candidate.sourceUrl === document.sourceUrl,
        );
        if (!existing) throw new Error(`duplicate source lookup failed: ${document.sourceUrl}`);
        existing.listedOnPages.push(page);
        continue;
      }
      seenUrls.add(document.sourceUrl);
      discovered.push({
        ...document,
        listedOnPages: [page],
        sourceOrder: discovered.length,
      });
    }
  }
  const pdfCount = discovered.filter(
    (document) => document.mediaType === "application/pdf",
  ).length;
  if (
    discovered.length !== expectedPdfCount + expectedWorkbookCount ||
    pdfCount !== expectedPdfCount
  ) {
    throw new Error(
      `aggregate source-set mismatch: expected ${expectedPdfCount} PDFs and ${expectedWorkbookCount} workbook`,
    );
  }

  const previousByUrl = new Map(
    (previousManifest?.documents ?? []).map((document) => [document.sourceUrl, document]),
  );
  const manifest = {
    schemaVersion: 2,
    libraryId: "NCAA-NAMCATS-ALL-PAGES",
    sourcePageUrls: Object.fromEntries(canonicalPageUrls),
    sourcePages,
    synchronizedAt: new Date().toISOString(),
    scope: {
      expectedPdfCount,
      expectedWorkbookCount,
      expectedPdfCountByPage: Object.fromEntries(expectedPdfCounts),
      duplicateListingsCollapsed: 1,
      ocrRequiredForPagesWithoutSearchableText: true,
      selfAssessmentPortalAccessed: false,
      authenticatedFormsSubmitted: false,
    },
    reviewPolicy: {
      eventDrivenReview: true,
      reconciliationIntervalMonths: 6,
      expertValidationIntervalMonths: 12,
      updateMode: "PROPOSE_DRAFT_ONLY",
    },
    documents: [],
  };

  for (let index = 0; index < discovered.length; index += 1) {
    const document = discovered[index];
    const filePath = join(options.vaultRoot, "files", document.fileName);
    const textPath = join(options.vaultRoot, "text", `${document.fileName}.txt`);
    const previous = previousByUrl.get(document.sourceUrl);
    let transferStatus = "DOWNLOADED";
    let responseMetadata = {
      etag: null,
      lastModified: null,
      responseContentType: null,
    };

    if (
      previous &&
      previous.observedUpdatedAt === document.observedUpdatedAt &&
      previous.observedSizeLabel === document.observedSizeLabel &&
      (await fileExists(filePath))
    ) {
      const existing = await inspectFile(filePath, document.mediaType);
      if (existing.sha256 === previous.sha256 && existing.byteSize === previous.byteSize) {
        transferStatus = "REUSED_PAGE_METADATA_UNCHANGED";
        responseMetadata = {
          etag: previous.etag ?? null,
          lastModified: previous.lastModified ?? null,
          responseContentType: previous.responseContentType ?? null,
        };
      }
    } else if (!previous && (await fileExists(filePath))) {
      await inspectFile(filePath, document.mediaType);
      transferStatus = "REUSED_EXISTING_LOCAL_VALID";
    }

    if (transferStatus === "DOWNLOADED") {
      console.log(`[${index + 1}/${discovered.length}] downloading ${document.fileName}`);
      responseMetadata = await downloadDocument(document, filePath);
    } else {
      console.log(`[${index + 1}/${discovered.length}] reusing ${document.fileName}`);
    }

    const inspection = await inspectFile(filePath, document.mediaType);
    let extractionStatus = "EXTRACTED";
    let extraction = null;
    if (
      previous?.sha256 === inspection.sha256 &&
      previous.extraction?.extractionStrategyVersion === 2 &&
      isCompleteExtractionStatus(previous.extractionStatus) &&
      (await fileExists(textPath))
    ) {
      extraction = previous.extraction;
      extractionStatus = previous.extractionStatus;
    } else {
      try {
        const extracted = await extractDocument(
          document,
          filePath,
          textPath,
          options.vaultRoot,
        );
        extractionStatus = extracted.status;
        extraction = extracted.details;
      } catch (error) {
        extractionStatus = "EXTRACTION_FAILED";
        extraction = { error: error instanceof Error ? error.message : String(error) };
      }
    }

    manifest.documents.push({
      id: manifestDocumentId(document),
      ...document,
      localRelativePath: relative(repositoryRoot, filePath),
      extractedTextRelativePath:
        isCompleteExtractionStatus(extractionStatus) ? relative(repositoryRoot, textPath) : null,
      byteSize: inspection.byteSize,
      sha256: inspection.sha256,
      transferStatus,
      extractionStatus,
      extraction,
      ...responseMetadata,
    });
    await writeManifest(options.manifestPath, manifest);
  }

  const failures = manifest.documents.filter(
    (document) => !isCompleteExtractionStatus(document.extractionStatus),
  );
  if (failures.length > 0) {
    throw new Error(
      `source synchronization downloaded all bytes but ${failures.length} extraction(s) failed`,
    );
  }
  return {
    documentCount: manifest.documents.length,
    totalBytes: manifest.documents.reduce((sum, document) => sum + document.byteSize, 0),
  };
}

async function main() {
  const options = parseArguments(process.argv.slice(2));
  const result = await synchronize(options);
  console.log(
    `ncaa-namcats-sync: ok (${result.documentCount} documents, ${result.totalBytes} bytes)`,
  );
}

if (import.meta.url === pathToFileURL(process.argv[1]).href) {
  main().catch((error) => {
    console.error(`ncaa-namcats-sync: ${error instanceof Error ? error.message : String(error)}`);
    process.exitCode = 1;
  });
}
