package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testUser struct {
	Username string `json:"username" validate:"required,min=3"`
	Email    string `json:"email,omitempty" validate:"required,email"`
	Age      int    `json:"age" validate:"gte=18"`
}

type testTags struct {
	Field1 string `json:"field1" validate:"lte=5"`
}

type testNoJson struct {
	Username string `validate:"required"`
}

type testDash struct {
	Field string `json:"-" validate:"required"`
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

func TestStructPtr(t *testing.T) {
	u := &testUser{
		Username: "alice",
		Email:    "alice@example.com",
		Age:      25,
	}
	assert.Nil(t, Struct(u))
}

func TestNonStruct(t *testing.T) {
	errs := Struct(123)
	assert.NotNil(t, errs)
	assert.Contains(t, errs, "_error")
}

func TestJsonTagEdgeCases(t *testing.T) {
	// Test - tag
	u1 := testDash{}
	errs1 := Struct(u1)
	assert.Contains(t, errs1, "Field")

	// Test empty tag
	type testEmpty struct {
		Field string `json:"" validate:"required"`
	}
	errs2 := Struct(testEmpty{})
	assert.Contains(t, errs2, "Field")

	// Test options like ,omitempty
	u3 := testUser{}
	errs3 := Struct(u3)
	assert.Contains(t, errs3, "email")
}

func TestPasswordValidationTags(t *testing.T) {
	type testPassword struct {
		Password string `json:"password" validate:"required,min=8,password_uppercase,password_number,password_special"`
	}

	assert.Nil(t, Struct(testPassword{Password: "Secret@1"}))

	errs := Struct(testPassword{Password: "password"})
	assert.Contains(t, errs["password"], "Must include at least one uppercase letter.")
	assert.Contains(t, errs["password"], "Must include at least one number.")
	assert.Contains(t, errs["password"], "Must include at least one special character.")
}

func TestMessageForTag(t *testing.T) {
	type testUnknown struct {
		Field string `validate:"url"`
	}
	errs := Struct(testUnknown{Field: "not-url"})
	assert.Equal(t, "Invalid value.", errs["Field"][0])
}
