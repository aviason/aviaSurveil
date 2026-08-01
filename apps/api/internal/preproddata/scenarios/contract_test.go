package scenarios_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"sort"
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/profiles"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/scenarios"
)

var requiredRoles = []string{
	"inspector",
	"leadInspector",
	"manager",
	"finance",
	"gm",
	"executiveDirector",
	"auditee",
	"admin",
}

var requiredLifecycleScenarios = []string{
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

var requiredDomainCoverage = []string{
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

var requiredIdentityCases = []string{
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

var requiredObjectStates = []string{
	"clean",
	"rejected",
	"expired",
	"delayed",
	"retrying",
	"unavailable",
}

var requiredSyncCases = []string{
	"offline-checkout",
	"causal-sync",
	"stale-revision",
	"duplicate-replay",
	"recovery-re-entry",
}

var requiredPrivacySurfaces = []string{
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

func TestEveryProfileCarriesTheExactRouteActionRoleAndScenarioCatalogs(
	t *testing.T,
) {
	catalog := loadCanonicalCatalog(t)
	for _, profileName := range []string{
		"smoke",
		"acceptance",
		"realistic",
		"stress",
	} {
		profile, err := profiles.Lookup(profileName, "1.0.0")
		if err != nil {
			t.Fatalf("lookup %s: %v", profileName, err)
		}
		stream, err := scenarios.NewStream(
			profile,
			[]byte("task-7-connected-scenarios"),
			catalog,
		)
		if err != nil {
			t.Fatalf("new %s stream: %v", profileName, err)
		}
		coverage := stream.Coverage()

		if got := routeIDs(coverage.Routes); !reflect.DeepEqual(
			got,
			catalogRouteIDs(catalog),
		) {
			t.Fatalf("%s route catalog differs", profileName)
		}
		if got := actionIDs(coverage.Actions); !reflect.DeepEqual(
			got,
			catalogActionIDs(catalog),
		) {
			t.Fatalf("%s visible-action catalog differs", profileName)
		}
		assertExactSet(t, profileName+" roles", coverage.Roles, requiredRoles)
		assertExactSet(
			t,
			profileName+" lifecycle scenarios",
			coverage.LifecycleScenarios,
			requiredLifecycleScenarios,
		)
		assertExactSet(
			t,
			profileName+" domain coverage",
			coverage.DomainCoverage,
			requiredDomainCoverage,
		)
		assertExactSet(
			t,
			profileName+" identity cases",
			coverage.IdentityCases,
			requiredIdentityCases,
		)
		assertExactSet(
			t,
			profileName+" object states",
			coverage.ObjectStates,
			requiredObjectStates,
		)
		assertExactSet(
			t,
			profileName+" sync cases",
			coverage.SyncCases,
			requiredSyncCases,
		)
		assertPrivacyMatrix(t, coverage.Privacy)
	}
}

func TestStreamObjectVersionMetadataMatchesTheSafeSyntheticJSON(
	t *testing.T,
) {
	profile, err := profiles.Lookup("smoke", "1.0.0")
	if err != nil {
		t.Fatalf("lookup smoke: %v", err)
	}
	stream, err := scenarios.NewStream(
		profile,
		[]byte("task-7-connected-scenarios"),
		loadCanonicalCatalog(t),
	)
	if err != nil {
		t.Fatalf("new stream: %v", err)
	}
	for {
		command, nextErr := stream.Next(context.Background())
		if nextErr != nil {
			t.Fatalf("stream ended before object versions: %v", nextErr)
		}
		if command.Family != "objectVersions" {
			continue
		}
		batch, err := scenarios.DecodeBatch(command)
		if err != nil {
			t.Fatalf("decode object-version batch: %v", err)
		}
		for _, record := range batch.Records {
			objectID, ok := record.Attributes["objectId"].(string)
			if !ok || objectID == "" {
				t.Fatalf("object version omits objectId: %#v", record)
			}
			content := fmt.Sprintf(
				`{"schemaVersion":"preprod-synthetic-object/v1","synthetic":true,"recordId":%q,"objectId":%q,"organizationId":%q,"binaryIncluded":false}`,
				record.RecordID,
				objectID,
				record.OrganizationID,
			)
			digest := sha256.Sum256([]byte(content))
			wantDigest := "sha256:" + hex.EncodeToString(digest[:])
			if got, _ := record.Attributes["contentDigest"].(string); got != wantDigest {
				t.Fatalf(
					"%s content digest = %q, expected %q",
					record.RecordID,
					got,
					wantDigest,
				)
			}
			sizeBytes, ok := record.Attributes["sizeBytes"].(float64)
			if !ok || sizeBytes != float64(len(content)) {
				t.Fatalf(
					"%s sizeBytes = %#v, expected %d",
					record.RecordID,
					record.Attributes["sizeBytes"],
					len(content),
				)
			}
			if included, ok := record.Attributes["binaryIncluded"].(bool); !ok ||
				included {
				t.Fatalf(
					"%s binaryIncluded = %#v",
					record.RecordID,
					record.Attributes["binaryIncluded"],
				)
			}
		}
		return
	}
}

func TestPrivacyCanariesReferenceExistingCrossOrganizationRecords(
	t *testing.T,
) {
	profile, err := profiles.Lookup("smoke", "1.0.0")
	if err != nil {
		t.Fatalf("lookup smoke: %v", err)
	}
	stream, err := scenarios.NewStream(
		profile,
		[]byte("task-7-connected-scenarios"),
		loadCanonicalCatalog(t),
	)
	if err != nil {
		t.Fatalf("new stream: %v", err)
	}
	privacy := stream.Coverage().Privacy
	routeRecords := make(map[string]scenarios.Record)
	for {
		command, nextErr := stream.Next(context.Background())
		if nextErr != nil {
			t.Fatalf("stream ended before route dispositions: %v", nextErr)
		}
		if command.Family != "routeDispositions" {
			continue
		}
		batch, err := scenarios.DecodeBatch(command)
		if err != nil {
			t.Fatalf("decode route dispositions: %v", err)
		}
		for _, record := range batch.Records {
			routeRecords[record.RecordID] = record
		}
		break
	}
	for _, assertion := range privacy {
		record, ok := routeRecords[assertion.RecordCanary]
		if !ok {
			t.Fatalf(
				"%s/%s canary %s does not exist",
				assertion.Surface,
				assertion.CanaryClass,
				assertion.RecordCanary,
			)
		}
		if record.OrganizationID != assertion.SourceOrganizationID {
			t.Fatalf(
				"%s/%s canary organization = %s, expected %s",
				assertion.Surface,
				assertion.CanaryClass,
				record.OrganizationID,
				assertion.SourceOrganizationID,
			)
		}
	}
}

func TestEveryProviderAccountHasExactRoleOrganizationAuthority(
	t *testing.T,
) {
	for _, profileName := range []string{
		"smoke",
		"acceptance",
		"realistic",
		"stress",
	} {
		profile, err := profiles.Lookup(profileName, "1.0.0")
		if err != nil {
			t.Fatalf("lookup %s: %v", profileName, err)
		}
		stream, err := scenarios.NewStream(
			profile,
			[]byte("task-7-connected-scenarios"),
			loadCanonicalCatalog(t),
		)
		if err != nil {
			t.Fatalf("new %s stream: %v", profileName, err)
		}
		checked := int64(0)
		for {
			command, nextErr := stream.Next(context.Background())
			if nextErr != nil {
				t.Fatalf(
					"%s stream ended before provider accounts: %v",
					profileName,
					nextErr,
				)
			}
			if command.Family != "providerAccounts" {
				continue
			}
			batch, err := scenarios.DecodeBatch(command)
			if err != nil {
				t.Fatalf(
					"decode %s provider accounts: %v",
					profileName,
					err,
				)
			}
			for _, record := range batch.Records {
				role, ok := record.Attributes["role"].(string)
				if !ok || role == "" {
					t.Fatalf(
						"%s/%s role = %#v",
						profileName,
						record.RecordID,
						record.Attributes["role"],
					)
				}
				if (role == "auditee") ==
					(record.OrganizationID == "CAA") {
					t.Fatalf(
						"%s/%s role %s has organization %s",
						profileName,
						record.RecordID,
						role,
						record.OrganizationID,
					)
				}
			}
			checked += int64(len(batch.Records))
			if checked == profile.ExpectedCounts["providerAccounts"] {
				break
			}
		}
	}
}

func TestSmokeStreamEmitsExactConnectedCountsDistributionsAndLinks(
	t *testing.T,
) {
	profile, err := profiles.Lookup("smoke", "1.0.0")
	if err != nil {
		t.Fatalf("lookup smoke: %v", err)
	}
	stream, err := scenarios.NewStream(
		profile,
		[]byte("task-7-connected-scenarios"),
		loadCanonicalCatalog(t),
	)
	if err != nil {
		t.Fatalf("new stream: %v", err)
	}

	counts := make(map[string]int64)
	distributions := make(map[string]map[string]int64)
	for family, expected := range profile.ExactDistributions {
		distributions[family] = make(map[string]int64, len(expected))
		for state := range expected {
			distributions[family][state] = 0
		}
	}
	relationshipTuples := make(map[string][][]string)
	versionedFamilies := map[string]bool{
		"desiredMembershipVersions": true,
		"checklistTemplateVersions": true,
		"checklistResponses":        true,
		"capRevisions":              true,
		"evidenceVersions":          true,
		"reportVersions":            true,
		"objectVersions":            true,
	}
	var objectStates, syncCases, identityCases []string

	for {
		command, nextErr := stream.Next(context.Background())
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			t.Fatalf("next command: %v", nextErr)
		}
		batch, decodeErr := scenarios.DecodeBatch(command)
		if decodeErr != nil {
			t.Fatalf("decode %s: %v", command.OperationID, decodeErr)
		}
		if command.Family != batch.Family {
			t.Fatalf("command family %q != batch family %q",
				command.Family, batch.Family)
		}
		for _, record := range batch.Records {
			counts[batch.Family]++
			if record.Family != batch.Family {
				t.Fatalf("record family %q != batch family %q",
					record.Family, batch.Family)
			}
			if record.RecordID == "" ||
				record.BusinessKey == "" ||
				record.EffectiveAt.IsZero() ||
				record.KnownAt.IsZero() ||
				record.KnownAt.Before(record.EffectiveAt) ||
				record.ActorMembershipID == "" ||
				record.OrganizationID == "" ||
				record.DecisionReason == "" ||
				len(record.RelationshipTuple) == 0 {
				t.Fatalf("incomplete causal record: %#v", record)
			}
			if versionedFamilies[batch.Family] && record.Revision > 1 &&
				record.PredecessorID == "" {
				t.Fatalf("%s revision %d has no predecessor",
					batch.Family, record.Revision)
			}
			relationshipTuples[batch.Family] = append(
				relationshipTuples[batch.Family],
				record.RelationshipTuple,
			)
			distribution := record.Distribution
			if distribution == "" {
				distribution = "generated"
			}
			if distributions[batch.Family] == nil {
				distributions[batch.Family] = make(map[string]int64)
			}
			distributions[batch.Family][distribution]++

			switch batch.Family {
			case "scannerJobs":
				objectStates = append(objectStates, stringAttribute(
					t,
					record.Attributes,
					"processingState",
				))
			case "syncChanges":
				syncCases = append(syncCases, stringAttribute(
					t,
					record.Attributes,
					"syncCase",
				))
			case "identityLifecycleCases":
				identityCases = append(identityCases, stringAttribute(
					t,
					record.Attributes,
					"caseKind",
				))
			}
		}
	}

	if !reflect.DeepEqual(counts, profile.ExpectedCounts) {
		t.Fatalf("actual counts differ:\nactual=%#v\nexpected=%#v",
			counts, profile.ExpectedCounts)
	}
	for family, expected := range profile.ExactDistributions {
		if !reflect.DeepEqual(distributions[family], expected) {
			t.Fatalf("%s distribution differs:\nactual=%#v\nexpected=%#v",
				family, distributions[family], expected)
		}
	}
	for family, tuples := range relationshipTuples {
		if len(tuples) != int(profile.ExpectedCounts[family]) {
			t.Fatalf("%s relationship tuple count = %d", family, len(tuples))
		}
	}
	assertContainsSet(t, "object processing states", objectStates, requiredObjectStates)
	assertContainsSet(t, "sync cases", syncCases, requiredSyncCases)
	assertContainsSet(t, "identity cases", identityCases, requiredIdentityCases)
}

func TestStreamResumesAtTheExactDeterministicOperationWithoutMaterializingScale(
	t *testing.T,
) {
	profile, err := profiles.Lookup("stress", "1.0.0")
	if err != nil {
		t.Fatalf("lookup stress: %v", err)
	}
	catalog := loadCanonicalCatalog(t)
	first, err := scenarios.NewStream(
		profile,
		[]byte("task-7-connected-scenarios"),
		catalog,
	)
	if err != nil {
		t.Fatalf("new first stream: %v", err)
	}
	var lastOperationID string
	for index := int64(0); index < 7; index++ {
		command, nextErr := first.Next(context.Background())
		if nextErr != nil {
			t.Fatalf("first stream command %d: %v", index, nextErr)
		}
		lastOperationID = command.OperationID
	}
	expected, err := first.Next(context.Background())
	if err != nil {
		t.Fatalf("first stream expected command: %v", err)
	}

	resumed, err := scenarios.NewStream(
		profile,
		[]byte("task-7-connected-scenarios"),
		catalog,
	)
	if err != nil {
		t.Fatalf("new resumed stream: %v", err)
	}
	if err := resumed.ResumeAfter(
		context.Background(),
		7,
		lastOperationID,
	); err != nil {
		t.Fatalf("resume: %v", err)
	}
	actual, err := resumed.Next(context.Background())
	if err != nil {
		t.Fatalf("resumed expected command: %v", err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("resumed command differs:\nactual=%#v\nexpected=%#v",
			actual, expected)
	}
}

func loadCanonicalCatalog(t *testing.T) scenarios.Catalog {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("resolve test source")
	}
	repoRoot := filepath.Clean(filepath.Join(
		filepath.Dir(file),
		"..",
		"..",
		"..",
		"..",
		"..",
	))
	routeSource, err := os.ReadFile(filepath.Join(
		repoRoot,
		"apps/web/src/parity/legacy-screen-source.json",
	))
	if err != nil {
		t.Fatalf("read route source: %v", err)
	}
	ledgerSource, err := os.ReadFile(filepath.Join(
		repoRoot,
		"tests/parity/behavior-ledger.json",
	))
	if err != nil {
		t.Fatalf("read behavior ledger: %v", err)
	}
	catalog, err := scenarios.ParseCatalogs(routeSource, ledgerSource)
	if err != nil {
		t.Fatalf("parse canonical catalogs: %v", err)
	}
	if len(catalog.Routes) != 86 {
		t.Fatalf("route count = %d", len(catalog.Routes))
	}
	if len(catalog.Actions) != 306 {
		t.Fatalf("visible action count = %d", len(catalog.Actions))
	}

	var ledger struct {
		ActionEvidence []struct {
			SurfaceID                string   `json:"surfaceId"`
			Scope                    string   `json:"scope"`
			Profiles                 []string `json:"profiles"`
			ControlKey               string   `json:"controlKey"`
			Assertion                string   `json:"assertion"`
			IncludeInScenarioCatalog *bool    `json:"includeInScenarioCatalog"`
		} `json:"actionEvidence"`
	}
	if err := json.Unmarshal(ledgerSource, &ledger); err != nil {
		t.Fatalf("decode behavior ledger independently: %v", err)
	}
	executableAssertions := map[string]bool{
		"assertNativeFormControlOutcome": true,
		"assertAccessibleStateOutcome":   true,
		"assertControlledSurfaceOutcome": true,
		"assertDurableControlOutcome":    true,
		"suggestedFilename":              true,
	}
	var expectedActionIDs []string
	for _, entry := range ledger.ActionEvidence {
		if entry.IncludeInScenarioCatalog != nil && !*entry.IncludeInScenarioCatalog {
			continue
		}
		if entry.Scope != "route" || !executableAssertions[entry.Assertion] {
			continue
		}
		if len(entry.Profiles) > 0 && !slices.Contains(entry.Profiles, "mock") {
			continue
		}
		expectedActionIDs = append(
			expectedActionIDs,
			entry.SurfaceID+"|"+entry.ControlKey,
		)
	}
	sort.Strings(expectedActionIDs)
	if !reflect.DeepEqual(catalogActionIDs(catalog), expectedActionIDs) {
		t.Fatalf("catalog actions do not match the executable ledger")
	}
	return catalog
}

func routeIDs(routes []scenarios.RouteCoverage) []string {
	result := make([]string, len(routes))
	for index, route := range routes {
		result[index] = route.AuditID + "|" + route.SurfaceID
	}
	sort.Strings(result)
	return result
}

func actionIDs(actions []scenarios.ActionCoverage) []string {
	result := make([]string, len(actions))
	for index, action := range actions {
		result[index] = action.ActionID
	}
	sort.Strings(result)
	return result
}

func catalogRouteIDs(catalog scenarios.Catalog) []string {
	result := make([]string, len(catalog.Routes))
	for index, route := range catalog.Routes {
		result[index] = route.AuditID + "|" + route.SurfaceID
	}
	sort.Strings(result)
	return result
}

func catalogActionIDs(catalog scenarios.Catalog) []string {
	result := make([]string, len(catalog.Actions))
	for index, action := range catalog.Actions {
		result[index] = action.ActionID
	}
	sort.Strings(result)
	return result
}

func assertPrivacyMatrix(
	t *testing.T,
	assertions []scenarios.PrivacyAssertion,
) {
	t.Helper()
	surfaces := make(map[string]map[string]bool)
	for _, assertion := range assertions {
		if assertion.SourceOrganizationID == assertion.ActorOrganizationID {
			t.Fatalf("privacy assertion is not cross-organization: %#v", assertion)
		}
		if assertion.ExpectedResult != "denied-no-exposure" ||
			assertion.RecordCanary == "" {
			t.Fatalf("privacy assertion is not fail-closed: %#v", assertion)
		}
		if surfaces[assertion.Surface] == nil {
			surfaces[assertion.Surface] = make(map[string]bool)
		}
		surfaces[assertion.Surface][assertion.CanaryClass] = true
	}
	for _, surface := range requiredPrivacySurfaces {
		for _, canaryClass := range []string{
			"auditee-a-from-b",
			"auditee-b-from-a",
			"caa-private-from-auditee",
		} {
			if !surfaces[surface][canaryClass] {
				t.Fatalf("%s has no %s privacy assertion",
					surface, canaryClass)
			}
		}
	}
}

func assertExactSet(
	t *testing.T,
	label string,
	actual, expected []string,
) {
	t.Helper()
	actual = append([]string(nil), actual...)
	expected = append([]string(nil), expected...)
	sort.Strings(actual)
	sort.Strings(expected)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("%s differs:\nactual=%#v\nexpected=%#v",
			label, actual, expected)
	}
}

func assertContainsSet(
	t *testing.T,
	label string,
	actual, required []string,
) {
	t.Helper()
	seen := make(map[string]bool)
	for _, value := range actual {
		seen[value] = true
	}
	for _, value := range required {
		if !seen[value] {
			t.Fatalf("%s is missing %q", label, value)
		}
	}
}

func stringAttribute(
	t *testing.T,
	attributes map[string]any,
	key string,
) string {
	t.Helper()
	value, ok := attributes[key].(string)
	if !ok || value == "" {
		t.Fatalf("attribute %q = %#v", key, attributes[key])
	}
	return value
}
