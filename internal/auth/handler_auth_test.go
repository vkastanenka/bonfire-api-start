package auth_test

// import (
// 	"bytes"
// 	"encoding/json"
// 	"net/http"
// 	"net/http/httptest"
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// 	"go.uber.org/mock/gomock"

// 	"bonfire-api/internal/apperr"
// 	"bonfire-api/internal/auth"
// )

// func TestAuthHandler_Register(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	// Assuming you have generated mocks for the Service and Validator interfaces
// 	mockService := auth.NewMockRegisterService(ctrl)
// 	mockValidator := auth.NewMockRequestValidator(ctrl)

// 	// Initialize the handler with mocked dependencies
// 	handler := auth.NewHandler(mockService, mockValidator)

// 	displayName := "TestUser"
// 	validReqBody := auth.RegisterRequest{
// 		Email:       "test@example.com",
// 		Username:    "testuser",
// 		Password:    "supersecret123",
// 		DisplayName: &displayName,
// 	}

// 	t.Run("Success - 201 Created", func(t *testing.T) {
// 		// 1. Mock Validation
// 		mockValidator.EXPECT().ValidateStruct(gomock.Any()).Return(nil)

// 		// 2. Mock Service Call
// 		mockService.EXPECT().
// 			Register(gomock.Any(), validReqBody).
// 			Return(nil)

// 		// 3. Setup Request
// 		bodyBytes, _ := json.Marshal(validReqBody)
// 		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(bodyBytes))
// 		req.Header.Set("Content-Type", "application/json")
// 		recorder := httptest.NewRecorder()

// 		// 4. Execute
// 		err := handler.Register(recorder, req)

// 		// 5. Assert
// 		require.NoError(t, err) // Your handler returns nil on success
// 		assert.Equal(t, http.StatusCreated, recorder.Code)

// 		var response map[string]string
// 		err = json.NewDecoder(recorder.Body).Decode(&response)
// 		require.NoError(t, err)
// 		assert.Equal(t, auth.RegisterOkMsg, response["message"])
// 	})

// 	t.Run("Failure - Invalid JSON Payload", func(t *testing.T) {
// 		// Sending malformed JSON
// 		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader([]byte(`{bad-json}`)))
// 		req.Header.Set("Content-Type", "application/json")
// 		recorder := httptest.NewRecorder()

// 		// Execute
// 		err := handler.Register(recorder, req)

// 		// Assert - Assuming httpio.DecodeJSON returns an error that bubbles up
// 		require.Error(t, err)
// 	})

// 	t.Run("Failure - Validation Error", func(t *testing.T) {
// 		mockErr := apperr.New(apperr.CodeBadRequest, "validation failed")

// 		mockValidator.EXPECT().ValidateStruct(gomock.Any()).Return(mockErr)

// 		bodyBytes, _ := json.Marshal(validReqBody)
// 		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(bodyBytes))
// 		req.Header.Set("Content-Type", "application/json")
// 		recorder := httptest.NewRecorder()

// 		// Execute
// 		err := handler.Register(recorder, req)

// 		// Assert
// 		require.Error(t, err)
// 		assert.Equal(t, mockErr, err)
// 	})

// 	t.Run("Failure - Service Returns Conflict", func(t *testing.T) {
// 		mockValidator.EXPECT().ValidateStruct(gomock.Any()).Return(nil)

// 		mockErr := apperr.New(apperr.CodeConflict, "username already taken")
// 		mockService.EXPECT().
// 			Register(gomock.Any(), validReqBody).
// 			Return(mockErr)

// 		bodyBytes, _ := json.Marshal(validReqBody)
// 		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(bodyBytes))
// 		req.Header.Set("Content-Type", "application/json")
// 		recorder := httptest.NewRecorder()

// 		// Execute
// 		err := handler.Register(recorder, req)

// 		// Assert
// 		require.Error(t, err)
// 		assert.Equal(t, mockErr, err)
// 	})
// }
