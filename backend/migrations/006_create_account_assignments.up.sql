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