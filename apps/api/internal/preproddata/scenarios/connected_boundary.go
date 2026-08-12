package scenarios

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/aviason/aviaSurveil/internal/preproddata"
	"github.com/aviason/aviaSurveil/internal/preproddata/profiles"
)

type ProviderAccount struct {
	ScenarioID      string
	SubjectID       string
	MembershipID    string
	Email           string
	OrganizationID  string
	Role            string
	Enabled         bool
	RequiredActions []string
}

type InvitationDelivery struct {
	InvitationID    string
	DeliveryID      string
	SubjectID       string
	Email           string
	RequiredActions []string
}

type ObjectVersion struct {
	VersionID      string
	ObjectID       string
	OrganizationID string
	Bucket         string
	Key            string
	ContentDigest  string
	SizeBytes      int64
	Content        []byte
}

type ScenarioStore interface {
	Initialize(context.Context) error
	Apply(context.Context, preproddata.AuthoritativeCommand) error
	Reconcile(context.Context) (preproddata.Reconciliation, error)
	Records(context.Context, string) ([]Record, error)
}

type resumableScenarioStore interface {
	Resume(context.Context) error
}

type scenarioRecordScanner interface {
	ScanRecords(context.Context, string, func(Record) error) error
}

type IdentityEndpoint interface {
	Preflight(context.Context) error
	EnsureProviderAccount(
		context.Context,
		ProviderAccount,
	) (ProviderAccount, error)
	ReconcileProviderAccounts(context.Context, []ProviderAccount) error
}

type resumableIdentityEndpoint interface {
	ResumePreflight(context.Context) error
}

type InvitationEndpoint interface {
	Preflight(context.Context) error
	EnsureInvitationDelivery(context.Context, InvitationDelivery) error
	ReconcileInvitationDeliveries(context.Context, []InvitationDelivery) error
}

type resumableInvitationEndpoint interface {
	ResumePreflight(context.Context) error
}

type ObjectEndpoint interface {
	Preflight(context.Context) error
	EnsureObjectVersion(context.Context, ObjectVersion) error
	ReconcileObjectVersions(context.Context, []ObjectVersion) error
}

type resumableObjectEndpoint interface {
	ResumePreflight(context.Context) error
}

type streamingObjectEndpoint interface {
	ReconcileObjectVersionStream(
		context.Context,
		func(func(ObjectVersion) error) error,
	) error
}

type ConnectedBoundaryConfig struct {
	Target      preproddata.TargetFingerprint
	Store       ScenarioStore
	Identity    IdentityEndpoint
	Invitations InvitationEndpoint
	Objects     ObjectEndpoint
}

type ConnectedBoundary struct {
	target               preproddata.TargetFingerprint
	store                ScenarioStore
	identity             IdentityEndpoint
	invitations          InvitationEndpoint
	objects              ObjectEndpoint
	subjectsByMembership map[string]string
	subjectsByScenarioID map[string]string
}

func NewConnectedBoundary(
	config ConnectedBoundaryConfig,
) (*ConnectedBoundary, error) {
	if err := config.Target.Validate(); err != nil {
		return nil, err
	}
	if config.Store == nil ||
		config.Identity == nil ||
		config.Invitations == nil ||
		config.Objects == nil {
		return nil, fmt.Errorf(
			"scenario store and all connected endpoints are required",
		)
	}
	return &ConnectedBoundary{
		target:               config.Target,
		store:                config.Store,
		identity:             config.Identity,
		invitations:          config.Invitations,
		objects:              config.Objects,
		subjectsByMembership: make(map[string]string),
		subjectsByScenarioID: make(map[string]string),
	}, nil
}

func (boundary *ConnectedBoundary) Preflight(
	ctx context.Context,
	target preproddata.TargetFingerprint,
	operation preproddata.Operation,
) error {
	if target != boundary.target {
		return fmt.Errorf("connected-scenario target differs from configuration")
	}
	switch operation {
	case preproddata.LoadEmptyTarget:
		if err := boundary.store.Initialize(ctx); err != nil {
			return err
		}
	case preproddata.ResumeRun:
		resumableStore, ok := boundary.store.(resumableScenarioStore)
		if !ok {
			return fmt.Errorf(
				"connected-scenario store does not support safe resume",
			)
		}
		if err := resumableStore.Resume(ctx); err != nil {
			return err
		}
	default:
		return fmt.Errorf(
			"connected-scenario boundary supports only load or resume",
		)
	}
	if operation == preproddata.ResumeRun {
		resumableIdentity, ok := boundary.identity.(resumableIdentityEndpoint)
		if !ok {
			return fmt.Errorf(
				"connected-scenario identity endpoint does not support safe resume",
			)
		}
		if err := resumableIdentity.ResumePreflight(ctx); err != nil {
			return err
		}
		resumableInvitations, ok := boundary.invitations.(resumableInvitationEndpoint)
		if !ok {
			return fmt.Errorf(
				"connected-scenario invitation endpoint does not support safe resume",
			)
		}
		if err := resumableInvitations.ResumePreflight(ctx); err != nil {
			return err
		}
		resumableObjects, ok := boundary.objects.(resumableObjectEndpoint)
		if !ok {
			return fmt.Errorf(
				"connected-scenario object endpoint does not support safe resume",
			)
		}
		if err := resumableObjects.ResumePreflight(ctx); err != nil {
			return err
		}
		return boundary.restoreProviderSubjects(ctx)
	}
	if err := boundary.identity.Preflight(ctx); err != nil {
		return err
	}
	if err := boundary.invitations.Preflight(ctx); err != nil {
		return err
	}
	return boundary.objects.Preflight(ctx)
}

func (boundary *ConnectedBoundary) restoreProviderSubjects(
	ctx context.Context,
) error {
	records, err := boundary.store.Records(ctx, "providerAccounts")
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return fmt.Errorf(
			"connected-scenario resume has no durable provider accounts",
		)
	}
	for _, record := range records {
		account, err := providerAccountFromRecord(record)
		if err != nil {
			return err
		}
		if err := boundary.bindProviderSubject(account); err != nil {
			return err
		}
	}
	return nil
}

func (boundary *ConnectedBoundary) Apply(
	ctx context.Context,
	command preproddata.AuthoritativeCommand,
) error {
	batch, err := DecodeBatch(command)
	if err != nil {
		return err
	}
	switch batch.Family {
	case "providerAccounts":
		for index := range batch.Records {
			account, err := providerAccountFromRecord(batch.Records[index])
			if err != nil {
				return err
			}
			if account.SubjectID != account.ScenarioID {
				return fmt.Errorf(
					"unbound provider account has unexpected provider subject",
				)
			}
			account.SubjectID = ""
			observed, err := boundary.identity.EnsureProviderAccount(
				ctx,
				account,
			)
			if err != nil {
				return err
			}
			if !sameProviderAccountBinding(account, observed) {
				return fmt.Errorf(
					"provider-assigned subject changed scenario account authority",
				)
			}
			if err := boundary.bindProviderSubject(observed); err != nil {
				return err
			}
			batch.Records[index].Attributes["providerSubjectId"] =
				observed.SubjectID
		}
		rewritten, err := commandWithBatch(command, batch)
		if err != nil {
			return err
		}
		return boundary.store.Apply(ctx, rewritten)
	default:
		if err := boundary.bindSubjectReferences(&batch); err != nil {
			return err
		}
		rewritten, err := commandWithBatch(command, batch)
		if err != nil {
			return err
		}
		if err := boundary.store.Apply(ctx, rewritten); err != nil {
			return err
		}
	}
	switch batch.Family {
	case "invitations":
		accounts, err := boundary.providerAccounts(ctx)
		if err != nil {
			return err
		}
		for _, record := range batch.Records {
			if record.Distribution != "delivered" {
				continue
			}
			delivery, err := invitationFromRecord(record, accounts)
			if err != nil {
				return err
			}
			if err := boundary.invitations.EnsureInvitationDelivery(
				ctx,
				delivery,
			); err != nil {
				return err
			}
		}
	case "objectVersions":
		for _, record := range batch.Records {
			version, err := objectVersionFromRecord(
				record,
				boundary.target,
			)
			if err != nil {
				return err
			}
			if err := boundary.objects.EnsureObjectVersion(
				ctx,
				version,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func (boundary *ConnectedBoundary) bindProviderSubject(
	account ProviderAccount,
) error {
	if !validProviderSubjectID(account.SubjectID) {
		return fmt.Errorf("provider assigned an invalid subject")
	}
	if current := boundary.subjectsByMembership[account.MembershipID]; current != "" && current != account.SubjectID {
		return fmt.Errorf("provider subject changed for retained membership")
	}
	if current := boundary.subjectsByScenarioID[account.ScenarioID]; current != "" && current != account.SubjectID {
		return fmt.Errorf("provider subject changed for scenario account")
	}
	boundary.subjectsByMembership[account.MembershipID] = account.SubjectID
	boundary.subjectsByScenarioID[account.ScenarioID] = account.SubjectID
	return nil
}

func (boundary *ConnectedBoundary) bindSubjectReferences(batch *Batch) error {
	for recordIndex := range batch.Records {
		record := &batch.Records[recordIndex]
		membershipID, _ := record.Attributes["membershipId"].(string)
		subjectID := boundary.subjectsByMembership[membershipID]
		for _, field := range []string{"subjectId", "providerSubjectId"} {
			value, exists := record.Attributes[field]
			if !exists {
				continue
			}
			if membershipID == "" || subjectID == "" {
				return fmt.Errorf(
					"scenario subject reference has no provider binding",
				)
			}
			if _, ok := value.(string); !ok {
				return fmt.Errorf("scenario subject reference is not a string")
			}
			record.Attributes[field] = subjectID
		}
		for tupleIndex, value := range record.RelationshipTuple {
			if replacement := boundary.subjectsByScenarioID[value]; replacement != "" {
				record.RelationshipTuple[tupleIndex] = replacement
			}
		}
	}
	return nil
}

func commandWithBatch(
	command preproddata.AuthoritativeCommand,
	batch Batch,
) (preproddata.AuthoritativeCommand, error) {
	payload, err := json.Marshal(batch)
	if err != nil {
		return preproddata.AuthoritativeCommand{}, err
	}
	command.Payload = payload
	return command, nil
}

func sameProviderAccountBinding(
	planned,
	observed ProviderAccount,
) bool {
	return observed.SubjectID != "" &&
		planned.ScenarioID == observed.ScenarioID &&
		planned.MembershipID == observed.MembershipID &&
		planned.Email == observed.Email &&
		planned.OrganizationID == observed.OrganizationID &&
		planned.Role == observed.Role &&
		planned.Enabled == observed.Enabled &&
		sameStrings(planned.RequiredActions, observed.RequiredActions)
}

func (boundary *ConnectedBoundary) Reconcile(
	ctx context.Context,
) (preproddata.Reconciliation, error) {
	accountsByMembership, err := boundary.providerAccounts(ctx)
	if err != nil {
		return preproddata.Reconciliation{}, err
	}
	accounts := make(
		[]ProviderAccount,
		0,
		len(accountsByMembership),
	)
	for _, account := range accountsByMembership {
		accounts = append(accounts, account)
	}
	sort.Slice(accounts, func(left, right int) bool {
		return accounts[left].SubjectID < accounts[right].SubjectID
	})
	if err := boundary.identity.ReconcileProviderAccounts(
		ctx,
		accounts,
	); err != nil {
		return preproddata.Reconciliation{}, err
	}

	invitationRecords, err := boundary.store.Records(ctx, "invitations")
	if err != nil {
		return preproddata.Reconciliation{}, err
	}
	deliveries := make([]InvitationDelivery, 0, len(invitationRecords))
	for _, record := range invitationRecords {
		if record.Distribution != "delivered" {
			continue
		}
		delivery, err := invitationFromRecord(
			record,
			accountsByMembership,
		)
		if err != nil {
			return preproddata.Reconciliation{}, err
		}
		deliveries = append(deliveries, delivery)
	}
	sort.Slice(deliveries, func(left, right int) bool {
		return deliveries[left].DeliveryID < deliveries[right].DeliveryID
	})
	if err := boundary.invitations.ReconcileInvitationDeliveries(
		ctx,
		deliveries,
	); err != nil {
		return preproddata.Reconciliation{}, err
	}

	recordScanner, recordsStream := boundary.store.(scenarioRecordScanner)
	objectStream, objectsStream := boundary.objects.(streamingObjectEndpoint)
	if recordsStream && objectsStream {
		if err := objectStream.ReconcileObjectVersionStream(
			ctx,
			func(yield func(ObjectVersion) error) error {
				return recordScanner.ScanRecords(
					ctx,
					"objectVersions",
					func(record Record) error {
						version, err := objectVersionFromRecord(
							record,
							boundary.target,
						)
						if err != nil {
							return err
						}
						return yield(version)
					},
				)
			},
		); err != nil {
			return preproddata.Reconciliation{}, err
		}
	} else {
		objectRecords, err := boundary.store.Records(ctx, "objectVersions")
		if err != nil {
			return preproddata.Reconciliation{}, err
		}
		versions := make([]ObjectVersion, 0, len(objectRecords))
		for _, record := range objectRecords {
			version, err := objectVersionFromRecord(record, boundary.target)
			if err != nil {
				return preproddata.Reconciliation{}, err
			}
			versions = append(versions, version)
		}
		sort.Slice(versions, func(left, right int) bool {
			return versions[left].VersionID < versions[right].VersionID
		})
		if err := boundary.objects.ReconcileObjectVersions(
			ctx,
			versions,
		); err != nil {
			return preproddata.Reconciliation{}, err
		}
	}
	return boundary.store.Reconcile(ctx)
}

func (boundary *ConnectedBoundary) providerAccounts(
	ctx context.Context,
) (map[string]ProviderAccount, error) {
	records, err := boundary.store.Records(ctx, "providerAccounts")
	if err != nil {
		return nil, err
	}
	accounts := make(map[string]ProviderAccount, len(records))
	for _, record := range records {
		account, err := providerAccountFromRecord(record)
		if err != nil {
			return nil, err
		}
		if _, exists := accounts[account.MembershipID]; exists {
			return nil, fmt.Errorf(
				"duplicate provider account membership %s",
				account.MembershipID,
			)
		}
		accounts[account.MembershipID] = account
	}
	return accounts, nil
}

func providerAccountFromRecord(
	record Record,
) (ProviderAccount, error) {
	subjectID, err := connectedString(record, "providerSubjectId")
	if err != nil {
		return ProviderAccount{}, err
	}
	membershipID, err := connectedString(record, "membershipId")
	if err != nil {
		return ProviderAccount{}, err
	}
	email, err := connectedString(record, "email")
	if err != nil {
		return ProviderAccount{}, err
	}
	role, err := connectedString(record, "role")
	if err != nil {
		return ProviderAccount{}, err
	}
	if !containsRole(role) {
		return ProviderAccount{}, fmt.Errorf(
			"provider account has unsupported role %q",
			role,
		)
	}
	observedState, err := connectedString(record, "observedState")
	if err != nil {
		return ProviderAccount{}, err
	}
	if observedState != "enabled" {
		return ProviderAccount{}, fmt.Errorf(
			"provider account has unsupported observed state %q",
			observedState,
		)
	}
	return ProviderAccount{
		ScenarioID:     record.RecordID,
		SubjectID:      subjectID,
		MembershipID:   membershipID,
		Email:          email,
		OrganizationID: record.OrganizationID,
		Role:           role,
		Enabled:        true,
		RequiredActions: []string{
			"UPDATE_PASSWORD",
			"VERIFY_EMAIL",
		},
	}, nil
}

func invitationFromRecord(
	record Record,
	accounts map[string]ProviderAccount,
) (InvitationDelivery, error) {
	membershipID, err := connectedString(record, "membershipId")
	if err != nil {
		return InvitationDelivery{}, err
	}
	account, ok := accounts[membershipID]
	if !ok {
		return InvitationDelivery{}, fmt.Errorf(
			"invitation references unknown provider membership %s",
			membershipID,
		)
	}
	deliveryID, err := connectedString(record, "deliveryId")
	if err != nil {
		return InvitationDelivery{}, err
	}
	requiredActions, err := connectedStrings(record, "requiredActions")
	if err != nil {
		return InvitationDelivery{}, err
	}
	return InvitationDelivery{
		InvitationID:    record.RecordID,
		DeliveryID:      deliveryID,
		SubjectID:       account.SubjectID,
		Email:           account.Email,
		RequiredActions: requiredActions,
	}, nil
}

func objectVersionFromRecord(
	record Record,
	target preproddata.TargetFingerprint,
) (ObjectVersion, error) {
	objectID, err := connectedString(record, "objectId")
	if err != nil {
		return ObjectVersion{}, err
	}
	binaryIncluded, ok := record.Attributes["binaryIncluded"].(bool)
	if !ok || binaryIncluded {
		return ObjectVersion{}, fmt.Errorf(
			"scenario object version must exclude binary fixtures",
		)
	}
	var targetSize int64
	profileName, hasProfile := record.Attributes["profile"].(string)
	profileVersion, hasVersion := record.Attributes["profileVersion"].(string)
	if hasProfile || hasVersion {
		if !hasProfile || !hasVersion {
			return ObjectVersion{}, fmt.Errorf(
				"scenario object version has incomplete profile identity",
			)
		}
		profile, err := profiles.Lookup(profileName, profileVersion)
		if err != nil {
			return ObjectVersion{}, err
		}
		ordinal, err := connectedInt64(record, "payloadOrdinal")
		if err != nil {
			return ObjectVersion{}, err
		}
		targetSize = objectPayloadSize(profile, ordinal-1)
	}
	payload, actualDigest := safeSyntheticObjectContent(
		record,
		objectID,
		targetSize,
	)
	contentDigest, err := connectedString(record, "contentDigest")
	if err != nil {
		return ObjectVersion{}, err
	}
	sizeBytes, err := connectedInt64(record, "sizeBytes")
	if err != nil {
		return ObjectVersion{}, err
	}
	if contentDigest != actualDigest || sizeBytes != int64(len(payload)) {
		return ObjectVersion{}, fmt.Errorf(
			"scenario object content metadata does not match safe JSON",
		)
	}
	return ObjectVersion{
		VersionID:      record.RecordID,
		ObjectID:       objectID,
		OrganizationID: record.OrganizationID,
		Bucket:         target.ObjectBucket,
		Key:            target.ObjectPrefix + "objects/" + record.RecordID + ".json",
		ContentDigest:  contentDigest,
		SizeBytes:      sizeBytes,
		Content:        payload,
	}, nil
}

func safeSyntheticObjectContent(
	record Record,
	objectID string,
	targetSize int64,
) ([]byte, string) {
	payload, _ := json.Marshal(struct {
		SchemaVersion  string `json:"schemaVersion"`
		Synthetic      bool   `json:"synthetic"`
		RecordID       string `json:"recordId"`
		ObjectID       string `json:"objectId"`
		OrganizationID string `json:"organizationId"`
		BinaryIncluded bool   `json:"binaryIncluded"`
	}{
		SchemaVersion:  "preprod-synthetic-object/v1",
		Synthetic:      true,
		RecordID:       record.RecordID,
		ObjectID:       objectID,
		OrganizationID: record.OrganizationID,
		BinaryIncluded: false,
	})
	if targetSize > int64(len(payload)) {
		prefix := append(
			append([]byte(nil), payload[:len(payload)-1]...),
			[]byte(`,"padding":"`)...,
		)
		suffix := []byte(`"}`)
		paddingSize := targetSize - int64(len(prefix)) - int64(len(suffix))
		if paddingSize > 0 {
			payload = append(
				prefix,
				[]byte(strings.Repeat("S", int(paddingSize)))...,
			)
			payload = append(payload, suffix...)
		}
	}
	digest := sha256.Sum256(payload)
	actualDigest := "sha256:" + hex.EncodeToString(digest[:])
	return payload, actualDigest
}

func connectedString(record Record, key string) (string, error) {
	value, ok := record.Attributes[key].(string)
	value = strings.TrimSpace(value)
	if !ok || value == "" {
		return "", fmt.Errorf(
			"scenario record %s/%s omits string attribute %s",
			record.Family,
			record.RecordID,
			key,
		)
	}
	return value, nil
}

func connectedStrings(record Record, key string) ([]string, error) {
	source, ok := record.Attributes[key].([]any)
	if !ok || len(source) == 0 {
		return nil, fmt.Errorf(
			"scenario record %s/%s omits string-list attribute %s",
			record.Family,
			record.RecordID,
			key,
		)
	}
	output := make([]string, len(source))
	seen := make(map[string]bool, len(source))
	for index, item := range source {
		value, ok := item.(string)
		value = strings.TrimSpace(value)
		if !ok || value == "" || seen[value] {
			return nil, fmt.Errorf(
				"scenario record %s/%s has invalid string-list attribute %s",
				record.Family,
				record.RecordID,
				key,
			)
		}
		seen[value] = true
		output[index] = value
	}
	return output, nil
}

func connectedInt64(record Record, key string) (int64, error) {
	switch value := record.Attributes[key].(type) {
	case int:
		if value > 0 {
			return int64(value), nil
		}
	case int64:
		if value > 0 {
			return value, nil
		}
	case float64:
		output := int64(value)
		if value == float64(output) && output > 0 {
			return output, nil
		}
	case json.Number:
		output, err := value.Int64()
		if err == nil && output > 0 {
			return output, nil
		}
	}
	return 0, fmt.Errorf(
		"scenario record %s/%s omits positive integer attribute %s",
		record.Family,
		record.RecordID,
		key,
	)
}

func containsRole(role string) bool {
	for _, allowed := range roles {
		if role == allowed {
			return true
		}
	}
	return false
}
