package request

import "time"

type Status string

const (
	StatusPending   Status = "pending"
	StatusApproved  Status = "approved"
	StatusAccepted  Status = "accepted"
	StatusCompleted Status = "completed"
	StatusDeclined  Status = "declined"
	StatusWithdrawn Status = "withdrawn"
)

type CostWaivedReason string

const (
	CostWaivedReasonNone             CostWaivedReason = ""
	CostWaivedReasonAfterEarlyWindow CostWaivedReason = "after_early_window"
	CostWaivedReasonSubscriber       CostWaivedReason = "subscriber"
)

type RewardStatus string

const (
	RewardStatusNone    RewardStatus = "none"
	RewardStatusPending RewardStatus = "pending"
	RewardStatusGranted RewardStatus = "granted"
	RewardStatusBlocked RewardStatus = "blocked"
)

type RewardBlockedReason string

const (
	RewardBlockedPhoneNotVerified  RewardBlockedReason = "phone_not_verified"
	RewardBlockedRepeatRecipient   RewardBlockedReason = "repeat_recipient"
	RewardBlockedSameDevice        RewardBlockedReason = "same_device"
	RewardBlockedSamePhoneIdentity RewardBlockedReason = "same_phone_identity"
	RewardBlockedRecipientDisputed RewardBlockedReason = "recipient_disputed"
	RewardBlockedSuspiciousIP      RewardBlockedReason = "suspicious_ip"
)

// IsOpen reports whether a request still holds reserved points and counts
// toward the item's active queue.
func (s Status) IsOpen() bool {
	switch s {
	case StatusPending, StatusApproved, StatusAccepted:
		return true
	default:
		return false
	}
}

type Request struct {
	ID                  string    `json:"id" dynamodbav:"id"`
	ItemID              string    `json:"itemId" dynamodbav:"itemId"`
	OwnerID             string    `json:"ownerId" dynamodbav:"ownerId"`
	RequesterID         string    `json:"requesterId" dynamodbav:"requesterId"`
	Message             string    `json:"message,omitempty" dynamodbav:"message,omitempty"`
	PointsReserved      int       `json:"pointsReserved" dynamodbav:"pointsReserved"`
	CostWaivedReason    string    `json:"costWaivedReason,omitempty" dynamodbav:"costWaivedReason,omitempty"`
	Rating              int       `json:"rating,omitempty" dynamodbav:"rating,omitempty"`
	RewardPoints        int       `json:"rewardPoints,omitempty" dynamodbav:"rewardPoints,omitempty"`
	RewardGranted       bool      `json:"rewardGranted" dynamodbav:"rewardGranted"`
	DeliveredAt         time.Time `json:"deliveredAt,omitempty" dynamodbav:"deliveredAt,omitempty"`
	RewardStatus        string    `json:"rewardStatus,omitempty" dynamodbav:"rewardStatus,omitempty"`
	RewardEligibleAt    time.Time `json:"rewardEligibleAt,omitempty" dynamodbav:"rewardEligibleAt,omitempty"`
	RewardBlockedReason string    `json:"rewardBlockedReason,omitempty" dynamodbav:"rewardBlockedReason,omitempty"`
	CreatedIP           string    `json:"-" dynamodbav:"createdIp,omitempty"`
	DeviceFingerprint   string    `json:"-" dynamodbav:"deviceFingerprint,omitempty"`
	Status              Status    `json:"status" dynamodbav:"status"`
	CreatedAt           time.Time `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt" dynamodbav:"updatedAt"`
}

type CreateInput struct {
	Message string `json:"message"`
}

type CompleteInput struct {
	Rating int `json:"rating"`
}
