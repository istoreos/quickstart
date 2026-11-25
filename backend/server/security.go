package server

import (
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"github.com/istoreos/quickstart/backend/internal/httpapi"
	"github.com/istoreos/quickstart/backend/modules/homeconfig"
)

const maxRequestBody = 1024 * 1024

// Protect the assembled LuCI HTTP router, including legacy handlers which only
// check that a forwarded session header exists. The internal protocol is separate.
func secureHTTP(next http.Handler, authorize homeconfig.Authorize) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Add("Vary", "Origin, Sec-Fetch-Site")
		access, err := authorize(r)
		if err != nil || !access.Read || !access.Write {
			drainRejectedBody(r)
			httpapi.RespondErrorString(httpapi.ForbiddenError, "", "", "Forbidden", w)
			return
		}
		// LuCI injects the session token even for a cross-site HTML form. Its
		// presence is authentication, not proof of the browser request's origin.
		if !sameOriginRequest(r) {
			drainRejectedBody(r)
			http.Error(w, "cross-origin request forbidden", http.StatusForbidden)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || mediaType != "application/json" {
				drainRejectedBody(r)
				http.Error(w, "application/json required", http.StatusUnsupportedMediaType)
				return
			}
		}
		if r.ContentLength > maxRequestBody {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
		next.ServeHTTP(w, r)
	})
}

func drainRejectedBody(r *http.Request) {
	// The legacy LuCI proxy uploads the entire request before reading the
	// response. Closing the socket with unread bytes produces a reset and a LuCI 500
	// instead of the intended denial. Consume only a bounded amount of input;
	// the HTTP server's ReadTimeout also bounds slow rejected uploads.
	if r.Body != nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(r.Body, maxRequestBody))
	}
}

func sameOriginRequest(r *http.Request) bool {
	// Older LuCI proxies forward CGI names with underscores intact.
	for _, name := range []string{"Sec-Fetch-Site", "Sec_Fetch_Site"} {
		if site := r.Header.Get(name); site != "" && site != "same-origin" && site != "none" {
			return false
		}
	}
	// Older browsers may omit Origin. JSON-only writes still cannot be sent
	// by a simple cross-origin form; this server does not enable CORS.
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.Host == "" || u.RawQuery != "" || u.Fragment != "" || u.Path != "" || !strings.EqualFold(u.Host, r.Host) {
			return false
		}
	}
	return true
}
