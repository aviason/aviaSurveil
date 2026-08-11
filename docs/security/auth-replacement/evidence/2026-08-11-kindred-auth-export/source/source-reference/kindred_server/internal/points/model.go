// Package points models a simple user point balance and ledger. Points are
// the in-app currency: users spend them to create threads and earn them when
// real deliveries are confirmed.
package points

import "time"

type Balance struct {
	UserID    string    `json:"userId" dynamodbav:"userId"`
	Available int       `json:"available" dynamodbav:"available"`
	UpdatedAt time.Time `json:"updatedAt" dynamodbav:"updatedAt"`
}

// Reason classifies why a ledger entry was written. Keeping this closed lets
// the audit log be machine-parseable.
type Reason string

const (
	ReasonReserve        Reason = "reserve"
	ReasonRefund         Reason = "refund"
	ReasonThreadCreate   Reason = "thread_create"
	ReasonDeliveryReward Reason = "delivery_reward"
	ReasonSignupBonus    Reason = "signup_bonus"
)

type Transaction struct {
	ID        string    `json:"id" dynamodbav:"id"`
	UserID    string    `json:"userId" dynamodbav:"userId"`
	Delta     int       `json:"delta" dynamodbav:"delta"`
	Reason    Reason    `json:"reason" dynamodbav:"reason"`
	RefID     string    `json:"refId,omitempty" dynamodbav:"refId,omitempty"`
	CreatedAt time.Time `json:"createdAt" dynamodbav:"createdAt"`
}
