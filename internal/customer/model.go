package customer

import (
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nyaruka/phonenumbers"
)

// ------------------------------------------------------
// Customer model
// ------------------------------------------------------

type Customer struct {
	ID        uuid.UUID `json:"customer_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	PANNumber *string   `json:"pan_number,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var (
	ErrInvalidEmail = errors.New("invalid email")
	ErrInvalidName  = errors.New("invalid name")
	ErrInvalidPhone = errors.New("invalid phone")
)

// ValidateForCreate ensures required fields are valid for customer creation.
func (c *Customer) ValidateForCreate() error {
	if strings.TrimSpace(c.Name) == "" {
		return ErrInvalidName
	}
	if _, err := mail.ParseAddress(c.Email); err != nil {
		return ErrInvalidEmail
	}
	if strings.TrimSpace(c.Phone) == "" {
		return ErrInvalidPhone
	}

	// ✅ Validate phone number using phonenumbers library
	num, err := phonenumbers.Parse(c.Phone, "IN") // "IN" → default region for India
	if err != nil || !phonenumbers.IsValidNumber(num) {
		return ErrInvalidPhone
	}

	return nil
}

// ValidateForUpdate ensures only valid data is updated.
func (c *Customer) ValidateForUpdate() error {
	if c.Name != "" && strings.TrimSpace(c.Name) == "" {
		return ErrInvalidName
	}
	if c.Email != "" {
		if _, err := mail.ParseAddress(c.Email); err != nil {
			return ErrInvalidEmail
		}
	}

	if c.Phone != "" {
		num, err := phonenumbers.Parse(c.Phone, "IN") // "IN" = default region (India)
		if err != nil || !phonenumbers.IsValidNumber(num) {
			return ErrInvalidPhone
		}
	}
	return nil
}

// ------------------------------------------------------
// Verification model (linked to customer)
// ------------------------------------------------------

type VerificationStatus string

const (
	StatusPending VerificationStatus = "PENDING"
	StatusDone    VerificationStatus = "DONE"
)

// Verification table represents PAN/document verification records for customers.
type Verification struct {
	ID         uuid.UUID          `json:"id"`
	CustomerID uuid.UUID          `json:"customer_id"`
	PANNumber  *string            `json:"pan_number"`
	Status     VerificationStatus `json:"status"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}
