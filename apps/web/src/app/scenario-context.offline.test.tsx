// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";
import "fake-indexeddb/auto";

import Dexie from "dexie";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { OfflineFieldDatabase } from "../offline/db";
import { IndexedDbFieldRepository } from "../offline/field-repository";
import { sha256InspectionAttachment } from "../offline/inspection-attachment-hash-worker";
import {
  InspectionAttachmentStore,
  type AttachmentFileWriter,
  type InspectionAttachmentFileSystem,
} from "../offline/opfs-inspection-attachment-store";
import { createMockBackend } from "../mock/create-mock-backend";
import { AppProviders } from "./providers";
import { ScenarioProvider, useScenario } from "./scenario-context";

const subjectId = "USR-INSPECTOR-AMINA";
const packageId = "PKG-CAB-2026-001";
const now = () => new Date("2026-06-15T09:00:00.000Z");
const databases = new Set<string>();

class TestAttachmentFileSystem implements InspectionAttachmentFileSystem {
  readonly files = new Map<string, Uint8Array>();

  async createWriter(path: string): Promise<AttachmentFileWriter> {
    this.files.set(path, new Uint8Array());
    return {
      write: async (chunk) => {
        const previous = this.files.get(path) ?? new Uint8Array();
        const next = new Uint8Array(previous.byteLength + chunk.byteLength);
        next.set(previous);
        next.set(chunk, previous.byteLength);
        this.files.set(path, next);
      },
      flush: async () => undefined,
    };
  }

  async read(path: string): Promise<Uint8Array> {
    const bytes = this.files.get(path);
    if (!bytes) throw new DOMException("missing", "NotFoundError");
    return bytes.slice();
  }

  async exists(path: string): Promise<boolean> {
    return this.files.has(path);
  }

  async promote(temporaryPath: string, finalPath: string): Promise<void> {
    const bytes = await this.read(temporaryPath);
    this.files.set(finalPath, bytes);
    this.files.delete(temporaryPath);
  }

  async list(directoryPath: string): Promise<string[]> {
    return [...this.files.keys()].filter((path) => path.startsWith(`${directoryPath}/`));
  }

  async remove(path: string): Promise<void> {
    this.files.delete(path);
  }
}

function FieldHarness({ questionId }: { questionId: string }) {
  const { actions, projection } = useScenario();
  return (
    <>
      <button type="button" onClick={() => void actions.loadPackage(packageId)}>Load local package</button>
      <button
        type="button"
        onClick={() => void actions.saveChecklistResponse("NON_COMPLIANT", "PBE record unavailable.", questionId)}
      >
        Save local response
      </button>
      <button
        type="button"
        onClick={() =>
          void actions.stageInspectionAttachment(
            new File(["candidate inspection attachment"], "pbe-serviceability.pdf", {
              type: "application/pdf",
            }),
          )
        }
      >
        Stage local attachment
      </button>
      <button type="button" onClick={() => void actions.createPotentialFinding(questionId)}>
        Create local Potential Finding
      </button>
      <button type="button" onClick={() => void actions.submitChecklist()}>
        Submit local checklist
      </button>
      <output data-testid="field-mode">{String(projection.fieldMode)}</output>
      <output data-testid="field-pending">{projection.fieldPendingOperationCount}</output>
      <output data-testid="field-answer">{projection.response?.answer ?? "none"}</output>
      <output data-testid="field-attachment">
        {projection.inspectionAttachments[0]?.stagingState ?? "none"}
      </output>
      <output data-testid="field-attachment-blocking">
        {projection.attachmentRecoveryBlocking.length}
      </output>
      <output data-testid="field-potential">{projection.potentialFinding?.status ?? "none"}</output>
      <output data-testid="field-checklist">{projection.checklistSubmission?.checklistStatus ?? "none"}</output>
    </>
  );
}

afterEach(async () => {
  cleanup();
  await Promise.all([...databases].map((name) => Dexie.delete(name)));
  databases.clear();
});

describe("ScenarioProvider field working set", () => {
  it("loads and mutates a checked-out package through FieldRepository without Backend writes", async () => {
    const backend = createMockBackend();
    const inspectionPackage = await backend.inspections.getPackage({ packageId });
    const checkout = await backend.inspections.checkout({
      operationId: "OP-TEST-CHECKOUT",
      packageId,
      expectedPackageVersion: inspectionPackage.packageVersion,
      deviceInstanceId: "DEVICE-TEST-001",
    });
    const questionId = checkout.inspectionPackage.questions.find((question) => question.assignedInspectorUserIds.includes(subjectId))?.id;
    if (!questionId) throw new Error("A test-assigned question is required.");
    const databaseName = `aviasurveil360-scenario-field-${crypto.randomUUID()}`;
    databases.add(databaseName);
    const database = new OfflineFieldDatabase({ name: databaseName });
    const repository = new IndexedDbFieldRepository({ database, subjectId, now });
    const attachmentFileSystem = new TestAttachmentFileSystem();
    const attachmentStore = new InspectionAttachmentStore({
      repository,
      fileSystem: attachmentFileSystem,
      hasher: { sha256: sha256InspectionAttachment },
      now,
    });
    await repository.checkoutPackage({
      ...checkout,
      checkedOutAt: now().toISOString(),
    });
    const backendResponseWrite = vi.spyOn(backend.inspections, "upsertChecklistResponse");
    const backendPotentialWrite = vi.spyOn(backend.potentialFindings, "create");
    const backendChecklistWrite = vi.spyOn(backend.inspections, "submitChecklist");

    render(
      <AppProviders
        runtime={{
          backend,
          buildProfile: "demo",
          environmentLabel: "field-test",
          fieldRepositoryForSubject: () => repository,
          inspectionAttachmentStoreForSubject: () => attachmentStore,
        }}
      >
        <ScenarioProvider>
          <FieldHarness questionId={questionId} />
        </ScenarioProvider>
      </AppProviders>,
    );

    await userEvent.click(screen.getByRole("button", { name: "Load local package" }));
    await waitFor(() => expect(screen.getByTestId("field-mode")).toHaveTextContent("true"));
    await userEvent.click(screen.getByRole("button", { name: "Save local response" }));
    await waitFor(() =>
      expect(screen.getByTestId("field-answer")).toHaveTextContent("NON_COMPLIANT"),
    );
    await userEvent.click(screen.getByRole("button", { name: "Create local Potential Finding" }));
    await waitFor(() =>
      expect(screen.getByTestId("field-potential")).toHaveTextContent("PENDING_LEAD_REVIEW"),
    );
    await userEvent.click(screen.getByRole("button", { name: "Stage local attachment" }));
    await waitFor(() => expect(screen.getByTestId("field-attachment")).toHaveTextContent("ready"));
    await userEvent.click(screen.getByRole("button", { name: "Submit local checklist" }));

    await waitFor(() =>
      expect(screen.getByTestId("field-checklist")).toHaveTextContent("SUBMITTED"),
    );
    expect(screen.getByTestId("field-pending")).toHaveTextContent("4");
    expect(backendResponseWrite).not.toHaveBeenCalled();
    expect(backendPotentialWrite).not.toHaveBeenCalled();
    expect(backendChecklistWrite).not.toHaveBeenCalled();
    expect(await repository.listOutbox(packageId)).toHaveLength(4);
    expect(attachmentFileSystem.files.size).toBe(1);
    expect((await repository.listAttachmentManifests())[0]).toMatchObject({
      potentialFindingLocalId: expect.stringMatching(/^PF-LOCAL-/),
    });

    attachmentFileSystem.files.clear();
    await userEvent.click(screen.getByRole("button", { name: "Load local package" }));
    await waitFor(() =>
      expect(screen.getByTestId("field-attachment-blocking")).toHaveTextContent("1"),
    );
    expect(await repository.getAttachmentManifest(
      (await repository.listAttachmentManifests())[0]!.attachmentId,
    )).toMatchObject({
      stagingState: "recovery_required",
      quarantineReason: "REFERENCED_BYTES_MISSING",
    });
  });

  it("binds field repositories and Inspection Attachment stores to the runtime subject", async () => {
    const backend = createMockBackend();
    const inspectionPackage = await backend.inspections.getPackage({ packageId });
    const checkout = await backend.inspections.checkout({
      operationId: "OP-UUID-CHECKOUT",
      packageId,
      expectedPackageVersion: inspectionPackage.packageVersion,
      deviceInstanceId: "DEVICE-UUID-001",
    });
    const uuidSubjectId = "154ec5ac-6f97-4f55-916f-d2f142fc6211";
    const uuidPackage = {
      ...checkout.inspectionPackage,
      questions: checkout.inspectionPackage.questions.map((question) => ({
        ...question,
        assignedInspectorUserIds: question.assignedInspectorUserIds.includes(subjectId)
          ? [uuidSubjectId]
          : question.assignedInspectorUserIds,
      })),
    };
    const questionId = uuidPackage.questions.find((question) => question.assignedInspectorUserIds.includes(uuidSubjectId))?.id;
    if (!questionId) throw new Error("A UUID-assigned question is required.");
    const databaseName = `aviasurveil360-scenario-field-${crypto.randomUUID()}`;
    databases.add(databaseName);
    const database = new OfflineFieldDatabase({ name: databaseName });
    const repository = new IndexedDbFieldRepository({
      database,
      subjectId: uuidSubjectId,
      now,
    });
    const attachmentFileSystem = new TestAttachmentFileSystem();
    const attachmentStore = new InspectionAttachmentStore({
      repository,
      fileSystem: attachmentFileSystem,
      hasher: { sha256: sha256InspectionAttachment },
      now,
    });
    await repository.checkoutPackage({
      inspectionPackage: uuidPackage,
      offlineGrant: {
        ...checkout.offlineGrant,
        grantId: "GRANT-UUID-SUBJECT",
        subjectId: uuidSubjectId,
        assignmentScope: {
          questionIds: ["CAB-LAV-001", "CAB-PAX-SEAT-001", "CAB-EMEQ-PBE-001"],
        },
      },
      checkedOutAt: now().toISOString(),
    });
    const requestedRepositorySubjects: string[] = [];
    const requestedStoreSubjects: string[] = [];

    render(
      <AppProviders
        runtime={{
          backend,
          buildProfile: "http",
          environmentLabel: "field-test",
          subjectId: uuidSubjectId,
          fieldRepositoryForSubject: (requestedSubjectId) => {
            requestedRepositorySubjects.push(requestedSubjectId);
            return repository;
          },
          inspectionAttachmentStoreForSubject: (requestedSubjectId) => {
            requestedStoreSubjects.push(requestedSubjectId);
            return attachmentStore;
          },
        }}
      >
        <ScenarioProvider>
          <FieldHarness questionId={questionId} />
        </ScenarioProvider>
      </AppProviders>,
    );

    await userEvent.click(screen.getByRole("button", { name: "Load local package" }));

    await waitFor(() => expect(screen.getByTestId("field-mode")).toHaveTextContent("true"));
    expect(requestedRepositorySubjects).toContain(uuidSubjectId);
    expect(requestedRepositorySubjects).not.toContain(subjectId);
    expect(requestedStoreSubjects).toContain(uuidSubjectId);
  });
});
