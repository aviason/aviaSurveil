import { sha256Canonical } from "./schema-migrations";
import {
  getBrowserOfflineFieldDatabase,
  type FoundationRow,
} from "./db";

export interface ProfileAuthorityRecord {
  subjectId: string;
  profileKeyId: string;
  privateKey: CryptoKey;
  publicKey: CryptoKey;
  publicJwk: JsonWebKey;
}

export interface ProfileAuthorityStorage {
  read(): Promise<ProfileAuthorityRecord | null>;
  write(value: ProfileAuthorityRecord): Promise<void>;
  hasBoundData?(): Promise<boolean>;
}

export interface ProfileAuthority {
  subjectId: string;
  profileKeyId: string;
  publicJwk: JsonWebKey;
  sign(payload: Uint8Array): Promise<string>;
  verify(payload: Uint8Array, signature: string): Promise<boolean>;
}

export class ProfileAuthorityError extends Error {
  constructor(readonly code: string, message: string) {
    super(message);
    this.name = "ProfileAuthorityError";
  }
}

function toBase64Url(value: ArrayBuffer): string {
  const bytes = new Uint8Array(value);
  let binary = "";
  for (const byte of bytes) binary += String.fromCharCode(byte);
  return btoa(binary).replaceAll("+", "-").replaceAll("/", "_").replace(/=+$/u, "");
}

function fromBase64Url(value: string): Uint8Array {
  const normalized = value.replaceAll("-", "+").replaceAll("_", "/") + "=".repeat((4 - value.length % 4) % 4);
  const binary = atob(normalized);
  return Uint8Array.from(binary, (character) => character.charCodeAt(0));
}

function ownedBuffer(value: Uint8Array): ArrayBuffer {
  const copy = new Uint8Array(value.byteLength);
  copy.set(value);
  return copy.buffer;
}

function authorityFromRecord(record: ProfileAuthorityRecord): ProfileAuthority {
  return {
    subjectId: record.subjectId,
    profileKeyId: record.profileKeyId,
    publicJwk: structuredClone(record.publicJwk),
    async sign(payload) {
      return toBase64Url(await crypto.subtle.sign(
        { name: "ECDSA", hash: "SHA-256" },
        record.privateKey,
        ownedBuffer(payload),
      ));
    },
    async verify(payload, signature) {
      return crypto.subtle.verify(
        { name: "ECDSA", hash: "SHA-256" },
        record.publicKey,
        ownedBuffer(fromBase64Url(signature)),
        ownedBuffer(payload),
      );
    },
  };
}

export async function createProfileAuthority(
  storage: ProfileAuthorityStorage,
  subjectId: string,
): Promise<ProfileAuthority> {
  if (!subjectId.trim()) throw new ProfileAuthorityError("PROFILE_SUBJECT_REQUIRED", "A subject is required.");
  const existing = await storage.read();
  if (existing) {
    if (existing.subjectId !== subjectId) {
      throw new ProfileAuthorityError(
        "PROFILE_KEY_SUBJECT_MISMATCH",
        "A persisted profile key cannot be rebound to another subject.",
      );
    }
    return authorityFromRecord(existing);
  }
  if (await storage.hasBoundData?.()) {
    throw new ProfileAuthorityError(
      "PROFILE_KEY_LOST",
      "Durable local work still references a profile key that is no longer available.",
    );
  }

  const generated = await crypto.subtle.generateKey(
    { name: "ECDSA", namedCurve: "P-256" },
    false,
    ["sign", "verify"],
  ) as CryptoKeyPair;
  const publicJwk = await crypto.subtle.exportKey("jwk", generated.publicKey);
  const profileKeyId = await sha256Canonical(publicJwk);
  const record: ProfileAuthorityRecord = {
    subjectId,
    profileKeyId,
    privateKey: generated.privateKey,
    publicKey: generated.publicKey,
    publicJwk,
  };
  await storage.write(record);
  return authorityFromRecord(record);
}

export async function createBrowserProfileAuthority(subjectId: string): Promise<ProfileAuthority> {
  const database = getBrowserOfflineFieldDatabase();
  const opened = await database.openForFieldUse();
  if (opened.mode === "read-only-recovery") {
    throw new ProfileAuthorityError("PROFILE_KEY_STORAGE_RECOVERY_REQUIRED", "Profile authority storage is read-only recovery.");
  }
  const key = `profile-authority:${subjectId}`;
  const storage: ProfileAuthorityStorage = {
    async read() {
      const row = await database.foundation.get(key) as FoundationRow<ProfileAuthorityRecord> | undefined;
      return row?.value ?? null;
    },
    async write(value) {
      await database.foundation.put({ key, value: structuredClone(value) });
    },
    async hasBoundData() {
      return (await database.offlineGrants.where("subjectId").equals(subjectId).count()) > 0;
    },
  };
  return createProfileAuthority(storage, subjectId);
}
