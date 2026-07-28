package integration_test

import (
	"context"
	"fmt"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/notifications"
)

type lifecycleIdentityProvider struct {
	provisionedSubjectID string
	provisioned          identity.KeycloakUser
	provisionError       error
	reconciledSubjectID  string
	reconcileMatched     bool
	reconciled           identity.KeycloakUser
	disabledSubjects     []string
	executeActions       []lifecycleExecuteActions
	resetMFASubjects     []string
	loggedOutSubjects    []string
	updatedAuthorities   []string
	enabledSubjects      []string
}

type lifecycleExecuteActions struct {
	subjectID       string
	actions         []string
	lifespanSeconds int
}

func (provider *lifecycleIdentityProvider) ProvisionUser(
	_ context.Context,
	user identity.KeycloakUser,
) (string, error) {
	provider.provisioned = user
	if provider.provisionError != nil {
		return "", provider.provisionError
	}
	return provider.provisionedSubjectID, nil
}

func (provider *lifecycleIdentityProvider) ReconcileProvisionedUser(
	_ context.Context,
	user identity.KeycloakUser,
) (string, bool, error) {
	provider.reconciled = user
	return provider.reconciledSubjectID, provider.reconcileMatched, nil
}

func (provider *lifecycleIdentityProvider) UpdateUserAuthority(
	_ context.Context,
	subjectID string,
	organizationID string,
	roles []identity.Role,
) error {
	provider.updatedAuthorities = append(
		provider.updatedAuthorities,
		fmt.Sprintf("%s:%s:%v", subjectID, organizationID, roles),
	)
	return nil
}

func (provider *lifecycleIdentityProvider) DisableUser(
	_ context.Context,
	subjectID string,
) error {
	provider.disabledSubjects = append(provider.disabledSubjects, subjectID)
	return nil
}

func (provider *lifecycleIdentityProvider) EnableUser(
	_ context.Context,
	subjectID string,
) error {
	provider.enabledSubjects = append(provider.enabledSubjects, subjectID)
	return nil
}

func (provider *lifecycleIdentityProvider) IssueExecuteActionsEmail(
	_ context.Context,
	subjectID string,
	actions []string,
	lifespanSeconds int,
) error {
	provider.executeActions = append(provider.executeActions, lifecycleExecuteActions{
		subjectID:       subjectID,
		actions:         append([]string(nil), actions...),
		lifespanSeconds: lifespanSeconds,
	})
	return nil
}

func (provider *lifecycleIdentityProvider) ResetUserMFA(
	_ context.Context,
	subjectID string,
) error {
	provider.resetMFASubjects = append(provider.resetMFASubjects, subjectID)
	return nil
}

func (provider *lifecycleIdentityProvider) ForceUserLogout(
	_ context.Context,
	subjectID string,
) error {
	provider.loggedOutSubjects = append(provider.loggedOutSubjects, subjectID)
	return nil
}

type notificationDeliveryAdapterFunc func(
	context.Context,
	notifications.EmailDelivery,
) error

func (adapter notificationDeliveryAdapterFunc) Deliver(
	ctx context.Context,
	delivery notifications.EmailDelivery,
) error {
	return adapter(ctx, delivery)
}

func scenarioIDGenerator() func(string) string {
	counts := map[string]int{}
	return func(prefix string) string {
		counts[prefix]++
		return fmt.Sprintf("%s-scenario-%03d", prefix, counts[prefix])
	}
}
