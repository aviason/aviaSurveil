package regulatory

import (
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
)

func TestBuildScopedReviewQueueFailsClosedForUnassignedSubjects(t *testing.T) {
	items := []ReviewQueueItem{{ReviewItemID: "R1", CandidateID: "C1", Kind: SourceReviewQueue, ScopeKey: "S1", CandidateOnly: true}}
	assignments := []identity.FunctionalAssignment{{SubjectID: "other", MembershipID: "m", Permission: identity.FunctionalPermissionRegulatorySourceOwner, EffectiveFrom: time.Now().Add(-time.Hour)}}
	result, err := BuildScopedReviewQueue(assignments, "subject", SourceReviewQueue, items)
	if err != nil || len(result) != 0 {
		t.Fatalf("unassigned queue leaked rows: result=%+v err=%v", result, err)
	}
}
