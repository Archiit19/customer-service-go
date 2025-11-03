package http

import (
	"net/http"
	"time"

	"github.com/Archiit19/customer-service-go/internal/customer"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter configures all routes
func NewRouter(svc *customer.Service) http.Handler {
	r := chi.NewRouter()

	// 🔹 Basic middlewares
	r.Use(
		middleware.RequestID,
		middleware.RealIP,
		middleware.Logger,
		middleware.Recoverer,
		middleware.Timeout(60*time.Second),
	)

	// 🔹 Health check
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Post("/v1/customers", createCustomerHandler(svc))

	r.Delete("/v1/customers/{id}", deleteCustomerHandler(svc))

	r.Patch("/v1/customers/{id}", patchCustomerHandler(svc))

	r.Get("/v1/customers", listCustomersHandler(svc))

	r.Get("/v1/customers/{id}", getCustomerHandler(svc))

	r.Get("/v1/customers/{id}/status", getCustomerKYCStatusHandler(svc))

	r.Patch("/v1/customers/{id}/verification", updateKYCHandler(svc))

	return r
}
