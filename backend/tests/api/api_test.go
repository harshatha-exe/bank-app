package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthAndSignup(t *testing.T) {
	f := newFixture(t)
	cookie, csrf := login(t, f, f.oneName, "fixture-password-123")
	if cookie.Value == "" || csrf == "" {
		t.Fatal("login did not return session and csrf credentials")
	}
	if rec := request(t, f.router, http.MethodPost, "/api/auth/login", map[string]string{"username": f.oneName, "password": "wrong-password"}, nil, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password status %d", rec.Code)
	}
	if rec := request(t, f.router, http.MethodGet, "/api/me", nil, nil, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status %d", rec.Code)
	}
	if rec := request(t, f.router, http.MethodPost, "/api/auth/logout", nil, cookie, csrf); rec.Code != http.StatusNoContent {
		t.Fatalf("logout status %d: %s", rec.Code, rec.Body.String())
	}
	if rec := request(t, f.router, http.MethodGet, "/api/me", nil, cookie, csrf); rec.Code != http.StatusUnauthorized {
		t.Fatalf("logged-out session status %d", rec.Code)
	}

	username := fmt.Sprintf("%s-signup", f.oneName)
	rec := request(t, f.router, http.MethodPost, "/api/auth/signup", map[string]any{"username": username, "password": "fixture-password-123", "fullName": "Signup User", "role": "ADMIN"}, nil, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("signup status %d: %s", rec.Code, rec.Body.String())
	}

	var role string
	if err := f.pool.QueryRow(context.Background(), `SELECT r.name FROM users u JOIN roles r ON r.id=u.role_id WHERE u.username=$1`, username).Scan(&role); err != nil {
		t.Fatal(err)
	}
	if role != "CUSTOMER" {
		t.Fatalf("signup role = %s", role)
	}
}

func TestOperationsAndAdmin(t *testing.T) {
	f := newFixture(t)
	customerCookie, customerCSRF := login(t, f, f.oneName, "fixture-password-123")
	transfer := requestWithKey(t, f.router, http.MethodPost, "/api/transfers", map[string]any{"sourceAccountId": f.oneAccount, "destinationAccountId": f.destination, "amount": "5.00"}, customerCookie, customerCSRF, "api-test-ops-transfer-01")
	if transfer.Code != http.StatusCreated {
		t.Fatalf("operations fixture transfer status %d: %s", transfer.Code, transfer.Body.String())
	}
	reference := responseData(t, transfer)["referenceNumber"].(string)
	opsCookie, opsCSRF := login(t, f, f.opsName, "fixture-password-123")
	accounts := request(t, f.router, http.MethodGet, "/api/operations/accounts", nil, opsCookie, opsCSRF)
	if accounts.Code != http.StatusOK || !strings.Contains(accounts.Body.String(), f.oneAccount.String()) || strings.Contains(accounts.Body.String(), f.twoAccount.String()) {
		t.Fatalf("assigned accounts response %d: %s", accounts.Code, accounts.Body.String())
	}
	for _, action := range []string{"freeze", "unfreeze"} {
		rec := request(t, f.router, http.MethodPost, "/api/operations/accounts/"+f.oneAccount.String()+"/"+action, nil, opsCookie, opsCSRF)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status %d: %s", action, rec.Code, rec.Body.String())
		}
	}
	if rec := request(t, f.router, http.MethodGet, "/api/operations/accounts/"+f.oneAccount.String()+"/transactions", nil, opsCookie, opsCSRF); rec.Code != http.StatusOK {
		t.Fatalf("operations transactions status %d", rec.Code)
	}
	review := request(t, f.router, http.MethodPatch, "/api/operations/transfers/"+reference+"/review", map[string]string{"reviewStatus": "UNDER_REVIEW"}, opsCookie, opsCSRF)
	if review.Code != http.StatusOK || !strings.Contains(review.Body.String(), "UNDER_REVIEW") {
		t.Fatalf("review status %d: %s", review.Code, review.Body.String())
	}
	adminCookie, adminCSRF := login(t, f, f.adminName, "fixture-password-123")
	created := request(t, f.router, http.MethodPost, "/api/admin/users", map[string]string{"username": f.oneName + "-managed", "role": "OPERATIONS", "temporaryPassword": "fixture-password-123"}, adminCookie, adminCSRF)
	if created.Code != http.StatusCreated {
		t.Fatalf("admin create status %d: %s", created.Code, created.Body.String())
	}
	managed := responseData(t, created)["id"].(string)
	assign := request(t, f.router, http.MethodPost, "/api/admin/account-assignments", map[string]string{"operationsUserId": managed, "accountId": f.destination.String()}, adminCookie, adminCSRF)
	if assign.Code != http.StatusCreated {
		t.Fatalf("admin assign status %d: %s", assign.Code, assign.Body.String())
	}
	if rec := request(t, f.router, http.MethodGet, "/api/admin/users?search="+f.oneName, nil, adminCookie, adminCSRF); rec.Code != http.StatusOK {
		t.Fatalf("admin list status %d", rec.Code)
	}
	if rec := request(t, f.router, http.MethodPatch, "/api/admin/users/"+managed+"/role", map[string]string{"role": "OPERATIONS"}, adminCookie, adminCSRF); rec.Code != http.StatusOK {
		t.Fatalf("admin role status %d: %s", rec.Code, rec.Body.String())
	}
	if rec := request(t, f.router, http.MethodPost, "/api/admin/users/"+managed+"/reset-password", map[string]string{"newPassword": "reset-password-123"}, adminCookie, adminCSRF); rec.Code != http.StatusNoContent {
		t.Fatalf("admin reset status %d: %s", rec.Code, rec.Body.String())
	}
	if rec := request(t, f.router, http.MethodGet, "/api/admin/config/transaction-limit", nil, adminCookie, adminCSRF); rec.Code != http.StatusOK {
		t.Fatalf("admin limit status %d", rec.Code)
	}
	if rec := request(t, f.router, http.MethodGet, "/api/admin/users", nil, customerCookie, customerCSRF); rec.Code != http.StatusForbidden {
		t.Fatalf("customer admin escalation status %d", rec.Code)
	}
	if rec := request(t, f.router, http.MethodGet, "/api/operations/accounts", nil, customerCookie, customerCSRF); rec.Code != http.StatusForbidden {
		t.Fatalf("customer operations escalation status %d", rec.Code)
	}
	if rec := request(t, f.router, http.MethodGet, "/api/admin/users", nil, opsCookie, opsCSRF); rec.Code != http.StatusForbidden {
		t.Fatalf("operations admin escalation status %d", rec.Code)
	}
}

func TestAccountsBOLAAndTransactions(t *testing.T) {
	f := newFixture(t)
	cookie, csrf := login(t, f, f.oneName, "fixture-password-123")
	if rec := request(t, f.router, http.MethodGet, "/api/accounts", nil, cookie, csrf); rec.Code != http.StatusOK {
		t.Fatalf("accounts list status %d", rec.Code)
	} else if strings.Contains(rec.Body.String(), "apitest-100000000001") {
		t.Fatal("unexpected account number in response")
	}
	for _, endpoint := range []string{
		"/api/accounts/" + f.twoAccount.String(),
		"/api/accounts/" + f.twoAccount.String() + "/transactions",
	} {
		rec := request(t, f.router, http.MethodGet, endpoint, nil, cookie, csrf)
		if rec.Code != http.StatusNotFound && rec.Code != http.StatusForbidden {
			t.Fatalf("%s status %d", endpoint, rec.Code)
		}
	}
	rec := request(t, f.router, http.MethodGet, "/api/accounts/"+f.oneAccount.String()+"/transactions?pageSize=101&sort=%27%20OR%20%271%27=%271", nil, cookie, csrf)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid transaction filters status %d", rec.Code)
	}
	otherCookie, otherCSRF := login(t, f, f.twoName, "fixture-password-123")
	otherTransfer := requestWithKey(t, f.router, http.MethodPost, "/api/transfers", map[string]any{"sourceAccountId": f.twoAccount, "destinationAccountId": f.destination, "amount": "1.00"}, otherCookie, otherCSRF, "api-test-bola-transfer-01")
	if otherTransfer.Code != http.StatusCreated {
		t.Fatalf("other customer transfer status %d: %s", otherTransfer.Code, otherTransfer.Body.String())
	}
	otherReference := responseData(t, otherTransfer)["referenceNumber"].(string)
	if rec := request(t, f.router, http.MethodGet, "/api/transfers/"+otherReference, nil, cookie, csrf); rec.Code != http.StatusNotFound && rec.Code != http.StatusForbidden {
		t.Fatalf("other customer transfer status %d", rec.Code)
	}
}

func TestBeneficiariesTransfersAndValidation(t *testing.T) {
	f := newFixture(t)
	cookie, csrf := login(t, f, f.oneName, "fixture-password-123")
	body := map[string]any{"destinationAccountId": f.twoAccount, "nickname": "<img src=x onerror=alert(1)>"}
	rec := request(t, f.router, http.MethodPost, "/api/beneficiaries", body, cookie, csrf)
	if rec.Code != http.StatusCreated {
		t.Fatalf("beneficiary status %d: %s", rec.Code, rec.Body.String())
	}
	data := responseData(t, rec)
	beneficiaryID, ok := data["id"].(string)
	twoAccountNumber := f.twoAccountNumber
	if twoAccountNumber == "" {
		if err := f.pool.QueryRow(context.Background(), "SELECT account_number FROM bank_accounts WHERE id=$1", f.twoAccount).Scan(&twoAccountNumber); err != nil {
			t.Fatal(err)
		}
	}
	last4 := twoAccountNumber[len(twoAccountNumber)-4:]
	expectedMasked := "******" + last4
	masked, _ := data["destinationAccountMasked"].(string)
	nickname, _ := data["nickname"].(string)
	if !ok || masked != expectedMasked || masked == twoAccountNumber || nickname != "<img src=x onerror=alert(1)>" {
		t.Fatalf("unexpected beneficiary response: %s", rec.Body.String())
	}
	if duplicate := request(t, f.router, http.MethodPost, "/api/beneficiaries", body, cookie, csrf); duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate status %d", duplicate.Code)
	}
	if own := request(t, f.router, http.MethodPost, "/api/beneficiaries", map[string]any{"destinationAccountId": f.oneAccount, "nickname": "own"}, cookie, csrf); own.Code != http.StatusNotFound && own.Code != http.StatusUnprocessableEntity {
		t.Fatalf("own beneficiary status %d", own.Code)
	}
	if status := request(t, f.router, http.MethodPatch, "/api/beneficiaries/"+beneficiaryID+"/status", map[string]string{"status": "DEACTIVATED"}, cookie, csrf); status.Code != http.StatusOK {
		t.Fatalf("deactivate status %d", status.Code)
	}
	if status := request(t, f.router, http.MethodPatch, "/api/beneficiaries/"+beneficiaryID+"/status", map[string]string{"status": "ACTIVE"}, cookie, csrf); status.Code != http.StatusOK {
		t.Fatalf("activate status %d", status.Code)
	}
	if list := request(t, f.router, http.MethodGet, "/api/beneficiaries", nil, cookie, csrf); list.Code != http.StatusOK || !strings.Contains(list.Body.String(), beneficiaryID) {
		t.Fatalf("beneficiary list status %d: %s", list.Code, list.Body.String())
	}
	key := "api-test-idempotency-0001"
	transfer := map[string]any{"sourceAccountId": f.oneAccount, "destinationAccountId": f.destination, "amount": "10.00", "purpose": "xss <script>alert(1)</script>"}
	rec = requestWithKey(t, f.router, http.MethodPost, "/api/transfers", transfer, cookie, csrf, key)
	if rec.Code != http.StatusCreated {
		t.Fatalf("transfer status %d: %s", rec.Code, rec.Body.String())
	}
	ref := responseData(t, rec)["referenceNumber"].(string)
	if got := request(t, f.router, http.MethodGet, "/api/transfers/"+ref, nil, cookie, csrf); got.Code != http.StatusOK {
		t.Fatalf("get transfer status %d", got.Code)
	}
	if got := request(t, f.router, http.MethodGet, "/api/transfers", nil, cookie, csrf); got.Code != http.StatusOK {
		t.Fatalf("list transfer status %d", got.Code)
	}
	viaBeneficiary := requestWithKey(t, f.router, http.MethodPost, "/api/transfers", map[string]any{"sourceAccountId": f.oneAccount, "destinationAccountId": f.twoAccount, "beneficiaryId": beneficiaryID, "amount": "10.00"}, cookie, csrf, "api-test-beneficiary-0001")
	if viaBeneficiary.Code != http.StatusCreated {
		t.Fatalf("beneficiary transfer status %d: %s", viaBeneficiary.Code, viaBeneficiary.Body.String())
	}
	if tx := request(t, f.router, http.MethodGet, "/api/accounts/"+f.oneAccount.String()+"/transactions?page=1&pageSize=1&sort=amount&direction=ASC&type=DEBIT&minAmount=1.00&maxAmount=20.00", nil, cookie, csrf); tx.Code != http.StatusOK || !strings.Contains(tx.Body.String(), `"pageSize":1`) {
		t.Fatalf("transaction filters status %d: %s", tx.Code, tx.Body.String())
	}
	conflict := requestWithKey(t, f.router, http.MethodPost, "/api/transfers", map[string]any{"sourceAccountId": f.oneAccount, "destinationAccountId": f.destination, "amount": "11.00"}, cookie, csrf, key)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("idempotency conflict status %d", conflict.Code)
	}
	for _, amount := range []string{"0", "-1.00"} {
		bad := requestWithKey(t, f.router, http.MethodPost, "/api/transfers", map[string]any{"sourceAccountId": f.oneAccount, "destinationAccountId": f.destination, "amount": amount}, cookie, csrf, "api-test-"+strings.ReplaceAll(amount, ".", "")+"-key")
		if bad.Code < 400 || bad.Code >= 500 {
			t.Fatalf("amount %s status %d", amount, bad.Code)
		}
	}
	var limit string
	if err := f.pool.QueryRow(context.Background(), `SELECT (max_transfer_amount + 0.01)::text FROM transaction_limits WHERE id=TRUE`).Scan(&limit); err != nil {
		t.Fatal(err)
	}
	over := requestWithKey(t, f.router, http.MethodPost, "/api/transfers", map[string]any{"sourceAccountId": f.oneAccount, "destinationAccountId": f.destination, "amount": limit}, cookie, csrf, "api-test-over-limit-01")
	if over.Code != http.StatusUnprocessableEntity {
		t.Fatalf("over-limit status %d", over.Code)
	}
	same := requestWithKey(t, f.router, http.MethodPost, "/api/transfers", map[string]any{"sourceAccountId": f.oneAccount, "destinationAccountId": f.oneAccount, "amount": "1.00"}, cookie, csrf, "api-test-same-account-01")
	if same.Code < 400 || same.Code >= 500 {
		t.Fatalf("same-account status %d", same.Code)
	}
}

func requestWithKey(t *testing.T, r http.Handler, method, url string, body any, cookie *http.Cookie, csrf, key string) *httptest.ResponseRecorder {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", key)
	req.AddCookie(cookie)
	req.Header.Set("X-CSRF-Token", csrf)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}
