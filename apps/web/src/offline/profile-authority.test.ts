import { describe, expect, it } from "vitest";

import {
  createProfileAuthority,
  type ProfileAuthorityRecord,
  type ProfileAuthorityStorage,
} from "./profile-authority";

class MemoryProfileAuthorityStorage implements ProfileAuthorityStorage {
  value: ProfileAuthorityRecord | null = null;
  boundData = false;

  async read(): Promise<ProfileAuthorityRecord | null> {
    return this.value;
  }

  async write(value: ProfileAuthorityRecord): Promise<void> {
    this.value = value;
  }

  async hasBoundData(): Promise<boolean> {
    return this.boundData;
  }
}

describe("profile proof-of-possession authority", () => {
  it("persists a non-exportable P-256 key and signs/verifies stable proof", async () => {
    const storage = new MemoryProfileAuthorityStorage();
    const first = await createProfileAuthority(storage, "SUBJECT-001");
    const second = await createProfileAuthority(storage, "SUBJECT-001");
    const payload = new TextEncoder().encode("lease:001|operation:001");
    const signature = await first.sign(payload);

    expect(first.profileKeyId).toMatch(/^sha256:[0-9a-f]{64}$/);
    expect(second.profileKeyId).toBe(first.profileKeyId);
    expect(storage.value?.privateKey.extractable).toBe(false);
    expect(await first.verify(payload, signature)).toBe(true);
    expect(await first.verify(new TextEncoder().encode("tampered"), signature)).toBe(false);
  });

  it("does not rebind a stored key to another subject", async () => {
    const storage = new MemoryProfileAuthorityStorage();
    await createProfileAuthority(storage, "SUBJECT-001");
    await expect(createProfileAuthority(storage, "SUBJECT-002")).rejects.toMatchObject({
      code: "PROFILE_KEY_SUBJECT_MISMATCH",
    });
  });

  it("does not mint a replacement key when durable local work still references the lost key", async () => {
    const storage = new MemoryProfileAuthorityStorage();
    storage.boundData = true;
    await expect(createProfileAuthority(storage, "SUBJECT-001")).rejects.toMatchObject({
      code: "PROFILE_KEY_LOST",
    });
  });
});
