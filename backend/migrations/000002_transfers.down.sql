ALTER TABLE transactions ALTER COLUMN balance_after TYPE NUMERIC(19,2);
ALTER TABLE transactions ALTER COLUMN amount TYPE NUMERIC(19,2);
ALTER TABLE ledger_entries ALTER COLUMN balance_after TYPE NUMERIC(19,2);
ALTER TABLE ledger_entries ALTER COLUMN amount TYPE NUMERIC(19,2);
ALTER TABLE fund_transfers ALTER COLUMN amount TYPE NUMERIC(19,2);
ALTER TABLE bank_accounts ALTER COLUMN balance TYPE NUMERIC(19,2);
ALTER TABLE fund_transfers DROP CONSTRAINT IF EXISTS fund_transfers_customer_id_idempotency_key_key;
ALTER TABLE fund_transfers ADD CONSTRAINT fund_transfers_initiated_by_idempotency_key_key UNIQUE (initiated_by, idempotency_key);
ALTER TABLE fund_transfers DROP COLUMN IF EXISTS customer_id;
