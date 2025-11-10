package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/Archiit19/customer-service-go/internal/customer"
	"github.com/Archiit19/customer-service-go/internal/logger"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc    *customer.Service
	logger logger.Logger
}

type createCustomerRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

type patchCustomerRequest struct {
	Name  *string `json:"name,omitempty"`
	Email *string `json:"email,omitempty"`
	Phone *string `json:"phone,omitempty"`
}

func NewHandler(svc *customer.Service, log logger.Logger) *Handler {
	return &Handler{svc: svc, logger: log}
}

func (h *Handler) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.Info(ctx, "http create customer received")
	var req createCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn(ctx, "http create customer decode failed", logger.Err(err))
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	c := &customer.Customer{
		Name:  req.Name,
		Email: req.Email,
		Phone: req.Phone,
	}
	created, err := h.svc.Create(ctx, c)
	if err != nil {
		switch {
		case errors.Is(err, customer.ErrInvalidEmail), errors.Is(err, customer.ErrInvalidName), errors.Is(err, customer.ErrInvalidPhone):
			h.logger.Warn(ctx, "http create customer validation failed", logger.Err(err))
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, customer.ErrConflict):
			h.logger.Warn(ctx, "http create customer conflict", logger.Err(err))
			writeError(w, http.StatusConflict, err.Error())
		default:
			h.logger.Error(ctx, "http create customer internal failure", logger.Err(err))
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
		"status_url":       fmt.Sprintf("/v1/customers/%s/status", created.ID),
		"verification_url": fmt.Sprintf("/v1/customers/%s/verification", created.ID),
	}
	h.logger.Info(ctx, "http create customer succeeded", logger.String("customer_id", created.ID.String()))
	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) GetCustomer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	h.logger.Info(ctx, "http get customer received", logger.String("customer_id", idStr))
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Warn(ctx, "http get customer invalid id", logger.Err(err), logger.String("customer_id", idStr))
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	cust, err := h.svc.Get(ctx, id)
	if err != nil {
		if errors.Is(err, customer.ErrNotFound) {
			h.logger.Warn(ctx, "http get customer not found", logger.String("customer_id", idStr))
			writeError(w, http.StatusNotFound, "not found")
		} else {
			h.logger.Error(ctx, "http get customer internal failure", logger.Err(err), logger.String("customer_id", idStr))
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}
	resp := map[string]any{
		"customer_id":      cust.ID,
		"name":             cust.Name,
		"email":            cust.Email,
		"phone":            cust.Phone,
		"created_at":       cust.CreatedAt,
		"updated_at":       cust.UpdatedAt,
		"pan_number":       cust.PANNumber,
		"status":           cust.Status,
		"status_url":       fmt.Sprintf("/v1/customers/%s/status", cust.ID),
		"verification_url": fmt.Sprintf("/v1/customers/%s/verification", cust.ID),
	}
	h.logger.Info(ctx, "http get customer succeeded", logger.String("customer_id", cust.ID.String()))
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListCustomers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	page := 1
	limit := 20
	if v := q.Get("page"); v != "" {
		if _, err := fmtSscanf(v, &page); err != nil || page < 1 {
			h.logger.Warn(ctx, "http list customers invalid page", logger.String("page", v))
			writeError(w, http.StatusBadRequest, "invalid page")
			return
		}
	}
	if v := q.Get("limit"); v != "" {
		if _, err := fmtSscanf(v, &limit); err != nil || limit < 1 {
			h.logger.Warn(ctx, "http list customers invalid limit", logger.String("limit", v))
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
	}
	h.logger.Info(ctx, "http list customers received", logger.Int("page", page), logger.Int("limit", limit))
	items, total, err := h.svc.List(ctx, page, limit)
	if err != nil {
		h.logger.Error(ctx, "http list customers internal failure", logger.Err(err))
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
	h.logger.Info(ctx, "http list customers succeeded", logger.Int("returned", len(out)), logger.Int("total", total))
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) PatchCustomer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	h.logger.Info(ctx, "http patch customer received", logger.String("customer_id", idStr))
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Warn(ctx, "http patch customer invalid id", logger.Err(err), logger.String("customer_id", idStr))
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req patchCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn(ctx, "http patch customer decode failed", logger.Err(err))
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	updated, err := h.svc.Update(ctx, id, req.Name, req.Email, req.Phone)
	if err != nil {
		if errors.Is(err, customer.ErrNotFound) {
			h.logger.Warn(ctx, "http patch customer not found", logger.String("customer_id", idStr))
			writeError(w, http.StatusNotFound, "not found")
		} else if errors.Is(err, customer.ErrConflict) {
			h.logger.Warn(ctx, "http patch customer conflict", logger.Err(err), logger.String("customer_id", idStr))
			writeError(w, http.StatusConflict, err.Error())
		} else {
			h.logger.Error(ctx, "http patch customer internal failure", logger.Err(err), logger.String("customer_id", idStr))
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}
	h.logger.Info(ctx, "http patch customer succeeded", logger.String("customer_id", updated.ID.String()))
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) DeleteCustomer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	h.logger.Info(ctx, "http delete customer received", logger.String("customer_id", idStr))
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Warn(ctx, "http delete customer invalid id", logger.Err(err), logger.String("customer_id", idStr))
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.svc.SoftDelete(ctx, id); err != nil {
		if errors.Is(err, customer.ErrNotFound) {
			h.logger.Warn(ctx, "http delete customer not found", logger.String("customer_id", idStr))
			writeError(w, http.StatusNotFound, "not found")
		} else {
			h.logger.Error(ctx, "http delete customer internal failure", logger.Err(err), logger.String("customer_id", idStr))
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}
	h.logger.Info(ctx, "http delete customer succeeded", logger.String("customer_id", idStr))
	writeJSON(w, http.StatusNoContent, nil)
}

func (h *Handler) GetCustomerKYCStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	h.logger.Info(ctx, "http get verification status received", logger.String("customer_id", id))
	verification, err := h.svc.GetVerificationByCustomerID(ctx, id)
	if err != nil {
		h.logger.Warn(ctx, "http get verification status failed", logger.Err(err), logger.String("customer_id", id))
		writeError(w, http.StatusNotFound, "verification record not found")
		return
	}
	h.logger.Info(ctx, "http get verification status succeeded", logger.String("customer_id", id))
	writeJSON(w, http.StatusOK, verification)
}

func (h *Handler) UpdateKYC(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	h.logger.Info(ctx, "http update verification received", logger.String("customer_id", id))
	var payload struct {
		PAN    string `json:"pan_number,omitempty"`
		Status string `json:"status,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.logger.Warn(ctx, "http update verification decode failed", logger.Err(err))
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if payload.PAN != "" {
		verification, err := h.svc.CreateVerification(ctx, id, payload.PAN)
		if err != nil {
			h.logger.Error(ctx, "http create verification failed", logger.Err(err), logger.String("customer_id", id))
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		h.logger.Info(ctx, "http create verification succeeded", logger.String("verification_id", verification.ID.String()), logger.String("customer_id", id))
		writeJSON(w, http.StatusCreated, verification)
		return
	}
	if payload.Status != "" {
		current, err := h.svc.GetVerificationByCustomerID(ctx, id)
		if err != nil {
			h.logger.Warn(ctx, "http update verification status unable to load current record", logger.Err(err), logger.String("customer_id", id))
			writeError(w, http.StatusNotFound, "verification record not found")
			return
		}
		if current.PANNumber == nil || *current.PANNumber == "" {
			h.logger.Warn(ctx, "http update verification status rejected: missing PAN", logger.String("customer_id", id), logger.String("status", payload.Status))
			writeError(w, http.StatusBadRequest, "pan_number must be provided before updating status")
			return
		}

		verification, err := h.svc.UpdateVerificationStatus(ctx, id, payload.Status)
		if err != nil {
			h.logger.Error(ctx, "http update verification status failed", logger.Err(err), logger.String("customer_id", id), logger.String("status", payload.Status))
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		h.logger.Info(ctx, "http update verification status succeeded", logger.String("verification_id", verification.ID.String()), logger.String("customer_id", id), logger.String("status", payload.Status))
		writeJSON(w, http.StatusOK, verification)
		return
	}
	h.logger.Warn(ctx, "http update verification nothing to update", logger.String("customer_id", id))
	writeError(w, http.StatusBadRequest, "nothing to update")
}

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

func getScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if s := r.Header.Get("X-Forwarded-Proto"); s != "" {
		return s
	}
	return "http"
}
