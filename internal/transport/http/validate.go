package http

import (
	"errors"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"

	"kibou/thisisyouryear/internal/domain"
)

// validate is the shared, concurrency-safe request validator. It reports field
// names using their JSON tag so error detail matches the wire contract.
var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name, _, _ := strings.Cut(fld.Tag.Get("json"), ",")
		if name == "-" {
			return ""
		}
		return name
	})
	return v
}

// ValidationError is a transport-level validation failure carrying per-field
// detail. It unwraps to domain.ErrValidation so the central error mapper treats
// it as a 422.
type ValidationError struct {
	msg    string
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	if e.msg != "" {
		return e.msg
	}
	return "validation failed"
}

func (e *ValidationError) Unwrap() error { return domain.ErrValidation }

// validateStruct validates v against its `validate` struct tags, returning a
// ValidationError with human-readable per-field messages on failure.
func validateStruct(v any) error {
	err := validate.Struct(v)
	if err == nil {
		return nil
	}
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		fields := make(map[string]string, len(verrs))
		for _, fe := range verrs {
			fields[fe.Field()] = describeRule(fe)
		}
		return &ValidationError{msg: "one or more fields are invalid", Fields: fields}
	}
	return &ValidationError{msg: "invalid request"}
}

func describeRule(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return "must be at least " + fe.Param() + " characters"
	case "max":
		return "must be at most " + fe.Param() + " characters"
	default:
		return "is invalid"
	}
}
