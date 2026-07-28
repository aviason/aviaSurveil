package notifications

import "fmt"

type IdentityActionKind string

const (
	IdentityActionInvitation IdentityActionKind = "INVITATION"
	IdentityActionRecovery   IdentityActionKind = "RECOVERY"
	IdentityActionMFAReset   IdentityActionKind = "MFA_RESET"
)

type IdentityDeliveryState string

const (
	IdentityDeliveryIssued           IdentityDeliveryState = "ISSUED"
	IdentityDeliveryAccepted         IdentityDeliveryState = "DELIVERY_ACCEPTED"
	IdentityDeliveryRetryableFailure IdentityDeliveryState = "RETRYABLE_FAILURE"
	IdentityDeliveryTerminalFailure  IdentityDeliveryState = "TERMINAL_FAILURE"
	IdentityDeliveryExpired          IdentityDeliveryState = "EXPIRED"
	IdentityDeliveryConsumed         IdentityDeliveryState = "CONSUMED"
	IdentityDeliveryCancelled        IdentityDeliveryState = "CANCELLED"
	IdentityDeliveryResetPending     IdentityDeliveryState = "RESET_PENDING"
	IdentityDeliveryResetCompleted   IdentityDeliveryState = "RESET_COMPLETED"
)

func ValidateIdentityDeliveryTransition(
	previous,
	next IdentityDeliveryState,
) error {
	allowed := map[IdentityDeliveryState]map[IdentityDeliveryState]bool{
		"": {
			IdentityDeliveryIssued:           true,
			IdentityDeliveryRetryableFailure: true,
			IdentityDeliveryTerminalFailure:  true,
			IdentityDeliveryResetPending:     true,
		},
		IdentityDeliveryIssued: {
			IdentityDeliveryAccepted:         true,
			IdentityDeliveryRetryableFailure: true,
			IdentityDeliveryTerminalFailure:  true,
			IdentityDeliveryCancelled:        true,
			IdentityDeliveryExpired:          true,
		},
		IdentityDeliveryAccepted: {
			IdentityDeliveryConsumed:  true,
			IdentityDeliveryExpired:   true,
			IdentityDeliveryCancelled: true,
		},
		IdentityDeliveryRetryableFailure: {
			IdentityDeliveryIssued:           true,
			IdentityDeliveryRetryableFailure: true,
			IdentityDeliveryTerminalFailure:  true,
			IdentityDeliveryResetPending:     true,
		},
		IdentityDeliveryResetPending: {
			IdentityDeliveryResetCompleted:   true,
			IdentityDeliveryRetryableFailure: true,
			IdentityDeliveryTerminalFailure:  true,
		},
	}
	if allowed[previous][next] {
		return nil
	}
	return fmt.Errorf(
		"invalid identity delivery transition %q -> %q",
		previous,
		next,
	)
}
