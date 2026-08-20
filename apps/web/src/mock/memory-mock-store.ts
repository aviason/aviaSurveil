import { canonicalJson, cloneValue, OperationIdReuseError } from "../backend/backend-contracts";
import { createCanonicalSeedState, createFullScreenScenarioSeedState, type MockState } from "./seed-data";

interface StoredOperation {
  payload: string;
  response: unknown;
}

interface PersistedMockStore {
  schemaVersion: 5;
  state: MockState;
  operations: [string, StoredOperation][];
}

interface PersistedMockStoreCandidate {
  schemaVersion?: number;
  state?: Partial<MockState>;
  operations?: [string, StoredOperation][];
}

const CURRENT_MOCK_STORE_SCHEMA_VERSION = 5;

function hydratePersistedState(state: Partial<MockState>, clock: () => string): MockState {
  const canonical = createCanonicalSeedState(clock());
  const adminWorkspace = state.adminWorkspace
    ? {
        ...state.adminWorkspace,
        regulatoryReferences: state.adminWorkspace.regulatoryReferences.map((reference) => {
          const canonicalReference = canonical.adminWorkspace.regulatoryReferences.find(
            (candidate) => candidate.id === reference.id,
          );
          const persistedReference = reference as typeof reference & {
            mappings?: typeof reference.mappings;
          };
          return {
            ...reference,
            mappings: persistedReference.mappings?.map((mapping) => {
              const canonicalMapping = canonicalReference?.mappings.find(
                (candidate) => candidate.id === mapping.id,
              );
              const persistedMapping = mapping as typeof mapping & {
                refreshPolicy?: typeof mapping.refreshPolicy;
                scopeRecommendation?: typeof mapping.scopeRecommendation;
              };
              return {
                ...mapping,
                refreshPolicy: persistedMapping.refreshPolicy
                  ?? canonicalMapping?.refreshPolicy
                  ?? canonical.adminWorkspace.regulatoryReferences[0]!.mappings[0]!.refreshPolicy,
                scopeRecommendation: persistedMapping.scopeRecommendation
                  ?? canonicalMapping?.scopeRecommendation
                  ?? canonical.adminWorkspace.regulatoryReferences[0]!.mappings[0]!.scopeRecommendation,
              };
            }) ?? canonicalReference?.mappings ?? [],
          };
        }),
      }
    : canonical.adminWorkspace;
  return {
    ...canonical,
    ...state,
    planningProposalDrafts: state.planningProposalDrafts ?? {},
    planningAuditPackageSetups: state.planningAuditPackageSetups ?? {},
    adminWorkspace,
    reportPublicMetadata: {
      ...canonical.reportPublicMetadata,
      ...(state.reportPublicMetadata ?? {}),
    },
    auditeeCoordinationResponses: state.auditeeCoordinationResponses ?? {},
    authorizedSyncChanges: state.authorizedSyncChanges ?? [],
  };
}

export class MemoryMockStore {
  private state: MockState;
  private readonly operations: Map<string, StoredOperation>;
  private persist: (() => void) | null = null;

  private constructor(
    state: MockState,
    readonly clock: () => string,
    operations: Iterable<[string, StoredOperation]> = [],
  ) {
    this.state = cloneValue(state);
    this.operations = new Map(operations);
  }

  static createCanonical({ clock }: { clock: () => string }): MemoryMockStore {
    return new MemoryMockStore(createCanonicalSeedState(clock()), clock);
  }

  static createFullScreenScenario({ clock }: { clock: () => string }): MemoryMockStore {
    return new MemoryMockStore(createFullScreenScenarioSeedState(clock()), clock);
  }

  static createPersistent({
    clock,
    storage,
    storageKey,
  }: {
    clock: () => string;
    storage: Storage;
    storageKey: string;
  }): MemoryMockStore {
    let persisted: PersistedMockStore | null = null;
    let migrated = false;
    try {
      const raw = storage.getItem(storageKey);
      const value = raw ? JSON.parse(raw) as PersistedMockStoreCandidate : null;
      if (
        value !== null &&
        ([1, 2, 3, 4, CURRENT_MOCK_STORE_SCHEMA_VERSION].includes(value.schemaVersion ?? -1)) &&
        value.state &&
        typeof value.state === "object" &&
        Array.isArray(value.operations)
      ) {
        migrated = value.schemaVersion !== CURRENT_MOCK_STORE_SCHEMA_VERSION
          || !value.state.reportPublicMetadata
          || !value.state.auditeeCoordinationResponses
          || !value.state.adminWorkspace;
        persisted = {
          schemaVersion: CURRENT_MOCK_STORE_SCHEMA_VERSION,
          state: hydratePersistedState(value.state, clock),
          operations: value.schemaVersion !== 1
            ? value.operations
            : [],
        };
      } else if (raw) {
        storage.removeItem(storageKey);
      }
    } catch {
      storage.removeItem(storageKey);
    }
    const store = persisted
      ? new MemoryMockStore(persisted.state, clock, persisted.operations)
      : MemoryMockStore.createCanonical({ clock });
    store.persist = () => {
      const snapshot: PersistedMockStore = {
        schemaVersion: CURRENT_MOCK_STORE_SCHEMA_VERSION,
        state: cloneValue(store.state),
        operations: [...store.operations.entries()].map(([key, operation]) => [key, cloneValue(operation)]),
      };
      storage.setItem(storageKey, JSON.stringify(snapshot));
    };
    if (migrated) store.persist();
    return store;
  }

  read<T>(reader: (state: Readonly<MockState>) => T): T {
    return cloneValue(reader(this.state));
  }

  execute<T>(
    operationId: string,
    semanticPayload: unknown,
    mutation: (draft: MockState) => T,
  ): T {
    const payload = canonicalJson(semanticPayload);
    const prior = this.operations.get(operationId);
    if (prior) {
      if (prior.payload !== payload) throw new OperationIdReuseError(operationId);
      return cloneValue(prior.response as T);
    }

    const draft = cloneValue(this.state);
    const response = mutation(draft);
    this.state = draft;
    this.operations.set(operationId, { payload, response: cloneValue(response) });
    this.persist?.();
    return cloneValue(response);
  }
}
