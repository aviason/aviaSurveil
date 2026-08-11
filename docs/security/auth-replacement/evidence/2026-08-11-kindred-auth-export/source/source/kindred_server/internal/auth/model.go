package auth

import (
	"time"

	"kindred_server/internal/user"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	DeviceID string `json:"deviceId,omitempty"`
}

type ReactivateRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	DeviceID string `json:"deviceId,omitempty"`
}

type RegisterRequest struct {
	Email       string          `json:"email"`
	Password    string          `json:"password"`
	DisplayName string          `json:"displayName"`
	Phone       string          `json:"phone,omitempty"`
	City        string          `json:"city,omitempty"`
	BirthYear   int             `json:"birthYear,omitempty"`
	Gender      string          `json:"gender,omitempty"`
	DeviceID    string          `json:"deviceId,omitempty"`
	Consents    map[string]bool `json:"consents,omitempty"`
}

type Claims struct {
	UserID       string    `json:"userId"`
	Email        string    `json:"email"`
	ExpiresAt    time.Time `json:"expiresAt"`
	TokenVersion int       `json:"tokenVersion"`
	DeviceID     string    `json:"deviceId,omitempty"`
}

type AuthResponse struct {
	Token            string        `json:"token"`
	ExpiresAt        time.Time     `json:"expiresAt"`
	RefreshToken     string        `json:"refreshToken"`
	RefreshExpiresAt time.Time     `json:"refreshExpiresAt"`
	User             user.Response `json:"user"`
}

type StartPhoneVerificationResponse struct {
	ExpiresAt        time.Time `json:"expiresAt"`
	VerificationCode string    `json:"verificationCode,omitempty"`
}

type VerifyPhoneRequest struct {
	Code string `json:"code"`
}

type StartPhoneChangeRequest struct {
	NewPhone string `json:"newPhone"`
	Password string `json:"password"`
}

type VerifyPhoneChangeRequest struct {
	Code string `json:"code"`
}

type PasswordStepUpRequest struct {
	Password string `json:"password"`
}

type DeleteAccountResponse struct {
	Status           user.AccountStatus `json:"status"`
	ScheduledPurgeAt time.Time          `json:"scheduledPurgeAt"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
	DeviceID     string `json:"deviceId,omitempty"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken,omitempty"`
}

type VerifyEmailRequest struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

type ResendVerificationRequest struct {
	Email string `json:"email"`
}

type RequestPasswordResetRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email"`
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}
