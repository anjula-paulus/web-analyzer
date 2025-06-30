package middleware

import (
	"log/slog"
	"net/http"
)

// NewCORSMiddleware middleware adds CORS headers
func NewCORSMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			if r.Method == "OPTIONS" {
				logger.Debug("CORS preflight request",
					"origin", r.Header.Get("Origin"),
					"remote_addr", r.RemoteAddr,
				)
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
