import type { components } from "../generated/transport/api-types";

import type { BackendRequestOptions } from "./backend";

type Schemas = components["schemas"];

export type AGADemoWorkspaceCapability = Schemas["AGADemoWorkspaceCapability"];
export type AGADemoWorkspaceQuery = Schemas["AGADemoWorkspaceQuery"];
export type AGADemoWorkspaceCommand = Schemas["AGADemoWorkspaceCommand"];
export type AGADemoWorkspaceQueryResponse = Schemas["AGADemoWorkspaceQueryResponse"];
export type AGADemoWorkspaceCommandResponse = Schemas["AGADemoWorkspaceCommandResponse"];
export type AGADemoWorkspaceDraft = Schemas["AGADemoWorkspaceDraft"];
export type AGADemoWorkspaceGeneration = Schemas["AGADemoWorkspaceGeneration"];
export type AGADemoWorkspaceProviderConfiguration = Schemas["AGADemoWorkspaceProviderConfiguration"];
export type AGADemoWorkspaceLifecycleQuestionSnapshot = Schemas["AGADemoWorkspaceLifecycleQuestionSnapshot"];
export type AGADemoWorkspaceLifecycleResponse = Schemas["AGADemoWorkspaceLifecycleResponse"];
export type AGADemoWorkspaceLifecyclePotentialFinding = Schemas["AGADemoWorkspaceLifecyclePotentialFinding"];
export type AGADemoWorkspaceLifecycleFinding = Schemas["AGADemoWorkspaceLifecycleFinding"];
export type AGADemoWorkspaceLifecycleCAPRevision = Schemas["AGADemoWorkspaceLifecycleCAPRevision"];
export type AGADemoWorkspaceLifecycleEvidenceVersion = Schemas["AGADemoWorkspaceLifecycleEvidenceVersion"];
export type AGADemoWorkspaceLifecycleVerificationDecision = Schemas["AGADemoWorkspaceLifecycleVerificationDecision"];
export type AGADemoWorkspaceLifecycleProjection = Schemas["AGADemoWorkspaceLifecycleProjection"];
export type AGADemoWorkspaceLifecycleCAAProjection = Schemas["AGADemoWorkspaceLifecycleCAAProjection"];
export type AGADemoWorkspaceLifecycleAuditeeProjection = Schemas["AGADemoWorkspaceLifecycleAuditeeProjection"];

export type AGADemoWorkspaceQueryOperation = AGADemoWorkspaceQuery["operationId"];
export type AGADemoWorkspaceCommandOperation = AGADemoWorkspaceCommand["operationId"];

export interface AGADemoWorkspaceBackend {
  capability(options?: BackendRequestOptions): Promise<AGADemoWorkspaceCapability>;
  classificationQuery(input: AGADemoWorkspaceQuery, options?: BackendRequestOptions): Promise<AGADemoWorkspaceQueryResponse>;
  classificationCommand(input: AGADemoWorkspaceCommand, options?: BackendRequestOptions): Promise<AGADemoWorkspaceCommandResponse>;
  recommendationCommand(input: AGADemoWorkspaceCommand, options?: BackendRequestOptions): Promise<AGADemoWorkspaceCommandResponse>;
  lifecycleQuery(input: AGADemoWorkspaceQuery, options?: BackendRequestOptions): Promise<AGADemoWorkspaceQueryResponse>;
  lifecycleCommand(input: AGADemoWorkspaceCommand, options?: BackendRequestOptions): Promise<AGADemoWorkspaceCommandResponse>;
  adminCommand(input: AGADemoWorkspaceCommand, options?: BackendRequestOptions): Promise<AGADemoWorkspaceCommandResponse>;
}

export function isWorkspaceCapabilityAvailable(value: AGADemoWorkspaceCapability | null | undefined): value is AGADemoWorkspaceCapability {
  return value?.available === true;
}
