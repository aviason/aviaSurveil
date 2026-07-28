package notifications

import "testing"

func TestIdentityDeliveryTransitionsPreserveAppendOnlyLifecycle(t *testing.T) {
	t.Parallel()
	allowed := []struct {
		previous IdentityDeliveryState
		next     IdentityDeliveryState
	}{
		{"", IdentityDeliveryIssued},
		{IdentityDeliveryIssued, IdentityDeliveryAccepted},
		{IdentityDeliveryAccepted, IdentityDeliveryConsumed},
		{IdentityDeliveryAccepted, IdentityDeliveryExpired},
		{IdentityDeliveryIssued, IdentityDeliveryCancelled},
		{"", IdentityDeliveryResetPending},
		{IdentityDeliveryResetPending, IdentityDeliveryResetCompleted},
		{IdentityDeliveryRetryableFailure, IdentityDeliveryIssued},
		{
			IdentityDeliveryRetryableFailure,
			IdentityDeliveryRetryableFailure,
		},
		{
			IdentityDeliveryRetryableFailure,
			IdentityDeliveryResetPending,
		},
	}
	for _, transition := range allowed {
		transition := transition
		t.Run(string(transition.previous)+"_"+string(transition.next), func(t *testing.T) {
			t.Parallel()
			if err := ValidateIdentityDeliveryTransition(
				transition.previous,
				transition.next,
			); err != nil {
				t.Fatalf("valid transition rejected: %v", err)
			}
		})
	}
}

func TestIdentityDeliveryTransitionsRejectTerminalRewrites(t *testing.T) {
	t.Parallel()
	terminalStates := []IdentityDeliveryState{
		IdentityDeliveryTerminalFailure,
		IdentityDeliveryExpired,
		IdentityDeliveryConsumed,
		IdentityDeliveryCancelled,
		IdentityDeliveryResetCompleted,
	}
	for _, terminalState := range terminalStates {
		terminalState := terminalState
		t.Run(string(terminalState), func(t *testing.T) {
			t.Parallel()
			if err := ValidateIdentityDeliveryTransition(
				terminalState,
				IdentityDeliveryIssued,
			); err == nil {
				t.Fatalf(
					"terminal state %q accepted a rewrite",
					terminalState,
				)
			}
		})
	}
}
