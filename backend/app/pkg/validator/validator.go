// Package validator wraps github.com/go-playground/validator/v10.
//
// Responsibility:
//   - Provide a single, reusable validator instance.
//   - Translate validation errors into a simple map[field]messages.
//
// Why wrap it?
//
//	We do not want Gin's binding and validation details to leak into
//	A wrapper lets us swap the underlying library later without touching business logic.
package validator

import (
	"reflect"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

// V is the shared validator instance.
var V = validator.New()

func init() {
	mustRegisterValidation("password_uppercase", hasUppercase)
	mustRegisterValidation("password_number", hasNumber)
	mustRegisterValidation("password_special", hasSpecialCharacter)
}

func mustRegisterValidation(tag string, fn validator.Func) {
	if err := V.RegisterValidation(tag, fn); err != nil {
		panic(err)
	}
}

func hasUppercase(fl validator.FieldLevel) bool {
	return stringHasUppercase(fl.Field().String())
}

func hasNumber(fl validator.FieldLevel) bool {
	return stringHasNumber(fl.Field().String())
}

func hasSpecialCharacter(fl validator.FieldLevel) bool {
	return stringHasSpecialCharacter(fl.Field().String())
}

// Struct validates a struct and returns field-error maps.
//
// The returned map uses json tag names when available, falling back to the Go field name.
func Struct(s interface{}) map[string][]string {
	err := V.Struct(s)
	if err == nil {
		return nil
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return map[string][]string{"_error": {err.Error()}}
	}

	errors := make(map[string][]string)
	structType := reflect.TypeOf(s)
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	for _, fieldErr := range validationErrors {
		fieldName := fieldErr.StructField()
		jsonName := jsonTag(structType, fieldName)
		errors[jsonName] = append(errors[jsonName], messageForTag(fieldErr.Tag()))
	}

	appendPasswordRuleErrors(s, structType, errors)

	return errors
}

func appendPasswordRuleErrors(s interface{}, structType reflect.Type, errors map[string][]string) {
	structValue := reflect.ValueOf(s)
	if structValue.Kind() == reflect.Ptr {
		structValue = structValue.Elem()
	}
	if !structValue.IsValid() || structValue.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		validateTag := field.Tag.Get("validate")
		if !containsValidationTag(validateTag, "password_uppercase") &&
			!containsValidationTag(validateTag, "password_number") &&
			!containsValidationTag(validateTag, "password_special") {
			continue
		}

		fieldValue := structValue.Field(i)
		if fieldValue.Kind() != reflect.String {
			continue
		}

		password := fieldValue.String()
		if password == "" {
			continue
		}

		jsonName := jsonTag(structType, field.Name)
		if containsValidationTag(validateTag, "password_uppercase") && !stringHasUppercase(password) {
			errors[jsonName] = appendUniqueMessage(errors[jsonName], messageForTag("password_uppercase"))
		}
		if containsValidationTag(validateTag, "password_number") && !stringHasNumber(password) {
			errors[jsonName] = appendUniqueMessage(errors[jsonName], messageForTag("password_number"))
		}
		if containsValidationTag(validateTag, "password_special") && !stringHasSpecialCharacter(password) {
			errors[jsonName] = appendUniqueMessage(errors[jsonName], messageForTag("password_special"))
		}
	}
}

func containsValidationTag(validateTag string, expected string) bool {
	for _, tag := range strings.Split(validateTag, ",") {
		name := tag
		if idx := strings.Index(name, "="); idx != -1 {
			name = name[:idx]
		}
		if name == expected {
			return true
		}
	}
	return false
}

func appendUniqueMessage(messages []string, message string) []string {
	for _, existing := range messages {
		if existing == message {
			return messages
		}
	}
	return append(messages, message)
}

func stringHasUppercase(value string) bool {
	for _, char := range value {
		if unicode.IsUpper(char) {
			return true
		}
	}
	return false
}

func stringHasNumber(value string) bool {
	for _, char := range value {
		if unicode.IsDigit(char) {
			return true
		}
	}
	return false
}

func stringHasSpecialCharacter(value string) bool {
	for _, char := range value {
		if unicode.IsPunct(char) || unicode.IsSymbol(char) {
			return true
		}
	}
	return false
}

// jsonTag returns the json tag for a struct field, or the field name if absent.
func jsonTag(structType reflect.Type, fieldName string) string {
	field, ok := structType.FieldByName(fieldName)
	if !ok {
		return fieldName
	}

	tag := field.Tag.Get("json")
	if tag == "" || tag == "-" {
		return fieldName
	}

	// Strip options like `json:"name,omitempty"`.
	if idx := strings.Index(tag, ","); idx != -1 {
		tag = tag[:idx]
	}
	return tag
}

// messageForTag returns a human-readable validation message.
func messageForTag(tag string) string {
	messages := map[string]string{
		"required":           "This field is required.",
		"email":              "Must be a valid email address.",
		"min":                "Value is too short.",
		"max":                "Value is too long.",
		"gte":                "Value is too small.",
		"lte":                "Value is too large.",
		"password_uppercase": "Must include at least one uppercase letter.",
		"password_number":    "Must include at least one number.",
		"password_special":   "Must include at least one special character.",
	}

	if msg, ok := messages[tag]; ok {
		return msg
	}
	return "Invalid value."
}
