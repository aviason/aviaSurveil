import type {
  Backend,
  BackendPrincipal,
  DemoBackend,
  GovernedRequiredOwnerView,
  GovernedValidationIssue,
  Role,
} from "../backend/backend";
export { DEMO_CAPABILITY_PERMISSION_MATRIX, type DemoCapabilityName } from "../backend/backend-contracts";
import { DEMO_MOCK_STORAGE_KEY } from "../app/demo-persistence";
import { MemoryMockStore } from "./memory-mock-store";
import { MockBackendEngine } from "./mock-engine";
import type { PriorAuditRecommendationProfile } from "./prior-audit-recommendations";

const defaultPrincipal: BackendPrincipal = {
  subjectId: "USR-INSPECTOR-AMINA",
  role: "inspector",
  organizationId: null,
};

export interface CreateMockBackendOptions {
  store?: MemoryMockStore;
  principal?: BackendPrincipal;
  clock?: () => string;
  governedRequiredOwners?: GovernedRequiredOwnerView[];
  governedBlockingIssues?: GovernedValidationIssue[];
  priorAuditProfile?: PriorAuditRecommendationProfile;
}

export function createMockBackend(options: CreateMockBackendOptions = {}): DemoBackend {
  const clock = options.clock ?? (() => "2026-06-15T09:00:00.000Z");
  const store = options.store ?? MemoryMockStore.createCanonical({ clock });
  return new MockBackendEngine(
    store,
    options.principal ?? defaultPrincipal,
    options.governedRequiredOwners,
    options.governedBlockingIssues,
    options.priorAuditProfile,
  );
}

export const DEMO_PRINCIPALS: Record<Role, BackendPrincipal> = {
  inspector: defaultPrincipal,
  leadInspector: {
    subjectId: "USR-LEAD-CANER",
    role: "leadInspector",
    organizationId: null,
  },
  manager: {
    subjectId: "USR-MANAGER-NORA",
    role: "manager",
    organizationId: null,
  },
  gm: { subjectId: "USR-GM-OMAR", role: "gm", organizationId: null },
  finance: { subjectId: "USR-FINANCE-LINA", role: "finance", organizationId: null },
  executiveDirector: {
    subjectId: "USR-ED-ZARA",
    role: "executiveDirector",
    organizationId: null,
  },
  auditee: {
    subjectId: "USR-AUDITEE-FLY",
    role: "auditee",
    organizationId: "ORG-FLY-NAMIBIA",
  },
  admin: { subjectId: "USR-ADMIN-ADA", role: "admin", organizationId: null },
};


function createRuntimeFromStore(store: MemoryMockStore, priorAuditProfile?: PriorAuditRecommendationProfile) {
  const sessions = new Map<Role, DemoBackend>();
  const backendForRole = (role: Role): DemoBackend => {
    const existing = sessions.get(role);
    if (existing) return existing;
    const backend = createMockBackend({ store, principal: DEMO_PRINCIPALS[role], priorAuditProfile });
    sessions.set(role, backend);
    return backend;
  };
  return {
    backend: backendForRole("inspector"),
    backendForRole,
    materializeSyntheticGovernedPackageForTest() {
      const manager = backendForRole("manager");
      if (!(manager instanceof MockBackendEngine)) {
        throw new Error("Synthetic governed test-profile materializer is unavailable.");
      }
      return manager.materializeSyntheticGovernedPackageForTest();
    },
  };
}

export function createMockBackendRuntime(clock = () => "2026-06-15T09:00:00.000Z", priorAuditProfile?: PriorAuditRecommendationProfile) {
  const store = MemoryMockStore.createCanonical({ clock });
  return createRuntimeFromStore(store, priorAuditProfile);
}

export function createMockBackendPersistentRuntime(
  storage: Storage,
  clock = () => "2026-06-15T09:00:00.000Z",
  priorAuditProfile?: PriorAuditRecommendationProfile,
) {
  return createRuntimeFromStore(MemoryMockStore.createPersistent({
    clock,
    storage,
    storageKey: DEMO_MOCK_STORAGE_KEY,
  }), priorAuditProfile);
}
