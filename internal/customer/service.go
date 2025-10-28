package customer

import (
    "context"
    "fmt"

    "github.com/google/uuid"
)

type Service struct {
    repo Repository
}

func NewService(repo Repository) *Service {
    return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, c *Customer) (*Customer, error) {
    if err := c.ValidateForCreate(); err != nil {
        return nil, err
    }
    return s.repo.Create(ctx, c)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Customer, error) {
    return s.repo.Get(ctx, id)
}

func (s *Service) List(ctx context.Context, kyc *KYCStatus, page, limit int) ([]Customer, int, error) {
    if page <= 0 {
        page = 1
    }
    if limit <= 0 || limit > 200 {
        limit = 20
    }
    offset := (page - 1) * limit
    return s.repo.List(ctx, kyc, offset, limit)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, upd UpdateCustomer) (*Customer, error) {
    return s.repo.Update(ctx, id, upd)
}

func (s *Service) UpdateKYC(ctx context.Context, id uuid.UUID, status KYCStatus) (*Customer, error) {
    if !status.Valid() {
        return nil, fmt.Errorf("invalid KYC status")
    }
    return s.repo.UpdateKYC(ctx, id, status)
}

func (s *Service) SoftDelete(ctx context.Context, id uuid.UUID) error {
    return s.repo.SoftDelete(ctx, id)
}
