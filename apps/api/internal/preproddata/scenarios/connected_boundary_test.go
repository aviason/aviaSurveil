package scenarios_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/scenarios"
)

func TestConnectedBoundaryMaterializesIdentityAndInvitationStateOutsidePostgres(
	t *testing.T,
) {
	ctx := context.Background()
	target := connectedTarget()
	store := newMemoryScenarioStore()
	identity := newMemoryIdentityEndpoint()
	invitations := newMemoryInvitationEndpoint()
	objects := newMemoryObjectEndpoint()
	boundary, err := scenarios.NewConnectedBoundary(
		scenarios.ConnectedBoundaryConfig{
			Target:      target,
			Store:       store,
			Identity:    identity,
			Invitations: invitations,
			Objects:     objects,
		},
	)
	if err != nil {
		t.Fatalf("new connected boundary: %v", err)
	}
	if err := boundary.Preflight(
		ctx,
		target,
		preproddata.LoadEmptyTarget,
	); err != nil {
		t.Fatalf("preflight connected boundary: %v", err)
	}

	account := scenarioRecord("providerAccounts", "subject-001")
	account.OrganizationID = "AUDITEE-A"
	account.Attributes = map[string]any{
		"providerSubjectId": "subject-001",
		"membershipId":      "membership-001",
		"observedState":     "enabled",
		"role":              "auditee",
		"email":             "user-0001@synthetic.invalid",
	}
	if err := boundary.Apply(
		ctx,
		scenarioCommand(t, account),
	); err != nil {
		t.Fatalf("apply provider account: %v", err)
	}

	invitation := scenarioRecord("invitations", "invitation-001")
	invitation.OrganizationID = "AUDITEE-A"
	invitation.Distribution = "delivered"
	invitation.Attributes = map[string]any{
		"membershipId": "membership-001",
		"deliveryId":   "delivery-001",
		"state":        "delivered",
		"requiredActions": []string{
			"UPDATE_PASSWORD",
			"VERIFY_EMAIL",
		},
	}
	if err := boundary.Apply(
		ctx,
		scenarioCommand(t, invitation),
	); err != nil {
		t.Fatalf("apply invitation: %v", err)
	}

	wantAccount := scenarios.ProviderAccount{
		ScenarioID:     "subject-001",
		SubjectID:      "provider-subject-001",
		MembershipID:   "membership-001",
		Email:          "user-0001@synthetic.invalid",
		OrganizationID: "AUDITEE-A",
		Role:           "auditee",
		Enabled:        true,
		RequiredActions: []string{
			"UPDATE_PASSWORD",
			"VERIFY_EMAIL",
		},
	}
	if got := identity.accounts["provider-subject-001"]; !equalProviderAccount(
		got,
		wantAccount,
	) {
		t.Fatalf("external provider account = %#v, expected %#v", got, wantAccount)
	}
	wantInvitation := scenarios.InvitationDelivery{
		InvitationID: "invitation-001",
		DeliveryID:   "delivery-001",
		SubjectID:    "provider-subject-001",
		Email:        "user-0001@synthetic.invalid",
		RequiredActions: []string{
			"UPDATE_PASSWORD",
			"VERIFY_EMAIL",
		},
	}
	if got := invitations.deliveries["delivery-001"]; !equalInvitation(
		got,
		wantInvitation,
	) {
		t.Fatalf(
			"external invitation delivery = %#v, expected %#v",
			got,
			wantInvitation,
		)
	}
	storedAccount := store.records["providerAccounts"]["subject-001"]
	if got := storedAccount.Attributes["providerSubjectId"]; got !=
		"provider-subject-001" {
		t.Fatalf("stored provider subject = %#v", got)
	}
}

func TestConnectedBoundaryWritesSafeDeterministicObjectVersions(
	t *testing.T,
) {
	ctx := context.Background()
	target := connectedTarget()
	store := newMemoryScenarioStore()
	objects := newMemoryObjectEndpoint()
	boundary, err := scenarios.NewConnectedBoundary(
		scenarios.ConnectedBoundaryConfig{
			Target:      target,
			Store:       store,
			Identity:    newMemoryIdentityEndpoint(),
			Invitations: newMemoryInvitationEndpoint(),
			Objects:     objects,
		},
	)
	if err != nil {
		t.Fatalf("new connected boundary: %v", err)
	}
	if err := boundary.Preflight(
		ctx,
		target,
		preproddata.LoadEmptyTarget,
	); err != nil {
		t.Fatalf("preflight connected boundary: %v", err)
	}

	const content = `{"schemaVersion":"preprod-synthetic-object/v1","synthetic":true,"recordId":"object-version-001","objectId":"object-001","organizationId":"AUDITEE-A","binaryIncluded":false}`
	contentHash := sha256.Sum256([]byte(content))
	contentDigest := "sha256:" + hex.EncodeToString(contentHash[:])
	version := scenarioRecord("objectVersions", "object-version-001")
	version.OrganizationID = "AUDITEE-A"
	version.Attributes = map[string]any{
		"objectId":       "object-001",
		"contentDigest":  contentDigest,
		"sizeBytes":      int64(len(content)),
		"binaryIncluded": false,
	}
	if err := boundary.Apply(
		ctx,
		scenarioCommand(t, version),
	); err != nil {
		t.Fatalf("apply object version: %v", err)
	}

	want := scenarios.ObjectVersion{
		VersionID:      "object-version-001",
		ObjectID:       "object-001",
		OrganizationID: "AUDITEE-A",
		Bucket:         "aviasurveil360-local-preprod",
		Key: "runs/run-task7-connected-smoke/objects/" +
			"object-version-001.json",
		ContentDigest: contentDigest,
		SizeBytes:     int64(len(content)),
		Content:       []byte(content),
	}
	if got := objects.versions["object-version-001"]; !reflect.DeepEqual(
		got,
		want,
	) {
		t.Fatalf("external object version = %#v, expected %#v", got, want)
	}
}

func TestConnectedBoundaryResumeRestoresProviderAssignedSubjectBindings(
	t *testing.T,
) {
	ctx := context.Background()
	target := connectedTarget()
	store := newMemoryScenarioStore()
	identity := newMemoryIdentityEndpoint()
	invitations := newMemoryInvitationEndpoint()
	objects := newMemoryObjectEndpoint()
	first, err := scenarios.NewConnectedBoundary(
		scenarios.ConnectedBoundaryConfig{
			Target:      target,
			Store:       store,
			Identity:    identity,
			Invitations: invitations,
			Objects:     objects,
		},
	)
	if err != nil {
		t.Fatalf("new first connected boundary: %v", err)
	}
	if err := first.Preflight(
		ctx,
		target,
		preproddata.LoadEmptyTarget,
	); err != nil {
		t.Fatalf("first preflight: %v", err)
	}
	account := scenarioRecord("providerAccounts", "subject-001")
	account.OrganizationID = "AUDITEE-A"
	account.Attributes = map[string]any{
		"providerSubjectId": "subject-001",
		"membershipId":      "membership-001",
		"observedState":     "enabled",
		"role":              "auditee",
		"email":             "user-0001@synthetic.invalid",
	}
	if err := first.Apply(ctx, scenarioCommand(t, account)); err != nil {
		t.Fatalf("apply provider account: %v", err)
	}

	resumed, err := scenarios.NewConnectedBoundary(
		scenarios.ConnectedBoundaryConfig{
			Target:      target,
			Store:       store,
			Identity:    identity,
			Invitations: invitations,
			Objects:     objects,
		},
	)
	if err != nil {
		t.Fatalf("new resumed connected boundary: %v", err)
	}
	if err := resumed.Preflight(
		ctx,
		target,
		preproddata.ResumeRun,
	); err != nil {
		t.Fatalf("resume preflight: %v", err)
	}
	membership := scenarioRecord(
		"desiredMembershipVersions",
		"membership-version-001",
	)
	membership.OrganizationID = "AUDITEE-A"
	membership.Attributes = map[string]any{
		"membershipId": "membership-001",
		"subjectId":    "subject-001",
		"roles":        []string{"auditee"},
	}
	if err := resumed.Apply(
		ctx,
		scenarioCommand(t, membership),
	); err != nil {
		t.Fatalf("apply resumed membership: %v", err)
	}
	stored := store.records["desiredMembershipVersions"]["membership-version-001"]
	if got := stored.Attributes["subjectId"]; got != "provider-subject-001" {
		t.Fatalf("resumed provider subject = %#v", got)
	}
}

func TestConnectedBoundaryRejectsExternalIdentityReconciliationDrift(
	t *testing.T,
) {
	ctx := context.Background()
	target := connectedTarget()
	store := newMemoryScenarioStore()
	identity := newMemoryIdentityEndpoint()
	boundary, err := scenarios.NewConnectedBoundary(
		scenarios.ConnectedBoundaryConfig{
			Target:      target,
			Store:       store,
			Identity:    identity,
			Invitations: newMemoryInvitationEndpoint(),
			Objects:     newMemoryObjectEndpoint(),
		},
	)
	if err != nil {
		t.Fatalf("new connected boundary: %v", err)
	}
	if err := boundary.Preflight(
		ctx,
		target,
		preproddata.LoadEmptyTarget,
	); err != nil {
		t.Fatalf("preflight connected boundary: %v", err)
	}
	account := scenarioRecord("providerAccounts", "subject-001")
	account.OrganizationID = "AUDITEE-A"
	account.Attributes = map[string]any{
		"providerSubjectId": "subject-001",
		"membershipId":      "membership-001",
		"observedState":     "enabled",
		"role":              "auditee",
		"email":             "user-0001@synthetic.invalid",
	}
	if err := boundary.Apply(
		ctx,
		scenarioCommand(t, account),
	); err != nil {
		t.Fatalf("apply provider account: %v", err)
	}
	drifted := identity.accounts["provider-subject-001"]
	drifted.Email = "wrong@synthetic.invalid"
	identity.accounts["provider-subject-001"] = drifted

	if _, err := boundary.Reconcile(ctx); err == nil {
		t.Fatalf("external identity drift was accepted")
	}
}

func connectedTarget() preproddata.TargetFingerprint {
	return preproddata.TargetFingerprint{
		Environment:              "local-preprod",
		DatabaseName:             "aviasurveil360_local_preprod",
		DatabaseOwner:            "aviasurveil360_preprod_loader",
		PostgresSystemIdentifier: "7421987349021349876",
		PostgresHost:             "preprod-postgres",
		PostgresPort:             5432,
		ComposeProject:           "aviasurveil360-local-preprod",
		KeycloakRealm:            "aviasurveil360-local-preprod",
		KeycloakDatabase:         "keycloak_local_preprod",
		KeycloakServiceClientID:  "aviasurveil360-local-preprod-lifecycle",
		MailpitNamespace:         "aviasurveil360-local-preprod",
		ObjectBucket:             "aviasurveil360-local-preprod",
		ObjectPrefix:             "runs/run-task7-connected-smoke/",
		LoaderQueueNamespace:     "aviasurveil360-local-preprod",
		ProfileName:              "smoke",
		ProfileVersion:           "1.0.0",
		RunID:                    "run-task7-connected-smoke",
		IntentDigest:             "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
}

func scenarioRecord(family, recordID string) scenarios.Record {
	effectiveAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return scenarios.Record{
		Family:            family,
		RecordID:          recordID,
		BusinessKey:       recordID,
		Revision:          1,
		Distribution:      "generated",
		EffectiveAt:       effectiveAt,
		KnownAt:           effectiveAt.Add(time.Second),
		ActorMembershipID: "membership-001",
		OrganizationID:    "CAA",
		DecisionReason:    "SYNTHETIC TEST DECISION",
		RelationshipTuple: []string{recordID},
		Attributes:        map[string]any{"synthetic": true},
	}
}

func scenarioCommand(
	t *testing.T,
	record scenarios.Record,
) preproddata.AuthoritativeCommand {
	t.Helper()
	payload, err := json.Marshal(scenarios.Batch{
		SchemaVersion: "preprod-connected-scenario-batch/v1",
		Family:        record.Family,
		Records:       []scenarios.Record{record},
	})
	if err != nil {
		t.Fatalf("encode scenario command: %v", err)
	}
	return preproddata.AuthoritativeCommand{
		Family:      record.Family,
		OperationID: "operation-" + record.RecordID,
		Payload:     payload,
	}
}

type memoryScenarioStore struct {
	records map[string]map[string]scenarios.Record
}

func newMemoryScenarioStore() *memoryScenarioStore {
	return &memoryScenarioStore{
		records: make(map[string]map[string]scenarios.Record),
	}
}

func (store *memoryScenarioStore) Initialize(context.Context) error {
	if len(store.records) != 0 {
		return fmt.Errorf("scenario store is not empty")
	}
	return nil
}

func (store *memoryScenarioStore) Resume(context.Context) error {
	if len(store.records) == 0 {
		return fmt.Errorf("scenario store has no resumable records")
	}
	return nil
}

func (store *memoryScenarioStore) Apply(
	_ context.Context,
	command preproddata.AuthoritativeCommand,
) error {
	batch, err := scenarios.DecodeBatch(command)
	if err != nil {
		return err
	}
	if store.records[batch.Family] == nil {
		store.records[batch.Family] = make(map[string]scenarios.Record)
	}
	for _, record := range batch.Records {
		store.records[batch.Family][record.RecordID] = record
	}
	return nil
}

func (store *memoryScenarioStore) Reconcile(
	context.Context,
) (preproddata.Reconciliation, error) {
	counts := make(map[string]int64, len(store.records))
	digests := make(map[string]string, len(store.records))
	for family, records := range store.records {
		counts[family] = int64(len(records))
		digests[family] = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	}
	return preproddata.Reconciliation{
		ActualCounts:        counts,
		RelationshipDigests: digests,
	}, nil
}

func (store *memoryScenarioStore) Records(
	_ context.Context,
	family string,
) ([]scenarios.Record, error) {
	records := maps.Values(store.records[family])
	output := slices.Collect(records)
	slices.SortFunc(output, func(left, right scenarios.Record) int {
		switch {
		case left.RecordID < right.RecordID:
			return -1
		case left.RecordID > right.RecordID:
			return 1
		default:
			return 0
		}
	})
	return output, nil
}

type memoryIdentityEndpoint struct {
	accounts map[string]scenarios.ProviderAccount
}

func newMemoryIdentityEndpoint() *memoryIdentityEndpoint {
	return &memoryIdentityEndpoint{
		accounts: make(map[string]scenarios.ProviderAccount),
	}
}

func (endpoint *memoryIdentityEndpoint) Preflight(context.Context) error {
	if len(endpoint.accounts) != 0 {
		return fmt.Errorf("identity endpoint is not empty")
	}
	return nil
}

func (endpoint *memoryIdentityEndpoint) ResumePreflight(context.Context) error {
	if len(endpoint.accounts) == 0 {
		return fmt.Errorf("identity endpoint has no resumable accounts")
	}
	return nil
}

func (endpoint *memoryIdentityEndpoint) EnsureProviderAccount(
	_ context.Context,
	account scenarios.ProviderAccount,
) (scenarios.ProviderAccount, error) {
	account.SubjectID = "provider-subject-001"
	endpoint.accounts[account.SubjectID] = account
	return account, nil
}

func (endpoint *memoryIdentityEndpoint) ReconcileProviderAccounts(
	_ context.Context,
	expected []scenarios.ProviderAccount,
) error {
	if len(endpoint.accounts) != len(expected) {
		return fmt.Errorf("provider account count differs")
	}
	for _, account := range expected {
		if !equalProviderAccount(endpoint.accounts[account.SubjectID], account) {
			return fmt.Errorf("provider account %s differs", account.SubjectID)
		}
	}
	return nil
}

type memoryInvitationEndpoint struct {
	deliveries map[string]scenarios.InvitationDelivery
}

func newMemoryInvitationEndpoint() *memoryInvitationEndpoint {
	return &memoryInvitationEndpoint{
		deliveries: make(map[string]scenarios.InvitationDelivery),
	}
}

func (endpoint *memoryInvitationEndpoint) Preflight(context.Context) error {
	if len(endpoint.deliveries) != 0 {
		return fmt.Errorf("invitation endpoint is not empty")
	}
	return nil
}

func (endpoint *memoryInvitationEndpoint) ResumePreflight(context.Context) error {
	return nil
}

func (endpoint *memoryInvitationEndpoint) EnsureInvitationDelivery(
	_ context.Context,
	delivery scenarios.InvitationDelivery,
) error {
	endpoint.deliveries[delivery.DeliveryID] = delivery
	return nil
}

func (endpoint *memoryInvitationEndpoint) ReconcileInvitationDeliveries(
	_ context.Context,
	expected []scenarios.InvitationDelivery,
) error {
	if len(endpoint.deliveries) != len(expected) {
		return fmt.Errorf("invitation delivery count differs")
	}
	for _, delivery := range expected {
		if !equalInvitation(
			endpoint.deliveries[delivery.DeliveryID],
			delivery,
		) {
			return fmt.Errorf(
				"invitation delivery %s differs",
				delivery.DeliveryID,
			)
		}
	}
	return nil
}

type memoryObjectEndpoint struct {
	versions map[string]scenarios.ObjectVersion
}

func newMemoryObjectEndpoint() *memoryObjectEndpoint {
	return &memoryObjectEndpoint{
		versions: make(map[string]scenarios.ObjectVersion),
	}
}

func (endpoint *memoryObjectEndpoint) Preflight(context.Context) error {
	if len(endpoint.versions) != 0 {
		return fmt.Errorf("object endpoint is not empty")
	}
	return nil
}

func (endpoint *memoryObjectEndpoint) ResumePreflight(context.Context) error {
	return nil
}

func (endpoint *memoryObjectEndpoint) EnsureObjectVersion(
	_ context.Context,
	version scenarios.ObjectVersion,
) error {
	endpoint.versions[version.VersionID] = version
	return nil
}

func (endpoint *memoryObjectEndpoint) ReconcileObjectVersions(
	_ context.Context,
	expected []scenarios.ObjectVersion,
) error {
	if len(endpoint.versions) != len(expected) {
		return fmt.Errorf("object version count differs")
	}
	for _, version := range expected {
		if !reflect.DeepEqual(endpoint.versions[version.VersionID], version) {
			return fmt.Errorf("object version %s differs", version.VersionID)
		}
	}
	return nil
}

func equalProviderAccount(
	left,
	right scenarios.ProviderAccount,
) bool {
	leftActions := append([]string(nil), left.RequiredActions...)
	rightActions := append([]string(nil), right.RequiredActions...)
	slices.Sort(leftActions)
	slices.Sort(rightActions)
	left.RequiredActions = leftActions
	right.RequiredActions = rightActions
	return reflect.DeepEqual(left, right)
}

func equalInvitation(
	left,
	right scenarios.InvitationDelivery,
) bool {
	leftActions := append([]string(nil), left.RequiredActions...)
	rightActions := append([]string(nil), right.RequiredActions...)
	slices.Sort(leftActions)
	slices.Sort(rightActions)
	left.RequiredActions = leftActions
	right.RequiredActions = rightActions
	return reflect.DeepEqual(left, right)
}
