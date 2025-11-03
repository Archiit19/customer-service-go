package customer

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Service handles all customer and verification operations
type Service struct {
	customerRepo *PGRepository
}

// NewService creates a new Service instance
func NewService(repo *PGRepository) *Service {
	return &Service{
		customerRepo: repo,
	}
}

// -------------------------
// Customer operations
// -------------------------

func (s *Service) Create(ctx context.Context, c *Customer) (*Customer, error) {
	if err := c.ValidateForCreate(); err != nil {
		return nil, err
	}
	return s.customerRepo.Create(ctx, c)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Customer, error) {
	return s.customerRepo.Get(ctx, id)
}

func (s *Service) List(ctx context.Context, page, limit int) ([]Customer, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	offset := (page - 1) * limit
	return s.customerRepo.List(ctx, offset, limit)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, name, email, phone *string) (*Customer, error) {
	upd := UpdateCustomer{
		Name:  name,
		Email: email,
		Phone: phone,
	}
	return s.customerRepo.Update(ctx, id, upd)
}

func (s *Service) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return s.customerRepo.SoftDelete(ctx, id)
}

// -------------------------
// Verification operations
// -------------------------

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
	return s.customerRepo.CreateVerification(ctx, v)
}

func (s *Service) GetVerificationByCustomerID(ctx context.Context, customerID string) (*Verification, error) {
	cid, err := uuid.Parse(customerID)
	if err != nil {
		return nil, err
	}
	return s.customerRepo.GetVerificationByCustomerID(ctx, cid)
}

func (s *Service) UpdateVerificationStatus(ctx context.Context, customerID, newStatus string) (*Verification, error) {
	cid, err := uuid.Parse(customerID)
	if err != nil {
		return nil, err
	}

	status := VerificationStatus(newStatus)
	if status != StatusPending && status != StatusDone {
		return nil, fmt.Errorf("invalid verification status")
	}

	if err := s.customerRepo.UpdateVerificationStatus(ctx, cid, status); err != nil {
		return nil, err
	}
	return s.customerRepo.GetVerificationByCustomerID(ctx, cid)
}
