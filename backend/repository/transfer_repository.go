package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TransferRepository struct{ pool *pgxpool.Pool }

type TransferInput struct {
	SourceAccountID, DestinationAccountID, BeneficiaryID uuid.UUID
	AmountPaise                                          int64
	Purpose                                              string
	IdempotencyKey                                       string
	ReferenceNumber                                      string
}

type TransferRecord struct {
	ID, SourceAccountID, DestinationAccountID, BeneficiaryID, InitiatedBy uuid.UUID
	ReferenceNumber, Amount, Purpose, Status, ReviewStatus, Currency      string
	DestinationAccountNumber                                              string
	CreatedAt                                                             time.Time
}

type OperationsTransferFilter struct {
	Reference, Status, ReviewStatus string
	DateFrom, DateTo                *time.Time
	Page, PageSize                  int
}

func NewTransferRepository(pool *pgxpool.Pool) *TransferRepository {
	return &TransferRepository{pool: pool}
}

func (r *TransferRepository) Create(ctx context.Context, userID, correlationID uuid.UUID, in TransferInput, ip string) (TransferRecord, bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return TransferRecord{}, false, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT a.id,a.customer_id,a.balance,a.currency,a.status
		FROM bank_accounts a WHERE a.id IN ($1,$2) ORDER BY a.id FOR UPDATE`, in.SourceAccountID, in.DestinationAccountID)
	if err != nil {
		return TransferRecord{}, false, err
	}
	type account struct {
		id, customer     uuid.UUID
		balance          int64
		currency, status string
	}
	accounts := map[uuid.UUID]account{}
	for rows.Next() {
		var a account
		var balance string
		if err := rows.Scan(&a.id, &a.customer, &balance, &a.currency, &a.status); err != nil {
			rows.Close()
			return TransferRecord{}, false, err
		}
		var whole, frac int64
		if _, err := fmt.Sscanf(balance, "%d.%d", &whole, &frac); err != nil {
			rows.Close()
			return TransferRecord{}, false, err
		}
		a.balance = whole*100 + frac
		accounts[a.id] = a
	}
	rows.Close()
	source, sourceOK := accounts[in.SourceAccountID]
	destination, destinationOK := accounts[in.DestinationAccountID]
	if !sourceOK || !destinationOK {
		return TransferRecord{}, false, ErrNotFound
	}
	if source.customer == uuid.Nil {
		return TransferRecord{}, false, ErrNotFound
	}
	var owner uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT c.id FROM customers c WHERE c.user_id=$1`, userID).Scan(&owner); err != nil {
		return TransferRecord{}, false, ErrNotFound
	}
	if source.customer != owner {
		return TransferRecord{}, false, ErrNotFound
	}
	var existing TransferRecord
	err = tx.QueryRow(ctx, `SELECT f.id,f.source_account_id,f.destination_account_id,COALESCE(f.beneficiary_id,'00000000-0000-0000-0000-000000000000'),f.initiated_by,f.reference_number,f.amount::text,COALESCE(f.purpose,''),f.status,f.review_status,f.currency,d.account_number,f.created_at
		FROM fund_transfers f JOIN bank_accounts d ON d.id=f.destination_account_id
		WHERE f.initiated_by=$1 AND f.idempotency_key=$2`, userID, in.IdempotencyKey).
		Scan(&existing.ID, &existing.SourceAccountID, &existing.DestinationAccountID, &existing.BeneficiaryID, &existing.InitiatedBy, &existing.ReferenceNumber, &existing.Amount, &existing.Purpose, &existing.Status, &existing.ReviewStatus, &existing.Currency, &existing.DestinationAccountNumber, &existing.CreatedAt)
	if err == nil {
		if existing.SourceAccountID != in.SourceAccountID || existing.DestinationAccountID != in.DestinationAccountID ||
			existing.BeneficiaryID != in.BeneficiaryID || existing.Amount != fmt.Sprintf("%d.%02d", in.AmountPaise/100, in.AmountPaise%100) || existing.Purpose != in.Purpose {
			return TransferRecord{}, false, ErrIdempotencyConflict
		}
		return existing, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return TransferRecord{}, false, err
	}
	if source.status == "FROZEN" || destination.status == "FROZEN" {
		return TransferRecord{}, false, ErrFrozenAccount
	}
	if source.balance < in.AmountPaise {
		return TransferRecord{}, false, ErrInsufficientFunds
	}
	if in.BeneficiaryID != uuid.Nil {
		var beneficiaryCustomer, destinationID uuid.UUID
		var status string
		if err := tx.QueryRow(ctx, `SELECT customer_id,destination_account_id,status FROM beneficiaries WHERE id=$1`, in.BeneficiaryID).Scan(&beneficiaryCustomer, &destinationID, &status); err != nil {
			return TransferRecord{}, false, ErrNotFound
		}
		if beneficiaryCustomer != owner || destinationID != in.DestinationAccountID {
			return TransferRecord{}, false, ErrNotFound
		}
		if status != "ACTIVE" {
			return TransferRecord{}, false, ErrBeneficiaryInactive
		}
	}
	newSource, newDestination := source.balance-in.AmountPaise, destination.balance+in.AmountPaise
	if _, err = tx.Exec(ctx, `UPDATE bank_accounts SET balance=$1::numeric/100,version=version+1,updated_at=NOW() WHERE id=$2`, newSource, source.id); err != nil {
		return TransferRecord{}, false, err
	}
	if _, err = tx.Exec(ctx, `UPDATE bank_accounts SET balance=$1::numeric/100,version=version+1,updated_at=NOW() WHERE id=$2`, newDestination, destination.id); err != nil {
		return TransferRecord{}, false, err
	}
	var out TransferRecord
	err = tx.QueryRow(ctx, `INSERT INTO fund_transfers(reference_number,source_account_id,destination_account_id,beneficiary_id,initiated_by,amount,currency,purpose,status,review_status,idempotency_key,completed_at)
		VALUES($1,$2,$3,NULLIF($4::uuid,'00000000-0000-0000-0000-000000000000'::uuid),$5,$6,'INR',$7,'COMPLETED','NONE',$8,NOW())
		RETURNING id,source_account_id,destination_account_id,COALESCE(beneficiary_id,'00000000-0000-0000-0000-000000000000'),initiated_by,reference_number,amount::text,COALESCE(purpose,''),status,review_status,currency,created_at`,
		in.ReferenceNumber, in.SourceAccountID, in.DestinationAccountID, in.BeneficiaryID, userID, fmt.Sprintf("%d.%02d", in.AmountPaise/100, in.AmountPaise%100), in.Purpose, in.IdempotencyKey).
		Scan(&out.ID, &out.SourceAccountID, &out.DestinationAccountID, &out.BeneficiaryID, &out.InitiatedBy, &out.ReferenceNumber, &out.Amount, &out.Purpose, &out.Status, &out.ReviewStatus, &out.Currency, &out.CreatedAt)
	if err != nil {
		return TransferRecord{}, false, err
	}
	if err = tx.QueryRow(ctx, `SELECT account_number FROM bank_accounts WHERE id=$1`, out.DestinationAccountID).Scan(&out.DestinationAccountNumber); err != nil {
		return TransferRecord{}, false, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO ledger_entries(transfer_id,account_id,entry_type,amount,balance_after) VALUES($1,$2,'DEBIT',$3::numeric/100,$4::numeric/100),($1,$5,'CREDIT',$3::numeric/100,$6::numeric/100)`, out.ID, source.id, in.AmountPaise, newSource, destination.id, newDestination); err != nil {
		return TransferRecord{}, false, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO transactions(account_id,transfer_id,transaction_type,amount,currency,description,balance_after) VALUES($1,$2,'DEBIT',$3::numeric/100,'INR',$4,$5::numeric/100),($6,$2,'CREDIT',$3::numeric/100,'INR',$4,$7::numeric/100)`, source.id, out.ID, in.AmountPaise, in.Purpose, newSource, destination.id, newDestination); err != nil {
		return TransferRecord{}, false, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audit_events(actor_user_id,event_type,entity_type,entity_id,action,result,ip_address,correlation_id) VALUES($1,'TRANSFER_COMPLETED','FUND_TRANSFER',$2,'CREATE','SUCCESS',NULLIF($3,'')::inet,$4)`, userID, out.ID, ip, correlationID); err != nil {
		return TransferRecord{}, false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return TransferRecord{}, false, err
	}
	return out, false, nil
}

func (r *TransferRepository) GetForCustomer(ctx context.Context, userID uuid.UUID, ref string) (TransferRecord, error) {
	return r.get(ctx, `f.reference_number=$1 AND f.initiated_by=$2`, ref, userID)
}
func (r *TransferRepository) GetForOperations(ctx context.Context, userID uuid.UUID, ref string) (TransferRecord, error) {
	return r.get(ctx, `f.reference_number=$1 AND EXISTS (SELECT 1 FROM account_assignments aa WHERE aa.user_id=$2 AND aa.account_id IN (f.source_account_id,f.destination_account_id))`, ref, userID)
}
func (r *TransferRepository) ListForCustomer(ctx context.Context, userID uuid.UUID, page, size int) ([]TransferRecord, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fund_transfers f WHERE f.initiated_by=$1`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.pool.Query(ctx, `SELECT f.id,f.source_account_id,f.destination_account_id,COALESCE(f.beneficiary_id,'00000000-0000-0000-0000-000000000000'),f.initiated_by,f.reference_number,f.amount::text,COALESCE(f.purpose,''),f.status,f.review_status,f.currency,d.account_number,f.created_at
		FROM fund_transfers f JOIN bank_accounts d ON d.id=f.destination_account_id WHERE f.initiated_by=$1
		ORDER BY f.created_at DESC,f.id DESC LIMIT $2 OFFSET $3`, userID, size, (page-1)*size)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []TransferRecord{}
	for rows.Next() {
		var item TransferRecord
		if err := rows.Scan(&item.ID, &item.SourceAccountID, &item.DestinationAccountID, &item.BeneficiaryID, &item.InitiatedBy, &item.ReferenceNumber, &item.Amount, &item.Purpose, &item.Status, &item.ReviewStatus, &item.Currency, &item.DestinationAccountNumber, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, rows.Err()
}

func (r *TransferRepository) ListForOperations(ctx context.Context, userID uuid.UUID, f OperationsTransferFilter) ([]TransferRecord, int, error) {
	args := []any{userID}
	conditions := []string{"EXISTS (SELECT 1 FROM account_assignments aa WHERE aa.user_id=$1 AND aa.account_id IN (f.source_account_id,f.destination_account_id))"}
	add := func(sql string, value any) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(sql, len(args)))
	}
	if f.Reference != "" {
		add("f.reference_number=$%d", f.Reference)
	}
	if f.Status != "" {
		add("f.status=$%d", f.Status)
	}
	if f.ReviewStatus != "" {
		add("f.review_status=$%d", f.ReviewStatus)
	}
	if f.DateFrom != nil {
		add("f.created_at >= $%d", *f.DateFrom)
	}
	if f.DateTo != nil {
		add("f.created_at <= $%d", *f.DateTo)
	}
	where := strings.Join(conditions, " AND ")
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fund_transfers f WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (f.Page - 1) * f.PageSize
	args = append(args, f.PageSize, offset)
	query := `SELECT f.id,f.source_account_id,f.destination_account_id,COALESCE(f.beneficiary_id,'00000000-0000-0000-0000-000000000000'),f.initiated_by,f.reference_number,f.amount::text,COALESCE(f.purpose,''),f.status,f.review_status,f.currency,d.account_number,f.created_at FROM fund_transfers f JOIN bank_accounts d ON d.id=f.destination_account_id WHERE ` + where + ` ORDER BY f.created_at DESC,f.id DESC LIMIT $` + fmt.Sprint(len(args)-1) + ` OFFSET $` + fmt.Sprint(len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var result []TransferRecord
	for rows.Next() {
		var item TransferRecord
		if err := rows.Scan(&item.ID, &item.SourceAccountID, &item.DestinationAccountID, &item.BeneficiaryID, &item.InitiatedBy, &item.ReferenceNumber, &item.Amount, &item.Purpose, &item.Status, &item.ReviewStatus, &item.Currency, &item.DestinationAccountNumber, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		result = append(result, item)
	}
	return result, total, rows.Err()
}

func (r *TransferRepository) Review(ctx context.Context, userID uuid.UUID, reference, reviewStatus string) (TransferRecord, string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return TransferRecord{}, "", err
	}
	defer tx.Rollback(ctx)
	var out TransferRecord
	var oldStatus string
	err = tx.QueryRow(ctx, `SELECT f.review_status FROM fund_transfers f
		WHERE f.reference_number=$2 AND EXISTS (SELECT 1 FROM account_assignments aa WHERE aa.user_id=$1 AND aa.account_id IN (f.source_account_id,f.destination_account_id)) FOR UPDATE`, userID, reference).Scan(&oldStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return TransferRecord{}, "", ErrNotFound
	}
	if err != nil {
		return TransferRecord{}, "", err
	}
	err = tx.QueryRow(ctx, `UPDATE fund_transfers f SET review_status=$3,updated_at=NOW()
		WHERE f.reference_number=$2 AND EXISTS (SELECT 1 FROM account_assignments aa WHERE aa.user_id=$1 AND aa.account_id IN (f.source_account_id,f.destination_account_id))
		RETURNING f.review_status,f.id,f.source_account_id,f.destination_account_id,COALESCE(f.beneficiary_id,'00000000-0000-0000-0000-000000000000'),f.initiated_by,f.reference_number,f.amount::text,COALESCE(f.purpose,''),f.status,f.currency,(SELECT account_number FROM bank_accounts WHERE id=f.destination_account_id),f.created_at`, userID, reference, reviewStatus).
		Scan(&out.ReviewStatus, &out.ID, &out.SourceAccountID, &out.DestinationAccountID, &out.BeneficiaryID, &out.InitiatedBy, &out.ReferenceNumber, &out.Amount, &out.Purpose, &out.Status, &out.Currency, &out.DestinationAccountNumber, &out.CreatedAt)
	if err != nil {
		return TransferRecord{}, "", err
	}
	if err = tx.Commit(ctx); err != nil {
		return TransferRecord{}, "", err
	}
	return out, oldStatus, nil
}
func (r *TransferRepository) get(ctx context.Context, condition string, ref string, userID uuid.UUID) (TransferRecord, error) {
	var out TransferRecord
	err := r.pool.QueryRow(ctx, `SELECT f.id,f.source_account_id,f.destination_account_id,COALESCE(f.beneficiary_id,'00000000-0000-0000-0000-000000000000'),f.initiated_by,f.reference_number,f.amount::text,COALESCE(f.purpose,''),f.status,f.review_status,f.currency,d.account_number,f.created_at FROM fund_transfers f JOIN bank_accounts d ON d.id=f.destination_account_id WHERE `+condition, ref, userID).
		Scan(&out.ID, &out.SourceAccountID, &out.DestinationAccountID, &out.BeneficiaryID, &out.InitiatedBy, &out.ReferenceNumber, &out.Amount, &out.Purpose, &out.Status, &out.ReviewStatus, &out.Currency, &out.DestinationAccountNumber, &out.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return TransferRecord{}, ErrNotFound
	}
	return out, err
}

var ErrIdempotencyConflict = errors.New("idempotency conflict")
var ErrFrozenAccount = errors.New("frozen account")
var ErrInsufficientFunds = errors.New("insufficient funds")
var ErrBeneficiaryInactive = errors.New("beneficiary inactive")
