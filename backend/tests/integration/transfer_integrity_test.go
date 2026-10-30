package integration

import (
	"context"
	"net/http"
	"sync"
	"testing"
)

func TestSuccessfulTransferIntegrity(t *testing.T) {
	f := createFixture(t)
	cookie, csrf := login(t, f)
	rec := transferRequest(t, f, cookie, csrf, "integrity-success-0001", "3000.00")
	if rec.Code != http.StatusCreated {
		t.Fatalf("transfer status %d: %s", rec.Code, rec.Body.String())
	}
	var transferID string
	if err := f.pool.QueryRow(context.Background(), "SELECT id::text FROM fund_transfers WHERE initiated_by=$1 AND idempotency_key=$2", f.user, "integrity-success-0001").Scan(&transferID); err != nil {
		t.Fatal(err)
	}
	var source, destination string
	if err := f.pool.QueryRow(context.Background(), "SELECT balance::text FROM bank_accounts WHERE id=$1", f.source).Scan(&source); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(context.Background(), "SELECT balance::text FROM bank_accounts WHERE id=$1", f.destination).Scan(&destination); err != nil {
		t.Fatal(err)
	}
	if source != "7000.00" || destination != "8000.00" {
		t.Fatalf("balances = %s/%s", source, destination)
	}
	var ledgers, transactions, audits int
	if err := f.pool.QueryRow(context.Background(), "SELECT count(*) FROM ledger_entries WHERE transfer_id=$1", transferID).Scan(&ledgers); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(context.Background(), "SELECT count(*) FROM transactions WHERE transfer_id=$1", transferID).Scan(&transactions); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(context.Background(), "SELECT count(*) FROM audit_events WHERE entity_id=$1::uuid", transferID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if ledgers != 2 || transactions != 2 || audits < 1 {
		t.Fatalf("counts ledger=%d transactions=%d audits=%d", ledgers, transactions, audits)
	}
}

func TestTransferFailureIdempotencyAndConcurrency(t *testing.T) {
	f := createFixture(t)
	cookie, csrf := login(t, f)
	before := accountBalance(t, f, f.source)
	if rec := transferRequest(t, f, cookie, csrf, "integrity-insufficient-01", "20000.00"); rec.Code < 400 || rec.Code >= 500 {
		t.Fatalf("insufficient funds status %d", rec.Code)
	}
	if accountBalance(t, f, f.source) != before {
		t.Fatal("insufficient transfer changed balance")
	}
	first := transferRequest(t, f, cookie, csrf, "integrity-repeat-0001", "10.00")
	second := transferRequest(t, f, cookie, csrf, "integrity-repeat-0001", "10.00")
	if first.Code != http.StatusCreated || second.Code != http.StatusCreated {
		t.Fatalf("duplicate statuses %d/%d", first.Code, second.Code)
	}
	if transferData(t, first)["referenceNumber"] != transferData(t, second)["referenceNumber"] {
		t.Fatal("duplicate request returned different reference")
	}
	var count int
	if err := f.pool.QueryRow(context.Background(), "SELECT count(*) FROM fund_transfers WHERE initiated_by=$1 AND idempotency_key=$2", f.user, "integrity-repeat-0001").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("duplicate transfer count %d", count)
	}
	resetBalance(t, f, "10000.00", "5000.00")
	var wg sync.WaitGroup
	statuses := make(chan int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c, token := login(t, f)
			statuses <- transferRequest(t, f, c, token, "integrity-concurrent-"+string(rune('a'+i)), "8000.00").Code
		}(i)
	}
	wg.Wait()
	close(statuses)
	success := 0
	for status := range statuses {
		if status == http.StatusCreated {
			success++
		}
	}
	if success != 1 || accountBalance(t, f, f.source) != "2000.00" {
		t.Fatalf("concurrent success=%d source=%s", success, accountBalance(t, f, f.source))
	}
}

func TestFrozenAccountsRejected(t *testing.T) {
	f := createFixture(t)
	cookie, csrf := login(t, f)
	_, err := f.pool.Exec(context.Background(), "UPDATE bank_accounts SET status='FROZEN' WHERE id=$1", f.source)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), "UPDATE bank_accounts SET status='ACTIVE' WHERE id IN ($1,$2)", f.source, f.destination)
	})
	if rec := transferRequest(t, f, cookie, csrf, "integrity-frozen-source", "1.00"); rec.Code < 400 || rec.Code >= 500 {
		t.Fatalf("frozen source status %d", rec.Code)
	}
	_, err = f.pool.Exec(context.Background(), "UPDATE bank_accounts SET status='ACTIVE' WHERE id=$1", f.source)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.pool.Exec(context.Background(), "UPDATE bank_accounts SET status='FROZEN' WHERE id=$1", f.destination)
	if err != nil {
		t.Fatal(err)
	}
	if rec := transferRequest(t, f, cookie, csrf, "integrity-frozen-destination", "1.00"); rec.Code < 400 || rec.Code >= 500 {
		t.Fatalf("frozen destination status %d", rec.Code)
	}
}

func accountBalance(t *testing.T, f *fixture, id any) string {
	t.Helper()
	var balance string
	if err := f.pool.QueryRow(context.Background(), "SELECT balance::text FROM bank_accounts WHERE id=$1", id).Scan(&balance); err != nil {
		t.Fatal(err)
	}
	return balance
}

func resetBalance(t *testing.T, f *fixture, source, destination string) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(), "UPDATE bank_accounts SET balance=$1,status='ACTIVE' WHERE id=$2", source, f.source); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(context.Background(), "UPDATE bank_accounts SET balance=$1,status='ACTIVE' WHERE id=$2", destination, f.destination); err != nil {
		t.Fatal(err)
	}
}
