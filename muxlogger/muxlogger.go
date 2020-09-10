package muxlogger

//inspired by: https://github.com/pytimer/mux-logrus

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type timer interface {
	Now() time.Time
	Since(time.Time) time.Duration
}

// realClock save request times
type realClock struct{}

func (rc *realClock) Now() time.Time {
	return time.Now()
}

func (rc *realClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

// LoggingMiddleware is a middleware handler that logs the request as it goes in and the response as it goes out.
type LoggingMiddleware struct {
	logger *logrus.Entry
	clock  timer
}

// NewLogger returns a new *LoggingMiddleware, yay!
func NewLogger(l *logrus.Entry) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: l,
		clock:  &realClock{},
	}
}

// realIP get the real IP from http request
func realIP(req *http.Request) string {
	ra := req.RemoteAddr
	if ip := req.Header.Get("X-Forwarded-For"); ip != "" {
		ra = strings.Split(ip, ", ")[0]
	} else if ip := req.Header.Get("X-Real-IP"); ip != "" {
		ra = ip
	} else {
		ra, _, _ = net.SplitHostPort(ra)
	}
	return ra
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

func (lw *loggingResponseWriter) WriteHeader(code int) {
	lw.statusCode = code
	lw.ResponseWriter.WriteHeader(code)
}

func (lw *loggingResponseWriter) Write(b []byte) (int, error) {
	return lw.ResponseWriter.Write(b)
}

// Middleware implement mux middleware interface
func (m *LoggingMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		entry := m.logger.WithFields(logrus.Fields{
			"path": r.URL.Path,
			"host": r.Host,
		})
		start := m.clock.Now()

		if remoteAddr := realIP(r); remoteAddr != "" {
			entry = entry.WithField("remoteAddr", remoteAddr)
		}

		lw := newLoggingResponseWriter(w)
		next.ServeHTTP(lw, r)

		latency := m.clock.Since(start)

		status := lw.statusCode
		entry = entry.WithFields(logrus.Fields{
			"status": status,
			"took":   latency,
		})
		if status < 499 {
			entry.Info("handled request")
		} else {
			entry.Warning("handled request")
		}
	})
}
