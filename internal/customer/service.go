package customer

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Service handles all customer and verification operations
type Service struct {
	customerRepo     Repository
	verificationRepo *VerificationRepository
}

// NewService creates a new Service instance
func NewService(cRepo Repository, vRepo *VerificationRepository) *Service {
	return &Service{
		customerRepo:     cRepo,
		verificationRepo: vRepo,
	}
}

// -------------------------
// Customer operations
// -------------------------

// Create adds a new customer after validation
func (s *Service) Create(ctx context.Context, c *Customer) (*Customer, error) {
	if err := c.ValidateForCreate(); err != nil {
		return nil, err
	}
	return s.customerRepo.Create(ctx, c)
}

// Get fetches a single customer by ID
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Customer, error) {
	return s.customerRepo.Get(ctx, id)
}

// List returns paginated list of customers
func (s *Service) List(ctx context.Context, page, limit int) ([]Customer, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	offset := (page - 1) * limit
	return s.customerRepo.List(ctx, nil, offset, limit)
}

// Update modifies customer details
func (s *Service) Update(ctx context.Context, id uuid.UUID, name, email, phone *string) (*Customer, error) {
	upd := UpdateCustomer{
		Name:  name,
		Email: email,
		Phone: phone,
	}
	return s.customerRepo.Update(ctx, id, upd)
}

// SoftDelete marks a customer as deleted (soft delete)
func (s *Service) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return s.customerRepo.SoftDelete(ctx, id)
}

// -------------------------
// Verification operations
// -------------------------

// CreateVerification creates a new verification record for a customer
func (s *Service) CreateVerification(ctx context.Context, customerID, pan string) (*Verification, error) {
	cid, err := uuid.Parse(customerID)
	if err != nil {
		return nil, err
	}

	v := &Verification{
		CustomerID: cid,
		PANNumber:  pan,
		Status:     StatusPending,
	}
	return s.verificationRepo.Create(ctx, v)
}

// GetVerificationByCustomerID retrieves a customer's verification info
func (s *Service) GetVerificationByCustomerID(ctx context.Context, customerID string) (*Verification, error) {
	cid, err := uuid.Parse(customerID)
	if err != nil {
		return nil, err
	}
	return s.verificationRepo.GetByCustomerID(ctx, cid)
}

// UpdateVerificationStatus updates a customer's verification status (e.g., PENDING → DONE)
func (s *Service) UpdateVerificationStatus(ctx context.Context, customerID, newStatus string) (*Verification, error) {
	cid, err := uuid.Parse(customerID)
	if err != nil {
		return nil, err
	}

	status := VerificationStatus(newStatus)
	if status != StatusPending && status != StatusDone {
		return nil, fmt.Errorf("invalid verification status")
	}

	if err := s.verificationRepo.UpdateStatus(ctx, cid, status); err != nil {
		return nil, err
	}
	return s.verificationRepo.GetByCustomerID(ctx, cid)
}
