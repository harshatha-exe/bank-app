ALTER TABLE fund_transfers ADD COLUMN IF NOT EXISTS customer_id UUID REFERENCES customers(id);
UPDATE fund_transfers f
SET customer_id = c.id
FROM bank_accounts a
JOIN customers c ON c.id = a.customer_id
WHERE f.customer_id IS NULL AND f.source_account_id = a.id;
ALTER TABLE fund_transfers ALTER COLUMN customer_id SET NOT NULL;
ALTER TABLE fund_transfers DROP CONSTRAINT IF EXISTS fund_transfers_initiated_by_idempotency_key_key;
ALTER TABLE fund_transfers ADD CONSTRAINT fund_transfers_customer_id_idempotency_key_key UNIQUE (customer_id, idempotency_key);
ALTER TABLE bank_accounts ALTER COLUMN balance TYPE NUMERIC(18,2);
ALTER TABLE fund_transfers ALTER COLUMN amount TYPE NUMERIC(18,2);
ALTER TABLE ledger_entries ALTER COLUMN amount TYPE NUMERIC(18,2);
ALTER TABLE ledger_entries ALTER COLUMN balance_after TYPE NUMERIC(18,2);
ALTER TABLE transactions ALTER COLUMN amount TYPE NUMERIC(18,2);
ALTER TABLE transactions ALTER COLUMN balance_after TYPE NUMERIC(18,2);
