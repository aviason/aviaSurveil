import {
  createContext,
  type PropsWithChildren,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import type {
  AssignmentSummary,
  CapRevisionView,
  ChecklistAnswer,
  ChecklistResponseView,
  EvidenceVersionView,
  FindingSeverity,
  FindingView,
  InspectionPackage,
  ManagerDashboardProjection,
  PotentialFindingView,
  ReportVersionView,
  ReviewCapOutput,
  ReviewEvidenceOutput,
  Role,
  SubmitCapInput,
  SubmitCapOutput,
  SubmitChecklistOutput,
  AuthorizedConflictDescriptor,
} from "../backend/backend";
import {
  createBrowserFieldRepository,
  toChecklistResponseView,
  type FieldPackageView,
  type IndexedDbFieldRepository,
} from "../offline/field-repository";
import {
  reconcileInspectionAttachments,
  type AttachmentRecoveryBlockingItem,
  type AttachmentRecoveryReport,
} from "../offline/attachment-recovery";
import type { AttachmentManifestRow, PotentialFindingDraftRow } from "../offline/db";
import {
  createBrowserInspectionAttachmentStore,
  type InspectionAttachmentStore,
} from "../offline/opfs-inspection-attachment-store";
import {
  createBrowserSyncBroadcast,
  createBrowserSyncOwnerLock,
  ForegroundSyncEngine,
  installForegroundSyncTriggers,
  type FieldSyncTrigger,
  type SyncStatusBroadcast,
} from "../offline/sync-engine";
import { getOrCreateDeviceInstanceId } from "../offline/storage-readiness";
import { useApplicationRuntime } from "./providers";

export interface ScenarioProjection {
  assignments: AssignmentSummary[];
  packageView: InspectionPackage | null;
  response: ChecklistResponseView | null;
  potentialFinding: PotentialFindingView | null;
  finding: FindingView | null;
  auditeeFindings: FindingView[];
  auditeeOrganizationName: string | null;
  checklistSubmission: SubmitChecklistOutput | null;
  capSubmission: SubmitCapOutput | null;
  capRevisions: CapRevisionView[];
  capReview: ReviewCapOutput | null;
  evidenceVersions: EvidenceVersionView[];
  evidenceReview: ReviewEvidenceOutput | null;
  report: ReportVersionView | null;
  dashboard: ManagerDashboardProjection | null;
  fieldMode: boolean;
  fieldPendingOperationCount: number;
  inspectionAttachments: AttachmentManifestRow[];
  uploadedInspectionAttachments: Array<{ attachmentId: string; fileName: string }>;
  attachmentRecoveryBlocking: AttachmentRecoveryBlockingItem[];
  attachmentRecoveryQuarantinedCount: number;
  fieldSyncStatus: "idle" | "synchronized" | "contended" | "retryable" | "conflict" | "forbidden" | "invalid" | "resnapshot-required";
  fieldSyncErrorCode: string | null;
  fieldSyncConflict: AuthorizedConflictDescriptor | null;
}

export interface ScenarioActions {
  loadAssignments(): Promise<void>;
  loadPackage(packageId?: string): Promise<void>;
  saveChecklistResponse(answer: ChecklistAnswer, comment: string, questionId?: string): Promise<void>;
  stageInspectionAttachment(file: File): Promise<void>;
  createPotentialFinding(questionId?: string): Promise<void>;
  submitChecklist(): Promise<void>;
  decidePotentialFinding(input: {
    decision: "RETURN" | "DISMISS";
    reason: string;
  }): Promise<void>;
  convertPotentialFinding(input: {
    severity: FindingSeverity;
    capRequired: boolean;
    evidenceRequired: boolean;
    dueDate: string | null;
  }): Promise<void>;
  refreshFinding(role: Role): Promise<void>;
  loadAuditeeFindings(): Promise<void>;
  selectAuditeeFinding(findingId: string): Promise<void>;
  submitCap(input: Omit<SubmitCapInput, "operationId" | "findingId" | "expectedFindingRevision">): Promise<void>;
  reviewCap(input: {
    decision: "ACCEPT" | "REJECT" | "REQUEST_MORE_INFORMATION";
    commentToAuditee: string;
    internalCaaNote: string;
  }): Promise<void>;
  submitEvidence(file: File): Promise<void>;
  loadEvidenceVersions(role: Role): Promise<void>;
  reviewEvidence(input: {
    decision: "CLOSE" | "PARTIALLY_CLOSE" | "NOT_CLOSE" | "REQUEST_MORE_INFORMATION";
    commentToAuditee: string;
    internalCaaNote: string;
  }): Promise<void>;
  authorizedClose(reason: string): Promise<void>;
  loadReport(role: Role): Promise<void>;
  issueReport(reason: string): Promise<void>;
  loadManagerDashboard(): Promise<void>;
  syncFieldWork(trigger?: FieldSyncTrigger): Promise<void>;
}

interface ScenarioContextValue {
  projection: ScenarioProjection;
  actions: ScenarioActions;
}

const ScenarioContext = createContext<ScenarioContextValue | null>(null);

const initialProjection: ScenarioProjection = {
  assignments: [],
  packageView: null,
  response: null,
  potentialFinding: null,
  finding: null,
  auditeeFindings: [],
  auditeeOrganizationName: null,
  checklistSubmission: null,
  capSubmission: null,
  capRevisions: [],
  capReview: null,
  evidenceVersions: [],
  evidenceReview: null,
  report: null,
  dashboard: null,
  fieldMode: false,
  fieldPendingOperationCount: 0,
  inspectionAttachments: [],
  uploadedInspectionAttachments: [],
  attachmentRecoveryBlocking: [],
  attachmentRecoveryQuarantinedCount: 0,
  fieldSyncStatus: "idle",
  fieldSyncErrorCode: null,
  fieldSyncConflict: null,
};

const DEMO_FIELD_SUBJECT_ID = "USR-INSPECTOR-AMINA";
const FIELD_PACKAGE_ID = "PKG-CAB-2026-001";
const FIELD_QUESTION_ID = "CAB-EMEQ-PBE-001";

function toPotentialFindingView(row: PotentialFindingDraftRow): PotentialFindingView {
  return {
    id: row.authoritativeEntityId ?? row.id,
    auditId: row.auditId,
    questionId: row.questionId,
    organizationId: row.organizationId,
    title: row.title,
    description: row.description,
    status: row.status,
    revision: row.baseRevision ?? 0,
    convertedFindingId: null,
  };
}

function fieldProjection(view: FieldPackageView, recovery?: AttachmentRecoveryReport) {
  const response = view.responses.find(
    (candidate) => candidate.questionId === FIELD_QUESTION_ID && !candidate.tombstoned,
  );
  const potentialFinding = view.potentialFindingDrafts.find(
    (candidate) => candidate.questionId === FIELD_QUESTION_ID,
  );
  const recoveryProjection = recovery
    ? {
        attachmentRecoveryBlocking: recovery.blocking,
        attachmentRecoveryQuarantinedCount:
          recovery.quarantinedAttachmentIds.length + recovery.quarantinedUnknownPaths.length,
      }
    : {};
  return {
    packageView: view.inspectionPackage,
    response: response ? toChecklistResponseView(response) : null,
    potentialFinding: potentialFinding ? toPotentialFindingView(potentialFinding) : null,
    fieldMode: true,
    fieldPendingOperationCount: view.pendingOperationCount,
    inspectionAttachments: view.attachmentManifests,
    ...recoveryProjection,
  };
}

export function ScenarioProvider({ children }: PropsWithChildren) {
  const runtime = useApplicationRuntime();
  const [projection, setProjection] = useState<ScenarioProjection>(initialProjection);
  const syncBroadcastRef = useRef<SyncStatusBroadcast | null>(null);
  const fieldSubjectId = runtime.subjectId ?? DEMO_FIELD_SUBJECT_ID;

  const backendFor = (role: Role) => runtime.backendForRole?.(role) ?? runtime.backend;
  const operationId = (prefix: string) => `${prefix}-${crypto.randomUUID()}`;
  const fieldOperationId = (prefix: string) => `${prefix}-${crypto.randomUUID()}`;
  const fieldRepository = (): IndexedDbFieldRepository =>
    runtime.fieldRepositoryForSubject?.(fieldSubjectId) ??
    createBrowserFieldRepository(
      fieldSubjectId,
      runtime.now ??
        (runtime.buildProfile === "demo"
        ? () => new Date("2026-06-15T09:00:00.000Z")
        : () => new Date()),
    );
  const inspectionAttachmentStore = (
    repository: IndexedDbFieldRepository,
  ): InspectionAttachmentStore =>
    runtime.inspectionAttachmentStoreForSubject?.(fieldSubjectId) ??
    createBrowserInspectionAttachmentStore(repository);

  const actions = useMemo<ScenarioActions>(
    () => ({
      async loadAssignments() {
        const result = await backendFor("inspector").assignments.list({});
        setProjection((current) => ({ ...current, assignments: result.items }));
      },

      async loadPackage(requestedPackageID = FIELD_PACKAGE_ID) {
        const isFieldPackage = requestedPackageID === FIELD_PACKAGE_ID;
        const repository = fieldRepository();
        const local = isFieldPackage ? await repository.loadPackage(FIELD_PACKAGE_ID) : null;
        if (local) {
          const attachmentStore = inspectionAttachmentStore(repository);
          const recovery = await reconcileInspectionAttachments({
            repository,
            fileSystem: attachmentStore.fileSystem,
            hasher: attachmentStore.hasher,
          });
          const recoveredLocal = await repository.loadPackage(FIELD_PACKAGE_ID);
          if (!recoveredLocal) throw new Error("Checked-out field package disappeared during recovery.");
          setProjection((current) => ({ ...current, ...fieldProjection(recoveredLocal, recovery) }));
          return;
        }
        const packageView = await backendFor("inspector").inspections.getPackage({
          packageId: requestedPackageID,
        });
        const response =
          packageView.questions.find((question) => question.currentResponse)?.currentResponse ?? null;
        setProjection((current) => ({
          ...current,
          packageView,
          response,
          fieldMode: false,
          fieldPendingOperationCount: 0,
          inspectionAttachments: [],
          uploadedInspectionAttachments: [],
          attachmentRecoveryBlocking: [],
          attachmentRecoveryQuarantinedCount: 0,
        }));
      },

      async saveChecklistResponse(answer, comment, requestedQuestionID = FIELD_QUESTION_ID) {
        if (projection.fieldMode) {
          if (projection.attachmentRecoveryBlocking.length > 0) {
            throw new Error("Resolve blocking Inspection Attachment recovery before editing.");
          }
          const repository = fieldRepository();
          await repository.saveChecklistResponse({
            operationId: fieldOperationId("OP-RESPONSE"),
            packageId: FIELD_PACKAGE_ID,
            responseId: "RESP-CAB-EMEQ-PBE-001",
            questionId: FIELD_QUESTION_ID,
            answer,
            comment,
          });
          const local = await repository.loadPackage(FIELD_PACKAGE_ID);
          if (!local) throw new Error("Checked-out field package disappeared after local commit.");
          setProjection((current) => ({ ...current, ...fieldProjection(local) }));
          return;
        }
        const existingResponse =
          projection.packageView?.questions.find((question) => question.id === requestedQuestionID)?.currentResponse ??
          (projection.response?.questionId === requestedQuestionID ? projection.response : null);
        const response = await backendFor("inspector").inspections.upsertChecklistResponse({
          operationId: operationId("OP-RESPONSE"),
          responseId: `RESP-${requestedQuestionID}`,
          auditId: projection.packageView?.auditId ?? "AUD-2026-001",
          questionId: requestedQuestionID,
          expectedResponseRevision: existingResponse?.revision ?? null,
          answer,
          comment,
        });
        setProjection((current) => ({ ...current, response }));
      },

      async stageInspectionAttachment(file) {
        if (!projection.response) throw new Error("Save the exact checklist response before staging an attachment.");
        if (projection.attachmentRecoveryBlocking.length > 0) {
          throw new Error("Resolve blocking Inspection Attachment recovery before staging bytes.");
        }
        const packageView = projection.packageView;
        if (!packageView) throw new Error("Checked-out field package is unavailable.");
        if (!projection.fieldMode) {
          const backend = backendFor("inspector");
          const digest = await crypto.subtle.digest("SHA-256", await file.arrayBuffer());
          const sha256 = `sha256:${Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, "0")).join("")}`;
          const requestedInspectionAttachmentId = `ATT-${crypto.randomUUID()}`;
          let inspectionAttachmentId = requestedInspectionAttachmentId;
          if (backend.mode === "http") {
            const deviceInstanceId = await getOrCreateDeviceInstanceId();
            const checkout = await backend.inspections.checkout({
              operationId: operationId("OP-ATTACHMENT-CHECKOUT"),
              packageId: packageView.id,
              expectedPackageVersion: packageView.packageVersion,
              deviceInstanceId,
            });
            const registrationOperationId = operationId("OP-ATTACHMENT-REGISTER");
            const registration = await backend.sync.pushOperation({
              operation: {
                operationId: registrationOperationId,
                protocolVersion: checkout.inspectionPackage.protocolVersion,
                offlineGrantId: checkout.offlineGrant.grantId,
                packageId: checkout.inspectionPackage.id,
                packageVersion: checkout.inspectionPackage.packageVersion,
                entityId: requestedInspectionAttachmentId,
                commandType: "REGISTER_INSPECTION_ATTACHMENT",
                baseRevision: null,
                deviceInstanceId: checkout.offlineGrant.deviceInstanceId,
                clientOccurredAt: (runtime.now ?? (() => new Date()))().toISOString(),
                payload: {
                  auditId: checkout.inspectionPackage.auditId,
                  checklistResponseId: projection.response.id,
                  potentialFindingOperationId: null,
                  fileName: file.name,
                  mediaType: file.type || "application/octet-stream",
                  byteSize: file.size,
                  sha256,
                },
              },
            });
            if ((registration.status !== "accepted" && registration.status !== "already_applied") || !registration.authoritativeEntityId) {
              throw new Error(`Inspection Attachment registration was not accepted (${registration.errorCode ?? registration.status}).`);
            }
            inspectionAttachmentId = registration.authoritativeEntityId;
          }
          const upload = await backend.inspectionAttachments.beginUpload({
            operationId: operationId("OP-ATTACHMENT-BEGIN"), inspectionAttachmentId,
            packageId: packageView.id, byteSize: file.size, sha256, fileName: file.name,
            declaredMediaType: file.type || "application/octet-stream",
          });
          if (backend.mode === "http") {
            const uploadResponse = await fetch(upload.uploadUrl, { method: "PUT", headers: upload.requiredHeaders, body: file });
            if (!uploadResponse.ok) throw new Error(`Inspection Attachment object upload failed with status ${uploadResponse.status}.`);
          }
          const completed = await backend.inspectionAttachments.completeUpload({
            operationId: operationId("OP-ATTACHMENT-COMPLETE"), uploadId: upload.uploadId, sha256, byteSize: file.size,
          });
          if (completed.inspectionAttachmentId !== inspectionAttachmentId) throw new Error("Inspection Attachment completion scope changed.");
          setProjection((current) => ({ ...current, uploadedInspectionAttachments: [
            ...current.uploadedInspectionAttachments, { attachmentId: inspectionAttachmentId, fileName: file.name },
          ] }));
          return;
        }
        const repository = fieldRepository();
        await inspectionAttachmentStore(repository).stage({
          attachmentId: `ATT-LOCAL-${crypto.randomUUID()}`,
          operationId: fieldOperationId("OP-ATTACHMENT"),
          packageId: packageView.id,
          checklistResponseId: projection.response.id,
          potentialFindingLocalId: projection.potentialFinding?.id ?? null,
          fileName: file.name,
          mediaType: file.type,
          bytes: new Uint8Array(await file.arrayBuffer()),
        });
        const local = await repository.loadPackage(packageView.id);
        if (!local) throw new Error("Checked-out field package disappeared after attachment staging.");
        setProjection((current) => ({ ...current, ...fieldProjection(local) }));
      },

      async createPotentialFinding(requestedQuestionID = FIELD_QUESTION_ID) {
        const selectedResponse =
          projection.packageView?.questions.find((question) => question.id === requestedQuestionID)?.currentResponse ??
          (projection.response?.questionId === requestedQuestionID ? projection.response : null);
        if (!selectedResponse) throw new Error("Save the exact checklist response first.");
        if (projection.fieldMode) {
          if (projection.attachmentRecoveryBlocking.length > 0) {
            throw new Error("Resolve blocking Inspection Attachment recovery before creating a Potential Finding.");
          }
          const repository = fieldRepository();
          await repository.createPotentialFindingDraft({
            operationId: fieldOperationId("OP-PF"),
            packageId: FIELD_PACKAGE_ID,
            localId: `PF-LOCAL-${crypto.randomUUID()}`,
            questionId: FIELD_QUESTION_ID,
            checklistResponseId: selectedResponse.id,
            title: "PBE serviceability and accessibility not confirmed",
            description:
              "The configured cabin check could not confirm that the PBE was serviceable and accessible.",
            requiredComment: selectedResponse.comment,
            inspectionAttachmentIds: projection.inspectionAttachments
              .filter((attachment) => attachment.stagingState === "ready")
              .map((attachment) => attachment.attachmentId),
          });
          const local = await repository.loadPackage(FIELD_PACKAGE_ID);
          if (!local) throw new Error("Checked-out field package disappeared after local commit.");
          setProjection((current) => ({ ...current, ...fieldProjection(local) }));
          return;
        }
        const potentialFinding = await backendFor("inspector").potentialFindings.create({
          operationId: operationId("OP-PF"),
          auditId: projection.packageView?.auditId ?? "AUD-2026-001",
          questionId: requestedQuestionID,
          checklistResponseId: selectedResponse.id,
          expectedChecklistResponseRevision: selectedResponse.revision,
          title: "PBE serviceability and accessibility not confirmed",
          description:
            "The configured cabin check could not confirm that the PBE was serviceable and accessible.",
          requiredComment: selectedResponse.comment,
          inspectionAttachmentIds: projection.uploadedInspectionAttachments.map((attachment) => attachment.attachmentId),
        });
        setProjection((current) => ({ ...current, potentialFinding }));
      },

      async submitChecklist() {
        const packageView = projection.packageView;
        if (!packageView) throw new Error("Inspection package is unavailable.");
        if (projection.fieldMode) {
          if (projection.attachmentRecoveryBlocking.length > 0) {
            throw new Error("Resolve blocking Inspection Attachment recovery before checklist submission.");
          }
          const repository = fieldRepository();
          const submission = await repository.submitChecklist({
            operationId: fieldOperationId("OP-CHECKLIST-SUBMIT"),
            packageId: packageView.id,
          });
          const local = await repository.loadPackage(packageView.id);
          if (!local) throw new Error("Checked-out field package disappeared after local commit.");
          setProjection((current) => ({
            ...current,
            ...fieldProjection(local),
            checklistSubmission: {
              auditId: submission.auditId,
              checklistStatus: submission.checklistStatus,
              checklistRevision: submission.checklistRevision,
            },
          }));
          return;
        }
        const checklistSubmission = await backendFor("inspector").inspections.submitChecklist({
          operationId: operationId("OP-CHECKLIST-SUBMIT"),
          auditId: packageView.auditId,
          expectedChecklistRevision: packageView.checklistRevision,
        });
        setProjection((current) => ({ ...current, checklistSubmission }));
      },

      async decidePotentialFinding(input) {
        if (!projection.potentialFinding) throw new Error("Potential Finding is unavailable.");
        const result = await backendFor("leadInspector").potentialFindings.decide({
          operationId: operationId("OP-PF-DECISION"),
          potentialFindingId: projection.potentialFinding.id,
          expectedPotentialFindingRevision: projection.potentialFinding.revision,
          decision: input.decision,
          reason: input.reason,
        });
        setProjection((current) => ({
          ...current,
          potentialFinding: result.potentialFinding,
          finding: result.finding ?? current.finding,
        }));
      },

      async convertPotentialFinding(input) {
        if (!projection.potentialFinding) throw new Error("Potential Finding is unavailable.");
        const result = await backendFor("leadInspector").potentialFindings.decide({
          operationId: operationId("OP-PF-CONVERT"),
          potentialFindingId: projection.potentialFinding.id,
          expectedPotentialFindingRevision: projection.potentialFinding.revision,
          decision: "CONVERT",
          severity: input.severity,
          capRequired: input.capRequired,
          evidenceRequired: input.evidenceRequired,
          dueDate: input.dueDate,
        });
        setProjection((current) => ({
          ...current,
          potentialFinding: result.potentialFinding,
          finding: result.finding,
        }));
      },

      async refreshFinding(role) {
        const finding = await backendFor(role).findings.get({ findingId: "FND-CAB-2026-001" });
        setProjection((current) => ({ ...current, finding }));
      },

      async loadAuditeeFindings() {
        const backend = backendFor("auditee");
        const [result, organizations] = await Promise.all([
          backend.findings.list({}),
          backend.organizations.list({}),
        ]);
        const auditeeOrganizationName = organizations.items[0]?.legalName ?? null;
        const selectedId =
          projection.finding && result.items.some((candidate) => candidate.id === projection.finding?.id)
            ? projection.finding.id
            : (result.items.find((candidate) => candidate.id === "FND-CAB-2026-001") ?? result.items[0])?.id;
        if (!selectedId) {
          setProjection((current) => ({
            ...current,
            auditeeFindings: result.items,
            auditeeOrganizationName,
            finding: null,
            capRevisions: [],
            evidenceVersions: [],
          }));
          return;
        }
        const [finding, capResult, evidenceVersions] = await Promise.all([
          backend.findings.get({ findingId: selectedId }),
          backend.caps.listRevisions({ findingId: selectedId }),
          backend.evidence.listVersions({ findingId: selectedId }),
        ]);
        setProjection((current) => ({
          ...current,
          auditeeFindings: result.items,
          auditeeOrganizationName,
          finding,
          capRevisions: capResult.items,
          evidenceVersions,
        }));
      },

      async selectAuditeeFinding(findingId) {
        const backend = backendFor("auditee");
        const [finding, capResult, evidenceVersions] = await Promise.all([
          backend.findings.get({ findingId }),
          backend.caps.listRevisions({ findingId }),
          backend.evidence.listVersions({ findingId }),
        ]);
        setProjection((current) => ({
          ...current,
          finding,
          capRevisions: capResult.items,
          evidenceVersions,
        }));
      },

      async submitCap(input) {
        if (!projection.finding) throw new Error("Finding is unavailable.");
        const capSubmission = await backendFor("auditee").caps.submit({
          operationId: operationId("OP-CAP-SUBMIT"),
          findingId: projection.finding.id,
          expectedFindingRevision: projection.finding.revision,
          ...input,
        });
        const backend = backendFor("auditee");
        const [finding, capResult] = await Promise.all([
          backend.findings.get({ findingId: projection.finding.id }),
          backend.caps.listRevisions({ findingId: projection.finding.id }),
        ]);
        setProjection((current) => ({
          ...current,
          capSubmission,
          finding,
          capRevisions: capResult.items,
          auditeeFindings: current.auditeeFindings.length
            ? current.auditeeFindings.map((item) => item.id === finding.id ? finding : item)
            : [finding],
        }));
      },

      async reviewCap(input) {
        if (!projection.finding || !projection.capSubmission) {
          throw new Error("Submitted CAP is unavailable.");
        }
        const capReview = await backendFor("leadInspector").caps.review({
          operationId: operationId("OP-CAP-REVIEW"),
          capRevisionId: projection.capSubmission.capRevisionId,
          expectedCapRevision: projection.capSubmission.capRevision,
          findingId: projection.finding.id,
          expectedFindingRevision: projection.finding.revision,
          ...input,
        });
        const finding = await backendFor("leadInspector").findings.get({
          findingId: projection.finding.id,
        });
        setProjection((current) => ({ ...current, capReview, finding }));
      },

      async submitEvidence(file) {
        if (!projection.finding) throw new Error("Finding is unavailable.");
        const backend = backendFor("auditee");
        const digest = await crypto.subtle.digest("SHA-256", await file.arrayBuffer());
        const sha256 = `sha256:${Array.from(new Uint8Array(digest), (byte) =>
          byte.toString(16).padStart(2, "0"),
        ).join("")}`;
        const upload = await backend.evidence.beginUpload({
          operationId: operationId("OP-EVIDENCE-BEGIN"),
          findingId: projection.finding.id,
          expectedFindingRevision: projection.finding.revision,
          fileName: file.name,
          declaredMediaType: file.type || "application/octet-stream",
          byteSize: file.size,
          sha256,
        });
        if (backend.mode === "http") {
          const uploadResponse = await fetch(upload.uploadUrl, {
            method: "PUT",
            headers: upload.requiredHeaders,
            body: file,
          });
          if (!uploadResponse.ok) {
            throw new Error(`Evidence object upload failed with status ${uploadResponse.status}.`);
          }
        }
        await backend.evidence.completeUpload({
          operationId: operationId("OP-EVIDENCE-COMPLETE"),
          uploadId: upload.uploadId,
          sha256,
          byteSize: file.size,
        });
        let evidenceVersions: EvidenceVersionView[] = [];
        let finding = await backend.findings.get({ findingId: projection.finding.id });
        for (let attempt = 0; attempt < 100; attempt += 1) {
          evidenceVersions = await backend.evidence.listVersions({ findingId: projection.finding.id });
          finding = await backend.findings.get({ findingId: projection.finding.id });
          if (backend.mode === "mock" || evidenceVersions.at(-1)?.scanState === "CLEAN") break;
          await new Promise((resolve) => setTimeout(resolve, 50));
        }
        if (evidenceVersions.at(-1)?.scanState !== "CLEAN") {
          throw new Error("Evidence scan did not reach CLEAN before the review timeout.");
        }
        setProjection((current) => ({
          ...current,
          evidenceVersions,
          finding,
          auditeeFindings: current.auditeeFindings.length
            ? current.auditeeFindings.map((item) => item.id === finding.id ? finding : item)
            : [finding],
        }));
      },

      async loadEvidenceVersions(role) {
        if (!projection.finding) throw new Error("Finding is unavailable.");
        const evidenceVersions = await backendFor(role).evidence.listVersions({
          findingId: projection.finding.id,
        });
        setProjection((current) => ({ ...current, evidenceVersions }));
      },

      async reviewEvidence(input) {
        if (!projection.finding) throw new Error("Finding is unavailable.");
        const latest = projection.evidenceVersions.at(-1);
        if (!latest) throw new Error("Evidence version is unavailable.");
        const backend = backendFor("leadInspector");
        const evidenceReview = await backend.evidence.review({
          operationId: operationId("OP-EVIDENCE-REVIEW"),
          evidenceVersionId: latest.id,
          expectedEvidenceVersionRevision: latest.revision,
          findingId: projection.finding.id,
          expectedFindingRevision: projection.finding.revision,
          ...input,
        });
        const [evidenceVersions, finding] = await Promise.all([
          backend.evidence.listVersions({ findingId: projection.finding.id }),
          backend.findings.get({ findingId: projection.finding.id }),
        ]);
        setProjection((current) => ({
          ...current,
          evidenceReview,
          evidenceVersions,
          finding,
        }));
      },

      async authorizedClose(reason) {
        if (!projection.finding) throw new Error("Finding is unavailable.");
        if (!reason.trim()) throw new Error("Authorized closure reason is required.");
        const finding = await backendFor("manager").findings.authorizedClose({
          operationId: operationId("OP-AUTHORIZED-CLOSE"),
          findingId: projection.finding.id,
          expectedFindingRevision: projection.finding.revision,
          reason,
        });
        setProjection((current) => ({ ...current, finding }));
      },

      async loadReport(role) {
        const report = await backendFor(role).reports.getVersion({
          reportVersionId: "RPT-CAB-2026-001-V1",
        });
        setProjection((current) => ({ ...current, report }));
      },

      async issueReport(reason) {
        const backend = backendFor("executiveDirector");
        const currentReport =
          projection.report ??
          (await backend.reports.getVersion({ reportVersionId: "RPT-CAB-2026-001-V1" }));
        const report = await backend.reports.decide({
          operationId: operationId("OP-REPORT-ISSUE"),
          reportVersionId: currentReport.reportVersionId,
          expectedReportVersionRevision: currentReport.revision,
          decision: "ISSUE_AND_LOCK",
          reason,
        });
        const finding = await backend.findings.get({ findingId: "FND-CAB-2026-001" });
        setProjection((current) => ({ ...current, report, finding }));
      },

      async loadManagerDashboard() {
        const backend = backendFor("manager");
        const [dashboard, findings, report] = await Promise.all([
          backend.dashboards.getManagerProjection({}),
          backend.findings.list({}),
          backend.reports.getVersion({ reportVersionId: "RPT-CAB-2026-001-V1" }),
        ]);
        const finding = findings.items.find((candidate) => candidate.id === "FND-CAB-2026-001");
        setProjection((current) => ({
          ...current,
          dashboard,
          finding: finding ?? current.finding,
          report,
        }));
      },

      async syncFieldWork(trigger = "manual") {
        if (runtime.buildProfile !== "http") return;
        const repository = fieldRepository();
        const local = await repository.loadPackage(FIELD_PACKAGE_ID);
        if (!local) return;
        const state = await repository.getSyncState(FIELD_PACKAGE_ID);
        if (!state) throw new Error("The field sync scope is unavailable.");
        const lock = createBrowserSyncOwnerLock();
        if (!lock) throw new Error("The approved managed-browser sync lock is unavailable.");
        const attachmentStore = inspectionAttachmentStore(repository);
        const engine = new ForegroundSyncEngine({
          backend: backendFor("inspector"),
          repository,
          lock,
          readAttachmentBytes: (path) => attachmentStore.fileSystem.read(path),
          broadcast: syncBroadcastRef.current?.broadcast,
        });
        const report = await engine.run(
          { packageId: FIELD_PACKAGE_ID, offlineGrantId: state.grantId },
          trigger,
        );
        let refreshed: FieldPackageView | null = null;
        if (report.status !== "resnapshot-required") {
          refreshed = await repository.loadPackage(FIELD_PACKAGE_ID);
        }
        setProjection((current) => ({
          ...current,
          ...(refreshed ? fieldProjection(refreshed) : {}),
          fieldSyncStatus: report.status,
          fieldSyncErrorCode: report.errorCode,
          fieldSyncConflict: report.conflict,
        }));
      },
    }),
    [projection, runtime],
  );

  const actionsRef = useRef(actions);
  actionsRef.current = actions;
  useEffect(() => {
    if (runtime.buildProfile !== "http") return;
    const broadcast = createBrowserSyncBroadcast();
    syncBroadcastRef.current = broadcast;
    const unsubscribe = broadcast?.subscribe((message) => {
      if (message.packageId !== FIELD_PACKAGE_ID) return;
      setProjection((current) => ({
        ...current,
        fieldSyncStatus: message.report.status,
        fieldSyncErrorCode: message.report.errorCode,
        fieldSyncConflict: message.report.conflict,
      }));
    });
    const triggers = installForegroundSyncTriggers({
      eventTarget: window,
      documentTarget: document,
      run: (trigger) => actionsRef.current.syncFieldWork(trigger).catch((cause) => {
        setProjection((current) => ({
          ...current,
          fieldSyncStatus: "retryable",
          fieldSyncErrorCode: cause instanceof Error ? cause.message : "SYNC_TRIGGER_FAILED",
        }));
      }),
    });
    return () => {
      triggers.close();
      unsubscribe?.();
      broadcast?.close();
      if (syncBroadcastRef.current === broadcast) syncBroadcastRef.current = null;
    };
  }, [runtime.buildProfile]);

  return <ScenarioContext.Provider value={{ projection, actions }}>{children}</ScenarioContext.Provider>;
}

export function useScenario(): ScenarioContextValue {
  const value = useContext(ScenarioContext);
  if (!value) throw new Error("Canonical scenario context is unavailable");
  return value;
}
