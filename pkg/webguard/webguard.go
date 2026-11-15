// Package webguard protects locally-served HTTP APIs against cross-site and
// remote abuse. It is shared by the asf-web API server and the MCP server —
// both expose endpoints that can trigger credential-spending LLM tasks, so
// both carry the same guard:
//
//   - the TCP peer (RemoteAddr) must be loopback — headers can't spoof it
//   - the Host header must be loopback (defeats DNS rebinding)
//   - the Origin header, when present, must be loopback (defeats CSRF)
//   - requests with bodies must declare Content-Type: application/json
//   - bodies are capped at 1 MiB
//   - if ASF_API_TOKEN is set, every guarded request must instead carry
//     "Authorization: Bearer <token>" (the deliberate path for non-local use)
//
// A local reverse proxy that accepts remote traffic would still satisfy the
// loopback checks — any such deployment must set ASF_API_TOKEN.
package webguard

import (
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// isLoopback reports whether a Host header, Origin host, or RemoteAddr
// refers to this machine (localhost / 127.0.0.1 / ::1).
func isLoopback(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

func deny(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// Guard wraps next, applying the protections above to every request whose
// path starts with pathPrefix. Requests outside the prefix pass through.
func Guard(next http.Handler, pathPrefix string) http.Handler {
	token := os.Getenv("ASF_API_TOKEN")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, pathPrefix) {
			next.ServeHTTP(w, r)
			return
		}

		if token != "" {
			if r.Header.Get("Authorization") != "Bearer "+token {
				deny(w, http.StatusUnauthorized, "missing or invalid token")
				return
			}
		} else if !isLoopback(r.RemoteAddr) || !isLoopback(r.Host) {
			deny(w, http.StatusForbidden, "non-local requests require ASF_API_TOKEN")
			return
		}

		if origin := r.Header.Get("Origin"); origin != "" && token == "" {
			u, err := url.Parse(origin)
			if err != nil || !isLoopback(u.Host) {
				deny(w, http.StatusForbidden, "cross-origin request refused")
				return
			}
		}

		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				deny(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		}

		next.ServeHTTP(w, r)
	})
}
