package httpio_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
)

// cancelableReader wraps a standard reader but honors context cancellation mid-stream
type cancelableReader struct {
	ctx context.Context
	r   io.Reader
}

func (cr cancelableReader) Read(p []byte) (int, error) {
	if err := cr.ctx.Err(); err != nil {
		return 0, err
	}
	return cr.r.Read(p)
}

type TestPayload struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestDecodeJSON(t *testing.T) {
	largePayload := `{"name": "` + strings.Repeat("a", 1048577) + `"}`

	tests := []struct {
		name           string
		body           string
		contentType    string
		setupCtx       func() (context.Context, context.CancelFunc)
		targetFactory  func() any
		expectedErr    bool
		expectedCode   apperr.Code
		expectedMsg    string
		expectedDetail map[string]string
		validate       func(t *testing.T, dst any)
	}{
		{
			name:          "Success - JSON Valid",
			body:          `{"name":"test","count":5}`,
			targetFactory: func() any { return &TestPayload{} },
			validate: func(t *testing.T, dst any) {
				payload := dst.(*TestPayload)
				assert.Equal(t, "test", payload.Name)
				assert.Equal(t, 5, payload.Count)
			},
		},
		{
			name:          "Failure - JSON Missing Content-Type",
			body:          `{"name":"test"}`,
			contentType:   "none",
			targetFactory: func() any { return &TestPayload{} },
			expectedErr:   true,
			expectedCode:  apperr.CodeUnsupportedMediaType,
			expectedMsg:   httpio.UnsupportedMediaTypeMsg,
		},
		{
			name:          "Failure - JSON Incorrect Media Type",
			body:          `{"name":"test"}`,
			contentType:   "application/xml",
			targetFactory: func() any { return &TestPayload{} },
			expectedErr:   true,
			expectedCode:  apperr.CodeUnsupportedMediaType,
			expectedMsg:   httpio.UnsupportedMediaTypeMsg,
		},
		{
			name:          "Failure - JSON Malformed Media Syntax",
			body:          `{"name":"test"}`,
			contentType:   "text/html; =invalid-syntax",
			targetFactory: func() any { return &TestPayload{} },
			expectedErr:   true,
			expectedCode:  apperr.CodeUnsupportedMediaType,
			expectedMsg:   httpio.UnsupportedMediaTypeMsg,
		},
		{
			name: "Failure - Context Cancelled Mid-Stream",
			body: `{"name":"test"}`,
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, func() {}
			},
			targetFactory: func() any { return &TestPayload{} },
			expectedErr:   true,
			expectedCode:  apperr.CodeClientClosedRequest,
			expectedMsg:   httpio.ClientClosedConnectionMsg,
		},
		{
			name: "Failure - Request Timeout Exceeded",
			body: `{"name":"test"}`,
			setupCtx: func() (context.Context, context.CancelFunc) {
				pastTime := time.Now().Add(-1 * time.Hour)
				return context.WithDeadline(context.Background(), pastTime)
			},
			targetFactory: func() any { return &TestPayload{} },
			expectedErr:   true,
			expectedCode:  apperr.CodeRequestTimeout,
			expectedMsg:   httpio.ReqTimeoutMsg,
		},
		{
			name:          "Failure - JSON Payload Too Large",
			body:          largePayload,
			targetFactory: func() any { return &TestPayload{} },
			expectedErr:   true,
			expectedCode:  apperr.CodePayloadTooLarge,
			expectedMsg:   httpio.PayloadTooLargeMsg,
		},
		{
			name:          "Failure - JSON Empty Body",
			body:          "",
			targetFactory: func() any { return &TestPayload{} },
			expectedErr:   true,
			expectedCode:  apperr.CodeInvalidInput,
			expectedMsg:   httpio.EmptyBodyMsg,
		},
		{
			name:          "Failure - JSON Malformed",
			body:          `{"name": "test", bad-json}`,
			targetFactory: func() any { return &TestPayload{} },
			expectedErr:   true,
			expectedCode:  apperr.CodeInvalidInput,
			expectedMsg:   httpio.MalformedJSONMsg,
		},
		{
			name:          "Failure - JSON Truncated",
			body:          `{"name": "test"`,
			targetFactory: func() any { return &TestPayload{} },
			expectedErr:   true,
			expectedCode:  apperr.CodeInvalidInput,
			expectedMsg:   httpio.TruncatedJSONMsg,
		},
		{
			name:          "Failure - JSON Type Mismatch (UnmarshalTypeError)",
			body:          `{"name": "test", "count": "five"}`,
			targetFactory: func() any { return &TestPayload{} },
			expectedErr:   true,
			expectedCode:  apperr.CodeInvalidInput,
			expectedDetail: map[string]string{
				"count": fmt.Sprintf(httpio.FieldTypeExpectationFmt, "int"),
			},
		},
		{
			name:          "Failure - JSON Unknown Field",
			body:          `{"name": "test", "count": 5, "admin": true}`,
			targetFactory: func() any { return &TestPayload{} },
			expectedErr:   true,
			expectedCode:  apperr.CodeInvalidInput,
			expectedMsg:   fmt.Sprintf(httpio.UnknownFieldFmt, "admin"),
		},
		{
			name: "Failure - Generic Fallback Internal Error",
			body: `{"name": "test"}`,
			targetFactory: func() any {
				// Passing a non-pointer target guarantees a json.InvalidUnmarshalError,
				// which correctly hits your default handler block.
				return TestPayload{}
			},
			expectedErr:  true,
			expectedCode: apperr.CodeInternal,
			expectedMsg:  httpio.DecodeErrMsg,
		},
		{
			name:          "Failure - Multiple JSON Values",
			body:          `{"name": "test"}{"name": "test2"}`,
			targetFactory: func() any { return &TestPayload{} },
			expectedErr:   true,
			expectedCode:  apperr.CodeInvalidInput,
			expectedMsg:   httpio.SingleValueRequiredMsg,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyReader io.Reader = strings.NewReader(tt.body)
			var ctx context.Context = context.Background()

			if tt.setupCtx != nil {
				var cancel context.CancelFunc
				ctx, cancel = tt.setupCtx()
				t.Cleanup(cancel)
				// Wrap body so read calls check the context lifecycle
				bodyReader = cancelableReader{ctx: ctx, r: bodyReader}
			}

			req := httptest.NewRequest(http.MethodPost, "/", bodyReader)
			req = req.WithContext(ctx)

			if tt.contentType != "none" {
				ct := tt.contentType
				if ct == "" {
					ct = "application/json"
				}
				req.Header.Set("Content-Type", ct)
			}

			recorder := httptest.NewRecorder()
			dst := tt.targetFactory()

			err := httpio.DecodeJSON(recorder, req, dst)

			if tt.expectedErr {
				require.Error(t, err)

				var appErr *apperr.Error
				require.True(t, errors.As(err, &appErr), "Expected error to be of type *apperr.Error")

				assert.Equal(t, tt.expectedCode, appErr.Code)

				if tt.expectedMsg != "" {
					assert.Contains(t, appErr.Message, tt.expectedMsg)
				}

				if tt.expectedDetail != nil {
					require.NotNil(t, appErr.Details)
					for k, v := range tt.expectedDetail {
						assert.Equal(t, v, appErr.Details[k])
					}
				}
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, dst)
				}
			}
		})
	}
}
