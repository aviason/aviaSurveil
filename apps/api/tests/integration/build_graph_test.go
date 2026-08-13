package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAPIModuleBuildGraph(t *testing.T) {
	moduleRoot := apiModuleRoot(t)

	command := exec.Command("go", "list", "-f", "{{with .Module}}{{.Path}}{{end}}|{{.ImportPath}}", "./cmd/api", "./cmd/worker")
	command.Dir = moduleRoot
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go list commands: %v\n%s", err, output)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if !strings.HasPrefix(line, "github.com/MarlonJD/aviaSurveil360/apps/api|") {
			t.Fatalf("command escaped the API module: %q", line)
		}
	}

	build := exec.Command("go", "build", "./cmd/api", "./cmd/worker")
	build.Dir = moduleRoot
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build commands: %v\n%s", err, output)
	}
}

func TestAPIModuleHasOneGoModuleAndCheckedGenerationInputs(t *testing.T) {
	moduleRoot := apiModuleRoot(t)
	var modules []string
	err := filepath.WalkDir(moduleRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Name() == "go.mod" {
			modules = append(modules, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk API module: %v", err)
	}
	if len(modules) != 1 || modules[0] != filepath.Join(moduleRoot, "go.mod") {
		t.Fatalf("go.mod files = %v", modules)
	}
	moduleContents, err := os.ReadFile(filepath.Join(moduleRoot, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	if !strings.Contains(string(moduleContents), "tool github.com/sqlc-dev/sqlc/cmd/sqlc") {
		t.Error("go.mod does not lock the sqlc generation tool")
	}

	for _, relativePath := range []string{
		"sqlc.yaml",
		"migrations/000001_foundation.up.sql",
		"migrations/000002_workflow_foundation.up.sql",
		"migrations/000003_authority_foundation.up.sql",
		"migrations/000004_evidence_upload_foundation.up.sql",
		"migrations/000005_sync_foundation.up.sql",
		"migrations/000006_first_production_routes.up.sql",
		"migrations/000007_full_workflow_projection.up.sql",
		"migrations/000008_communications_documents.up.sql",
		"migrations/000009_notifications_risk_admin.up.sql",
		"migrations/000010_identity_settings.up.sql",
		"migrations/000011_preliminary_report_versions.up.sql",
		"internal/httpapi/generated/api.gen.go",
		"internal/platform/session/session.go",
		"internal/platform/idempotency/idempotency.go",
		"internal/platform/auditevent/audit_event.go",
		"internal/platform/outbox/outbox.go",
		"internal/organizations/store/postgres/queries.sql.go",
		"internal/inspections/store/postgres/queries.sql.go",
		"internal/planning/store/postgres/queries.sql.go",
		"internal/configuration/store/postgres/queries.sql.go",
		"internal/assignments/store/postgres/queries.sql.go",
		"internal/communications/store/postgres/queries.sql.go",
		"internal/documents/store/postgres/queries.sql.go",
		"internal/notifications/store/postgres/queries.sql.go",
		"internal/risk/store/postgres/queries.sql.go",
		"internal/administration/store/postgres/queries.sql.go",
	} {
		if _, err := os.Stat(filepath.Join(moduleRoot, relativePath)); err != nil {
			t.Errorf("required generated/build input %s: %v", relativePath, err)
		}
	}
}

func TestLocalPostgreSQLProfileIsPinnedAndSelfCleaning(t *testing.T) {
	repositoryRoot := filepath.Clean(filepath.Join(apiModuleRoot(t), "..", ".."))
	composePath := filepath.Join(repositoryRoot, "deploy", "local", "compose.test.yaml")
	composeContents, err := os.ReadFile(composePath)
	if err != nil {
		t.Fatalf("read local compose profile: %v", err)
	}
	compose := string(composeContents)
	for _, required := range []string{
		"postgres:", "object-store:", "MINIO_API_CORS_ALLOW_ORIGIN: http://127.0.0.1:4174",
		"healthcheck:", "volumes:", "sha256:",
	} {
		if !strings.Contains(compose, required) {
			t.Errorf("compose profile does not contain %q", required)
		}
	}
	if strings.Contains(compose, ":latest") {
		t.Error("compose profile uses an unpinned latest tag")
	}

	scriptContents, err := os.ReadFile(filepath.Join(repositoryRoot, "scripts", "test-http-profile.sh"))
	if err != nil {
		t.Fatalf("read HTTP profile script: %v", err)
	}
	script := string(scriptContents)
	for _, required := range []string{
		"down --volumes", "trap cleanup EXIT", "go -C", "check-contracts.sh", "check-sqlc.sh",
		"GOTMPDIR", "SHARED_GO_CACHE", "seed_task_go_cache", "cp -al", "API_RACE_PACKAGES",
		"test-agaapplicability-race-shards.sh",
		"build -tags canonicaltest -o \"${RUNTIME_DIRECTORY}/api\" ./cmd/api", "build -o \"${RUNTIME_DIRECTORY}/worker\" ./cmd/worker",
		"exec \"${RUNTIME_DIRECTORY}/api\"", "exec \"${RUNTIME_DIRECTORY}/worker\"",
		"test:contract:http", "test:e2e:mock", "test:e2e:http",
	} {
		if !strings.Contains(script, required) {
			t.Errorf("HTTP profile script does not contain %q", required)
		}
	}
}

func apiModuleRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve integration test path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}
