package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

var ErrJsonEncode = errors.New("json encode failed")

// WriteJsonResponseBody marshals the given content and makes a JSON response body with the given status code.
// It has two errors: ErrJsonEncode for JSON encoding error and a normal error for network error.
// It is preferrable to set http.Error() only to ErrJsonEncode and not to normal error, since it already sends a response
// when the normal error occurs.
func WriteJsonResponseBody(w http.ResponseWriter, content any, statusCode int) error {
	body, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrJsonEncode, err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, err = w.Write(body)
	if err != nil {
		return err
	}
	return nil
}
