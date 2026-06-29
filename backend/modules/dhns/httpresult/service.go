package httpresult

import (
	"net/http"
	"strings"
)

func IsOK(statusCode int, body []byte, readErr error) bool {
	if readErr != nil {
		return false
	}
	if statusCode != http.StatusOK {
		return false
	}
	return strings.Contains(string(body), "OK")
}
