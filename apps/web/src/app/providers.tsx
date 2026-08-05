import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createContext, type ComponentType, type PropsWithChildren, type ReactElement, useContext, useState } from "react";

import type { Backend, Role } from "../backend/backend";
import type { IdentityMode, SessionClient } from "../auth/session-client";
import type { IndexedDbFieldRepository } from "../offline/field-repository";
import type { InspectionAttachmentStore } from "../offline/opfs-inspection-attachment-store";

export type BuildProfile = "demo" | "http";

export interface ApplicationRuntime {
  backend: Backend;
  backendForRole?: (role: Role) => Backend;
  buildProfile: BuildProfile;
  environmentLabel: string;
  now?: () => Date;
  identityMode?: IdentityMode;
  sessionClient?: SessionClient;
  subjectId?: string;
  beforeSubjectChange?: (reason: "LOGOUT" | "USER_SWITCH") => Promise<void>;
  fieldRepositoryForSubject?: (subjectId: string) => IndexedDbFieldRepository;
  inspectionAttachmentStoreForSubject?: (subjectId: string) => InspectionAttachmentStore;
  /** HTTP-only route extension; the memory-mock entry never imports it. */
  supplementalRouteElements?: readonly ReactElement[];
  /** HTTP-only capability-gated navigation extension. */
  supplementalNavigation?: ComponentType<{ activeRole: Role; onNavigate?: () => void }>;
}

const ApplicationRuntimeContext = createContext<ApplicationRuntime | null>(null);

export function AppProviders({
  runtime,
  children,
}: PropsWithChildren<{ runtime: ApplicationRuntime }>) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: { retry: false, staleTime: Number.POSITIVE_INFINITY },
          mutations: { retry: false },
        },
      }),
  );

  return (
    <QueryClientProvider client={queryClient}>
      <ApplicationRuntimeProvider runtime={runtime}>{children}</ApplicationRuntimeProvider>
    </QueryClientProvider>
  );
}

export function ApplicationRuntimeProvider({
  runtime,
  children,
}: PropsWithChildren<{ runtime: ApplicationRuntime }>) {
  return (
    <ApplicationRuntimeContext.Provider value={runtime}>
      {children}
    </ApplicationRuntimeContext.Provider>
  );
}

export function useApplicationRuntime(): ApplicationRuntime {
  const runtime = useContext(ApplicationRuntimeContext);
  if (!runtime) throw new Error("Application runtime is unavailable");
  return runtime;
}

export function useOptionalApplicationRuntime(): ApplicationRuntime | null {
  return useContext(ApplicationRuntimeContext);
}

export function useBackendForRole(role: Role): Backend {
  const runtime = useApplicationRuntime();
  return runtime.backendForRole?.(role) ?? runtime.backend;
}
