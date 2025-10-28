-- Customers table
CREATE TABLE IF NOT EXISTS customers (
    id UUID PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(20) NOT NULL,
    kyc_status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

-- Enforce allowed KYC values
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'kyc_status_check'
    ) THEN
        ALTER TABLE customers
            ADD CONSTRAINT kyc_status_check
            CHECK (kyc_status IN ('PENDING','VERIFIED','REJECTED'));
    END IF;
END$$;

-- Uniqueness (ignoring soft-deleted rows)
CREATE UNIQUE INDEX IF NOT EXISTS ux_customers_email ON customers (lower(email)) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS ux_customers_phone ON customers (phone) WHERE deleted_at IS NULL;
