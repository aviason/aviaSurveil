package subscription

import (
	"errors"
	"time"
)

const PremiumMonthlyProductID = "com.radlof.kindred.premium.monthly_sub"

var (
	ErrNotFound           = errors.New("subscription not found")
	ErrInvalidTransaction = errors.New("invalid storekit transaction")
)

type Entitlement struct {
	UserID                string    `json:"userId" dynamodbav:"userId"`
	ProductID             string    `json:"productId" dynamodbav:"productId"`
	TransactionID         string    `json:"transactionId" dynamodbav:"transactionId"`
	OriginalTransactionID string    `json:"originalTransactionId" dynamodbav:"originalTransactionId"`
	ExpiresAt             time.Time `json:"expiresAt" dynamodbav:"expiresAt"`
	UpdatedAt             time.Time `json:"updatedAt" dynamodbav:"updatedAt"`
}

type Status struct {
	IsActive                        bool       `json:"isActive"`
	ProductID                       string     `json:"productId,omitempty"`
	ExpiresAt                       *time.Time `json:"expiresAt,omitempty"`
	FreeEarlyRequestsRemainingToday int        `json:"freeEarlyRequestsRemainingToday"`
	DailyFreeEarlyRequestLimit      int        `json:"dailyFreeEarlyRequestLimit"`
}

type StoreKitTransactionRequest struct {
	SignedTransactionInfo string `json:"signedTransactionInfo"`
}

type StoreKitTransactionResponse struct {
	Entitlement Status `json:"entitlement"`
}

type VerifiedTransaction struct {
	ProductID             string
	BundleID              string
	TransactionID         string
	OriginalTransactionID string
	ExpiresAt             time.Time
}
