package errors

import (
	"errors"
)

var (
	InternalServerErr = errors.New("internal server error")
	BadRequestErr     = errors.New("bad request")
	UnauthorizedErr   = errors.New("credentials are invalid or missing")
	ForbiddenErr      = errors.New("you do not have permission to access this resource")
)

func ErrorMessages() map[string]string {
	return map[string]string{
		"required": "This field is required",
		"min":      "Should be more than the limit",
		"max":      "Should be less that the limit",
	}
}
