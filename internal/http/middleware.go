package http

import (
	"net/http"
	"time"

	"github.com/Archiit19/customer-service-go/internal/logger"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.size += n
	return n, err
}

func WithRequestContext(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := middleware.GetReqID(r.Context())
			if requestID == "" {
				requestID = uuid.NewString()
			}
			ctx := logger.WithRequestID(r.Context(), requestID)
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			start := time.Now()
			defer func() {
				duration := time.Since(start)
				log.Info(ctx, "http request completed", logger.String("method", r.Method), logger.String("path", r.URL.Path), logger.Int("status", rw.status), logger.Int("bytes", rw.size), logger.Duration("duration", duration))
			}()
			log.Info(ctx, "http request started", logger.String("method", r.Method), logger.String("path", r.URL.Path), logger.String("remote_addr", r.RemoteAddr))
			next.ServeHTTP(rw, r.WithContext(ctx))
		})
	}
}

func Recovery(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					ctx := r.Context()
					log.Error(ctx, "http panic recovered", logger.Any("panic", rec))
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
