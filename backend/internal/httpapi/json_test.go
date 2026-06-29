package httpapi

import (
	"net/http"
	"strings"
	"testing"
)

type decodeJSONFixture struct {
	Name string `json:"name"`
}

func TestDecodeJSONDecodesValidBody(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"name":"ok"}`))
	if err != nil {
		t.Fatal(err)
	}

	got, err := DecodeJSON[decodeJSONFixture](req)
	if err != nil {
		t.Fatalf("DecodeJSON returned error: %v", err)
	}
	if got.Name != "ok" {
		t.Fatalf("Name = %q, want ok", got.Name)
	}
}

func TestDecodeJSONRejectsMalformedBody(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"name":`))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := DecodeJSON[decodeJSONFixture](req); err == nil {
		t.Fatal("DecodeJSON returned nil error for malformed JSON")
	}
}

func TestDecodeJSONRejectsTrailingBody(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"name":"ok"} {"name":"extra"}`))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := DecodeJSON[decodeJSONFixture](req); err == nil {
		t.Fatal("DecodeJSON returned nil error for trailing JSON")
	}
}
