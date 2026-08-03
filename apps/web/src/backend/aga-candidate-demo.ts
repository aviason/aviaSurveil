import type { components } from "../generated/transport/api-types";

import type { BackendRequestOptions, PageOutput } from "./backend";

type Schemas = components["schemas"];

export interface AGACandidateDemoCapability {
  available: boolean;
  labels: readonly string[];
}

export interface AGACandidateDemoSummary {
  packageDigest: string;
  formCount: number;
  questionCount: number;
  sourceRequirements: readonly string[];
  labels: readonly string[];
}

export type AGACandidateDemoForm = Schemas["AGACandidateDemoForm"];
export type AGACandidateDemoQuestion = Schemas["AGACandidateDemoQuestion"];

/**
 * Read-only boundary for the tagged, disposable preprod candidate projection.
 * It deliberately exposes neither commands nor a browser-local fallback.
 */
export interface AGACandidateDemoBackend {
  capability(input: Record<string, never>, options?: BackendRequestOptions): Promise<AGACandidateDemoCapability>;
  summary(input: Record<string, never>, options?: BackendRequestOptions): Promise<AGACandidateDemoSummary>;
  listForms(input: { cursor?: string; limit?: number }, options?: BackendRequestOptions): Promise<PageOutput<AGACandidateDemoForm>>;
  getForm(input: { formCode: string }, options?: BackendRequestOptions): Promise<AGACandidateDemoForm>;
  listQuestions(input: { cursor?: string; limit?: number; formCode?: string; sourceGapCategory?: "PROPOSAL_PRESENT_REVIEW_REQUIRED" | "UNMAPPED_NO_QUESTION_LEVEL_SOURCE_PROPOSAL"; riskBand?: "PROPOSED_CONTROL_ASSURANCE" | "PROPOSED_HIGH_OPERATIONAL" | "PROPOSED_REVIEW_REQUIRED" | "PROPOSED_SAFETY_CRITICAL" }, options?: BackendRequestOptions): Promise<PageOutput<AGACandidateDemoQuestion>>;
}
