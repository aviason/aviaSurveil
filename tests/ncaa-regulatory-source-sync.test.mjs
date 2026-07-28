import assert from "node:assert/strict";
import test from "node:test";

import {
  combinePdfPageText,
  discoverPageDocuments,
  discoverPageOneDocuments,
} from "../scripts/regulatory/sync-ncaa-namcats.mjs";

function pageFixture(page = 1, pdfCount = 20) {
  const index = page === 1 ? `
    <a href="publications/SYSDocs/NAMCARs-NAMCATS Index 20241001.xlsx">
      NAMCARs - NAMCATS Index file
    </a>
  ` : "";
  const rows = Array.from({ length: pdfCount }, (_value, indexValue) => {
    const number = indexValue + 1;
    return `
      <tr>
        <td><a href="publications/legislations/NAMCATS/Page ${page} Test Part ${number}.pdf">Page ${page} Test Part ${number}.pdf</a></td>
        <td><span class="destination-details__list__text">2026-07-${String(number).padStart(2, "0")} 12:00:00</span></td>
        <td>${number}.00 MB</td>
        <td>PDF</td>
        <td><a href="publications/legislations/NAMCATS/Page ${page} Test Part ${number}.pdf">Download</a></td>
      </tr>
    `;
  }).join("\n");
  return `<html><body>${index}<table>${rows}</table></body></html>`;
}

test("discovers the exact bounded page-1 source set with stable metadata", () => {
  const documents = discoverPageOneDocuments(pageFixture());

  assert.equal(documents.length, 21);
  assert.equal(documents.filter((document) => document.mediaType === "application/pdf").length, 20);
  assert.deepEqual(documents[0], {
    title: "NAMCARs - NAMCATS Index file",
    fileName: "NAMCARs-NAMCATS Index 20241001.xlsx",
    sourceUrl:
      "https://www.ncaa.com.na/publications/SYSDocs/NAMCARs-NAMCATS%20Index%2020241001.xlsx",
    sourcePage: 1,
    mediaType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    observedUpdatedAt: null,
    observedSizeLabel: null,
    sourceOrderWithinPage: 0,
  });
  assert.equal(documents[1].observedUpdatedAt, "2026-07-01 12:00:00");
  assert.equal(documents[1].observedSizeLabel, "1.00 MB");
});

test("discovers exact bounded page-2 and page-3 source sets", () => {
  const pageTwo = discoverPageDocuments(pageFixture(2, 20), 2);
  const pageThree = discoverPageDocuments(pageFixture(3, 18), 3);

  assert.equal(pageTwo.length, 20);
  assert.equal(pageThree.length, 18);
  assert.equal(pageTwo[0].sourcePage, 2);
  assert.equal(pageThree[17].sourcePage, 3);
  assert.equal(pageThree[17].sourceOrderWithinPage, 17);
});

test("fails closed when a page set is incomplete or leaves the NCAA publications path", () => {
  assert.throws(
    () => discoverPageOneDocuments(pageFixture().replace(/<tr>[\s\S]*?<\/tr>/u, "")),
    /source-set mismatch/,
  );
  assert.throws(
    () =>
      discoverPageOneDocuments(
        pageFixture().replace(
          "publications/legislations/NAMCATS/Page 1 Test Part 1.pdf",
          "https://example.com/Page 1 Test Part 1.pdf",
        ),
      ),
    /unapproved source origin/,
  );
  assert.throws(
    () => discoverPageDocuments(pageFixture(3, 17), 3),
    /source-set mismatch for page 3/,
  );
});

test("combines searchable text with OCR only for pages that need it", () => {
  const combined = combinePdfPageText(
    [
      "--- PAGE 1 [SEARCHABLE] ---",
      "",
      "Embedded requirement text",
      "",
      "--- PAGE 2 [SEARCHABLE] ---",
      "",
      "",
      "",
      "--- PAGE 3 [SEARCHABLE] ---",
      "",
      "Embedded evidence text",
    ].join("\n"),
    [
      "--- PAGE 2 [OCR] ---",
      "",
      "Scanned implementation text",
    ].join("\n"),
    3,
  );

  assert.equal(combined.pagesWithContextText, 3);
  assert.match(combined.text, /PAGE 1 \[SEARCHABLE\][\s\S]*Embedded requirement text/);
  assert.match(combined.text, /PAGE 2 \[OCR\][\s\S]*Scanned implementation text/);
  assert.match(combined.text, /PAGE 3 \[SEARCHABLE\][\s\S]*Embedded evidence text/);
});
