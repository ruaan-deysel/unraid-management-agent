package api

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/ruaan-deysel/unraid-management-agent/daemon/logger"
)

func corsMiddleware(allowedOrigin string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := allowedOrigin
			if origin == "" {
				origin = "*"
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// statusRecorder wraps http.ResponseWriter to capture the response status code.
// It preserves the http.Hijacker interface so that WebSocket upgrades still work.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

// Hijack delegates to the underlying ResponseWriter's Hijack method
// so that WebSocket upgrades (which require http.Hijacker) continue to work.
func (sr *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := sr.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not implement http.Hijacker")
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		logger.Debug("%s %s %d %v", r.Method, r.URL.Path, rec.status, time.Since(start))
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.LogPanicWithStack("HTTP handler", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// securityHeadersMiddleware adds standard security headers to every response.
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), camera=(), microphone=()")
		next.ServeHTTP(w, r)
	})
}

// bodySizeLimitMiddleware limits the maximum request body size to prevent
// memory exhaustion attacks (CWE-770).
const maxRequestBodySize int64 = 1 * 1024 * 1024 // 1 MB

func bodySizeLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
		}
		next.ServeHTTP(w, r)
	})
}

// csrfMiddleware validates the Origin header on state-changing requests to
// prevent cross-site request forgery. Non-browser clients that don't send
// an Origin header are allowed through.
func csrfMiddleware(allowedOrigin string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only check state-changing methods
			if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
				next.ServeHTTP(w, r)
				return
			}

			origin := r.Header.Get("Origin")
			if origin == "" {
				// Non-browser clients (curl, API tools) don't send Origin
				next.ServeHTTP(w, r)
				return
			}

			// Parse and validate origin
			parsed, err := url.Parse(origin)
			if err != nil {
				http.Error(w, "Forbidden: invalid origin", http.StatusForbidden)
				return
			}

			if allowedOrigin != "*" {
				// Strict mode: origin must match configured value
				if origin == allowedOrigin {
					next.ServeHTTP(w, r)
					return
				}
				http.Error(w, "Forbidden: origin not allowed", http.StatusForbidden)
				return
			}

			// Wildcard mode: verify origin host matches request host to
			// prevent drive-by CSRF from external websites.
			originHost := parsed.Hostname()
			requestHost := stripHostPort(r.Host)
			if originHost == requestHost {
				next.ServeHTTP(w, r)
				return
			}

			// Allow localhost aliases to match each other
			if isLocalhost(originHost) && isLocalhost(requestHost) {
				next.ServeHTTP(w, r)
				return
			}

			http.Error(w, "Forbidden: origin not allowed", http.StatusForbidden)
		})
	}
}

// stripHostPort removes the port portion from a host:port string.
func stripHostPort(hostport string) string {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport // already just a host
	}
	return host
}

// isLocalhost returns true if the host is a loopback address or "localhost".
func isLocalhost(host string) bool {
	switch strings.ToLower(host) {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}
