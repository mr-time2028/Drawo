// Package domain contains pure business entities.
package domain

import "time"

// OTPType defines which channel the verification code is for.
type OTPType string

const (
	OTPEmail OTPType = "email"
	OTPPhone OTPType = "phone"
)

// OTP represents a temporary verification code stored in a fast-access cache.
type OTP struct {
	Identifier string    // Email address or Phone number
	Type       OTPType   // Email or Phone
	Code       string    // The 6-digit random code
	ExpiresAt  time.Time // Expiration timestamp
}

// IsExpired checks if the code has passed its validity window.
func (o *OTP) IsExpired() bool {
	return time.Now().After(o.ExpiresAt)
}
