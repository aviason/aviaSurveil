package agademoworkspace

import (
	"strings"
	"testing"
)

func TestWorkspaceRoleMatrixIsClosed(t *testing.T) {
	if err := ValidateWorkspaceRoleMatrix(WorkspaceRoleMatrix()); err != nil {
		t.Fatal(err)
	}
	ddl := WorkspaceRuntimeGrantDDL()
	for _, forbidden := range []string{"GRANT ALL", "preprod_normal_api", "preprod_aga_demo_reader", "preprod_aga_demo_writer", "CREATE TABLE"} {
		if strings.Contains(ddl, forbidden) {
			t.Fatalf("workspace runtime DDL contains forbidden token %q", forbidden)
		}
	}
	if !strings.Contains(ddl, "EXECUTE ON FUNCTION") || !strings.Contains(ddl, WorkspaceCommandRole) {
		t.Fatal("command EXECUTE boundary is missing")
	}
}
