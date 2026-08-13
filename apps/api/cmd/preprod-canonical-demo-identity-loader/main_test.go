package main

import (
	"testing"
)

func TestValidateFixtureRequiresExactRoleAndAuthorityMatrix(t *testing.T) {
	fixture := identityFixture{SchemaVersion: fixtureSchema}
	roles := []struct {
		role, organization string
	}{
		{"admin", "CAA"}, {"auditee", "ORG-FLY-NAMIBIA"},
		{"auditee", "ORG-FLY-NAMIBIA"}, {"executiveDirector", "CAA"},
		{"finance", "CAA"}, {"gm", "CAA"}, {"inspector", "CAA"},
		{"leadInspector", "CAA"}, {"manager", "CAA"},
	}
	for index, item := range roles {
		user := fixtureUser{
			ScenarioID:     "synthetic-fixture-user-" + string(rune('a'+index)),
			MembershipID:   "CANONICAL-DEMO-MEMBERSHIP-USER-" + string(rune('A'+index)),
			Email:          "fixture-user-" + string(rune('a'+index)) + "@synthetic.invalid",
			DisplayName:    "Fixture User",
			OrganizationID: item.organization,
			Role:           item.role,
		}
		if item.role == "manager" {
			user.DepartmentMembership = &departmentMembership{
				ID:                   "CANONICAL-DEMO-DEPARTMENT-MANAGER",
				DepartmentID:         "FLIGHT_OPERATIONS_INSPECTORATE",
				OrganizationalUnitID: "FLIGHT_OPERATIONS_INSPECTORATE",
			}
		}
		fixture.Users = append(fixture.Users, user)
	}
	if err := validateFixture(fixture); err != nil {
		t.Fatalf("valid fixture rejected: %v", err)
	}
	fixture.Users[0].OrganizationID = "ORG-FLY-NAMIBIA"
	if err := validateFixture(fixture); err == nil {
		t.Fatal("admin/provider authority drift was accepted")
	}
}

func TestValidateIssuerRequiresExactFirstPartyPath(t *testing.T) {
	want := "https://fixture.trycloudflare.com/identity"
	if got, err := validateIssuer(want); err != nil || got != want {
		t.Fatalf("valid issuer rejected: got=%q err=%v", got, err)
	}
	for _, candidate := range []string{
		"https://fixture.trycloudflare.com",
		"https://fixture.trycloudflare.com/identity/realms/other",
		"https://fixture.trycloudflare.com/identity?x=1",
		"http://fixture.trycloudflare.com/identity",
	} {
		if _, err := validateIssuer(candidate); err == nil {
			t.Fatalf("invalid issuer accepted: %s", candidate)
		}
	}
	if got, err := validateIssuer("http://localhost:8445/identity"); err != nil || got != "http://localhost:8445/identity" {
		t.Fatalf("loopback HTTP issuer rejected: got=%q err=%v", got, err)
	}
}
