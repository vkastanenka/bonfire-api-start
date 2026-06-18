package validator_test

// import (
// 	"bonfire-api/internal/apperr"
// 	"bonfire-api/internal/validator"
// 	"errors"
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// )

// type testStruct struct {
// 	Name     string `json:"name" validate:"required"`
// 	Email    string `json:"email" validate:"required,email"`
// 	Age      int    `json:"age" validate:"min=18"`
// 	Username string `json:"username" validate:"alphanum"`
// 	Address  struct {
// 		City string `json:"city" validate:"required"`
// 	} `json:"address"`
// }

// func TestValidator_ValidateStruct(t *testing.T) {
// 	v := validator.New()

// 	tests := []struct {
// 		name          string
// 		input         interface{}
// 		expectErr     bool
// 		expectedCode  apperr.Code
// 		expectedField string // The field expected to have an error
// 	}{
// 		{
// 			name: "Success - valid input",
// 			input: testStruct{
// 				Name:     "John Doe",
// 				Email:    "john@example.com",
// 				Age:      25,
// 				Username: "johndoe1",
// 			},
// 			expectErr: false,
// 		},
// 		{
// 			name: "Failure - required field empty",
// 			input: testStruct{
// 				Name: "", // Missing
// 			},
// 			expectErr:     true,
// 			expectedCode:  apperr.CodeInvalidInput,
// 			expectedField: "name",
// 		},
// 		{
// 			name: "Failure - whitespace field",
// 			input: testStruct{
// 				Name: "   ",
// 			},
// 			expectErr:     true,
// 			expectedCode:  apperr.CodeInvalidInput,
// 			expectedField: "name",
// 		},
// 		{
// 			name: "Failure - invalid email",
// 			input: testStruct{
// 				Name:  "John",
// 				Email: "not-an-email",
// 			},
// 			expectErr:     true,
// 			expectedCode:  apperr.CodeInvalidInput,
// 			expectedField: "email",
// 		},
// 		{
// 			name: "Failure - min constraint",
// 			input: testStruct{
// 				Name:  "John",
// 				Email: "john@test.com",
// 				Age:   10,
// 			},
// 			expectErr:     true,
// 			expectedCode:  apperr.CodeInvalidInput,
// 			expectedField: "age",
// 		},
// 		{
// 			name: "Failure - nested struct validation",
// 			input: testStruct{
// 				Name:  "John",
// 				Email: "john@test.com",
// 				Age:   20,
// 				Address: struct {
// 					City string `json:"city" validate:"required"`
// 				}{City: ""},
// 			},
// 			expectErr:     true,
// 			expectedCode:  apperr.CodeInvalidInput,
// 			expectedField: "address.city",
// 		},
// 		{
// 			name:         "Failure - invalid validation target (non-struct)",
// 			input:        "this is a string, not a struct",
// 			expectErr:    true,
// 			expectedCode: apperr.CodeInternal,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			err := v.ValidateStruct(tt.input)

// 			if !tt.expectErr {
// 				assert.NoError(t, err)
// 				return
// 			}

// 			require.Error(t, err)

// 			// Verify error type is our custom domain error
// 			var appErr *apperr.Error
// 			assert.True(t, errors.As(err, &appErr), "Error should be of type apperr.Error")
// 			assert.Equal(t, tt.expectedCode, appErr.Code)

// 			// If specific field errors are expected, check them
// 			if tt.expectedField != "" {
// 				found := false
// 				for _, fe := range appErr.ValidationErrors {
// 					if fe.Field == tt.expectedField {
// 						found = true
// 						assert.NotEmpty(t, fe.Message, "Error message should not be empty")
// 					}
// 				}
// 				assert.True(t, found, "Expected field %s not found in validation errors", tt.expectedField)
// 			}
// 		})
// 	}
// }
