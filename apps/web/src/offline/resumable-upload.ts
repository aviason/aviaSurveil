export interface UploadPart {
  partNumber: number;
  offset: number;
  byteSize: number;
  bytes: Uint8Array;
  sha256: string;
}

export interface UploadPartReceipt {
  partNumber: number;
  byteSize: number;
  sha256: string;
  acknowledgedOffset: number;
  objectVersion: string;
}

function sha256Hex(bytes: Uint8Array): Promise<string> {
  const owned = new Uint8Array(bytes.byteLength);
  owned.set(bytes);
  return crypto.subtle.digest("SHA-256", owned.buffer).then((digest) =>
    `sha256:${Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, "0")).join("")}`,
  );
}

export async function buildUploadParts(bytes: Uint8Array, partSize: number): Promise<UploadPart[]> {
  if (!Number.isSafeInteger(partSize) || partSize <= 0) {
    throw new Error("Upload part size must be a positive integer.");
  }
  const parts: UploadPart[] = [];
  for (let offset = 0, partNumber = 1; offset < bytes.byteLength; offset += partSize, partNumber += 1) {
    const partBytes = bytes.slice(offset, Math.min(bytes.byteLength, offset + partSize));
    parts.push({
      partNumber,
      offset,
      byteSize: partBytes.byteLength,
      bytes: partBytes,
      sha256: await sha256Hex(partBytes),
    });
  }
  return parts;
}

export function partOperationId(
  uploadSessionId: string,
  sessionEpoch: number,
  partNumber: number,
  partSha256: string,
): string {
  return `upload-part:${uploadSessionId}:epoch-${sessionEpoch}:part-${partNumber}:${partSha256}`;
}

export function missingUploadParts(
  parts: UploadPart[],
  receipts: UploadPartReceipt[],
): UploadPart[] {
  const acknowledged = new Map(receipts.map((receipt) => [receipt.partNumber, receipt]));
  return parts.filter((part) => {
    const receipt = acknowledged.get(part.partNumber);
    return !receipt || receipt.byteSize !== part.byteSize || receipt.sha256 !== part.sha256 || !receipt.objectVersion;
  });
}
