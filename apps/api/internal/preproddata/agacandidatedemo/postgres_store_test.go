package agacandidatedemo_test

import (
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agacandidatedemo"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestOverlayDDLIsDedicatedImmutableAndReaderOnly(t *testing.T) {
	ddl := agacandidatedemo.OverlaySchemaDDL
	for _, value := range []string{"preprod_aga_demo.package_intents", "preprod_aga_demo.packages", "preprod_aga_demo.forms", "preprod_aga_demo.questions", "preprod_aga_demo.package_seals", "OWNER TO preprod_aga_demo_owner", "reject_mutation", "reject_child_after_seal", "question_source_proposals_after_seal", "sealed_packages", "GRANT USAGE ON SCHEMA preprod_aga_demo TO preprod_aga_demo_writer", "GRANT USAGE ON SCHEMA preprod_aga_demo TO preprod_aga_demo_reader", "GRANT SELECT ON preprod_aga_demo.sealed_packages TO preprod_aga_demo_reader", "REVOKE ALL ON ALL TABLES IN SCHEMA preprod_aga_demo FROM PUBLIC, preprod_normal_api", "GRANT SELECT, INSERT ON ALL TABLES IN SCHEMA preprod_aga_demo TO preprod_aga_demo_writer"} {
		if !strings.Contains(ddl, value) {
			t.Fatalf("missing DDL boundary %q", value)
		}
	}
	for _, forbidden := range []string{"checklist_import", "existing_checklist_candidates", "findings", "audits", "outbox"} {
		if strings.Contains(strings.ToLower(ddl), forbidden) {
			t.Fatalf("forbidden governed table target %q", forbidden)
		}
	}
	for _, forbiddenGrant := range []string{"UPDATE, DELETE", "TRUNCATE", "GRANT SELECT ON preprod_aga_demo.sealed_packages TO preprod_normal_api"} {
		if strings.Contains(ddl, forbiddenGrant) {
			t.Fatalf("unsafe overlay privilege %q", forbiddenGrant)
		}
	}
}

func TestPostgresStoreMutatesOnlyDedicatedOverlaySchema(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test path")
	}
	content, err := os.ReadFile(filepath.Join(filepath.Dir(file), "postgres_store.go"))
	if err != nil {
		t.Fatal(err)
	}
	mutations := regexp.MustCompile(`(?i)INSERT INTO\s+([^\s(]+)`).FindAllStringSubmatch(string(content), -1)
	if len(mutations) == 0 {
		t.Fatal("expected dedicated overlay mutations")
	}
	for _, mutation := range mutations {
		if !strings.Contains(mutation[1], "preprod_aga_demo") {
			t.Fatalf("mutation escapes the dedicated overlay schema: %q", mutation[0])
		}
	}
	for _, forbiddenImport := range []string{"internal/checklistintake", "internal/checklistgovernance", "internal/regulatory", "internal/datafeed"} {
		if strings.Contains(string(content), forbiddenImport) {
			t.Fatalf("PostgreSQL overlay must not invoke real domain service %q", forbiddenImport)
		}
	}
}
