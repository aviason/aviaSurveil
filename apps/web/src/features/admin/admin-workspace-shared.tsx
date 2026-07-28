import { useEffect, useState, type PropsWithChildren, type ReactNode } from "react";

import { useBackendForRole } from "../../app/providers";
import type { AdminWorkspaceBackend } from "../../backend/backend";
import { WorkspaceShell } from "../shared/workspace-shell";

export function useAdminWorkspace(): AdminWorkspaceBackend {
  const capability = useBackendForRole("admin").adminWorkspace;
  if (!capability) throw new Error("Admin workspace capability is unavailable until Plan 2 activates this route.");
  return capability;
}

export function useAdminLoad<T>(loader: () => Promise<T>, dependencies: readonly unknown[]): { data: T | null; error: string | null; reload(): void } {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [revision, setRevision] = useState(0);
  useEffect(() => {
    let active = true;
    setError(null);
    void loader().then((value) => { if (active) setData(value); }).catch((cause) => {
      if (active) setError(cause instanceof Error ? cause.message : "The Admin demo projection could not be loaded.");
    });
    return () => { active = false; };
    // Callers provide the complete stable dependency list.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...dependencies, revision]);
  return { data, error, reload: () => setRevision((value) => value + 1) };
}

export function AdminPage({
  testId,
  routeLabel,
  title,
  description,
  action,
  children,
}: PropsWithChildren<{ testId: string; routeLabel: string; title: string; description: string; action?: ReactNode }>) {
  return (
    <WorkspaceShell roleLabel="Admin Preview" routeLabel={routeLabel}>
      <div className="admin-secondary-page" data-testid={testId}>
        <header className="admin-secondary-page__header workbench-page-header">
          <div><p className="eyebrow">Administration</p><h1>{title}</h1><p>{description}</p></div>
          {action ? <div className="admin-secondary-page__actions">{action}</div> : null}
        </header>
        {children}
      </div>
    </WorkspaceShell>
  );
}

export function AdminError({ message }: { message: string | null }) {
  return message ? <p className="command-error" role="alert">{message}</p> : null;
}

export function DisabledAdminAction({ label, reason }: { label: string; reason: string }) {
  return (
    <span className="admin-disabled-action">
      <button aria-label={`${label} unavailable: ${reason}`} disabled title={reason} type="button">{label}</button>
      <small>{reason}</small>
    </span>
  );
}

export function AdminGuardrails() {
  return (
    <div className="admin-guardrails" role="note" aria-label="Regulatory library guardrails">
      <span>Candidate-only regulatory library</span><span>Public source baseline captured locally</span><span>Not a legal decision</span><span>No autonomous publication</span>
    </div>
  );
}
