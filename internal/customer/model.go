package customer

import (
    "errors"
    "net/mail"
    "strings"
    "time"

    "github.com/google/uuid"
)

type KYCStatus string

const (
    KYCStatusPending  KYCStatus = "PENDING"
    KYCStatusVerified KYCStatus = "VERIFIED"
    KYCStatusRejected KYCStatus = "REJECTED"
)

func (s KYCStatus) Valid() bool {
    switch s {
    case KYCStatusPending, KYCStatusVerified, KYCStatusRejected:
        return true
    default:
        return false
    }
}

type Customer struct {
    ID        uuid.UUID `json:"customer_id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    Phone     string    `json:"phone"`
    KYCStatus KYCStatus `json:"kyc_status"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

var (
    ErrInvalidEmail = errors.New("invalid email")
    ErrInvalidName  = errors.New("invalid name")
    ErrInvalidPhone = errors.New("invalid phone")
    ErrInvalidKYC   = errors.New("invalid kyc status")
)

func (c *Customer) ValidateForCreate() error {
    if strings.TrimSpace(c.Name) == "" {
        return ErrInvalidName
    }
    if _, err := mail.ParseAddress(c.Email); err != nil {
        return ErrInvalidEmail
    }
    if len(strings.TrimSpace(c.Phone)) < 6 {
        return ErrInvalidPhone
    }
    if c.KYCStatus == "" {
        c.KYCStatus = KYCStatusPending
    }
    if !c.KYCStatus.Valid() {
        return ErrInvalidKYC
    }
    return nil
}

func (c *Customer) ValidateForUpdate() error {
    if c.Name != "" && strings.TrimSpace(c.Name) == "" {
        return ErrInvalidName
    }
    if c.Email != "" {
        if _, err := mail.ParseAddress(c.Email); err != nil {
            return ErrInvalidEmail
        }
    }
    if c.Phone != "" && len(strings.TrimSpace(c.Phone)) < 6 {
        return ErrInvalidPhone
    }
    return nil
}
