package user

import "time"

type AccountStatus string

const (
	AccountStatusActive          AccountStatus = "active"
	AccountStatusDeactivated     AccountStatus = "deactivated"
	AccountStatusDeletionPending AccountStatus = "deletion_pending"
	AccountStatusDeleted         AccountStatus = "deleted"
)

type User struct {
	ID            string    `json:"id" dynamodbav:"id"`
	Email         string    `json:"email" dynamodbav:"email"`
	DisplayName   string    `json:"displayName" dynamodbav:"displayName"`
	Phone         string    `json:"phone,omitempty" dynamodbav:"phone,omitempty"`
	PhoneVerified bool      `json:"phoneVerified" dynamodbav:"phoneVerified"`
	City          string    `json:"city,omitempty" dynamodbav:"city,omitempty"`
	BirthYear     int       `json:"birthYear,omitempty" dynamodbav:"birthYear,omitempty"`
	Gender        string    `json:"gender,omitempty" dynamodbav:"gender,omitempty"`
	PasswordHash  string    `json:"-" dynamodbav:"passwordHash"`
	CreatedAt     time.Time `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt" dynamodbav:"updatedAt"`

	EmailVerified              bool      `json:"emailVerified" dynamodbav:"emailVerified"`
	EmailVerificationTokenHash string    `json:"-" dynamodbav:"emailVerificationTokenHash,omitempty"`
	EmailVerificationExpiresAt time.Time `json:"-" dynamodbav:"emailVerificationExpiresAt,omitempty"`

	PhoneVerificationCodeHash         string    `json:"-" dynamodbav:"phoneVerificationCodeHash,omitempty"`
	PhoneVerificationExpiresAt        time.Time `json:"-" dynamodbav:"phoneVerificationExpiresAt,omitempty"`
	PendingPhone                      string    `json:"-" dynamodbav:"pendingPhone,omitempty"`
	PendingPhoneVerificationCodeHash  string    `json:"-" dynamodbav:"pendingPhoneVerificationCodeHash,omitempty"`
	PendingPhoneVerificationExpiresAt time.Time `json:"-" dynamodbav:"pendingPhoneVerificationExpiresAt,omitempty"`

	PasswordResetTokenHash string    `json:"-" dynamodbav:"passwordResetTokenHash,omitempty"`
	PasswordResetExpiresAt time.Time `json:"-" dynamodbav:"passwordResetExpiresAt,omitempty"`

	RefreshTokenHash              string    `json:"-" dynamodbav:"refreshTokenHash,omitempty"`
	RefreshTokenExpiresAt         time.Time `json:"-" dynamodbav:"refreshTokenExpiresAt,omitempty"`
	RefreshTokenAbsoluteExpiresAt time.Time `json:"-" dynamodbav:"refreshTokenAbsoluteExpiresAt,omitempty"`
	RefreshTokenDeviceID          string    `json:"-" dynamodbav:"refreshTokenDeviceID,omitempty"`

	// TokenVersion is incremented on Logout to invalidate all outstanding
	// access tokens. The auth middleware compares the value carried in the
	// bearer token's claims against this and rejects on mismatch.
	TokenVersion int `json:"-" dynamodbav:"tokenVersion,omitempty"`

	FailedLoginAttempts int       `json:"-" dynamodbav:"failedLoginAttempts,omitempty"`
	LockedUntil         time.Time `json:"-" dynamodbav:"lockedUntil,omitempty"`

	// ProfilePhotoKey is the S3 object key for this user's avatar. Empty
	// means no photo set. The presigned GET URL is computed at view time
	// (see internal/user handler) so URL expiry is self-healing.
	ProfilePhotoKey string `json:"-" dynamodbav:"profilePhotoKey,omitempty"`

	AccountStatus       AccountStatus `json:"accountStatus" dynamodbav:"accountStatus,omitempty"`
	DeactivatedAt       time.Time     `json:"deactivatedAt,omitempty" dynamodbav:"deactivatedAt,omitempty"`
	DeletionRequestedAt time.Time     `json:"deletionRequestedAt,omitempty" dynamodbav:"deletionRequestedAt,omitempty"`
	ScheduledPurgeAt    time.Time     `json:"scheduledPurgeAt,omitempty" dynamodbav:"scheduledPurgeAt,omitempty"`
}

type CreateRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
	Phone       string `json:"phone,omitempty"`
	City        string `json:"city,omitempty"`
	BirthYear   int    `json:"birthYear,omitempty"`
	Gender      string `json:"gender,omitempty"`
}

type Response struct {
	ID            string        `json:"id"`
	Email         string        `json:"email"`
	DisplayName   string        `json:"displayName"`
	Phone         string        `json:"phone,omitempty"`
	PhoneVerified bool          `json:"phoneVerified"`
	EmailVerified bool          `json:"emailVerified"`
	City          string        `json:"city,omitempty"`
	BirthYear     int           `json:"birthYear,omitempty"`
	Gender        string        `json:"gender,omitempty"`
	CreatedAt     time.Time     `json:"createdAt"`
	UpdatedAt     time.Time     `json:"updatedAt"`
	AccountStatus AccountStatus `json:"accountStatus,omitempty"`
}

func ToResponse(u User) Response {
	status := u.AccountStatus
	if status == "" {
		status = AccountStatusActive
	}
	return Response{
		ID:            u.ID,
		Email:         u.Email,
		DisplayName:   u.DisplayName,
		Phone:         u.Phone,
		PhoneVerified: u.PhoneVerified,
		EmailVerified: u.EmailVerified,
		City:          u.City,
		BirthYear:     u.BirthYear,
		Gender:        u.Gender,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
		AccountStatus: status,
	}
}

func (u User) Status() AccountStatus {
	if u.AccountStatus == "" {
		return AccountStatusActive
	}
	return u.AccountStatus
}
