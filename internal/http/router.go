package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/yourname/customer-service/internal/customer"
)

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

	// 🔹 Customer CRUD (flat definitions)
	r.Post("/api/v1/customers", createCustomerHandler(svc))
	r.Get("/api/v1/customers", listCustomersHandler(svc))
	r.Get("/api/v1/customers/{id}", getCustomerHandler(svc))
	r.Put("/api/v1/customers/{id}", updateCustomerHandler(svc))
	r.Delete("/api/v1/customers/{id}", deleteCustomerHandler(svc))
	r.Patch("/api/v1/customers/{id}/kyc", updateKYCHandler(svc))

	return r
}
