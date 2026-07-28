package administration

import (
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
)

func TestTask3LifecycleCommandContractRequiresReasonRevisionAndExactActions(t *testing.T) {
	t.Parallel()

	actions := []UserLifecycleAction{
		UserLifecycleUpdateRoles,
		UserLifecycleSuspend,
		UserLifecycleReactivate,
		UserLifecycleDeactivate,
		UserLifecycleTransferOrganization,
		UserLifecycleResendInvitation,
		UserLifecycleResetPassword,
		UserLifecycleResetMFA,
		UserLifecycleForceLogout,
	}
	for _, action := range actions {
		var effectiveAt *time.Time
		if action == UserLifecycleTransferOrganization {
			value := time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)
			effectiveAt = &value
		}
		command := RequestUserLifecycleCommand{
			OperationID:                "operation-" + string(action),
			IdempotencyKey:             "idempotency-" + string(action),
			SubjectID:                  "provider-subject-001",
			Action:                     action,
			Roles:                      []identity.Role{identity.RoleAuditee},
			OrganizationID:             "airline-xyz",
			Reason:                     "Approved identity lifecycle decision.",
			ExpectedMembershipRevision: 4,
			EffectiveAt:                effectiveAt,
		}
		if err := validateLifecycleCommand(command); err != nil {
			t.Errorf("action %s rejected: %v", action, err)
		}
	}

	provision := RequestUserLifecycleCommand{
		OperationID:    "operation-provision",
		IdempotencyKey: "idempotency-provision",
		Action:         UserLifecycleProvision,
		Roles:          []identity.Role{identity.RoleAuditee},
		OrganizationID: "airline-xyz",
		Email:          "new.auditee@example.test",
		DisplayName:    "New Auditee",
		Reason:         "Approved new membership.",
	}
	if err := validateLifecycleCommand(provision); err != nil {
		t.Fatalf("revision-zero provisioning rejected: %v", err)
	}

	for name, mutate := range map[string]func(*RequestUserLifecycleCommand){
		"missing reason": func(command *RequestUserLifecycleCommand) {
			command.Reason = ""
		},
		"negative revision": func(command *RequestUserLifecycleCommand) {
			command.ExpectedMembershipRevision = -1
		},
		"multiple roles": func(command *RequestUserLifecycleCommand) {
			command.Roles = []identity.Role{
				identity.RoleInspector,
				identity.RoleLeadInspector,
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			invalid := provision
			mutate(&invalid)
			if err := validateLifecycleCommand(invalid); err == nil {
				t.Fatalf("invalid lifecycle command was accepted: %+v", invalid)
			}
		})
	}
}
