package errors

import (
	"drawo/pkg/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"net/http"
	"reflect"
	"strings"
)

var errorsList map[string][]string

func Init() {
	errorsList = make(map[string][]string)
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

type TypedError struct {
	Error   error
	Field   string
	Message string
}

func HandleTypedError(sErr *TypedError) (status int, message *gin.H) {
	var c int
	var m *gin.H

	if sErr.Field == "" {
		m = &gin.H{"message": sErr.Message}
	} else {
		m = &gin.H{"message": gin.H{sErr.Field: []string{sErr.Message}}}
	}

	switch sErr.Error {
	case BadRequestErr:
		c = http.StatusBadRequest
	case UnauthorizedErr:
		c = http.StatusUnauthorized
	case InternalServerErr:
		c = http.StatusInternalServerError
		m = &gin.H{"message": InternalServerErr.Error()}
		fmt.Println(sErr.Error.Error(), ":", sErr.Message)
	}

	return c, m
}

func HandleJsonError(err error, obj any) (status int, message *gin.H) {
	Init()
	SetFromErrors(err, obj)
	validationErr := Get()

	if validationErr != nil && len(validationErr) > 0 {
		return http.StatusBadRequest, &gin.H{"message": validationErr}
	} else {
		return http.StatusBadRequest, &gin.H{"message": err.Error()}
	}
}
