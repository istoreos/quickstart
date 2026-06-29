package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
)

func DecodeJSON[T any](r *http.Request) (T, error) {
	var req T
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return req, err
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return req, err
	}
	return req, nil
}
