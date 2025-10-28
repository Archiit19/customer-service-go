package http

import (
    "encoding/json"
    "errors"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"
    "github.com/yourname/customer-service/internal/customer"
)

// POST /customers
type createCustomerRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Phone string `json:"phone"`
    // kyc_status is optional at creation; defaults to PENDING
    KYCStatus *string `json:"kyc_status,omitempty"`
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
        if req.KYCStatus != nil {
            c.KYCStatus = customer.KYCStatus(*req.KYCStatus)
        }

        created, err := svc.Create(r.Context(), c)
        if err != nil {
            switch {
            case errors.Is(err, customer.ErrInvalidEmail),
                 errors.Is(err, customer.ErrInvalidName),
                 errors.Is(err, customer.ErrInvalidPhone),
                 errors.Is(err, customer.ErrInvalidKYC):
                writeError(w, http.StatusBadRequest, err.Error())
            case errors.Is(err, customer.ErrConflict):
                writeError(w, http.StatusConflict, err.Error())
            default:
                writeError(w, http.StatusInternalServerError, "internal error")
            }
            return
        }
        writeJSON(w, http.StatusCreated, created)
    }
}

// GET /customers/{id}
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
        writeJSON(w, http.StatusOK, c)
    }
}

// GET /customers?kyc_status=&page=&limit=
func listCustomersHandler(svc *customer.Service) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        q := r.URL.Query()

        var kyc *customer.KYCStatus
        if v := q.Get("kyc_status"); v != "" {
            ks := customer.KYCStatus(v)
            if !ks.Valid() {
                writeError(w, http.StatusBadRequest, "invalid kyc_status")
                return
            }
            kyc = &ks
        }

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

        items, total, err := svc.List(r.Context(), kyc, page, limit)
        if err != nil {
            writeError(w, http.StatusInternalServerError, "internal error")
            return
        }
        resp := map[string]any{
            "page":  page,
            "limit": limit,
            "total": total,
            "data":  items,
        }
        writeJSON(w, http.StatusOK, resp)
    }
}

// helper because fmt.Sscanf requires format; use simple parse
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

// PUT /customers/{id}
type updateCustomerRequest struct {
    Name  *string `json:"name,omitempty"`
    Email *string `json:"email,omitempty"`
    Phone *string `json:"phone,omitempty"`
}

func updateCustomerHandler(svc *customer.Service) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        idStr := chi.URLParam(r, "id")
        id, err := uuid.Parse(idStr)
        if err != nil {
            writeError(w, http.StatusBadRequest, "invalid id")
            return
        }
        var req updateCustomerRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            writeError(w, http.StatusBadRequest, "invalid JSON body")
            return
        }
        // basic validation via model
        tmp := customer.Customer{}
        if req.Name != nil {
            tmp.Name = *req.Name
        }
        if req.Email != nil {
            tmp.Email = *req.Email
        }
        if req.Phone != nil {
            tmp.Phone = *req.Phone
        }
        if err := tmp.ValidateForUpdate(); err != nil {
            writeError(w, http.StatusBadRequest, err.Error())
            return
        }

        updated, err := svc.Update(r.Context(), id, customer.UpdateCustomer{
            Name:  req.Name,
            Email: req.Email,
            Phone: req.Phone,
        })
        if err != nil {
            switch {
            case errors.Is(err, customer.ErrNotFound):
                writeError(w, http.StatusNotFound, "not found")
            case errors.Is(err, customer.ErrConflict):
                writeError(w, http.StatusConflict, "conflict")
            default:
                writeError(w, http.StatusInternalServerError, "internal error")
            }
            return
        }
        writeJSON(w, http.StatusOK, updated)
    }
}

// PATCH /customers/{id}/kyc
type kycUpdateRequest struct {
    KYCStatus string `json:"kyc_status"`
}

func updateKYCHandler(svc *customer.Service) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        idStr := chi.URLParam(r, "id")
        id, err := uuid.Parse(idStr)
        if err != nil {
            writeError(w, http.StatusBadRequest, "invalid id")
            return
        }
        var req kycUpdateRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            writeError(w, http.StatusBadRequest, "invalid JSON body")
            return
        }
        status := customer.KYCStatus(req.KYCStatus)
        if !status.Valid() {
            writeError(w, http.StatusBadRequest, "invalid kyc_status")
            return
        }
        updated, err := svc.UpdateKYC(r.Context(), id, status)
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

// DELETE /customers/{id}
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
