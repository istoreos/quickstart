package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQuickstartURLNormalizesRelativePath(t *testing.T) {
	base, err := normalizeBaseURL("192.168.30.244", "http")
	if err != nil {
		t.Fatalf("normalizeBaseURL error = %v", err)
	}

	got := quickstartURL(base, "/system/status/")
	want := "http://192.168.30.244/cgi-bin/luci/istore/system/status/"
	if got != want {
		t.Fatalf("quickstartURL = %q, want %q", got, want)
	}
}

func TestQuickstartURLKeepsFullIstorePath(t *testing.T) {
	base, err := normalizeBaseURL("https://router.local/", "http")
	if err != nil {
		t.Fatalf("normalizeBaseURL error = %v", err)
	}

	got := quickstartURL(base, "/cgi-bin/luci/istore/network/status/")
	want := "https://router.local/cgi-bin/luci/istore/network/status/"
	if got != want {
		t.Fatalf("quickstartURL = %q, want %q", got, want)
	}
}

func TestLoginStoresSysauthCookie(t *testing.T) {
	var sawUsername, sawPassword string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cgi-bin/luci/" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm error = %v", err)
		}
		sawUsername = r.Form.Get("luci_username")
		sawPassword = r.Form.Get("luci_password")
		http.SetCookie(w, &http.Cookie{Name: "sysauth", Value: "sid-1", Path: "/"})
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := newRouterClient(server.URL, "http", false)
	if err != nil {
		t.Fatalf("newRouterClient error = %v", err)
	}
	if err := client.login(context.Background(), "root", "secret"); err != nil {
		t.Fatalf("login error = %v", err)
	}
	if sawUsername != "root" || sawPassword != "secret" {
		t.Fatalf("login form = %q/%q, want root/secret", sawUsername, sawPassword)
	}
	if !client.hasAuthCookie() {
		t.Fatalf("expected auth cookie after login")
	}
}

func TestDoQuickstartRejectsNonZeroSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cgi-bin/luci/istore/system/reboot/" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": -1001,
			"error":   "Forbidden",
			"scope":   "system",
		})
	}))
	defer server.Close()

	client, err := newRouterClient(server.URL, "http", false)
	if err != nil {
		t.Fatalf("newRouterClient error = %v", err)
	}
	_, err = client.doQuickstart(context.Background(), "GET", "/system/reboot/", "")
	if err == nil {
		t.Fatalf("expected non-zero success to return an error")
	}
}

func TestDoQuickstartPostSendsJSONBody(t *testing.T) {
	var gotMethod, gotContentType string
	var gotBody map[string]bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		if r.URL.Path != "/cgi-bin/luci/istore/system/auto-check-update/" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("Decode body error = %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":0,"result":{"ok":true}}`))
	}))
	defer server.Close()

	client, err := newRouterClient(server.URL, "http", false)
	if err != nil {
		t.Fatalf("newRouterClient error = %v", err)
	}
	data, err := client.doQuickstart(context.Background(), "POST", "/system/auto-check-update/", `{"enable":true}`)
	if err != nil {
		t.Fatalf("doQuickstart error = %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want POST", gotMethod)
	}
	if gotContentType != "application/json;charset=utf-8" {
		t.Fatalf("Content-Type = %q, want application/json;charset=utf-8", gotContentType)
	}
	if !gotBody["enable"] {
		t.Fatalf("body = %#v, want enable true", gotBody)
	}
	if string(data) != `{"success":0,"result":{"ok":true}}` {
		t.Fatalf("data = %q", string(data))
	}
}

func TestDoQuickstartReturnsHTTPErrorWithBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	client, err := newRouterClient(server.URL, "http", false)
	if err != nil {
		t.Fatalf("newRouterClient error = %v", err)
	}
	data, err := client.doQuickstart(context.Background(), "GET", "/missing/", "")
	if err == nil {
		t.Fatalf("expected HTTP error")
	}
	if string(data) != "not found\n" {
		t.Fatalf("data = %q, want HTTP response body", string(data))
	}
}

func TestResolvePasswordFromEnv(t *testing.T) {
	got, err := resolvePassword("", "ROUTER_PASSWORD", func(name string) string {
		if name == "ROUTER_PASSWORD" {
			return "secret"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("resolvePassword error = %v", err)
	}
	if got != "secret" {
		t.Fatalf("password = %q, want secret", got)
	}
}

func TestRequireConfirmationBlocksPostWithoutWriteConfirmation(t *testing.T) {
	err := requireConfirmation(http.MethodPost, "/system/auto-check-update/", false, false, func(string) string { return "" })
	if err == nil {
		t.Fatalf("expected POST without write confirmation to fail")
	}
}

func TestRequireConfirmationAllowsPostWithWriteConfirmation(t *testing.T) {
	err := requireConfirmation(http.MethodPost, "/system/auto-check-update/", true, false, func(string) string { return "" })
	if err != nil {
		t.Fatalf("requireConfirmation error = %v", err)
	}
}

func TestRequireConfirmationBlocksDangerWithoutDangerConfirmation(t *testing.T) {
	err := requireConfirmation(http.MethodPost, "/system/reboot/", true, false, func(string) string { return "" })
	if err == nil {
		t.Fatalf("expected dangerous POST without danger confirmation to fail")
	}
}

func TestRequireConfirmationAllowsDangerWithEnvConfirmation(t *testing.T) {
	err := requireConfirmation(http.MethodPost, "/system/reboot/", true, false, func(name string) string {
		if name == "CONFIRM_QSCTL_DANGER" {
			return "YES"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("requireConfirmation error = %v", err)
	}
}
