import type { Backend, ReportVersionView } from "../../backend/backend";

export type ReportKind = "PRELIMINARY" | "FINAL";

function reportKind(report: ReportVersionView): ReportKind {
  return report.kind;
}

export async function discoverCAAReportVersions(
  backend: Pick<Backend, "mode" | "documents" | "reports">,
  kinds: readonly ReportKind[] = ["PRELIMINARY", "FINAL"],
): Promise<ReportVersionView[]> {
  const documents = await backend.documents.list({});
  const reportVersionIds = documents.items
    .filter((document) => document.kind === "REPORT")
    .map((document) => document.id);
  if (!reportVersionIds.length) return [];

  const reports = await Promise.all(
    reportVersionIds.map((reportVersionId) => backend.reports.getVersion({ reportVersionId })),
  );
  return reports.filter((report) => kinds.includes(reportKind(report)));
}
