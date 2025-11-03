package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/Archiit19/customer-service-go/internal/customer"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// -------------------------------
// POST /v1/customers
// -------------------------------
type createCustomerRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

func createCustomerHandler(svc *customer.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createCustomerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		c := &customer.Customer{
			Name:  req.Name,
			Email: req.Email,
			Phone: req.Phone,
		}

		created, err := svc.Create(r.Context(), c)
		if err != nil {
			switch {
			case errors.Is(err, customer.ErrInvalidEmail),
				errors.Is(err, customer.ErrInvalidName),
				errors.Is(err, customer.ErrInvalidPhone):
				writeError(w, http.StatusBadRequest, err.Error())
			case errors.Is(err, customer.ErrConflict):
				writeError(w, http.StatusConflict, err.Error())
			default:
				writeError(w, http.StatusInternalServerError, "internal error")
			}
			return
		}

		resp := map[string]any{
			"customer_id":      created.ID,
			"name":             created.Name,
			"email":            created.Email,
			"phone":            created.Phone,
			"created_at":       created.CreatedAt,
			"updated_at":       created.UpdatedAt,
			"status_url":       fmt.Sprintf("/v1/customers/%s/status", c.ID),
			"verification_url": fmt.Sprintf("/v1/customers/%s/verification", c.ID),
		}
		writeJSON(w, http.StatusCreated, resp)
	}
}

func getScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if s := r.Header.Get("X-Forwarded-Proto"); s != "" {
		return s
	}
	return "http"
}

// -------------------------------
// GET /v1/customers/{id}
// -------------------------------
func getCustomerHandler(svc *customer.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		c, err := svc.Get(r.Context(), id)
		if err != nil {
			if errors.Is(err, customer.ErrNotFound) {
				writeError(w, http.StatusNotFound, "not found")
			} else {
				writeError(w, http.StatusInternalServerError, "internal error")
			}
			return
		}

		resp := map[string]any{
			"customer_id":      c.ID,
			"name":             c.Name,
			"email":            c.Email,
			"phone":            c.Phone,
			"created_at":       c.CreatedAt,
			"updated_at":       c.UpdatedAt,
			"status_url":       fmt.Sprintf("/v1/customers/%s/status", c.ID),
			"verification_url": fmt.Sprintf("/v1/customers/%s/verification", c.ID),
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

// -------------------------------
// GET /v1/customers?status=&page=&limit=
// -------------------------------
func listCustomersHandler(svc *customer.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		page := 1
		limit := 20

		if v := q.Get("page"); v != "" {
			if _, err := fmtSscanf(v, &page); err != nil || page < 1 {
				writeError(w, http.StatusBadRequest, "invalid page")
				return
			}
		}
		if v := q.Get("limit"); v != "" {
			if _, err := fmtSscanf(v, &limit); err != nil || limit < 1 {
				writeError(w, http.StatusBadRequest, "invalid limit")
				return
			}
		}

		items, total, err := svc.List(r.Context(), page, limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		var out []map[string]any
		for _, c := range items {
			out = append(out, map[string]any{
				"customer_id":      c.ID,
				"name":             c.Name,
				"email":            c.Email,
				"phone":            c.Phone,
				"created_at":       c.CreatedAt,
				"updated_at":       c.UpdatedAt,
				"status_url":       fmt.Sprintf("/v1/customers/%s/status", c.ID),
				"verification_url": fmt.Sprintf("/v1/customers/%s/verification", c.ID),
			})
		}
		resp := map[string]any{
			"page":  page,
			"limit": limit,
			"total": total,
			"data":  out,
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

// -------------------------------
// helper: parse integer safely
// -------------------------------
func fmtSscanf(s string, dst *int) (int, error) {
	var n int
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, errors.New("not a number")
		}
		n = n*10 + int(s[i]-'0')
	}
	*dst = n
	return 1, nil
}

// -------------------------------
// PATCH /v1/customers/{id}
// -------------------------------
type patchCustomerRequest struct {
	Name  *string `json:"name,omitempty"`
	Email *string `json:"email,omitempty"`
	Phone *string `json:"phone,omitempty"`
}

func patchCustomerHandler(svc *customer.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var req patchCustomerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		updated, err := svc.Update(r.Context(), id, req.Name, req.Email, req.Phone)
		if err != nil {
			if errors.Is(err, customer.ErrNotFound) {
				writeError(w, http.StatusNotFound, "not found")
			} else {
				writeError(w, http.StatusInternalServerError, "internal error")
			}
			return
		}
		writeJSON(w, http.StatusOK, updated)
	}
}

// -------------------------------
// GET /v1/customers/{id}/status
// -------------------------------
func getCustomerKYCStatusHandler(svc *customer.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		verification, err := svc.GetVerificationByCustomerID(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "verification record not found")
			return
		}
		writeJSON(w, http.StatusOK, verification)
	}
}

// -------------------------------
// PATCH /v1/customers/{id}/verification
// -------------------------------
func updateKYCHandler(svc *customer.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var payload struct {
			PAN    string `json:"pan_number,omitempty"`
			Status string `json:"status,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		if payload.PAN != "" {
			v, err := svc.CreateVerification(r.Context(), id, payload.PAN)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, v)
			return
		}

		if payload.Status != "" {
			v, err := svc.UpdateVerificationStatus(r.Context(), id, payload.Status)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, v)
			return
		}

		writeError(w, http.StatusBadRequest, "nothing to update")
	}
}

// -------------------------------
// DELETE /v1/customers/{id}
// -------------------------------
func deleteCustomerHandler(svc *customer.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		if err := svc.SoftDelete(r.Context(), id); err != nil {
			if errors.Is(err, customer.ErrNotFound) {
				writeError(w, http.StatusNotFound, "not found")
			} else {
				writeError(w, http.StatusInternalServerError, "internal error")
			}
			return
		}
		writeJSON(w, http.StatusNoContent, nil)
	}
}
