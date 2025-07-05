package decoder

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DecodeJSONBody is a helper function to decode JSON request bodies with robust error handling.
// It checks for unknown fields, syntax errors, type mismatches, and empty bodies.
func DecodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	// Use http.MaxBytesReader to prevent overly large request bodies (e.g., 1MB).
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	// Create a new decoder and disallow unknown fields in the JSON.
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	// Attempt to decode the request body into the destination struct.
	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError

		switch {
		// Handle syntax errors (e.g., malformed JSON).
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf("Request body contains badly-formed JSON (at character %d)", syntaxError.Offset)
			http.Error(w, msg, http.StatusBadRequest)
			return err

		// Handle cases where a JSON value has the wrong type.
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				msg := fmt.Sprintf("Request body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
				http.Error(w, msg, http.StatusBadRequest)
				return err
			}
			msg := fmt.Sprintf("Request body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)
			http.Error(w, msg, http.StatusBadRequest)
			return err

		// Handle an empty request body.
		case errors.Is(err, io.EOF):
			msg := "Request body must not be empty"
			http.Error(w, msg, http.StatusBadRequest)
			return err

		// Handle the case of an unknown field in the JSON.
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			msg := fmt.Sprintf("Request body contains unknown key %s", fieldName)
			http.Error(w, msg, http.StatusBadRequest)
			return err

		// Handle cases where the destination is not a valid pointer. This is a developer error.
		case errors.As(err, &invalidUnmarshalError):
			panic(err)

		// For other decoding errors, return a generic bad request.
		default:
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return err
		}
	}

	// Check if the request body contains more than one JSON object.
	err = dec.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		msg := "request body must only contain a single JSON object"
		http.Error(w, msg, http.StatusBadRequest)
		return errors.New(msg)
	}

	return nil
}
