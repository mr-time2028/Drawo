package errors

import (
	"drawo/pkg/json"
	"errors"
	"github.com/go-playground/validator/v10"
	"reflect"
	"strings"
)

type ServiceError struct {
	Error   error
	Field   string
	Message string
}

var errorsList map[string][]string

func Init() {
	errorsList = make(map[string][]string)
}

func SetFromErrors(err error, obj interface{}) {
	var validationErrors validator.ValidationErrors

	if errors.As(err, &validationErrors) {
		structType := reflect.TypeOf(obj)

		for _, fieldError := range validationErrors {
			jsonTag := json.GetJSONTag(structType, fieldError.StructField())
			Add(jsonTag, GetErrorMessage(fieldError.Tag()))
		}
	}
}

func Add(key string, value string) {
	errorsList[strings.ToLower(key)] = append(errorsList[strings.ToLower(key)], value)
}

func GetErrorMessage(tag string) string {
	return ErrorMessages()[tag]
}

func Get() map[string][]string {
	return errorsList
}
