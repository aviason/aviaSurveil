package agacandidatedemo_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestOverlayRoleProvisioningDeniesNormalAPIAndPublic(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test path")
	}
	path := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../../deploy/local/preprod/aga-demo-role-provision.sql"))
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read role provisioner: %v", err)
	}
	text := string(content)
	for _, required := range []string{"preprod_aga_demo_owner", "preprod_aga_demo_writer", "preprod_aga_demo_reader", "preprod_normal_api", "REVOKE ALL ON SCHEMA preprod_aga_demo FROM PUBLIC, preprod_normal_api", "GRANT CONNECT ON DATABASE aviasurveil360_local_preprod TO preprod_normal_api, preprod_aga_demo_reader, preprod_aga_demo_writer", "GRANT USAGE ON SCHEMA preprod_aga_demo TO preprod_aga_demo_reader", "GRANT SELECT ON caa_department_memberships, caa_department_status_facts, caa_organizational_unit_status_facts, caa_organizational_units TO preprod_normal_api", "ALTER ROLE preprod_aga_demo_reader SET default_transaction_read_only = on"} {
		if !strings.Contains(text, required) {
			t.Fatalf("missing role boundary %q", required)
		}
	}
	if strings.Contains(text, "GRANT ALL ON SCHEMA preprod_aga_demo TO preprod_normal_api") {
		t.Fatal("normal API overlay grant is forbidden")
	}
}
