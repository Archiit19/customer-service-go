-- Add index on deleted_at for soft-delete filtering
CREATE INDEX IF NOT EXISTS idx_customers_deleted_at
    ON customers (deleted_at);

-- Ensure each customer has exactly one verification record
ALTER TABLE verifications
    ADD CONSTRAINT unique_customer_verification
        UNIQUE (customer_id);

-- Ensure PAN number is unique across all customers
ALTER TABLE verifications
    ADD CONSTRAINT unique_pan_number
        UNIQUE (pan_number);

-- Index to quickly join customers → verifications
CREATE INDEX IF NOT EXISTS idx_verifications_customer_id
    ON verifications (customer_id);

-- Index on status for faster filtering in dashboards/queries
CREATE INDEX IF NOT EXISTS idx_verifications_status
    ON verifications (status);