package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testUser struct {
	Username string `json:"username" validate:"required,min=3"`
	Email    string `json:"email" validate:"required,email"`
	Age      int    `json:"age" validate:"gte=18"`
}

func TestStructValid(t *testing.T) {
	u := testUser{
		Username: "alice",
		Email:    "alice@example.com",
		Age:      25,
	}
	assert.Nil(t, Struct(u))
}

func TestStructInvalid(t *testing.T) {
	u := testUser{
		Username: "al",
		Email:    "not-an-email",
		Age:      16,
	}
	 errs := Struct(u)
	assert.NotNil(t, errs)
	assert.Contains(t, errs, "username")
	assert.Contains(t, errs, "email")
	assert.Contains(t, errs, "age")
}
