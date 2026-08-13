import { useEffect, useState } from "react";
import { Link } from "react-router-dom";

import { useBackendForRole } from "../../app/providers";
import type {
  ChecklistAnswer,
  ChecklistTemplateVersionDetailView,
  ChecklistTemplateVersionView,
} from "../../backend/backend";
import { CommandError, errorMessage, WorkspaceShell } from "../shared/workspace-shell";
import { AdminPage } from "./admin-workspace-shared";

type AdminPreviewMode = "register" | "preview";

const answerLabels: Record<ChecklistAnswer, string> = {
  COMPLIANT: "Compliant",
  NON_COMPLIANT: "Non-Compliant",
  OBSERVATION: "Observation",
  NOT_APPLICABLE: "Not Applicable",
  NOT_CHECKED: "Not Checked",
};

function answers(values: ChecklistAnswer[]): string {
  return values.map((value) => answerLabels[value]).join(" · ");
}

function templateDomain(templateId: string): string {
  return templateId === "CABIN" ? "Cabin Safety" : templateId;
}

export function AdminConfigurationPage() {
  const backend = useBackendForRole("admin");
  const [templates, setTemplates] = useState<ChecklistTemplateVersionView[]>([]);
  const [detail, setDetail] = useState<ChecklistTemplateVersionDetailView | null>(null);
  const [mode, setMode] = useState<AdminPreviewMode>("preview");
  const [error, setError] = useState<string | null>(null);

  async function loadDetail(templateVersionId: string): Promise<void> {
    setError(null);
    try {
      const next = await backend.configuration.getChecklistTemplateVersion({ templateVersionId });
      if (!Array.isArray(next.questions)) throw new Error("Checklist template detail is malformed.");
      setDetail(next);
      setMode("preview");
    } catch (cause) {
      setDetail(null);
      setError(errorMessage(cause));
    }
  }

  useEffect(() => {
    let active = true;
    void backend.configuration.listChecklistTemplateVersions({ limit: 100 })
      .then(async (output) => {
        if (!active) return;
        setTemplates(output.items);
        if (!output.items.length) {
          setMode("register");
          return;
        }
        await loadDetail(output.items[0]!.id);
      })
      .catch((cause) => {
        if (active) {
          setMode("register");
          setError(errorMessage(cause));
        }
      });
    return () => { active = false; };
    // The role-scoped Backend is stable for the mounted route.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <WorkspaceShell roleLabel="Admin Preview" routeLabel="Template Preview">
      <div className="admin-template-workspace" data-testid="admin-template-preview-page">
        {mode === "register" ? (
          <>
            <header className="admin-page-head workbench-page-header">
              <div><h1>Checklist Templates</h1><p>Published checklist versions available for read-only preview.</p></div>
            </header>
            <CommandError message={error} />
            {templates.length ? (
              <div className="admin-table-scroll">
                <table aria-label="Published checklist template versions" className="admin-template-register">
                  <thead><tr><th>Template</th><th>Domain</th><th>Version</th><th>Items</th><th>Status</th><th>Action</th></tr></thead>
                  <tbody>{templates.map((template) => (
                    <tr key={template.id}>
                      <td><b>{template.title}</b><small>{template.id}</small></td>
                      <td>{templateDomain(template.templateId)}</td>
                      <td>Version {template.version}</td>
                      <td>{template.questionCount}</td>
                      <td><span className="admin-published-badge"><i />Published</span></td>
                      <td><button aria-label={`Preview ${template.id}`} onClick={() => void loadDetail(template.id)} type="button">Preview</button></td>
                    </tr>
                  ))}</tbody>
                </table>
              </div>
            ) : <p className="admin-empty-state">No published checklist template versions are available.</p>}
          </>
        ) : (
          <>
            <header className="admin-page-head workbench-page-header">
              <div><h1>Template Preview — Cabin Inspection</h1><p>Read-only preview of the published checklist template.</p></div>
              <Link className="admin-action-link" to="/admin/template-library">Back to templates</Link>
            </header>
            <CommandError message={error} />
            {detail ? (
              <>
                <section aria-label="Published template summary" className="admin-template-summary">
                  <div><span>Template ID</span><b>{detail.id}</b></div>
                  <div><span>Domain</span><b>{templateDomain(detail.templateId)}</b></div>
                  <div><span>Version</span><b data-testid="template-version">Version {detail.version}</b></div>
                  <div><span>Owner</span><b>{templateDomain(detail.templateId)} Section</b></div>
                  <div><span>Items</span><b>{detail.questionCount}</b></div>
                  <div><span>Status</span><b>Published</b><small>{detail.publishedAt.slice(0, 10)}</small></div>
                </section>
                <p className="admin-source-profile">Source workbook profile: 126 Cabin Inspection rows across 6 sections. This demo runs a curated {detail.questionCount}-question subset; the source workbook remains a mock/configured checklist reference, not a live import or legal source.</p>
                <div className="admin-section-jumps" aria-label="Template section jumps">{[...new Set(detail.questions.map((question) => question.sectionId))].map((sectionId) => <a href={`#admin-template-section-${sectionId.replace(/[^a-z0-9]+/gi, "-").toLocaleLowerCase()}`} key={sectionId}>{sectionId}</a>)}</div>
                <div className="admin-table-scroll admin-question-table-scroll">
                  <table aria-label="Published checklist questions" className="admin-question-table">
                    <thead><tr><th>Row</th><th>Question</th><th>Regulatory reference</th><th>Expected evidence</th><th>Trace</th></tr></thead>
                    <tbody>{detail.questions.map((question, index) => (
                      <tr id={`admin-template-section-${question.sectionId.replace(/[^a-z0-9]+/gi, "-").toLocaleLowerCase()}`} key={question.id}>
                        <td><span className="admin-row-number">{index + 1}</span></td>
                        <td><b>{question.prompt}</b><small>{question.id}</small><em className="admin-allowed-answers admin-allowed-answers--question">Allowed answers: {answers(question.allowedAnswers)}</em></td>
                        <td>{question.regulatoryReference ?? "No configured regulatory reference"}</td>
                        <td>{question.expectedEvidence ?? "No expected Evidence configured"}<small>Comment required for {answers(question.commentRequiredFor)}</small></td>
                        <td><span className="admin-published-badge"><i />published</span><em className="admin-allowed-answers admin-allowed-answers--trace">Allowed answers: {answers(question.allowedAnswers)}</em></td>
                      </tr>
                    ))}</tbody>
                  </table>
                </div>
              </>
            ) : null}
          </>
        )}
      </div>
    </WorkspaceShell>
  );
}

export function AdminConfigurationsPage() {
  return (
    <AdminPage testId="admin-configurations-page" routeLabel="Configurations" title="Configurations" description="Demo-only configured rules are separate from production-required integrations and undeclared commands.">
      <div className="admin-configuration-grid">
        <section className="admin-record-card"><h2>Configured demo rules</h2><ul><li>Finding severity: Level 1 Critical, Level 2 Major, Level 3 Minor, Observation</li><li>Finding → CAP → Evidence → CAA Review → Closure lifecycle</li><li>Due Date, Target, Due Soon, and Overdue language</li><li>Oversight Health Index is advisory only and never makes a legal, enforcement, certificate, or closure decision</li></ul></section>
        <section className="admin-record-card"><h2>Notification Rules</h2><p>This section aliases the same configuration surface without creating a second active route.</p><p>Configured 30 / 15 / 7 day and Due Date in-app reminder rules are demo records.</p><p>No real email or SMS delivery is configured.</p></section>
        <section className="admin-record-card"><h2>Production-required integrations</h2><ul><li>Configured identity-provider administration</li><li>Real email / SMS provider</li><li>Real regulatory ingestion and records governance</li></ul><p>No fake editable or Save control is shown because Task 10 declares no configuration mutation.</p></section>
      </div>
    </AdminPage>
  );
}
