package httpio

import (
	"encoding/json"
	"net/http"
)

// DecodeJSON reads the request body and parses it into the target destination.
// If decoding fails, it automatically writes a standard 400 Bad Request JSON
// error to the client and returns false.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	err := json.NewDecoder(r.Body).Decode(dst)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request payload."})
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
