package httpapi

import (
	"context"
	"net/http"
	"runtime/debug"
	"time"

	"solo-0014-zone-rollout-control/internal/domain"
)

type requestIDContextKey struct{}

type trackingWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *trackingWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *trackingWriter) Write(payload []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	count, err := w.ResponseWriter.Write(payload)
	w.bytes += count
	return count, err
}

func (s *Server) withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = s.ids.New("req")
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDContextKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		writer := &trackingWriter{ResponseWriter: w}
		next.ServeHTTP(writer, r)
		status := writer.status
		if status == 0 {
			status = http.StatusOK
		}
		s.logger.Info("request complete",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"bytes", writer.bytes,
			"duration_ms", time.Since(started).Milliseconds(),
			"request_id", requestID(r.Context()),
		)
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logger.Error("request panic",
					"panic", recovered,
					"stack", string(debug.Stack()),
					"request_id", requestID(r.Context()),
				)
				sendError(w, r, domain.NewError(domain.CodeInternal, "request handling failed", nil))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func requestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDContextKey{}).(string)
	return value
}
