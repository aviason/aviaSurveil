package agademoworkspace

import (
	"context"
	"strings"
	"testing"
)

func TestWorkspaceLoaderReconcilesAndSeals(t *testing.T) {
	store := NewMemoryStore()
	if _, err := store.LoadAndSeal(context.Background(), LoadInput{}); err == nil {
		t.Fatal("invalid loader input was accepted")
	}
}

func TestWorkspaceLoaderPersistsBothPassProjections(t *testing.T) {
	if len(WorkspaceSchemaObjectNames()) == 0 || WorkspaceSchemaName == "" {
		t.Fatal("workspace schema contract is empty")
	}
	if WorkspaceLoaderRole == WorkspaceReaderRole {
		t.Fatal("loader and reader roles must be distinct")
	}
}

func TestPostgresRecommendationSnapshotRequiresCommandStore(t *testing.T) {
	store := &PostgresStore{}
	if _, _, err := store.PutRecommendationSnapshot(context.Background(), RecommendationSnapshot{}); err == nil {
		t.Fatal("reader-only PostgreSQL store accepted recommendation snapshot write")
	}
}

func TestRecommendationSnapshotRelationIsAppendOnly(t *testing.T) {
	ddl := WorkspaceAppendOnlyTriggerDDL()
	if !strings.Contains(ddl, "recommendation_snapshots_append_only") {
		t.Fatal("recommendation snapshot relation is missing its append-only trigger")
	}
}
