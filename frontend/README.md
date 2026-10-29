# Bank App frontend (React + Tailwind CSS)

Web front end for the Bank App backend in `../backend`.
React 18, React Router 6, Vite 5, Tailwind CSS 3. No icon or UI component libraries.
Every screen calls the real API; errors shown come from the backend (with its correlation ID).

## Run it

1. Start the backend first (see the root README: `make db-up`, `make migrate-up`, `make run`).
   Check it is up: `curl -s localhost:8080/health`
2. In this folder (Node.js 18 or newer):

       npm install
       npm run dev

3. Open http://localhost:5173 and sign in with a demo user from the root README.

The dev server forwards every `/api` request to the backend, so the browser makes no
cross-origin calls (the backend has no CORS settings). If the backend is not on port 8080:

       VITE_API_TARGET=http://localhost:9090 npm run dev

Production build: `npm run build` (output in `dist/`, which is not committed).

## Screens

- Customer: overview, account statement (filters, paging), beneficiaries, send money (review step), transfers, transfer detail.
- Operations: assigned accounts (freeze or unfreeze), account transactions, transfer review.
- Admin: create and manage users, assign accounts to operations users, transfer limit, audit log.
- Every role: "Your user ID" with a Copy button in the sidebar. The ID alone gives no access; the backend decides what each role may do.

## Limits of the current API (the UI does not hide them)

- Transfers lists transfers the signed-in customer sent. Money received shows as a credit in the account statement and balance.
- There is no list-users, list-all-accounts or recipient-lookup endpoint, so beneficiaries and admin actions use pasted IDs (account ID or user ID, not the masked account number).
- Operations users only see accounts an admin has assigned to them, so their screens start empty.
- Signup creates a customer login with no bank account.

## Sending money needs the transfer fix

At backend commit 41ad906 the transfer handler never reads the request body, so `POST /api/transfers`
fails with a validation error. The fix restores `httpx.BindJSON` in `backend/internal/transfer/handler.go`.
Without it, the Send money screen shows the backend's error.

## Retry protection

Each send uses an `Idempotency-Key`. If the connection drops, the screen says the result is unknown and
reuses the same key on retry (kept in sessionStorage for about 10 minutes), so the bank treats it as one transfer.
This is demo-level protection, not a production guarantee.
