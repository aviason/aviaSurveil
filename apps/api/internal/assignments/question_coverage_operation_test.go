package assignments

import (
	"fmt"
	"testing"
)

func TestApplyQuestionCoverageOperationBuildsLargeExactSetInBoundedBatches(t *testing.T) {
	all := make([]QuestionAssignment, 0, 1310)
	for index := 0; index < 1310; index++ {
		all = append(all, QuestionAssignment{
			QuestionID: fmt.Sprintf("qv:synthetic:%04d", index+1),
			SubjectID:  "inspector-001",
		})
	}

	current := []QuestionAssignment{}
	for start := 0; start < len(all); start += 500 {
		end := min(start+500, len(all))
		var err error
		current, err = applyQuestionCoverageOperation(current, all[start:end], QuestionCoverageAdd)
		if err != nil {
			t.Fatalf("add coverage batch %d: %v", start/500+1, err)
		}
	}

	if len(current) != 1310 {
		t.Fatalf("coverage rows = %d, want 1310", len(current))
	}
	if current[0] != all[0] || current[len(current)-1] != all[len(all)-1] {
		t.Fatalf("coverage ordering/identity drifted: first=%+v last=%+v", current[0], current[len(current)-1])
	}
}

func TestApplyQuestionCoverageOperationSupportsExactRemovalAndReplacement(t *testing.T) {
	initial := []QuestionAssignment{
		{QuestionID: "qv-1", SubjectID: "inspector-a"},
		{QuestionID: "qv-2", SubjectID: "inspector-a"},
	}
	removed, err := applyQuestionCoverageOperation(initial, initial[:1], QuestionCoverageRemove)
	if err != nil {
		t.Fatalf("remove coverage: %v", err)
	}
	if len(removed) != 1 || removed[0] != initial[1] {
		t.Fatalf("removed coverage = %+v", removed)
	}
	replacement := []QuestionAssignment{{QuestionID: "qv-3", SubjectID: "inspector-b"}}
	replaced, err := applyQuestionCoverageOperation(removed, replacement, QuestionCoverageReplace)
	if err != nil {
		t.Fatalf("replace coverage: %v", err)
	}
	if len(replaced) != 1 || replaced[0] != replacement[0] {
		t.Fatalf("replaced coverage = %+v", replaced)
	}
}
