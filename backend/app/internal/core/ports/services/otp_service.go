// Package services defines application use cases and business logic.
package services

import (
	"context"
	"log/slog"

	"drawo/internal/core/domain"
)

// OTPService defines the contract for sending verification codes to users.
type OTPService interface {
	// SendCode transmits a 6-digit code via the specified channel.
	SendCode(ctx context.Context, otp *domain.OTP) error
}

// mockOTPService is a development-only provider that logs codes to the console.
type mockOTPService struct{}

// NewMockOTPService creates a provider for local development.
func NewMockOTPService() OTPService {
	return &mockOTPService{}
}

// SendCode implements the logic for local testing.
func (s *mockOTPService) SendCode(ctx context.Context, otp *domain.OTP) error {
	// -------------------------------------------------------------------------
	// PRODUCTION CONFIGURATION GUIDE:
	// -------------------------------------------------------------------------
	// To use a real provider (like Twilio, SendGrid, or AWS SES):
	//
	// 1. Create a new implementation file (e.g., `internal/infrastructure/msg/sendgrid.go`).
	// 2. Add your Provider API Key to the `.env` file and `config` struct.
	// 3. Implement the SendCode method using the provider's SDK.
	// 4. Update `internal/infrastructure/di/container.go` to use your new
	//    provider instead of this mock.
	// -------------------------------------------------------------------------

	slog.Info("MOCK OTP SENT",
		slog.String("to", otp.Identifier),
		slog.String("type", string(otp.Type)),
		slog.String("code", otp.Code),
	)

	return nil
}
