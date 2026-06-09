package httpio

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
)

// DecodeJSON reads the request body and parses it into the target destination.
// If decoding fails, it automatically writes a standard 400 Bad Request JSON
// error to the client and returns false.
func DecodeJSON(w http.ResponseWriter, r *http.Request, data interface{}) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1048576) // 1MB req body limit
	defer r.Body.Close()                             // Close network stream before DecodeJSON finishes

	dec := json.NewDecoder(r.Body) // Create decoder to read from stream
	dec.DisallowUnknownFields()    // Prevent unknown fields in the req body

	// dec.Decode(dst) parses JSON data into the struct
	if err := dec.Decode(data); err != nil {
		log.Printf("[ERROR] JSON decoding failed: %v\n", err)

		statusCode := http.StatusBadRequest
		msg := "Invalid request body."

		var maxBytesErr *http.MaxBytesError
		var syntaxErr *json.SyntaxError

		// Check if the body exceeded the MaxBytesReader limit
		if errors.As(err, &maxBytesErr) {
			statusCode = http.StatusRequestEntityTooLarge
			msg = "Request body too large. Limit is 1MB."
			// Check if the JSON syntax itself is broken (e.g., missing a comma)
		} else if errors.As(err, &syntaxErr) {
			msg = "Malformed request body JSON syntax."
			// Check if they passed an unexpected field (DisallowUnknownFields triggered)
		} else if err.Error() != "" && errors.Is(err, io.EOF) == false {
			msg = "Unknown field present in request body."
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]string{"error": msg})
		return false
	}
	return true
}

// RespondJSON wraps payload formatting and writes a JSON response to the client.
func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
