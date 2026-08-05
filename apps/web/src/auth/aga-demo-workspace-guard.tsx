import { useEffect, useState, type ReactNode } from "react";

import { useApplicationRuntime } from "../app/providers";
import type { AGADemoWorkspaceCapability } from "../backend/aga-demo-workspace";
import type { Role } from "../backend/backend";
import { RoleGuard } from "./role-guard";

type CapabilityState =
  | { kind: "checking" }
  | { kind: "available"; capability: AGADemoWorkspaceCapability }
  | { kind: "restored" }
  | { kind: "unavailable" };

export function AGADemoWorkspaceGuard({
  requiredRole,
  children,
}: {
  requiredRole: Role;
  children: (capability: AGADemoWorkspaceCapability) => ReactNode;
}) {
  return (
    <RoleGuard requiredRole={requiredRole}>
      <AGADemoWorkspaceCapabilityGate>{children}</AGADemoWorkspaceCapabilityGate>
    </RoleGuard>
  );
}

function AGADemoWorkspaceCapabilityGate({
  children,
}: {
  children: (capability: AGADemoWorkspaceCapability) => ReactNode;
}) {
  const { backend } = useApplicationRuntime();
  const capability = backend.agaDemoWorkspace;
  const [state, setState] = useState<CapabilityState>({ kind: "checking" });

  useEffect(() => {
    const controller = new AbortController();
    const invalidateRestoredPage = (event: PageTransitionEvent) => {
      if (!event.persisted) return;
      controller.abort();
      setState({ kind: "restored" });
    };
    window.addEventListener("pageshow", invalidateRestoredPage);
    if (!capability) {
      setState({ kind: "unavailable" });
      return () => {
        controller.abort();
        window.removeEventListener("pageshow", invalidateRestoredPage);
      };
    }
    setState({ kind: "checking" });
    void capability
      .capability({ signal: controller.signal })
      .then((result) => {
        if (controller.signal.aborted || !result.available) {
          setState({ kind: "unavailable" });
          return;
        }
        setState({ kind: "available", capability: result });
      })
      .catch(() => {
        if (!controller.signal.aborted) setState({ kind: "unavailable" });
      });
    return () => {
      controller.abort();
      window.removeEventListener("pageshow", invalidateRestoredPage);
    };
  }, [capability]);

  if (state.kind === "checking") return <p data-testid="aga-workspace-capability-checking" role="status">Checking supplemental workspace capability…</p>;
  if (state.kind === "restored") return <p role="alert">This restored page was cleared; reload the server-authoritative workspace to continue.</p>;
  if (state.kind === "unavailable") return <p data-testid="aga-workspace-capability-unavailable" role="status">This supplemental workspace is not available for the current session.</p>;
  return <>{children(state.capability)}</>;
}
