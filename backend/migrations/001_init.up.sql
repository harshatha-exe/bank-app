CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(30) UNIQUE NOT NULL CHECK (name IN ('CUSTOMER', 'OPERATIONS', 'ADMIN')),
    description TEXT
);

CREATE INDEX idx_roles_name ON roles (name);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role_id UUID NOT NULL REFERENCES roles(id),
    status VARCHAR(20) NOT NULL CHECK (status IN ('ACTIVE', 'DEACTIVATED')),
    force_password_change BOOLEAN NOT NULL DEFAULT FALSE,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_username ON users (username);
CREATE INDEX idx_users_role_id ON users (role_id);

CREATE TABLE customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES users(id),
    customer_number VARCHAR(30) UNIQUE NOT NULL,
    full_name VARCHAR(150) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_customers_user_id ON customers (user_id);

CREATE TABLE bank_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    account_number VARCHAR(30) UNIQUE NOT NULL,
    account_type VARCHAR(30) NOT NULL CHECK (account_type IN ('SAVINGS', 'CURRENT')),
    balance NUMERIC(19,2) NOT NULL CHECK (balance >= 0),
    currency VARCHAR(3) NOT NULL CHECK (currency = 'INR'),
    status VARCHAR(20) NOT NULL CHECK (status IN ('ACTIVE', 'FROZEN')),
    version BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_bank_accounts_customer_id ON bank_accounts (customer_id);
CREATE INDEX idx_bank_accounts_status ON bank_accounts (status);

CREATE TABLE beneficiaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    destination_account_id UUID NOT NULL REFERENCES bank_accounts(id),
    nickname VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('ACTIVE', 'DEACTIVATED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (customer_id, destination_account_id)
);

CREATE INDEX idx_beneficiaries_customer_id ON beneficiaries (customer_id);

CREATE TABLE account_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    account_id UUID NOT NULL REFERENCES bank_accounts(id),
    assigned_by UUID NOT NULL REFERENCES users(id),
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, account_id)
);

CREATE INDEX idx_account_assignments_user_id ON account_assignments (user_id);
CREATE INDEX idx_account_assignments_account_id ON account_assignments (account_id);

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

CREATE TABLE ledger_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transfer_id UUID NOT NULL REFERENCES fund_transfers(id),
    account_id UUID NOT NULL REFERENCES bank_accounts(id),
    entry_type VARCHAR(10) NOT NULL CHECK (entry_type IN ('DEBIT', 'CREDIT')),
    amount NUMERIC(19,2) NOT NULL CHECK (amount > 0),
    balance_after NUMERIC(19,2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ledger_entries_transfer_id ON ledger_entries (transfer_id);
CREATE INDEX idx_ledger_entries_account_id_created_at ON ledger_entries (account_id, created_at);

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES bank_accounts(id),
    transfer_id UUID NOT NULL REFERENCES fund_transfers(id),
    transaction_type VARCHAR(20) NOT NULL CHECK (transaction_type IN ('DEBIT', 'CREDIT')),
    amount NUMERIC(19,2) NOT NULL CHECK (amount > 0),
    currency VARCHAR(3) NOT NULL,
    description VARCHAR(255),
    balance_after NUMERIC(19,2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transactions_account_id_created_at ON transactions (account_id, created_at);
CREATE INDEX idx_transactions_transfer_id ON transactions (transfer_id);

CREATE TABLE audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id UUID REFERENCES users(id),
    event_type VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50),
    entity_id UUID,
    action VARCHAR(50) NOT NULL,
    result VARCHAR(30) NOT NULL,
    ip_address INET,
    correlation_id UUID NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_events_actor_user_id ON audit_events (actor_user_id);
CREATE INDEX idx_audit_events_event_type_created_at ON audit_events (event_type, created_at);
CREATE INDEX idx_audit_events_correlation_id ON audit_events (correlation_id);

CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_token_hash TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_sessions_session_token_hash ON sessions (session_token_hash);
CREATE INDEX idx_sessions_user_id ON sessions (user_id);
CREATE INDEX idx_sessions_expires_at ON sessions (expires_at);

INSERT INTO roles (id, name, description) VALUES
('00000000-0000-0000-0000-000000000001', 'CUSTOMER', 'Retail banking customer'),
('00000000-0000-0000-0000-000000000002', 'OPERATIONS', 'Bank operations user'),
('00000000-0000-0000-0000-000000000003', 'ADMIN', 'System administrator');

INSERT INTO users (id, username, password_hash, role_id, status, force_password_change)
VALUES
('10000000-0000-0000-0000-000000000001', 'demo-admin',
 '$argon2id$v=19$m=65536,t=3,p=4$XBZuxhlBdjEIA8YErz3Yhg$qACEeur7NKLfS3a31NapdPuNYFPjY8JP12YF1TJlZFs',
 '00000000-0000-0000-0000-000000000003', 'ACTIVE', FALSE),
('10000000-0000-0000-0000-000000000002', 'demo-operations',
 '$argon2id$v=19$m=65536,t=3,p=4$dPRaw4Y6y+56o8Q7gP0hpg$wh/06ICdfsYcOst2v3k3ijB+e3fFhmS//oxFHgBqc8U',
 '00000000-0000-0000-0000-000000000002', 'ACTIVE', FALSE),
('10000000-0000-0000-0000-000000000003', 'demo-customer',
 '$argon2id$v=19$m=65536,t=3,p=4$EPoRzaIK0fMrST0fIMWK1Q$EpMhLdSRInggfiLn9yN30Q+8sreC1ZBrn1BFpzZByWM',
 '00000000-0000-0000-0000-000000000001', 'ACTIVE', FALSE),
('10000000-0000-0000-0000-000000000004', 'demo-customer-b',
 '$argon2id$v=19$m=65536,t=3,p=4$EPoRzaIK0fMrST0fIMWK1Q$EpMhLdSRInggfiLn9yN30Q+8sreC1ZBrn1BFpzZByWM',
 '00000000-0000-0000-0000-000000000001', 'ACTIVE', FALSE);

INSERT INTO customers (id, user_id, customer_number, full_name) VALUES
('20000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000003', 'CUST-DEMO-001', 'Asha Demo'),
('20000000-0000-0000-0000-000000000002', '10000000-0000-0000-0000-000000000004', 'CUST-DEMO-002', 'Bharat Demo');

INSERT INTO bank_accounts (id, customer_id, account_number, account_type, balance, currency, status)
VALUES
('30000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001', '******1234', 'SAVINGS', 100000.00, 'INR', 'ACTIVE'),
('30000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000001', '******5678', 'CURRENT', 50000.00, 'INR', 'ACTIVE'),
('30000000-0000-0000-0000-000000000003', '20000000-0000-0000-0000-000000000002', '******9876', 'SAVINGS', 75000.00, 'INR', 'ACTIVE'),
('30000000-0000-0000-0000-000000000004', '20000000-0000-0000-0000-000000000002', '******4321', 'SAVINGS', 25000.00, 'INR', 'ACTIVE');

INSERT INTO fund_transfers
    (id, reference_number, source_account_id, destination_account_id, initiated_by, amount, currency,
     purpose, status, review_status, idempotency_key, created_at, completed_at)
VALUES
('40000000-0000-0000-0000-000000000001', 'FT-DEMO-0001',
 '30000000-0000-0000-0000-000000000001', '30000000-0000-0000-0000-000000000003',
 '10000000-0000-0000-0000-000000000003', 1000.00, 'INR', 'Demo transfer', 'COMPLETED', 'NONE', 'seed-1',
 NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'),
('40000000-0000-0000-0000-000000000002', 'FT-DEMO-0002',
 '30000000-0000-0000-0000-000000000002', '30000000-0000-0000-0000-000000000004',
 '10000000-0000-0000-0000-000000000004', 500.00, 'INR', 'Demo transfer', 'COMPLETED', 'NONE', 'seed-2',
 NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day');

INSERT INTO ledger_entries (transfer_id, account_id, entry_type, amount, balance_after, created_at) VALUES
('40000000-0000-0000-0000-000000000001', '30000000-0000-0000-0000-000000000001', 'DEBIT', 1000.00, 100000.00, NOW() - INTERVAL '2 days'),
('40000000-0000-0000-0000-000000000001', '30000000-0000-0000-0000-000000000003', 'CREDIT', 1000.00, 75000.00, NOW() - INTERVAL '2 days'),
('40000000-0000-0000-0000-000000000002', '30000000-0000-0000-0000-000000000002', 'DEBIT', 500.00, 50000.00, NOW() - INTERVAL '1 day'),
('40000000-0000-0000-0000-000000000002', '30000000-0000-0000-0000-000000000004', 'CREDIT', 500.00, 25000.00, NOW() - INTERVAL '1 day');

INSERT INTO transactions
    (account_id, transfer_id, transaction_type, amount, currency, description, balance_after, created_at)
VALUES
('30000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000001', 'DEBIT', 1000.00, 'INR', 'Demo transfer', 100000.00, NOW() - INTERVAL '2 days'),
('30000000-0000-0000-0000-000000000003', '40000000-0000-0000-0000-000000000001', 'CREDIT', 1000.00, 'INR', 'Demo transfer', 75000.00, NOW() - INTERVAL '2 days'),
('30000000-0000-0000-0000-000000000002', '40000000-0000-0000-0000-000000000002', 'DEBIT', 500.00, 'INR', 'Demo transfer', 50000.00, NOW() - INTERVAL '1 day'),
('30000000-0000-0000-0000-000000000004', '40000000-0000-0000-0000-000000000002', 'CREDIT', 500.00, 'INR', 'Demo transfer', 25000.00, NOW() - INTERVAL '1 day');
