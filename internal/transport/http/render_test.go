package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"kibou/thisisyouryear/internal/domain"
)

func TestRespondError_Mapping(t *testing.T) {
	cases := map[string]struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		"validation":          {fmt.Errorf("bad: %w", domain.ErrValidation), http.StatusUnprocessableEntity, "validation_error"},
		"invalid credentials": {domain.ErrInvalidCredentials, http.StatusUnauthorized, "unauthorized"},
		"unauthenticated":     {domain.ErrUnauthenticated, http.StatusUnauthorized, "unauthorized"},
		"not found":           {fmt.Errorf("x: %w", domain.ErrNotFound), http.StatusNotFound, "not_found"},
		"conflict":            {fmt.Errorf("x: %w", domain.ErrConflict), http.StatusConflict, "conflict"},
		"unclassified":        {errors.New("boom"), http.StatusInternalServerError, "internal_error"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			respondError(rec, nil, req, tc.err)

			require.Equal(t, tc.wantStatus, rec.Code)
			var body errorBody
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			require.Equal(t, tc.wantCode, body.Error.Code)
			require.NotEmpty(t, body.Error.Message)
		})
	}
}

func TestRespondError_UnclassifiedHidesDetail(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	respondError(rec, nil, req, errors.New("secret internal detail"))
	require.NotContains(t, rec.Body.String(), "secret internal detail")
}

func TestRespondError_ValidationCarriesFields(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	ve := &ValidationError{msg: "invalid", Fields: map[string]string{"email": "is required"}}
	respondError(rec, nil, req, ve)

	var body errorBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	require.Equal(t, "is required", body.Error.Fields["email"])
}

func TestRespondError_DomainFieldErrorCarriesField(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	err := fmt.Errorf("register: %w", &domain.FieldError{Field: "inviteCode", Message: "is not valid"})
	respondError(rec, nil, req, err)

	var body errorBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	require.Equal(t, "validation_error", body.Error.Code)
	require.Equal(t, "is not valid", body.Error.Fields["inviteCode"])
	require.True(t, domain.IsValidation(err))
}

func TestDecodeJSON_RejectsUnknownFields(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"nope": 1}`))
	var dst RegisterRequest
	err := decodeJSON(rec, req, &dst)
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestDecodeJSON_RejectsMalformed(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{not json`))
	var dst RegisterRequest
	err := decodeJSON(rec, req, &dst)
	require.ErrorIs(t, err, domain.ErrValidation)
}
