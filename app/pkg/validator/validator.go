// Package validator wraps github.com/go-playground/validator/v10.
//
// Responsibility:
//   - Provide a single, reusable validator instance.
//   - Translate validation errors into a simple map[field]messages.
//
// Why wrap it?
//   We do not want Gin's binding and validation details to leak into 
//   A wrapper lets us swap the underlying library later without touching business logic.
package validator

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// V is the shared validator instance.
var V = validator.New()

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

	return errors
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
		"required": "This field is required.",
		"email":    "Must be a valid email address.",
		"min":      "Value is too short.",
		"max":      "Value is too long.",
		"gte":      "Value is too small.",
		"lte":      "Value is too large.",
	}

	if msg, ok := messages[tag]; ok {
		return msg
	}
	return "Invalid value."
}
