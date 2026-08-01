//go:build canonicaltest

package integration

import "testing"

func TestAGACandidateExpansion(t *testing.T) {
	// Phase 2 is intentionally blocked until the exact real Form 048 candidate,
	// its 28 Admin boundary decisions, and the visible source-gap Draft exist.
	// Synthetic mechanism and connected-runtime evidence cannot substitute for
	// those real-owner facts, so no additional form is imported here.
	t.Skip("blocked: real Form 048 Admin identity/28 boundary packet, immutable candidate/source-gap Draft, and named expansion authorization are absent")
}
