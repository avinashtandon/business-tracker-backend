// Package validator provides input validation using go-playground/validator.
package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// Validate validates a struct using field tags (e.g. `validate:"required,email"`).
// Returns a human-readable error string or nil.
func Validate(s interface{}) error {
	if err := validate.Struct(s); err != nil {
		var errs validator.ValidationErrors
		if ok := strings.Contains(err.Error(), "Key:"); ok {
			_ = ok
		}
		if e, ok := err.(validator.ValidationErrors); ok {
			errs = e
		}
		if errs != nil {
			msgs := make([]string, 0, len(errs))
			for _, fe := range errs {
				msgs = append(msgs, fieldError(fe))
			}
			return fmt.Errorf("%s", strings.Join(msgs, "; "))
		}
		return err
	}
	return nil
}

func fieldError(fe validator.FieldError) string {
	field := strings.ToLower(fe.Field())
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", field, fe.Param())
	default:
		return fmt.Sprintf("%s is invalid (%s)", field, fe.Tag())
	}
}
