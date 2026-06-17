package apperr_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bonfire-api/internal/apperr"
)

func TestNew_AndOptions(t *testing.T) {
	underlyingErr := errors.New("database connection failed")

	tests := []struct {
		name     string
		code     apperr.Code
		msg      string
		opts     []apperr.Option
		validate func(t *testing.T, err error)
	}{
		{
			name: "Success - Error Without Options",
			code: apperr.CodeNotFound,
			msg:  apperr.CodeNotFound.Message(),
			validate: func(t *testing.T, err error) {
				require.Error(t, err)
				var appErr *apperr.Error
				require.True(t, errors.As(err, &appErr))

				assert.Equal(t, appErr.Code, apperr.CodeNotFound)
				assert.Equal(t, appErr.Message, apperr.CodeNotFound.Message())
				assert.Nil(t, appErr.Err)
				assert.Nil(t, appErr.Details)
				assert.Empty(t, appErr.ValidationErrors)
			},
		},
		{
			name: "Success - Error With Wrapped Underlying Error",
			code: apperr.CodeInternal,
			msg:  apperr.CodeInternal.Message(),
			opts: []apperr.Option{
				apperr.WithErr(underlyingErr),
			},
			validate: func(t *testing.T, err error) {
				var appErr *apperr.Error
				require.True(t, errors.As(err, &appErr))
				assert.Equal(t, appErr.Err, underlyingErr)
			},
		},
		{
			name: "Success - Error With Details Chained",
			code: apperr.CodeBadRequest,
			msg:  apperr.CodeBadRequest.Message(),
			opts: []apperr.Option{
				apperr.WithDetails("user_id", 123),
				apperr.WithDetails("action", "delete"),
			},
			validate: func(t *testing.T, err error) {
				var appErr *apperr.Error
				require.True(t, errors.As(err, &appErr))
				require.NotNil(t, appErr.Details)
				assert.Equal(t, 123, appErr.Details["user_id"])
				assert.Equal(t, "delete", appErr.Details["action"])
			},
		},
		{
			name: "Success - Error With Validation Errors Mixed",
			code: apperr.CodeUnprocessableEntity,
			msg:  apperr.CodeUnprocessableEntity.Message(),
			opts: []apperr.Option{
				apperr.WithValidationErr("email", "must be a valid email"),
				apperr.WithValidationErrors([]apperr.ValidationError{
					{Field: "age", Message: "must be over 18"},
					{Field: "password", Message: "too short"},
				}),
			},
			validate: func(t *testing.T, err error) {
				var appErr *apperr.Error
				require.True(t, errors.As(err, &appErr))
				require.Len(t, appErr.ValidationErrors, 3)

				assert.Equal(t, "email", appErr.ValidationErrors[0].Field)
				assert.Equal(t, "must be a valid email", appErr.ValidationErrors[0].Message)

				assert.Equal(t, "age", appErr.ValidationErrors[1].Field)
				assert.Equal(t, "password", appErr.ValidationErrors[2].Field)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := apperr.New(tt.code, tt.msg, tt.opts...)
			tt.validate(t, err)
		})
	}
}

func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *apperr.Error
		expected string
	}{
		{
			name: "Format - Without Internal Error",
			err: &apperr.Error{
				Code:    apperr.CodeUnauthorized,
				Message: "Token expired",
			},
			expected: "[UNAUTHORIZED] Token expired",
		},
		{
			name: "Format - With Internal Error",
			err: &apperr.Error{
				Code:    "INTERNAL_DB",
				Message: "Query failed",
				Err:     errors.New("timeout parsing response"),
			},
			expected: "[INTERNAL_DB] Query failed: timeout parsing response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestErrorCode(t *testing.T) {
	appErr := apperr.New("CUSTOM_CODE", "Something broke")
	wrappedAppErr := fmt.Errorf("wrapped context: %w", appErr)

	tests := []struct {
		name     string
		err      error
		expected apperr.Code
	}{
		{
			name:     "Success - Nil Error",
			err:      nil,
			expected: "",
		},
		{
			name:     "Success - Direct App Error",
			err:      appErr,
			expected: "CUSTOM_CODE",
		},
		{
			name:     "Success - Wrapped App Error (errors.As works)",
			err:      wrappedAppErr,
			expected: "CUSTOM_CODE",
		},
		{
			name:     "Fallback - Standard Standard Error Returns Internal",
			err:      errors.New("standard go error"),
			expected: apperr.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := apperr.ErrorCode(tt.err)
			assert.Equal(t, tt.expected, code)
		})
	}
}

func TestError_IsCode(t *testing.T) {
	err := &apperr.Error{Code: "EXPECTED_CODE"}

	assert.True(t, err.IsCode("EXPECTED_CODE"), "Should return true for matching code")
	assert.False(t, err.IsCode("OTHER_CODE"), "Should return false for non-matching code")
}

func TestError_Unwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := apperr.New("CODE", "message", apperr.WithErr(cause))

	assert.True(t, errors.Is(err, cause), "errors.Is should traverse the wrapped error successfully")

	unwrapped := errors.Unwrap(err)
	assert.Equal(t, cause, unwrapped)
}

func TestError_ToResponse(t *testing.T) {
	err := &apperr.Error{
		Code:    "VALIDATION_FAILED",
		Message: "Input is invalid",
		Details: map[string]any{"attempt": 1},
		ValidationErrors: []apperr.ValidationError{
			{Field: "username", Message: "taken"},
		},
		Timestamp: "2026-06-17T16:54:27Z",
		RequestID: "req-123",
		TraceID:   "trace-456",
		Err:       errors.New("should be ignored"),
	}

	resp := err.ToResponse()

	assert.Equal(t, "VALIDATION_FAILED", resp.Code)
	assert.Equal(t, "Input is invalid", resp.Message)
	assert.Equal(t, err.Details, resp.Details)
	assert.Equal(t, err.ValidationErrors, resp.ValidationErrors)
	assert.Equal(t, "2026-06-17T16:54:27Z", resp.Timestamp)
	assert.Equal(t, "req-123", resp.RequestID)
	assert.Equal(t, "trace-456", resp.TraceID)
}
