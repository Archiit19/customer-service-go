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

// ----------------------------------------------------------------------
// Common errors
// ----------------------------------------------------------------------
var (
	ErrNotFound             = errors.New("customer not found")
	ErrConflict             = errors.New("conflict: email or phone already exists")
	ErrVerificationNotFound = errors.New("verification not found")
	ErrPANAlreadyExists     = errors.New("PAN already exists")
)

// ----------------------------------------------------------------------
// Repository interface
// ----------------------------------------------------------------------

type Repository interface {
	// Customer operations
	Create(ctx context.Context, c *Customer) (*Customer, error)
	Get(ctx context.Context, id uuid.UUID) (*Customer, error)
	List(ctx context.Context, offset, limit int) ([]Customer, int, error)
	Update(ctx context.Context, id uuid.UUID, upd UpdateCustomer) (*Customer, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error

	// Verification operations
	CreateVerification(ctx context.Context, v *Verification) (*Verification, error)
	GetVerificationByCustomerID(ctx context.Context, cid uuid.UUID) (*Verification, error)
	UpdateVerificationStatus(ctx context.Context, cid uuid.UUID, status VerificationStatus) error
}

// ----------------------------------------------------------------------
// PGRepository implements Repository using pgxpool
// ----------------------------------------------------------------------

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

// ----------------------------------------------------------------------
// Customer operations
// ----------------------------------------------------------------------

type UpdateCustomer struct {
	Name  *string
	Email *string
	Phone *string
}

// Check for unique constraint violation
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// Create a new customer
func (r *PGRepository) Create(ctx context.Context, c *Customer) (*Customer, error) {
	c.ID = uuid.New()
	q := `
		INSERT INTO customers (id, name, email, phone)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, email, phone, created_at, updated_at;
	`
	row := r.pool.QueryRow(ctx, q, c.ID, c.Name, strings.ToLower(c.Email), c.Phone)
	var out Customer
	if err := row.Scan(&out.ID, &out.Name, &out.Email, &out.Phone, &out.CreatedAt, &out.UpdatedAt); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}

	//  create corresponding verification record
	_, err := r.pool.Exec(ctx,
		`INSERT INTO verifications (customer_id, status, pan_number) VALUES ($1, 'NOT_FOUND', NULL);`,
		out.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create verification: %w", err)
	}

	out.Status = "NOT_FOUND"
	out.PANNumber = nil
	return &out, nil
}

// Get customer by ID
func (r *PGRepository) Get(ctx context.Context, id uuid.UUID) (*Customer, error) {
	q := `
		SELECT c.id, c.name, c.email, c.phone,
		       v.pan_number, v.status,
		       c.created_at, c.updated_at
		FROM customers c
		LEFT JOIN verifications v ON v.customer_id = c.id
		WHERE c.id = $1 AND c.deleted_at IS NULL;
	`
	var c Customer
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&c.ID, &c.Name, &c.Email, &c.Phone,
		&c.PANNumber, &c.Status,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &c, err
}

// List customers with pagination
func (r *PGRepository) List(ctx context.Context, offset, limit int) ([]Customer, int, error) {
	countSQL := `SELECT COUNT(*) FROM customers WHERE deleted_at IS NULL;`
	var total int
	if err := r.pool.QueryRow(ctx, countSQL).Scan(&total); err != nil {
		return nil, 0, err
	}

	q := `
		SELECT c.id, c.name, c.email, c.phone,
		       v.pan_number, v.status,
		       c.created_at, c.updated_at
		FROM customers c
		LEFT JOIN verifications v ON v.customer_id = c.id
		WHERE c.deleted_at IS NULL
		ORDER BY c.created_at DESC
		LIMIT $1 OFFSET $2;
	`
	rows, err := r.pool.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var res []Customer
	for rows.Next() {
		var c Customer
		if err := rows.Scan(
			&c.ID, &c.Name, &c.Email, &c.Phone,
			&c.PANNumber, &c.Status,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		res = append(res, c)
	}
	return res, total, nil
}

// Update customer details
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
	setParts = append(setParts, "updated_at = now()")

	if len(setParts) == 0 {
		return r.Get(ctx, id) // nothing to update
	}

	q := fmt.Sprintf(`
		UPDATE customers
		SET %s
		WHERE id = $%d AND deleted_at IS NULL
		RETURNING id, name, email, phone, created_at, updated_at;
	`, strings.Join(setParts, ", "), argi)
	args = append(args, id)

	var out Customer
	err := r.pool.QueryRow(ctx, q, args...).Scan(&out.ID, &out.Name, &out.Email, &out.Phone, &out.CreatedAt, &out.UpdatedAt)
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

// Soft delete (mark as deleted)
func (r *PGRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	q := `
		UPDATE customers
		SET deleted_at = now(), updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL;
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

// ----------------------------------------------------------------------
// Verification operations
// ----------------------------------------------------------------------

// Create verification record
func (r *PGRepository) CreateVerification(ctx context.Context, v *Verification) (*Verification, error) {
	q := `
		INSERT INTO verifications (customer_id, pan_number, status)
		VALUES ($1, $2, $3)
		RETURNING id, customer_id, pan_number, status, created_at, updated_at;
	`
	row := r.pool.QueryRow(ctx, q, v.CustomerID, v.PANNumber, v.Status)
	err := row.Scan(&v.ID, &v.CustomerID, &v.PANNumber, &v.Status, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}

// Get verification by customer ID
func (r *PGRepository) GetVerificationByCustomerID(ctx context.Context, cid uuid.UUID) (*Verification, error) {
	q := `
		SELECT id, customer_id, pan_number, status, created_at, updated_at
		FROM verifications
		WHERE customer_id=$1;
	`
	var v Verification
	err := r.pool.QueryRow(ctx, q, cid).Scan(&v.ID, &v.CustomerID, &v.PANNumber, &v.Status, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVerificationNotFound
		}
		return nil, err
	}
	return &v, nil
}

// Update verification status
func (r *PGRepository) UpdateVerificationStatus(ctx context.Context, cid uuid.UUID, status VerificationStatus) error {
	q := `
		UPDATE verifications
		SET status=$2, updated_at=now()
		WHERE customer_id=$1;
	`
	_, err := r.pool.Exec(ctx, q, cid, status)
	return err
}
