//go:build canonicaltest

package integration

import (
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/checklistintake"
)

func TestGovernedChecklistIntakeLifecycle(t *testing.T) {
	if checklistintake.AGAZipPDFV1().Version != checklistintake.PolicyAGAZipPDFV1 {
		t.Fatal("intake policy version drifted")
	}
}
