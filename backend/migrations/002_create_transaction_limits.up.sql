CREATE TABLE transaction_limits (
    id BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id),   -- CHECK(id) allows only ONE row ever
    max_transfer_amount NUMERIC(19,2) NOT NULL CHECK (max_transfer_amount > 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'INR' CHECK (currency = 'INR'),
    updated_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO transaction_limits (max_transfer_amount) VALUES (100000.00);