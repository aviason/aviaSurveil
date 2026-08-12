package scenarios

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aviason/aviaSurveil/internal/preproddata"
)

const batchSchemaVersion = "preprod-connected-scenario-batch/v1"

type Batch struct {
	SchemaVersion string   `json:"schemaVersion"`
	Family        string   `json:"family"`
	Records       []Record `json:"records"`
}

type Record struct {
	Family            string         `json:"family"`
	RecordID          string         `json:"recordId"`
	BusinessKey       string         `json:"businessKey"`
	Revision          int64          `json:"revision"`
	PredecessorID     string         `json:"predecessorId,omitempty"`
	Distribution      string         `json:"distribution"`
	EffectiveAt       time.Time      `json:"effectiveAt"`
	KnownAt           time.Time      `json:"knownAt"`
	ActorMembershipID string         `json:"actorMembershipId"`
	OrganizationID    string         `json:"organizationId"`
	DecisionReason    string         `json:"decisionReason"`
	RelationshipTuple []string       `json:"relationshipTuple"`
	Attributes        map[string]any `json:"attributes"`
}

type Coverage struct {
	Routes             []RouteCoverage    `json:"routes"`
	Actions            []ActionCoverage   `json:"actions"`
	Roles              []string           `json:"roles"`
	LifecycleScenarios []string           `json:"lifecycleScenarios"`
	DomainCoverage     []string           `json:"domainCoverage"`
	IdentityCases      []string           `json:"identityCases"`
	ObjectStates       []string           `json:"objectStates"`
	SyncCases          []string           `json:"syncCases"`
	Privacy            []PrivacyAssertion `json:"privacy"`
}

type PrivacyAssertion struct {
	Surface              string `json:"surface"`
	CanaryClass          string `json:"canaryClass"`
	SourceOrganizationID string `json:"sourceOrganizationId"`
	ActorOrganizationID  string `json:"actorOrganizationId"`
	RecordCanary         string `json:"recordCanary"`
	ExpectedResult       string `json:"expectedResult"`
}

func DecodeBatch(command preproddata.AuthoritativeCommand) (Batch, error) {
	var batch Batch
	if err := json.Unmarshal(command.Payload, &batch); err != nil {
		return Batch{}, fmt.Errorf("decode scenario batch: %w", err)
	}
	if batch.SchemaVersion != batchSchemaVersion ||
		batch.Family == "" ||
		batch.Family != command.Family ||
		len(batch.Records) == 0 {
		return Batch{}, fmt.Errorf("invalid connected-scenario batch")
	}
	for _, record := range batch.Records {
		if record.Family != batch.Family ||
			record.RecordID == "" ||
			record.BusinessKey == "" ||
			record.Revision < 1 ||
			record.EffectiveAt.IsZero() ||
			record.KnownAt.Before(record.EffectiveAt) ||
			record.ActorMembershipID == "" ||
			record.OrganizationID == "" ||
			record.DecisionReason == "" ||
			len(record.RelationshipTuple) == 0 {
			return Batch{}, fmt.Errorf("invalid connected-scenario record")
		}
	}
	return batch, nil
}
