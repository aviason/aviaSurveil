import { describe, expect, it } from "vitest";

import {
  buildUploadParts,
  missingUploadParts,
  partOperationId,
  type UploadPartReceipt,
} from "./resumable-upload";

describe("resumable attachment upload planning", () => {
  it("keeps stable part identities and resumes only parts without server receipts", async () => {
    const bytes = new Uint8Array([0, 1, 2, 3, 4, 5, 6, 7, 8, 9]);
    const parts = await buildUploadParts(bytes, 4);

    expect(parts.map((part) => [part.partNumber, part.offset, part.byteSize])).toEqual([
      [1, 0, 4],
      [2, 4, 4],
      [3, 8, 2],
    ]);
    expect(partOperationId("UPLOAD-001", 2, parts[1]!.partNumber, parts[1]!.sha256)).toBe(
      partOperationId("UPLOAD-001", 2, parts[1]!.partNumber, parts[1]!.sha256),
    );

    const acknowledged: UploadPartReceipt[] = [{
      partNumber: 1,
      byteSize: 4,
      sha256: parts[0]!.sha256,
      acknowledgedOffset: 4,
      objectVersion: "version-1",
    }];
    expect(missingUploadParts(parts, acknowledged).map((part) => part.partNumber)).toEqual([2, 3]);
  });

  it("does not treat a same-number part with a different hash as acknowledged", async () => {
    const parts = await buildUploadParts(new Uint8Array([1, 2, 3, 4]), 4);
    expect(missingUploadParts(parts, [{
      partNumber: 1,
      byteSize: 4,
      sha256: "sha256:" + "0".repeat(64),
      acknowledgedOffset: 4,
      objectVersion: "version-wrong",
    }])).toHaveLength(1);
  });
});
