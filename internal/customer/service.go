package customer

import (
	"context"
	"fmt"

	"github.com/Archiit19/customer-service-go/internal/logger"
	"github.com/google/uuid"
)

// Service handles all customer and verification operations
type Service struct {
	customerRepo Repository
	logger       logger.Logger
}

// NewService creates a new Service instance
func NewService(repo Repository, log logger.Logger) *Service {
	return &Service{
		customerRepo: repo,
		logger:       log,
	}
}

func (s *Service) Create(ctx context.Context, c *Customer) (*Customer, error) {
	s.logger.Info(ctx, "service create customer invoked")
	if err := c.ValidateForCreate(); err != nil {
		s.logger.Warn(ctx, "service create customer validation failed", logger.Err(err))
		return nil, err
	}
	customer, err := s.customerRepo.Create(ctx, c)
	if err != nil {
		s.logger.Error(ctx, "service create customer failed", logger.Err(err))
		return nil, err
	}
	s.logger.Info(ctx, "service create customer succeeded", logger.String("customer_id", customer.ID.String()))
	return customer, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Customer, error) {
	s.logger.Info(ctx, "service get customer invoked", logger.String("customer_id", id.String()))
	customer, err := s.customerRepo.Get(ctx, id)
	if err != nil {
		s.logger.Error(ctx, "service get customer failed", logger.Err(err), logger.String("customer_id", id.String()))
		return nil, err
	}
	s.logger.Debug(ctx, "service get customer succeeded", logger.String("customer_id", customer.ID.String()))
	return customer, nil
}

func (s *Service) List(ctx context.Context, page, limit int) ([]Customer, int, error) {
	s.logger.Info(ctx, "service list customers invoked", logger.Int("page", page), logger.Int("limit", limit))
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	offset := (page - 1) * limit
	items, total, err := s.customerRepo.List(ctx, offset, limit)
	if err != nil {
		s.logger.Error(ctx, "service list customers failed", logger.Err(err))
		return nil, 0, err
	}
	s.logger.Info(ctx, "service list customers succeeded", logger.Int("returned", len(items)), logger.Int("total", total))
	return items, total, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, name, email, phone *string) (*Customer, error) {
	s.logger.Info(ctx, "service update customer invoked", logger.String("customer_id", id.String()))
	upd := UpdateCustomer{
		Name:  name,
		Email: email,
		Phone: phone,
	}
	customer, err := s.customerRepo.Update(ctx, id, upd)
	if err != nil {
		s.logger.Error(ctx, "service update customer failed", logger.Err(err), logger.String("customer_id", id.String()))
		return nil, err
	}
	s.logger.Info(ctx, "service update customer succeeded", logger.String("customer_id", customer.ID.String()))
	return customer, nil
}

func (s *Service) SoftDelete(ctx context.Context, id uuid.UUID) error {
	s.logger.Info(ctx, "service soft delete customer invoked", logger.String("customer_id", id.String()))
	if err := s.customerRepo.SoftDelete(ctx, id); err != nil {
		s.logger.Error(ctx, "service soft delete customer failed", logger.Err(err), logger.String("customer_id", id.String()))
		return err
	}
	s.logger.Info(ctx, "service soft delete customer succeeded", logger.String("customer_id", id.String()))
	return nil
}

func (s *Service) CreateVerification(ctx context.Context, customerID, pan string) (*Verification, error) {
	s.logger.Info(ctx, "service create verification invoked", logger.String("customer_id", customerID))
	cid, err := uuid.Parse(customerID)
	if err != nil {
		s.logger.Warn(ctx, "service create verification invalid id", logger.Err(err), logger.String("customer_id", customerID))
		return nil, err
	}

	v := &Verification{
		CustomerID: cid,
		PANNumber:  &pan,
		Status:     StatusPending,
	}
	verification, err := s.customerRepo.CreateVerification(ctx, v)
	if err != nil {
		s.logger.Error(ctx, "service create verification failed", logger.Err(err), logger.String("customer_id", customerID))
		return nil, err
	}
	s.logger.Info(ctx, "service create verification succeeded", logger.String("verification_id", verification.ID.String()), logger.String("customer_id", customerID))
	return verification, nil
}

func (s *Service) GetVerificationByCustomerID(ctx context.Context, customerID string) (*Verification, error) {
	s.logger.Info(ctx, "service get verification invoked", logger.String("customer_id", customerID))
	cid, err := uuid.Parse(customerID)
	if err != nil {
		s.logger.Warn(ctx, "service get verification invalid id", logger.Err(err), logger.String("customer_id", customerID))
		return nil, err
	}
	verification, err := s.customerRepo.GetVerificationByCustomerID(ctx, cid)
	if err != nil {
		s.logger.Error(ctx, "service get verification failed", logger.Err(err), logger.String("customer_id", customerID))
		return nil, err
	}
	s.logger.Debug(ctx, "service get verification succeeded", logger.String("verification_id", verification.ID.String()), logger.String("customer_id", customerID))
	return verification, nil
}

func (s *Service) UpdateVerificationStatus(ctx context.Context, customerID, newStatus string) (*Verification, error) {
	s.logger.Info(ctx, "service update verification status invoked", logger.String("customer_id", customerID), logger.String("status", newStatus))
	cid, err := uuid.Parse(customerID)
	if err != nil {
		s.logger.Warn(ctx, "service update verification invalid id", logger.Err(err), logger.String("customer_id", customerID))
		return nil, err
	}

	status := VerificationStatus(newStatus)
	if !IsValidStatus(VerificationStatus(status)) {
		s.logger.Warn(ctx, "service update verification invalid status", logger.String("status", newStatus))
		return nil, fmt.Errorf("invalid verification status")
	}

	if err := s.customerRepo.UpdateVerificationStatus(ctx, cid, status); err != nil {
		s.logger.Error(ctx, "service update verification status failed", logger.Err(err), logger.String("customer_id", customerID))
		return nil, err
	}
	verification, err := s.customerRepo.GetVerificationByCustomerID(ctx, cid)
	if err != nil {
		s.logger.Error(ctx, "service get verification after update failed", logger.Err(err), logger.String("customer_id", customerID))
		return nil, err
	}
	s.logger.Info(ctx, "service update verification status succeeded", logger.String("verification_id", verification.ID.String()), logger.String("customer_id", customerID), logger.String("status", string(verification.Status)))
	return verification, nil
}
