package http

import (
	"net/http"
	"time"

	"github.com/Archiit19/customer-service-go/internal/customer"
	"github.com/Archiit19/customer-service-go/internal/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter configures all routes
func NewRouter(svc *customer.Service, log logger.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(
		middleware.RequestID,
		middleware.RealIP,
		WithRequestContext(log),
		Recovery(log),
		middleware.Timeout(60*time.Second),
	)

	h := NewHandler(svc, log)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		log.Info(r.Context(), "health check")
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Post("/v1/customers", h.CreateCustomer)
	r.Delete("/v1/customers/{id}", h.DeleteCustomer)
	r.Patch("/v1/customers/{id}", h.PatchCustomer)
	r.Get("/v1/customers", h.ListCustomers)
	r.Get("/v1/customers/{id}", h.GetCustomer)
	r.Get("/v1/customers/{id}/status", h.GetCustomerKYCStatus)
	r.Patch("/v1/customers/{id}/verification", h.UpdateKYC)
	return r
}
