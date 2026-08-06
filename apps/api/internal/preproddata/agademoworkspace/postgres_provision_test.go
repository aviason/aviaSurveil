package agademoworkspace

import (
	"strings"
	"testing"
)

func TestWorkspaceProvisionDDLContainsClosedFamilies(t *testing.T) {
	for _, relation := range WorkspaceSchemaObjectNames() {
		if !strings.Contains(WorkspaceSchemaDDL, "preprod_aga_demo_workspace."+relation) {
			t.Fatalf("DDL does not contain workspace relation %s", relation)
		}
	}
	for _, forbidden := range []string{"INSERT INTO Provider", "INSERT INTO Identity", "preprod_aga_demo.", "public."} {
		if strings.Contains(WorkspaceSchemaDDL, forbidden) {
			t.Fatalf("workspace DDL crosses boundary with %q", forbidden)
		}
	}
	if !strings.Contains(WorkspaceAppendOnlyTriggerDDL(), "question_versions_append_only") {
		t.Fatal("append-only question version trigger missing")
	}
	if !strings.Contains(WorkspaceAppendOnlyTriggerDDL(), "credential_revocation_receipts_append_only") {
		t.Fatal("credential revocation receipt trigger missing")
	}
	if !strings.Contains(WorkspaceAppendOnlyTriggerDDL(), "batch_preview_consumptions_append_only") {
		t.Fatal("batch preview consumption trigger missing")
	}
	if !strings.Contains(WorkspaceSchemaDDL, "credential_revocation_receipts") {
		t.Fatal("credential revocation receipt relation missing")
	}
	if !strings.Contains(WorkspaceSchemaDDL, "batch_preview_consumptions") {
		t.Fatal("batch preview consumption relation missing")
	}
}
