package httpio_test

import (
	"bonfire-api/internal/httpio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRespondJSON(t *testing.T) {
	t.Run("Success - valid data", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := map[string]string{"message": "success"}

		httpio.RespondJSON(w, http.StatusOK, data)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "success", resp["message"])
	})

	t.Run("Failure - unmarshalable data triggers fallback", func(t *testing.T) {
		w := httptest.NewRecorder()

		// Channels cannot be serialized into JSON, forcing an error in the encoder
		invalidData := make(chan int)

		httpio.RespondJSON(w, http.StatusOK, invalidData)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), `"message":"An unexpected error occurred."`)
	})

	t.Run("Buffer Reuse - ensures Reset() is working", func(t *testing.T) {
		// This test calls the function twice sequentially to ensure the
		// buffer pool reuse doesn't leak data from the first call to the second.

		// Call 1
		w1 := httptest.NewRecorder()
		httpio.RespondJSON(w1, http.StatusOK, map[string]string{"a": "b"})

		// Call 2
		w2 := httptest.NewRecorder()
		httpio.RespondJSON(w2, http.StatusOK, map[string]string{"c": "d"})

		assert.Equal(t, `{"a":"b"}`, strings.TrimSpace(w1.Body.String()))
		assert.Equal(t, `{"c":"d"}`, strings.TrimSpace(w2.Body.String()), "Second call should not contain data from first call")
	})
}
