package httpresult

import (
	"errors"
	"net/http"
	"testing"
)

func TestIsOKReturnsTrueForHTTP200BodyContainingOK(t *testing.T) {
	t.Parallel()

	if !IsOK(http.StatusOK, []byte("OK\n"), nil) {
		t.Fatal("expected OK response to be accepted")
	}

	if !IsOK(http.StatusOK, []byte("prefix OK suffix"), nil) {
		t.Fatal("expected body containing OK to be accepted")
	}
}

func TestIsOKRejectsNonOKStatusReadErrorOrMissingMarker(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		status int
		body   []byte
		err    error
	}{
		{name: "bad status", status: http.StatusBadGateway, body: []byte("OK")},
		{name: "read error", status: http.StatusOK, body: []byte("OK"), err: errors.New("read failed")},
		{name: "missing marker", status: http.StatusOK, body: []byte("NO")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if IsOK(tc.status, tc.body, tc.err) {
				t.Fatalf("expected response to be rejected: %#v", tc)
			}
		})
	}
}
