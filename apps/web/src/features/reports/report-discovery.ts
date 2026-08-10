import type { Backend, ReportVersionView } from "../../backend/backend";

export type ReportKind = "PRELIMINARY" | "FINAL";

function reportKind(report: ReportVersionView): ReportKind {
  return report.reportId.startsWith("PR-") ? "PRELIMINARY" : "FINAL";
}

export async function discoverCAAReportVersions(
  backend: Pick<Backend, "mode" | "documents" | "reports">,
  kinds: readonly ReportKind[] = ["PRELIMINARY", "FINAL"],
): Promise<ReportVersionView[]> {
  let reportVersionIds: string[];
  try {
    const documents = await backend.documents.list({});
    reportVersionIds = documents.items
      .filter((document) => document.kind === "REPORT")
      .map((document) => document.id);
  } catch (cause) {
    if (backend.mode !== "mock") throw cause;
    reportVersionIds = [
      ...(kinds.includes("PRELIMINARY") ? ["PR-2026-018-V1", "PR-2026-018-V0"] : []),
      ...(kinds.includes("FINAL") ? ["RPT-CAB-2026-001-V1"] : []),
    ];
  }
  if (!reportVersionIds.length) return [];

  const reports = await Promise.all(
    reportVersionIds.map((reportVersionId) => backend.reports.getVersion({ reportVersionId })),
  );
  return reports.filter((report) => kinds.includes(reportKind(report)));
}
