package scenarios

import "testing"

func TestOIDCQualificationAcceptsTheExactAccountAcrossSyntheticLifecycleDrift(
	t *testing.T,
) {
	account := ProviderAccount{
		SubjectID:      "synthetic-subject-auditee",
		MembershipID:   "synthetic-membership-auditee",
		OrganizationID: "synthetic-organization-auditee",
		Role:           "auditee",
	}

	organizationID, roles, err := qualificationAuthorityFor(
		account,
		2,
		account.SubjectID,
		"CAA",
		[]string{"admin"},
		"DEACTIVATED",
	)
	if err != nil {
		t.Fatalf(
			"exact provider-bound account was rejected because its frozen lifecycle revision drifted: %v",
			err,
		)
	}
	if organizationID != "synthetic-organization-auditee" ||
		len(roles) != 1 || roles[0] != "auditee" {
		t.Fatalf(
			"qualification authority = organization %q roles %v",
			organizationID,
			roles,
		)
	}
}
