//go:build canonicaltest

package integration

import (
	"testing"

	"github.com/aviason/aviaSurveil/internal/checklistintake"
)

func TestGovernedChecklistIntakeLifecycle(t *testing.T) {
	if checklistintake.AGAZipPDFV1().Version != checklistintake.PolicyAGAZipPDFV1 {
		t.Fatal("intake policy version drifted")
	}
}
