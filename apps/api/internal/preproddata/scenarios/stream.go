package scenarios

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/profiles"
)

const recordsPerBatch int64 = 128

var familyOrder = []string{
	"organizations",
	"providerAccounts",
	"desiredMembershipVersions",
	"applicationProfiles",
	"invitations",
	"recoveryRequests",
	"mfaEnrollments",
	"sessions",
	"surveillancePlans",
	"planningApprovals",
	"audits",
	"assignments",
	"checklistTemplates",
	"checklistTemplateVersions",
	"checklistQuestions",
	"inspectionPackages",
	"checklistResponses",
	"potentialFindings",
	"findings",
	"capRevisions",
	"evidenceReferences",
	"objects",
	"objectVersions",
	"evidenceVersions",
	"reviewDecisions",
	"reportVersions",
	"communications",
	"notifications",
	"outboxMessages",
	"deliveryJobs",
	"scannerJobs",
	"renderJobs",
	"calendarRecords",
	"offlineGrants",
	"auditEvents",
	"syncChanges",
	"routeDispositions",
	"visibleActionDispositions",
	"identityLifecycleCases",
	"lifecycleScenarioCases",
}

var roles = []string{
	"inspector",
	"leadInspector",
	"manager",
	"finance",
	"gm",
	"executiveDirector",
	"auditee",
	"admin",
}

var lifecycleScenarios = []string{
	"planned",
	"active",
	"overdue",
	"returned",
	"rejected",
	"corrected",
	"superseded",
	"reopened",
	"partially-closed",
	"not-closed",
	"authorized-closed",
	"verified-closed",
}

var domainCoverage = []string{
	"planning",
	"finance-review",
	"gm-review",
	"executive-review",
	"assignment",
	"checklist",
	"potential-finding",
	"finding",
	"cap",
	"evidence",
	"report",
	"communication",
	"notification",
	"calendar",
	"closure",
	"reopen",
	"correction",
	"supersession",
}

var identityCases = []string{
	"requested",
	"invited",
	"activated",
	"suspended",
	"deactivated",
	"reactivated",
	"transferred",
	"role-changed",
	"mfa-reset",
	"forced-logout",
	"expired-invitation",
	"provider-unavailable",
	"provider-drift",
}

var objectStates = []string{
	"clean",
	"rejected",
	"expired",
	"delayed",
	"retrying",
	"unavailable",
}

var syncCases = []string{
	"offline-checkout",
	"causal-sync",
	"stale-revision",
	"duplicate-replay",
	"recovery-re-entry",
}

var privacySurfaces = []string{
	"list",
	"direct-id",
	"search",
	"filter",
	"count",
	"pagination",
	"cache",
	"report-pdf",
	"download",
	"notification",
	"calendar",
	"offline-sync",
	"raw-wire",
	"logs",
	"evidence",
}

type Stream struct {
	profile      profiles.Profile
	generator    *preproddata.Generator
	catalog      Catalog
	coverage     Coverage
	familyIndex  int
	recordIndex  int64
	commandIndex int64
}

func NewStream(
	profile profiles.Profile,
	seed []byte,
	catalog Catalog,
) (*Stream, error) {
	if err := profiles.ValidateFrozen(profile); err != nil {
		return nil, err
	}
	if err := validateCatalog(profile, catalog); err != nil {
		return nil, err
	}
	generator, err := preproddata.NewGenerator(profile, seed)
	if err != nil {
		return nil, err
	}
	stream := &Stream{
		profile:   profile,
		generator: generator,
		catalog:   cloneCatalog(catalog),
	}
	stream.coverage = buildCoverage(generator, catalog)
	return stream, nil
}

func (stream *Stream) Coverage() Coverage {
	return cloneCoverage(stream.coverage)
}

func (stream *Stream) Next(
	_ context.Context,
) (preproddata.AuthoritativeCommand, error) {
	for stream.familyIndex < len(familyOrder) {
		family := familyOrder[stream.familyIndex]
		count := stream.profile.ExpectedCounts[family]
		if stream.recordIndex >= count {
			stream.familyIndex++
			stream.recordIndex = 0
			continue
		}
		start := stream.recordIndex
		end := min(start+recordsPerBatch, count)
		records := make([]Record, 0, end-start)
		for index := start; index < end; index++ {
			records = append(records, stream.record(family, index))
		}
		payload, err := json.Marshal(Batch{
			SchemaVersion: batchSchemaVersion,
			Family:        family,
			Records:       records,
		})
		if err != nil {
			return preproddata.AuthoritativeCommand{}, err
		}
		command := preproddata.AuthoritativeCommand{
			Family:      family,
			OperationID: stream.operationID(family, start),
			Payload:     payload,
		}
		stream.recordIndex = end
		stream.commandIndex++
		return command, nil
	}
	return preproddata.AuthoritativeCommand{}, io.EOF
}

func (stream *Stream) ResumeAfter(
	_ context.Context,
	appliedCommands int64,
	lastOperationID string,
) error {
	if appliedCommands < 0 || appliedCommands > stream.totalCommands() {
		return fmt.Errorf("resume position is outside the scenario stream")
	}
	if appliedCommands == 0 {
		if lastOperationID != "" {
			return fmt.Errorf("zero resume position has an operation ID")
		}
		stream.familyIndex = 0
		stream.recordIndex = 0
		stream.commandIndex = 0
		return nil
	}

	remaining := appliedCommands
	var expectedLast string
	for familyIndex, family := range familyOrder {
		count := stream.profile.ExpectedCounts[family]
		batches := (count + recordsPerBatch - 1) / recordsPerBatch
		if remaining > batches {
			if batches > 0 {
				expectedLast = stream.operationID(
					family,
					(batches-1)*recordsPerBatch,
				)
			}
			remaining -= batches
			continue
		}
		if remaining > 0 {
			expectedLast = stream.operationID(
				family,
				(remaining-1)*recordsPerBatch,
			)
		}
		recordIndex := remaining * recordsPerBatch
		if recordIndex >= count {
			stream.familyIndex = familyIndex + 1
			stream.recordIndex = 0
		} else {
			stream.familyIndex = familyIndex
			stream.recordIndex = recordIndex
		}
		stream.commandIndex = appliedCommands
		if expectedLast != lastOperationID {
			return fmt.Errorf(
				"resume operation does not match the scenario stream",
			)
		}
		return nil
	}
	if expectedLast != lastOperationID {
		return fmt.Errorf(
			"resume operation does not match the scenario stream",
		)
	}
	stream.familyIndex = len(familyOrder)
	stream.recordIndex = 0
	stream.commandIndex = appliedCommands
	return nil
}

func (stream *Stream) totalCommands() int64 {
	var total int64
	for _, family := range familyOrder {
		count := stream.profile.ExpectedCounts[family]
		total += (count + recordsPerBatch - 1) / recordsPerBatch
	}
	return total
}

func (stream *Stream) operationID(family string, start int64) string {
	return stream.generator.ID("operation-"+family, start)
}

func (stream *Stream) record(family string, index int64) Record {
	recordID := stream.recordID(family, index)
	entityIndex, revision, predecessor := stream.versionIdentity(
		family,
		index,
	)
	businessKey := recordID
	if revision > 1 || predecessor != "" || isVersionedFamily(family) {
		businessKey = stream.generator.ID(family+"-business", entityIndex)
	}
	distribution := stream.distribution(family, index)
	effectiveAt := stream.generator.Instant(index * 2)
	knownAt := stream.generator.Instant(index*2 + 1)
	organizationID := stream.organizationID(index)
	actorMembershipID := stream.membershipID(index)
	record := Record{
		Family:            family,
		RecordID:          recordID,
		BusinessKey:       businessKey,
		Revision:          revision,
		PredecessorID:     predecessor,
		Distribution:      distribution,
		EffectiveAt:       effectiveAt,
		KnownAt:           knownAt,
		ActorMembershipID: actorMembershipID,
		OrganizationID:    organizationID,
		DecisionReason:    stream.generator.Text(family+" decision reason", index),
		Attributes: map[string]any{
			"synthetic":      true,
			"profile":        stream.profile.Name,
			"profileVersion": stream.profile.Version,
		},
	}
	record.RelationshipTuple = stream.decorate(
		&record,
		entityIndex,
		index,
	)
	return record
}

func (stream *Stream) decorate(
	record *Record,
	entityIndex, index int64,
) []string {
	family := record.Family
	state := record.Distribution
	organizationID := record.OrganizationID
	membershipID := stream.membershipID(index)
	subjectID := stream.subjectID(index)
	role := roles[index%int64(len(roles))]
	auditID := stream.linkedID("audits", index)
	responseID := stream.linkedID("checklistResponses", index)
	findingID := stream.linkedID("findings", index)
	capRevisionID := stream.linkedID("capRevisions", index)
	evidenceID := stream.linkedID("evidenceReferences", index)
	objectID := stream.linkedID("objects", entityIndex)
	objectVersionID := stream.linkedID("objectVersions", index)
	reportVersionID := stream.linkedID("reportVersions", index)
	notificationID := stream.linkedID("notifications", index)
	outboxID := stream.linkedID("outboxMessages", index)
	sessionID := stream.linkedID("sessions", index)

	switch family {
	case "organizations":
		organizationType := "auditee"
		if index == 0 {
			organizationType = "caa"
		}
		record.OrganizationID = record.RecordID
		record.Attributes["organizationType"] = organizationType
		record.Attributes["legalName"] = stream.generator.Text(
			"organization",
			index,
		)
		return []string{record.RecordID, organizationType}
	case "providerAccounts":
		observedState := "enabled"
		record.OrganizationID = stream.organizationForRole(state, index)
		record.Attributes["providerSubjectId"] = record.RecordID
		record.Attributes["membershipId"] = membershipID
		record.Attributes["observedState"] = observedState
		record.Attributes["role"] = state
		record.Attributes["email"] = stream.generator.SyntheticEmail(
			"user",
			index,
		)
		return []string{record.RecordID, membershipID, observedState}
	case "desiredMembershipVersions":
		record.OrganizationID = stream.organizationForRole(role, index)
		record.Attributes["membershipId"] = membershipID
		record.Attributes["subjectId"] = subjectID
		record.Attributes["roles"] = []string{role}
		record.Attributes["state"] = state
		record.Attributes["providerState"] = "in-sync"
		return []string{
			membershipID,
			strconv.FormatInt(record.Revision, 10),
			record.OrganizationID,
			role,
			state,
		}
	case "applicationProfiles":
		record.OrganizationID = stream.organizationForRole(role, index)
		record.Attributes["membershipId"] = membershipID
		record.Attributes["displayName"] = stream.generator.Text(
			"application profile",
			index,
		)
		return []string{record.RecordID, membershipID, record.OrganizationID}
	case "invitations":
		deliveryID := stream.generator.ID("invitation-delivery", index)
		record.Attributes["membershipId"] = membershipID
		record.Attributes["deliveryId"] = deliveryID
		record.Attributes["state"] = state
		record.Attributes["requiredActions"] = []string{
			"UPDATE_PASSWORD",
			"VERIFY_EMAIL",
		}
		return []string{record.RecordID, membershipID, deliveryID, state}
	case "recoveryRequests":
		record.Attributes["membershipId"] = membershipID
		record.Attributes["state"] = state
		return []string{
			record.RecordID,
			membershipID,
			record.EffectiveAt.Format(time.RFC3339Nano),
			state,
		}
	case "mfaEnrollments":
		observedState := state
		record.Attributes["membershipId"] = membershipID
		record.Attributes["providerSubjectId"] = subjectID
		record.Attributes["observedState"] = observedState
		return []string{membershipID, subjectID, observedState}
	case "sessions":
		sessionState := []string{
			"active",
			"revoked",
			"expired",
			"denied-stale-authority",
		}[index%4]
		record.Attributes["membershipId"] = membershipID
		record.Attributes["membershipRevision"] = int64(1)
		record.Attributes["state"] = sessionState
		return []string{record.RecordID, membershipID, "1", sessionState}
	case "offlineGrants":
		packageID := stream.linkedID("inspectionPackages", index)
		record.Attributes["sessionId"] = sessionID
		record.Attributes["packageId"] = packageID
		record.Attributes["membershipRevision"] = int64(1)
		record.Attributes["syncCase"] = syncCases[index%int64(len(syncCases))]
		expiresAt := record.EffectiveAt.Add(8 * time.Hour)
		return []string{
			record.RecordID,
			sessionID,
			"1",
			expiresAt.Format(time.RFC3339Nano),
		}
	case "surveillancePlans":
		record.Attributes["state"] = state
		return []string{
			record.RecordID,
			organizationID,
			strconv.FormatInt(record.Revision, 10),
			state,
		}
	case "planningApprovals":
		planningID := stream.linkedID("surveillancePlans", index)
		record.Attributes["planningItemId"] = planningID
		record.Attributes["actorRole"] = role
		return []string{
			record.RecordID,
			planningID,
			role,
			strconv.FormatInt(record.Revision, 10),
		}
	case "audits":
		planningID := stream.linkedID("surveillancePlans", index)
		record.Attributes["planningItemId"] = planningID
		record.Attributes["state"] = state
		return []string{
			record.RecordID,
			planningID,
			organizationID,
			strconv.FormatInt(record.Revision, 10),
		}
	case "assignments":
		assignmentQuestionID := stream.linkedID(
			"checklistQuestions",
			index/stream.profile.ExpectedCounts["audits"],
		)
		record.Attributes["auditId"] = auditID
		record.Attributes["membershipId"] = membershipID
		record.Attributes["questionId"] = assignmentQuestionID
		return []string{
			record.RecordID,
			auditID,
			membershipID,
			assignmentQuestionID,
		}
	case "checklistTemplates":
		record.OrganizationID = stream.organizationID(0)
		return []string{record.RecordID, record.OrganizationID}
	case "checklistTemplateVersions":
		templateID := stream.linkedID("checklistTemplates", entityIndex)
		record.Attributes["templateId"] = templateID
		return []string{
			record.RecordID,
			templateID,
			strconv.FormatInt(record.Revision, 10),
			record.PredecessorID,
		}
	case "checklistQuestions":
		templateVersionID := stream.linkedID(
			"checklistTemplateVersions",
			index,
		)
		sectionID := stream.generator.ID("checklist-section", index%4)
		record.Attributes["templateVersionId"] = templateVersionID
		record.Attributes["sectionId"] = sectionID
		return []string{record.RecordID, templateVersionID, sectionID}
	case "inspectionPackages":
		templateVersionID := stream.linkedID(
			"checklistTemplateVersions",
			index,
		)
		record.Attributes["auditId"] = auditID
		record.Attributes["templateVersionId"] = templateVersionID
		return []string{record.RecordID, auditID, templateVersionID}
	case "checklistResponses":
		responseQuestionID := stream.linkedID(
			"checklistQuestions",
			index/stream.profile.ExpectedCounts["audits"],
		)
		record.Attributes["auditId"] = auditID
		record.Attributes["questionId"] = responseQuestionID
		record.Attributes["membershipId"] = membershipID
		return []string{
			record.RecordID,
			auditID,
			responseQuestionID,
			membershipID,
			strconv.FormatInt(record.Revision, 10),
		}
	case "potentialFindings":
		record.Attributes["responseId"] = responseID
		record.Attributes["auditId"] = auditID
		record.Attributes["state"] = state
		return []string{record.RecordID, responseID, auditID, state}
	case "findings":
		potentialFindingID := stream.linkedID("potentialFindings", index)
		record.Attributes["potentialFindingId"] = potentialFindingID
		record.Attributes["auditId"] = auditID
		record.Attributes["state"] = state
		return []string{
			record.RecordID,
			potentialFindingID,
			auditID,
			organizationID,
			state,
		}
	case "capRevisions":
		record.Attributes["findingId"] = findingID
		record.Attributes["state"] = state
		return []string{
			record.RecordID,
			findingID,
			strconv.FormatInt(record.Revision, 10),
			record.PredecessorID,
			state,
		}
	case "evidenceReferences":
		record.Attributes["findingId"] = findingID
		record.Attributes["capRevisionId"] = capRevisionID
		return []string{record.RecordID, findingID, capRevisionID}
	case "evidenceVersions":
		record.Attributes["evidenceId"] = evidenceID
		record.Attributes["objectVersionId"] = objectVersionID
		record.Attributes["state"] = state
		return []string{
			record.RecordID,
			evidenceID,
			strconv.FormatInt(record.Revision, 10),
			objectVersionID,
			state,
		}
	case "reviewDecisions":
		decision := []string{"accepted", "returned", "rejected"}[index%3]
		record.Attributes["recordId"] = findingID
		record.Attributes["decision"] = decision
		return []string{
			record.RecordID,
			findingID,
			"1",
			membershipID,
			decision,
		}
	case "reportVersions":
		record.Attributes["auditId"] = auditID
		record.Attributes["state"] = state
		return []string{
			record.RecordID,
			auditID,
			strconv.FormatInt(record.Revision, 10),
			record.PredecessorID,
			state,
		}
	case "communications":
		recipientMembershipID := stream.membershipID(index + 1)
		record.Attributes["senderMembershipId"] = membershipID
		record.Attributes["recipientMembershipId"] = recipientMembershipID
		record.Attributes["visibility"] = []string{
			"auditee-visible",
			"caa-private",
		}[index%2]
		return []string{
			record.RecordID,
			organizationID,
			membershipID,
			recipientMembershipID,
		}
	case "notifications":
		eventID := stream.linkedID("auditEvents", index)
		record.Attributes["recipientMembershipId"] = membershipID
		record.Attributes["eventId"] = eventID
		return []string{
			record.RecordID,
			organizationID,
			membershipID,
			eventID,
		}
	case "auditEvents":
		entityType := domainCoverage[index%int64(len(domainCoverage))]
		entityID := stream.generator.ID(entityType, index)
		record.Attributes["entityType"] = entityType
		record.Attributes["entityId"] = entityID
		return []string{
			record.RecordID,
			entityType,
			entityID,
			"1",
			membershipID,
		}
	case "outboxMessages":
		aggregateType := domainCoverage[index%int64(len(domainCoverage))]
		aggregateID := stream.generator.ID(aggregateType, index)
		record.Attributes["aggregateType"] = aggregateType
		record.Attributes["aggregateId"] = aggregateID
		return []string{
			record.RecordID,
			aggregateType,
			aggregateID,
			"1",
		}
	case "deliveryJobs":
		jobState := []string{
			"delivered",
			"delayed",
			"retrying",
			"unavailable",
		}[index%4]
		record.Attributes["outboxId"] = outboxID
		record.Attributes["notificationId"] = notificationID
		record.Attributes["state"] = jobState
		return []string{
			record.RecordID,
			outboxID,
			notificationID,
			jobState,
		}
	case "scannerJobs":
		processingState := objectStates[index%int64(len(objectStates))]
		record.Attributes["objectVersionId"] = objectVersionID
		record.Attributes["processingState"] = processingState
		record.Attributes["binaryIncluded"] = false
		return []string{record.RecordID, objectVersionID, processingState}
	case "renderJobs":
		jobState := []string{
			"succeeded",
			"delayed",
			"retrying",
			"unavailable",
		}[index%4]
		record.Attributes["reportVersionId"] = reportVersionID
		record.Attributes["objectVersionId"] = objectVersionID
		record.Attributes["state"] = jobState
		return []string{
			record.RecordID,
			reportVersionID,
			objectVersionID,
			jobState,
		}
	case "objects":
		record.Attributes["recordType"] = "evidence"
		record.Attributes["recordId"] = evidenceID
		return []string{
			record.RecordID,
			organizationID,
			"evidence",
			evidenceID,
		}
	case "objectVersions":
		record.Attributes["objectId"] = objectID
		record.Attributes["binaryIncluded"] = false
		record.Attributes["payloadOrdinal"] = index + 1
		content, contentDigest := safeSyntheticObjectContent(
			*record,
			objectID,
			objectPayloadSize(stream.profile, index),
		)
		record.Attributes["contentDigest"] = contentDigest
		record.Attributes["sizeBytes"] = int64(len(content))
		return []string{
			record.RecordID,
			objectID,
			strconv.FormatInt(record.Revision, 10),
			contentDigest,
		}
	case "calendarRecords":
		recordType := []string{
			"audit",
			"finding-due",
			"cap-due",
			"report-review",
		}[index%4]
		entityID := stream.generator.ID(recordType, index)
		record.Attributes["recordType"] = recordType
		record.Attributes["recordId"] = entityID
		return []string{
			record.RecordID,
			organizationID,
			recordType,
			entityID,
		}
	case "syncChanges":
		syncCase := syncCases[index%int64(len(syncCases))]
		entityType := []string{
			"checklist-response",
			"potential-finding",
			"attachment",
		}[index%3]
		entityID := stream.generator.ID(entityType, index)
		record.Attributes["syncCase"] = syncCase
		record.Attributes["entityType"] = entityType
		record.Attributes["entityId"] = entityID
		record.Attributes["causalPredecessorId"] = stream.generator.ID(
			"sync-causal-predecessor",
			max(index-1, 0),
		)
		return []string{
			record.RecordID,
			membershipID,
			entityType,
			entityID,
			"1",
		}
	case "routeDispositions":
		if index < int64(len(privacySurfaces)*3) {
			switch index % 3 {
			case 0:
				record.OrganizationID = stream.recordID("organizations", 1)
			case 1:
				record.OrganizationID = stream.recordID("organizations", 2)
			case 2:
				record.OrganizationID = stream.recordID("organizations", 0)
			}
		}
		route := stream.catalog.Routes[index]
		actorRole := route.Role
		if state == "denied" {
			actorRole = roles[(slicesIndex(roles, route.Role)+1)%len(roles)]
		}
		scenarioID := stream.generator.ID(
			"lifecycle-scenario",
			index%int64(len(lifecycleScenarios)),
		)
		record.Attributes["surfaceId"] = route.SurfaceID
		record.Attributes["screenName"] = route.ScreenName
		record.Attributes["ownerRole"] = route.Role
		record.Attributes["actorRole"] = actorRole
		record.Attributes["disposition"] = state
		record.Attributes["scenarioId"] = scenarioID
		record.Attributes["stateAssertion"] = explicitStateAssertion(state)
		if state == "authorized-data" {
			backingFamily := routeBackingFamily(route.SurfaceID)
			record.Attributes["backingFamily"] = backingFamily
			record.Attributes["backingRecordId"] = stream.linkedID(
				backingFamily,
				index,
			)
		}
		if index == 0 {
			record.Attributes["privacyMatrix"] = stream.coverage.Privacy
		}
		return []string{
			route.AuditID,
			actorRole,
			state,
			scenarioID,
		}
	case "visibleActionDispositions":
		action := stream.catalog.Actions[index]
		route := stream.routeForSurface(action.SurfaceID)
		record.Attributes["actionId"] = action.ActionID
		record.Attributes["surfaceId"] = action.SurfaceID
		record.Attributes["controlKey"] = action.ControlKey
		record.Attributes["boundary"] = action.Boundary
		record.Attributes["role"] = route.Role
		record.Attributes["disposition"] = state
		record.Attributes["stateAssertion"] = explicitStateAssertion(state)
		return []string{
			route.AuditID,
			action.ActionID,
			route.Role,
			state,
		}
	case "identityLifecycleCases":
		caseKind := identityCaseKind(state, index)
		providerState := "enabled"
		sessionState := "active"
		switch caseKind {
		case "suspended", "deactivated", "forced-logout", "mfa-reset":
			sessionState = "revoked"
		case "provider-unavailable":
			providerState = "unavailable"
			sessionState = "denied-stale-authority"
		case "provider-drift":
			providerState = "drifted"
			sessionState = "denied-stale-authority"
		case "requested", "invited", "expired-invitation":
			sessionState = "none"
		}
		record.Attributes["scenarioId"] = record.RecordID
		record.Attributes["membershipId"] = membershipID
		record.Attributes["caseKind"] = caseKind
		record.Attributes["providerState"] = providerState
		record.Attributes["sessionState"] = sessionState
		return []string{
			record.RecordID,
			membershipID,
			state,
			providerState,
			sessionState,
		}
	case "lifecycleScenarioCases":
		recordType := domainCoverage[index%int64(len(domainCoverage))]
		entityID := stream.generator.ID(recordType, index)
		record.Attributes["scenarioId"] = record.RecordID
		record.Attributes["recordType"] = recordType
		record.Attributes["recordId"] = entityID
		record.Attributes["lifecycleState"] = state
		record.Attributes["coveredDomainStates"] = domainCoverage
		return []string{record.RecordID, recordType, entityID, state}
	}
	panic("unsupported scenario family " + family)
}

func (stream *Stream) distribution(family string, index int64) string {
	distribution, ok := stream.profile.ExactDistributions[family]
	if !ok {
		return "generated"
	}
	keys := make([]string, 0, len(distribution))
	for key := range distribution {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var cursor int64
	for _, key := range keys {
		cursor += distribution[key]
		if index < cursor {
			return key
		}
	}
	panic("distribution index outside frozen count")
}

func (stream *Stream) versionIdentity(
	family string,
	index int64,
) (entityIndex, revision int64, predecessor string) {
	base := stream.versionBase(family)
	if base < 1 {
		return index, 1, ""
	}
	entityIndex = index % base
	revision = index/base + 1
	if revision > 1 {
		predecessor = stream.recordID(family, index-base)
	}
	return entityIndex, revision, predecessor
}

func (stream *Stream) versionBase(family string) int64 {
	switch family {
	case "desiredMembershipVersions":
		return stream.profile.ExpectedCounts["providerAccounts"]
	case "checklistTemplateVersions":
		return stream.profile.ExpectedCounts["checklistTemplates"]
	case "checklistResponses":
		return max(
			1,
			stream.profile.ExpectedCounts["checklistResponses"]*2/3,
		)
	case "capRevisions":
		return stream.profile.ExpectedCounts["findings"]
	case "evidenceVersions":
		return stream.profile.ExpectedCounts["evidenceReferences"]
	case "reportVersions":
		return stream.profile.ExpectedCounts["audits"]
	case "objectVersions":
		return stream.profile.ExpectedCounts["objects"]
	default:
		return 0
	}
}

func objectPayloadSize(
	profile profiles.Profile,
	index int64,
) int64 {
	if profile.Name != "stress" {
		return 0
	}
	count := profile.ExpectedCounts["objectVersions"]
	if index < 0 || index >= count || count < 1 {
		return 0
	}
	base := profile.ResourceEnvelope.ObjectBytes / count
	remainder := profile.ResourceEnvelope.ObjectBytes % count
	if index < remainder {
		return base + 1
	}
	return base
}

func (stream *Stream) recordID(family string, index int64) string {
	if family == "organizations" && index == 0 {
		return "CAA"
	}
	return stream.generator.ID(family, index)
}

func (stream *Stream) linkedID(family string, index int64) string {
	count := stream.profile.ExpectedCounts[family]
	if count < 1 {
		panic("linked scenario family has no records: " + family)
	}
	return stream.recordID(family, index%count)
}

func (stream *Stream) organizationID(index int64) string {
	count := stream.profile.ExpectedCounts["organizations"]
	if count <= 1 {
		return stream.recordID("organizations", 0)
	}
	return stream.recordID("organizations", 1+index%(count-1))
}

func (stream *Stream) organizationForRole(role string, index int64) string {
	if role != "auditee" {
		return stream.recordID("organizations", 0)
	}
	return stream.organizationID(index)
}

func (stream *Stream) membershipID(index int64) string {
	count := stream.profile.ExpectedCounts["providerAccounts"]
	return stream.generator.ID("membership", index%count)
}

func (stream *Stream) subjectID(index int64) string {
	return stream.linkedID("providerAccounts", index)
}

func (stream *Stream) routeForSurface(surfaceID string) RouteCoverage {
	for _, route := range stream.catalog.Routes {
		if route.SurfaceID == surfaceID {
			return route
		}
	}
	panic("action references unknown route " + surfaceID)
}

func validateCatalog(profile profiles.Profile, catalog Catalog) error {
	if profile.Catalogs.RouteCount != 86 ||
		len(catalog.Routes) != profile.Catalogs.RouteCount ||
		len(catalog.Actions) != 306 {
		return fmt.Errorf("scenario catalogs do not match the frozen profile")
	}
	seenRoles := make(map[string]bool)
	for _, route := range catalog.Routes {
		seenRoles[route.Role] = true
	}
	for _, role := range profile.Catalogs.Roles {
		if !seenRoles[role] {
			return fmt.Errorf("scenario route catalog omits role %s", role)
		}
	}
	for _, family := range familyOrder {
		if _, ok := profile.ExpectedCounts[family]; !ok {
			return fmt.Errorf("scenario stream omits frozen family %s", family)
		}
	}
	if len(profile.ExpectedCounts) != len(familyOrder) {
		return fmt.Errorf("scenario stream family catalog is incomplete")
	}
	return nil
}

func buildCoverage(
	generator *preproddata.Generator,
	catalog Catalog,
) Coverage {
	organizationA := generator.ID("organizations", 1)
	organizationB := generator.ID("organizations", 2)
	caaOrganization := "CAA"
	var privacy []PrivacyAssertion
	for surfaceIndex, surface := range privacySurfaces {
		canaryOffsets := []int{0, 1, 2}
		for canaryIndex, assertion := range []PrivacyAssertion{
			{
				CanaryClass:          "auditee-a-from-b",
				SourceOrganizationID: organizationA,
				ActorOrganizationID:  organizationB,
			},
			{
				CanaryClass:          "auditee-b-from-a",
				SourceOrganizationID: organizationB,
				ActorOrganizationID:  organizationA,
			},
			{
				CanaryClass:          "caa-private-from-auditee",
				SourceOrganizationID: caaOrganization,
				ActorOrganizationID:  organizationA,
			},
		} {
			assertion.Surface = surface
			assertion.RecordCanary = generator.ID(
				"routeDispositions",
				int64(surfaceIndex*3+canaryOffsets[canaryIndex]),
			)
			assertion.ExpectedResult = "denied-no-exposure"
			privacy = append(privacy, assertion)
		}
	}
	return Coverage{
		Routes:  cloneCatalog(catalog).Routes,
		Actions: cloneCatalog(catalog).Actions,
		Roles:   append([]string(nil), roles...),
		LifecycleScenarios: append(
			[]string(nil),
			lifecycleScenarios...,
		),
		DomainCoverage: append([]string(nil), domainCoverage...),
		IdentityCases:  append([]string(nil), identityCases...),
		ObjectStates:   append([]string(nil), objectStates...),
		SyncCases:      append([]string(nil), syncCases...),
		Privacy:        privacy,
	}
}

func cloneCatalog(source Catalog) Catalog {
	return Catalog{
		Routes:  append([]RouteCoverage(nil), source.Routes...),
		Actions: append([]ActionCoverage(nil), source.Actions...),
	}
}

func cloneCoverage(source Coverage) Coverage {
	return Coverage{
		Routes:  append([]RouteCoverage(nil), source.Routes...),
		Actions: append([]ActionCoverage(nil), source.Actions...),
		Roles:   append([]string(nil), source.Roles...),
		LifecycleScenarios: append(
			[]string(nil),
			source.LifecycleScenarios...,
		),
		DomainCoverage: append([]string(nil), source.DomainCoverage...),
		IdentityCases:  append([]string(nil), source.IdentityCases...),
		ObjectStates:   append([]string(nil), source.ObjectStates...),
		SyncCases:      append([]string(nil), source.SyncCases...),
		Privacy:        append([]PrivacyAssertion(nil), source.Privacy...),
	}
}

func isVersionedFamily(family string) bool {
	switch family {
	case "desiredMembershipVersions",
		"checklistTemplateVersions",
		"checklistResponses",
		"capRevisions",
		"evidenceVersions",
		"reportVersions",
		"objectVersions":
		return true
	default:
		return false
	}
}

func identityCaseKind(state string, index int64) string {
	switch state {
	case "active":
		if index%2 == 0 {
			return "activated"
		}
		return "reactivated"
	case "invitation-expired":
		return "expired-invitation"
	case "reactivation-pending", "recovered":
		return "reactivated"
	default:
		return state
	}
}

func explicitStateAssertion(disposition string) string {
	switch disposition {
	case "authorized-data", "executable":
		return "meaningful-authorized-data"
	case "intentional-empty":
		return "intentional-empty-state"
	case "denied", "disabled-by-role":
		return "exact-authority-denial"
	case "disabled-by-state":
		return "exact-lifecycle-state-denial"
	default:
		return "explicit-" + disposition
	}
}

func routeBackingFamily(surfaceID string) string {
	switch {
	case strings.Contains(surfaceID, "finding"),
		strings.Contains(surfaceID, "cap"):
		return "findings"
	case strings.Contains(surfaceID, "report"):
		return "reportVersions"
	case strings.Contains(surfaceID, "message"):
		return "communications"
	case strings.Contains(surfaceID, "calendar"):
		return "calendarRecords"
	case strings.Contains(surfaceID, "audit"),
		strings.Contains(surfaceID, "inspection"):
		return "audits"
	case strings.Contains(surfaceID, "template"),
		strings.Contains(surfaceID, "checklist"),
		strings.Contains(surfaceID, "question"):
		return "checklistTemplateVersions"
	case strings.Contains(surfaceID, "user"),
		strings.Contains(surfaceID, "profile"),
		strings.Contains(surfaceID, "setting"):
		return "applicationProfiles"
	case strings.Contains(surfaceID, "notification"):
		return "notifications"
	case strings.Contains(surfaceID, "document"),
		strings.Contains(surfaceID, "evidence"):
		return "evidenceVersions"
	case strings.Contains(surfaceID, "planning"),
		strings.Contains(surfaceID, "plan"):
		return "surveillancePlans"
	default:
		return "organizations"
	}
}

func slicesIndex(values []string, value string) int {
	for index, candidate := range values {
		if candidate == value {
			return index
		}
	}
	return 0
}
