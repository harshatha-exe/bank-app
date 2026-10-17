CREATE TABLE fund_transfers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reference_number VARCHAR(40) UNIQUE NOT NULL,
    source_account_id UUID NOT NULL REFERENCES bank_accounts(id),
    destination_account_id UUID NOT NULL REFERENCES bank_accounts(id),
    beneficiary_id UUID REFERENCES beneficiaries(id),
    initiated_by UUID NOT NULL REFERENCES users(id),
    amount NUMERIC(19,2) NOT NULL CHECK (amount > 0),
    currency VARCHAR(3) NOT NULL,
    purpose VARCHAR(255),
    status VARCHAR(20) NOT NULL CHECK (status IN ('PENDING', 'COMPLETED', 'FAILED')),
    review_status VARCHAR(20) NOT NULL CHECK (review_status IN ('NONE', 'UNDER_REVIEW', 'RESOLVED')),
    idempotency_key VARCHAR(100) NOT NULL,
    failure_code VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (initiated_by, idempotency_key),
    CHECK (source_account_id <> destination_account_id)
);

CREATE INDEX idx_fund_transfers_source_account_id ON fund_transfers (source_account_id);
CREATE INDEX idx_fund_transfers_destination_account_id ON fund_transfers (destination_account_id);
CREATE INDEX idx_fund_transfers_initiated_by ON fund_transfers (initiated_by);
CREATE INDEX idx_fund_transfers_created_at ON fund_transfers (created_at);