//go:build canonicaltest

package integration

import "testing"

func TestGovernedChecklistIntakeRecovery(t *testing.T) {
	t.Skip("blocked: disposable PostgreSQL/object-store crash-recovery profile is not available in this candidate workspace")
}
