package customer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("customer not found")
	ErrConflict = errors.New("conflict: email or phone already exists")
)

type Repository interface {
	Create(ctx context.Context, c *Customer) (*Customer, error)
	Get(ctx context.Context, id uuid.UUID) (*Customer, error)
	List(ctx context.Context, kyc *KYCStatus, offset, limit int) ([]Customer, int, error)
	Update(ctx context.Context, id uuid.UUID, upd UpdateCustomer) (*Customer, error)
	UpdateKYC(ctx context.Context, id uuid.UUID, status KYCStatus) (*Customer, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

type UpdateCustomer struct {
	Name  *string
	Email *string
	Phone *string
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func (r *PGRepository) Create(ctx context.Context, c *Customer) (*Customer, error) {
	q := `
        INSERT INTO customers (id, name, email, phone, kyc_status)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id, name, email, phone, kyc_status, created_at, updated_at
    `
	c.ID = uuid.New()
	row := r.pool.QueryRow(ctx, q, c.ID, c.Name, strings.ToLower(c.Email), c.Phone, string(c.KYCStatus))
	var out Customer
	if err := row.Scan(&out.ID, &out.Name, &out.Email, &out.Phone, &out.KYCStatus, &out.CreatedAt, &out.UpdatedAt); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return &out, nil
}

func (r *PGRepository) Get(ctx context.Context, id uuid.UUID) (*Customer, error) {
	q := `
        SELECT id, name, email, phone, kyc_status, created_at, updated_at
        FROM customers
        WHERE id = $1 AND deleted_at IS NULL
    `
	var out Customer
	err := r.pool.QueryRow(ctx, q, id).Scan(&out.ID, &out.Name, &out.Email, &out.Phone, &out.KYCStatus, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *PGRepository) List(ctx context.Context, kyc *KYCStatus, offset, limit int) ([]Customer, int, error) {
	baseWhere := "WHERE deleted_at IS NULL"
	args := []any{}
	if kyc != nil && kyc.Valid() {
		baseWhere += " AND kyc_status = $1"
		args = append(args, string(*kyc))
	}

	countSQL := fmt.Sprintf(`SELECT count(*) FROM customers %s`, baseWhere)
	var total int
	if err := r.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// pagination
	if kyc != nil && kyc.Valid() {
		args = append(args, limit, offset)
	} else {
		args = append(args, limit, offset)
	}
	listSQL := fmt.Sprintf(`
        SELECT id, name, email, phone, kyc_status, created_at, updated_at
        FROM customers
        %s
        ORDER BY created_at DESC
        LIMIT $%d OFFSET $%d
    `, baseWhere, len(args)-1, len(args))

	rows, err := r.pool.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var res []Customer
	for rows.Next() {
		var c Customer
		if err := rows.Scan(&c.ID, &c.Name, &c.Email, &c.Phone, &c.KYCStatus, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		res = append(res, c)
	}
	if rows.Err() != nil {
		return nil, 0, rows.Err()
	}
	return res, total, nil
}

func (r *PGRepository) Update(ctx context.Context, id uuid.UUID, upd UpdateCustomer) (*Customer, error) {
	setParts := []string{}
	args := []any{}
	argi := 1

	if upd.Name != nil {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argi))
		args = append(args, *upd.Name)
		argi++
	}
	if upd.Email != nil {
		setParts = append(setParts, fmt.Sprintf("email = $%d", argi))
		args = append(args, strings.ToLower(*upd.Email))
		argi++
	}
	if upd.Phone != nil {
		setParts = append(setParts, fmt.Sprintf("phone = $%d", argi))
		args = append(args, *upd.Phone)
		argi++
	}
	setParts = append(setParts, fmt.Sprintf("updated_at = now()"))

	if len(setParts) == 0 {
		return r.Get(ctx, id) // nothing to update
	}

	q := fmt.Sprintf(`
        UPDATE customers
        SET %s
        WHERE id = $%d AND deleted_at IS NULL
        RETURNING id, name, email, phone, kyc_status, created_at, updated_at
    `, strings.Join(setParts, ", "), argi)
	args = append(args, id)

	var out Customer
	err := r.pool.QueryRow(ctx, q, args...).Scan(&out.ID, &out.Name, &out.Email, &out.Phone, &out.KYCStatus, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return &out, nil
}

func (r *PGRepository) UpdateKYC(ctx context.Context, id uuid.UUID, status KYCStatus) (*Customer, error) {
	q := `
        UPDATE customers
        SET kyc_status = $2, updated_at = now()
        WHERE id = $1 AND deleted_at IS NULL
        RETURNING id, name, email, phone, kyc_status, created_at, updated_at
    `
	var out Customer
	err := r.pool.QueryRow(ctx, q, id, string(status)).Scan(&out.ID, &out.Name, &out.Email, &out.Phone, &out.KYCStatus, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *PGRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	q := `
        UPDATE customers
        SET deleted_at = now(), updated_at = now()
        WHERE id = $1 AND deleted_at IS NULL
    `
	ct, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
