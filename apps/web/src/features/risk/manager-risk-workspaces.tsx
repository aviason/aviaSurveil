import { useEffect, useMemo, useState, type ReactNode } from "react";
import { Link } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type {
  FindingView,
  ManagementRiskLevel,
  OrganizationSummary,
  RiskManagementProjectionView,
  RiskOverviewView,
} from "../../backend/backend";
import { CommandError, errorMessage, PageHeader, WorkspaceShell } from "../shared/workspace-shell";

interface IntelligenceProjection {
  organizations: OrganizationSummary[];
  findings: FindingView[];
  overviews: Record<string, RiskOverviewView>;
  management: RiskManagementProjectionView;
}

const emptyManagementProjection: RiskManagementProjectionView = {
  findings: [],
  capEffectiveness: [],
  generatedAt: "2026-06-15T09:00:00.000Z",
  revision: 1,
};

const emptyProjection: IntelligenceProjection = {
  organizations: [],
  findings: [],
  overviews: {},
  management: emptyManagementProjection,
};

function useIntelligenceProjection() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("manager") ?? runtime.backend, [runtime]);
  const [projection, setProjection] = useState<IntelligenceProjection>(emptyProjection);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    if (!backend.risk) {
      setError("Risk projection is unavailable in this build profile.");
      return () => { cancelled = true; };
    }
    void Promise.all([
      backend.organizations.list({ limit: 100 }),
      backend.findings.list({ limit: 100 }),
      backend.risk.getManagementProjection({}),
    ]).then(async ([organizationPage, findingPage, management]) => {
      if (!backend.risk) throw new Error("Risk projection is unavailable in this build profile.");
      const organizationOverviews = await Promise.all(
        organizationPage.items.map(async (organization) => [
          organization.id,
          await backend.risk!.getOverview({ organizationId: organization.id }),
        ] as const),
      );
      if (!cancelled) {
        setProjection({
          organizations: organizationPage.items,
          findings: findingPage.items,
          overviews: Object.fromEntries(organizationOverviews),
          management,
        });
      }
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend]);

  return { projection, error };
}

function AdvisoryBoundary({ children }: { children?: ReactNode }) {
  return (
    <div className="manager-intelligence-boundary" role="note">
      <b>Management indicator only.</b> Not a legal decision. No automatic legal, enforcement,
      certification, suspension, closure, or compliance decision is made by this workspace.
      {children}
    </div>
  );
}

const riskLevels: readonly ManagementRiskLevel[] = ["HIGH", "MEDIUM", "LOW", "VERY_LOW"];

function riskLabel(level: ManagementRiskLevel): string {
  return level === "VERY_LOW" ? "Very Low" : level.charAt(0) + level.slice(1).toLowerCase();
}

function IndicatorCards({ records }: { records: RiskManagementProjectionView["findings"] }) {
  const indicators: ReadonlyArray<readonly [string, number]> = [
    ["Total Findings", records.length],
    ...riskLevels.map((level) => [riskLabel(level), records.filter((record) => record.riskLevel === level).length] as const),
    ["Overdue CAPs", records.filter((record) => record.capRequired && record.dueState === "OVERDUE" && record.status !== "CLOSED").length],
  ];
  return (
    <section aria-label="Risk indicators" className="manager-intelligence-indicators">
      {indicators.map(([label, value]) => <article key={label}><span>{label}</span><strong>{value}</strong><small>Derived from Finding records</small></article>)}
    </section>
  );
}

function csvCell(value: string): string {
  return `"${value.replaceAll('"', '""')}"`;
}

function csvFor(records: RiskManagementProjectionView["findings"]): string {
  const rows = records.map((record) => [
    record.findingId,
    record.findingNumber,
    record.organizationId,
    record.inspectionId,
    record.department ?? "Unavailable",
    riskLabel(record.riskLevel),
    record.status,
    record.dueState,
  ].map(csvCell).join(","));
  return [
    "Finding ID,Finding Number,Organization ID,Inspection ID,Department,Risk Level,Status,Due State",
    ...rows,
  ].join("\n");
}

function OrganizationProfileAction({ organizationId }: { organizationId: string }) {
  if (organizationId === "ORG-FLY-NAMIBIA") {
    return <Link aria-label="Open risk profile for ORG-FLY-NAMIBIA" to="/department-manager/organizations/ORG-FLY-NAMIBIA/risk-profile">Open profile</Link>;
  }
  return (
    <button
      aria-label={`Risk profile unavailable for ${organizationId}`}
      disabled
      title={`Organization ${organizationId} has no declared Department Manager risk-profile route in Plan 1.`}
      type="button"
    >
      Profile unavailable
    </button>
  );
}

export function ManagerRiskDashboardPage() {
  const { projection, error } = useIntelligenceProjection();
  const [dateRange, setDateRange] = useState("all");
  const [department, setDepartment] = useState("all");
  const [inspection, setInspection] = useState("all");
  const [riskLevel, setRiskLevel] = useState("all");
  const [exportStatus, setExportStatus] = useState<string | null>(null);
  const departments = [...new Set(projection.management.findings.map((record) => record.department ?? "unavailable"))];
  const inspections = [...new Set(projection.management.findings.map((record) => record.inspectionId))];
  const generatedAt = new Date(projection.management.generatedAt).getTime();
  const filtered = projection.management.findings.filter((record) => {
    if (department !== "all" && (record.department ?? "unavailable") !== department) return false;
    if (inspection !== "all" && record.inspectionId !== inspection) return false;
    if (riskLevel !== "all" && record.riskLevel !== riskLevel) return false;
    if (dateRange === "all") return true;
    if (!record.issuedAt) return false;
    const issuedAt = new Date(record.issuedAt).getTime();
    const elapsedDays = (generatedAt - issuedAt) / 86_400_000;
    if (dateRange === "last-30") return elapsedDays >= 0 && elapsedDays <= 30;
    if (dateRange === "last-90") return elapsedDays >= 0 && elapsedDays <= 90;
    return new Date(record.issuedAt).getUTCFullYear() === new Date(projection.management.generatedAt).getUTCFullYear();
  });
  const distribution = Object.fromEntries(riskLevels.map((level) => [level, filtered.filter((record) => record.riskLevel === level).length])) as Record<ManagementRiskLevel, number>;
  const trend = [...new Set(filtered.map((record) => record.issuedAt?.slice(0, 7) ?? "Date unavailable"))]
    .map((period) => ({ period, count: filtered.filter((record) => (record.issuedAt?.slice(0, 7) ?? "Date unavailable") === period).length }));
  const filtersAreDefault = dateRange === "all" && department === "all" && inspection === "all" && riskLevel === "all";
  const resetFilters = () => {
    setDateRange("all");
    setDepartment("all");
    setInspection("all");
    setRiskLevel("all");
    setExportStatus(null);
  };
  const csvHref = `data:text/csv;charset=utf-8,${encodeURIComponent(csvFor(filtered))}`;

  return (
    <WorkspaceShell roleLabel="Department Manager" routeLabel="Risk Dashboard">
      <div className="manager-intelligence-page" data-testid="manager-risk-dashboard-page">
        <PageHeader eyebrow="Department intelligence" title="Risk Dashboard" description="Review Finding exposure, Due Dates, and CAP attention using advisory management indicators." />
        <CommandError message={error} />
        <AdvisoryBoundary />
        <section aria-label="Risk filters" className="manager-intelligence-filters">
          <label>Date Range<select aria-label="Date Range" value={dateRange} onChange={(event) => setDateRange(event.target.value)}><option value="all">All dates</option><option value="last-30">Last 30 days</option><option value="last-90">Last 90 days</option><option value="year">Current year</option></select></label>
          <label>Department<select aria-label="Department" value={department} onChange={(event) => setDepartment(event.target.value)}><option value="all">All departments</option>{departments.map((value) => <option key={value} value={value}>{value === "unavailable" ? "Unavailable in Backend contract" : value}</option>)}</select></label>
          <label>Inspection<select aria-label="Inspection" value={inspection} onChange={(event) => setInspection(event.target.value)}><option value="all">All inspections</option>{inspections.map((value) => <option key={value} value={value}>{value}</option>)}</select></label>
          <label>Risk Level<select aria-label="Risk Level" value={riskLevel} onChange={(event) => setRiskLevel(event.target.value)}><option value="all">All risk levels</option>{riskLevels.map((value) => <option key={value} value={value}>{riskLabel(value)}</option>)}</select></label>
          <div className="manager-intelligence-filter-actions">
            <button
              aria-label={filtersAreDefault ? "Reset risk filters unavailable" : undefined}
              disabled={filtersAreDefault}
              onClick={resetFilters}
              title={filtersAreDefault ? "Risk filters are already at their defaults." : undefined}
              type="button"
            >
              Reset filters
            </button>
            <a download="AviaSurveil360_Department_Risk_Summary.csv" href={csvHref} onClick={() => setExportStatus(`CSV prepared for ${filtered.length} Finding records`)}>Export CSV</a>
          </div>
          <span data-testid="active-risk-filters">{dateRange} · {department} · {inspection} · {riskLevel}</span>
          {exportStatus ? <p role="status">{exportStatus}</p> : null}
        </section>
        <IndicatorCards records={filtered} />
        <section aria-label="Management attention" className="manager-intelligence-grid">
          <article className="manager-intelligence-panel"><p className="eyebrow">Exact severity mapping</p><h2>Findings by Risk</h2><ul>{riskLevels.map((level) => <li key={level}><span>{riskLabel(level)}</span><b>{distribution[level]}</b></li>)}</ul><small>Level 1 → High, Level 2 → Medium, Level 3 → Low, Observation → Very Low.</small></article>
          <article className="manager-intelligence-panel"><p className="eyebrow">Issued Finding records</p><h2>Risk Trend</h2>{trend.length ? <ul>{trend.map((item) => <li key={item.period}><span>{item.period}</span><b>{item.count}</b></li>)}</ul> : <p>No issued Finding dates match.</p>}</article>
          <article className="manager-intelligence-panel manager-intelligence-panel--wide"><p className="eyebrow">Likelihood × impact</p><h2>Risk Exposure Matrix</h2><div aria-label="Advisory risk matrix" data-testid="risk-exposure-matrix"><p>Likelihood and impact values are unavailable in the Backend contract; cells are intentionally unscored.</p><div className="manager-risk-matrix">{Array.from({ length: 25 }, (_, index) => <span data-matrix-cell key={index}><b>—</b><small>L{Math.floor(index / 5) + 1} × I{index % 5 + 1}</small></span>)}</div></div></article>
          <article className="manager-intelligence-panel"><p className="eyebrow">Record concentration</p><h2>Top Risky Areas</h2><p>Department unavailable in Backend contract · {filtered.length} Finding records</p></article>
          <article className="manager-intelligence-panel"><p className="eyebrow">Department scope</p><h2>Department Risk Distribution</h2><p>Unavailable in Backend contract · {filtered.length} Finding records</p></article>
          <article className="manager-intelligence-panel"><p className="eyebrow">Due Date attention</p><h2>Overdue CAPs by Risk</h2><ul>{riskLevels.map((level) => <li key={level}><span>{riskLabel(level)}</span><b>{filtered.filter((record) => record.riskLevel === level && record.capRequired && record.dueState === "OVERDUE" && record.status !== "CLOSED").length}</b></li>)}</ul></article>
          <article className="manager-intelligence-panel manager-intelligence-panel--wide"><p className="eyebrow">Exact record identity</p><h2>Recent Finding records</h2>{filtered.length ? <div className="manager-intelligence-table-wrap"><table><thead><tr><th>Finding</th><th>Organization</th><th>Inspection</th><th>Risk</th><th>Status</th><th>Action</th></tr></thead><tbody>{filtered.map((record) => <tr key={record.findingId}><td><b>{record.findingId}</b><small>{record.findingNumber}</small></td><td>{record.organizationName}<small>{record.organizationId}</small></td><td>{record.inspectionId}<small>{record.inspectionTitle ?? "Title unavailable"}</small></td><td>{riskLabel(record.riskLevel)}</td><td>{record.status}</td><td><Link to={`/department-manager/findings-review?organizationId=${record.organizationId}&findingId=${record.findingId}`}>Open Finding</Link></td></tr>)}</tbody></table></div> : <p>No Finding records match the active risk filters.</p>}</article>
          <article className="manager-intelligence-panel manager-intelligence-panel--wide"><p className="eyebrow">Priority review</p><h2>Management attention</h2><p>{filtered.some((record) => record.dueState === "OVERDUE") ? "Open the overdue Finding and verify its CAP / Evidence next action." : "Review organization profiles and upcoming oversight work."}</p><Link to="/department-manager/safety-intelligence">Open Safety Intelligence</Link></article>
          <article className="manager-intelligence-panel"><p className="eyebrow">State safety monitoring</p><h2>SSP / NASP</h2><p>Track objectives, SPI trends, and accountable sections.</p><Link to="/department-manager/ssp-nasp">Open SSP / NASP</Link></article>
          <article className="manager-intelligence-panel"><p className="eyebrow">Readiness support</p><h2>USOAP Readiness</h2><p>Review configured PQ / CE evidence gaps without an official EI claim.</p><Link to="/department-manager/usoap-readiness">Open USOAP Readiness</Link></article>
          <article className="manager-intelligence-panel"><p className="eyebrow">Post-closure monitoring</p><h2>CAP Effectiveness</h2><p>Monitor recurrence separately from CAP acceptance and Finding closure.</p><Link to="/department-manager/cap-effectiveness">Open CAP Effectiveness</Link></article>
          {projection.organizations.map((organization) => <article className="manager-intelligence-panel" key={organization.id}><p className="eyebrow">Organization profile</p><h2>{organization.legalName}</h2><p>{organization.id} · {projection.overviews[organization.id]?.openFindingCount ?? 0} open Findings</p><OrganizationProfileAction organizationId={organization.id} /></article>)}
        </section>
      </div>
    </WorkspaceShell>
  );
}

export function ManagerSafetyIntelligencePage() {
  const { projection, error } = useIntelligenceProjection();
  const [filter, setFilter] = useState("all");
  const signals = projection.organizations.filter((organization) => {
    const overview = projection.overviews[organization.id];
    if (filter === "overdue") return (overview?.overdueFindingCount ?? 0) > 0;
    if (filter === "repeat") return (overview?.repeatFindingCount ?? 0) > 0;
    return true;
  });
  return (
    <WorkspaceShell roleLabel="Department Manager" routeLabel="Safety Intelligence">
      <div className="manager-intelligence-page" data-testid="manager-safety-intelligence-page">
        <PageHeader eyebrow="Management signals" title="Safety Intelligence Dashboard" description="Decide which risk, delay, or workload signal needs management attention." />
        <CommandError message={error} /><AdvisoryBoundary />
        <section aria-label="Safety intelligence filters" className="manager-intelligence-filters"><label>Signal filter<select aria-label="Signal filter" value={filter} onChange={(event) => setFilter(event.target.value)}><option value="all">All signals</option><option value="overdue">Overdue</option><option value="repeat">Repeat</option></select></label><output data-testid="active-signal-filter">{filter}</output></section>
        <p className="eyebrow">Management attention</p>
        <section aria-label="Management attention" className="manager-signal-list">{signals.map((organization) => { const overview = projection.overviews[organization.id]; return <article key={organization.id}><div><p className="eyebrow">Mock risk indicator</p><h2>{organization.legalName}</h2><p>{organization.id} · Recommended action: review exact Finding and planning context.</p></div><dl><div><dt>Open Findings</dt><dd>{overview?.openFindingCount ?? 0}</dd></div><div><dt>Overdue</dt><dd>{overview?.overdueFindingCount ?? 0}</dd></div></dl><OrganizationProfileAction organizationId={organization.id} /></article>; })}{signals.length === 0 ? <p>No safety intelligence signals match this filter.</p> : null}</section>
      </div>
    </WorkspaceShell>
  );
}

export function OrganizationRiskProfilePage() {
  const { projection, error } = useIntelligenceProjection();
  const organization = projection.organizations.find((item) => item.id === "ORG-FLY-NAMIBIA") ?? null;
  const overview = organization ? projection.overviews[organization.id] ?? null : null;
  return (
    <WorkspaceShell roleLabel="Department Manager" routeLabel="Organization Risk Profile">
      <div className="manager-intelligence-page" data-testid="organization-risk-profile-page">
        <PageHeader eyebrow="Exact Organization profile" title={`Organization Risk Profile — ${organization?.legalName ?? "Fly Namibia"}`} description="Understand why this Organization needs oversight attention before planning or opening an inspection." />
        <CommandError message={error} /><AdvisoryBoundary />
        <section className="organization-risk-summary"><div><span>Oversight Health</span><strong>Unavailable</strong><small>Oversight Health value is unavailable in the Backend contract.</small></div><dl><div><dt>Organization ID</dt><dd>ORG-FLY-NAMIBIA</dd></div><div><dt>Open Findings</dt><dd>{overview?.openFindingCount ?? 0}</dd></div><div><dt>Overdue</dt><dd>{overview?.overdueFindingCount ?? 0}</dd></div><div><dt>Repeat signals</dt><dd>{overview?.repeatFindingCount ?? 0}</dd></div></dl></section>
        <section aria-label="Organization risk actions" className="manager-intelligence-actions"><Link aria-label="Open Fly Namibia organization record" to="/department-manager/organizations/ORG-FLY-NAMIBIA">Open organization record</Link><Link aria-label="Open Findings Review for ORG-FLY-NAMIBIA" to="/department-manager/findings-review?organizationId=ORG-FLY-NAMIBIA">Open Findings Review</Link><Link to="/department-manager/safety-intelligence">Back to Safety Intelligence</Link></section>
      </div>
    </WorkspaceShell>
  );
}

const sspObjectives = [
  { id: "SSP-OBJ-001", title: "Strengthen cabin emergency equipment oversight follow-up", spi: "Overdue CAP ratio", current: "18%", target: "< 10%", action: "NASP-ACT-001", owner: "Cabin Safety Section", status: "In progress" },
  { id: "SSP-OBJ-002", title: "Improve evidence-based closure discipline", spi: "Evidence accepted before closure", current: "Demo workflow enforced", target: "100%", action: "NASP-ACT-002", owner: "Oversight Quality Team", status: "Planned" },
] as const;

export function ManagerSspNaspPage() {
  return <WorkspaceShell roleLabel="Department Manager" routeLabel="SSP / NASP"><div className="manager-intelligence-page" data-testid="manager-ssp-nasp-page"><PageHeader eyebrow="State safety monitoring" title="SSP/NASP Management Dashboard" description="Track safety objectives, SPI trends, NASP actions, and responsible sections." /><AdvisoryBoundary><span> Supports monitoring; it does not determine State safety performance automatically.</span></AdvisoryBoundary><section className="manager-dossier-list">{sspObjectives.map((item) => <article key={item.id}><p className="eyebrow">{item.id} · Monitoring indicator</p><h2>{item.title}</h2><div className="manager-intelligence-table-wrap"><table><thead><tr><th>SPI</th><th>Current</th><th>Target</th></tr></thead><tbody><tr><td>{item.spi}</td><td>{item.current}</td><td>{item.target}</td></tr></tbody></table><table><thead><tr><th>NASP action</th><th>Owner</th><th>Status</th></tr></thead><tbody><tr><td>{item.action}</td><td>{item.owner}</td><td>{item.status}</td></tr></tbody></table></div></article>)}</section><Link to="/department-manager/risk-dashboard">Back to Risk Dashboard</Link></div></WorkspaceShell>;
}

export function ManagerUsoapReadinessPage() {
  return <WorkspaceShell roleLabel="Department Manager" routeLabel="USOAP Readiness"><div className="manager-intelligence-page" data-testid="manager-usoap-readiness-page"><PageHeader eyebrow="Configured readiness support" title="USOAP Readiness Workspace" description="See PQ / CE gaps, missing Evidence, and readiness history without claiming an official EI outcome." /><AdvisoryBoundary><span> No official EI score or automatic compliance conclusion.</span></AdvisoryBoundary><section className="manager-intelligence-table-wrap"><table><thead><tr><th>PQ readiness item</th><th>Critical Element</th><th>Audit area</th><th>Linked CAP / Finding</th><th>Status</th><th>Verification history</th></tr></thead><tbody><tr><td><b>PQ-CAB-MOCK-001</b><small>Mock readiness record; not an official ICAO assessment.</small></td><td>CE-7</td><td>Cabin Safety</td><td>FND-SKYCARGO-2026-099</td><td>Missing evidence</td><td>15 Jun 2026 · Gap remains under review.</td></tr></tbody></table></section><Link to="/department-manager/risk-dashboard">Back to Risk Dashboard</Link></div></WorkspaceShell>;
}

export function ManagerCapEffectivenessPage() {
  const { projection, error } = useIntelligenceProjection();
  const records = projection.management.capEffectiveness;
  return <WorkspaceShell roleLabel="Department Manager" routeLabel="CAP Effectiveness"><div className="manager-intelligence-page" data-testid="manager-cap-effectiveness-page"><PageHeader eyebrow="Post-closure monitoring" title="CAP Effectiveness" description="Review whether accepted CAP actions worked and whether recurrence needs attention." /><CommandError message={error} /><AdvisoryBoundary><span> CAP acceptance is not Finding closure. Effectiveness review remains separate from Evidence acceptance and authorized verification.</span></AdvisoryBoundary><section className="manager-intelligence-table-wrap"><table><thead><tr><th>Finding</th><th>CAP revision</th><th>Organization</th><th>Closure / verification</th><th>Action</th></tr></thead><tbody>{records.length ? records.map((record) => <tr key={record.capRevisionId}><td><b>{record.findingId}</b><small>{record.findingNumber} · {record.findingStatus}</small></td><td><b>{record.capId}</b><small>{record.capRevisionId} · revision {record.capRevision} · {record.capStatus}</small></td><td>{record.organizationName}<small>{record.organizationId}</small></td><td><b>{record.state === "NOT_ELIGIBLE" ? "Unavailable" : "Pending post-closure verification"}</b><small>{record.reason}</small></td><td><Link to={`/department-manager/findings-review?organizationId=${record.organizationId}&findingId=${record.findingId}`}>Open {record.findingId}</Link></td></tr>) : <tr><td colSpan={5}>No CAP revision records are available.</td></tr>}</tbody></table></section><Link to="/department-manager/risk-dashboard">Back to Risk Dashboard</Link></div></WorkspaceShell>;
}
