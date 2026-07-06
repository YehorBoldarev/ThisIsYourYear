package http

import (
	"testing"

	"github.com/stretchr/testify/require"

	"kibou/thisisyouryear/internal/domain"
)

func TestValidateStruct_ReportsJSONFieldNames(t *testing.T) {
	err := validateStruct(RegisterRequest{Email: "not-an-email", DisplayName: "", Password: "short"})
	require.Error(t, err)
	require.ErrorIs(t, err, domain.ErrValidation)

	var ve *ValidationError
	require.ErrorAs(t, err, &ve)
	require.Contains(t, ve.Fields, "email", "field keys use the json tag name")
	require.Contains(t, ve.Fields, "displayName")
	require.Contains(t, ve.Fields, "password")
	require.Equal(t, "must be a valid email address", ve.Fields["email"])
}

func TestValidateStruct_Valid(t *testing.T) {
	err := validateStruct(RegisterRequest{
		Email:       "a@b.com",
		DisplayName: "Alice",
		Password:    "password123",
	})
	require.NoError(t, err)
}
