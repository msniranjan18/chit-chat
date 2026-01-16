package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func LoggingMiddleware(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log request details
		logger.Debug("HTTP request received",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"query", r.URL.RawQuery)

		// Create a response writer wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call the next handler
		next.ServeHTTP(rw, r)

		// Log response details
		duration := time.Since(start)

		// Determine log level based on status code
		logLevel := slog.LevelInfo
		if rw.statusCode >= 400 && rw.statusCode < 500 {
			logLevel = slog.LevelWarn
		} else if rw.statusCode >= 500 {
			logLevel = slog.LevelError
		} else if rw.statusCode >= 200 && rw.statusCode < 300 {
			logLevel = slog.LevelInfo
		}

		// Log with appropriate level
		logger.LogAttrs(r.Context(), logLevel, "HTTP request completed",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rw.statusCode),
			slog.Duration("duration", duration),
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
		)

		// Log additional details for errors
		if rw.statusCode >= 400 {
			logger.LogAttrs(r.Context(), slog.LevelDebug, "Request details for error",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("query", r.URL.RawQuery),
				slog.Any("headers", getHeadersForLogging(r.Header)),
			)
		}
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Helper function to get headers for logging (sanitized)
func getHeadersForLogging(headers http.Header) map[string][]string {
	// Create a copy of headers for logging
	logHeaders := make(map[string][]string)

	// Filter out sensitive headers
	sensitiveHeaders := map[string]bool{
		"authorization":  true,
		"cookie":         true,
		"set-cookie":     true,
		"x-api-key":      true,
		"x-access-token": true,
	}

	for key, values := range headers {
		lowerKey := http.CanonicalHeaderKey(key)
		if !sensitiveHeaders[lowerKey] {
			logHeaders[lowerKey] = values
		} else {
			// Replace sensitive headers with masked value
			logHeaders[lowerKey] = []string{"[MASKED]"}
		}
	}

	return logHeaders
}
