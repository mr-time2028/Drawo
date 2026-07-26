package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"drawo/internal/core/domain"
)

func TestMockOTPService(t *testing.T) {
	svc := NewMockOTPService()
	err := svc.SendCode(context.Background(), &domain.OTP{
		Identifier: "test",
		Type:       domain.OTPEmail,
		Code:       "123456",
	})
	assert.NoError(t, err)
}
