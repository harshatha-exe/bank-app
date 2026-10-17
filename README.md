# Bank App

## Running locally

Copy the demo environment file, then start PostgreSQL, apply migrations, and run
the backend locally:

```sh
cp -n .env.example .env
docker compose down -v
make db-up
make migrate-up
make run
```

The backend health check is available at `http://localhost:8080/health` and
PostgreSQL is available on port `5432`. The `make run` and migration targets
load variables from `.env` automatically. The `down -v` command removes the
old database volume so the consolidated initial migration can run from a clean
database.

## Demo credentials

These credentials are **DEMO ONLY**. Change or remove them before using this
application outside local development:

| Role | Username | Password |
| --- | --- | --- |
| Admin | `demo-admin` | `DemoAdminPassword123!` |
| Operations | `demo-operations` | `DemoOperationsPassword123!` |
| Customer | `demo-customer` | `DemoCustomerPassword123!` |
| Customer | `demo-customer-b` | `DemoCustomerPassword123!` |

The seed data is for local demonstration only. Signup creates a customer
profile without a bank account. Login returns a CSRF token in `meta.csrfToken`;
send it as `X-CSRF-Token` on authenticated state-changing requests.

Useful commands:

```sh
make db-up
make db-down
make logs
make migrate-up
make migrate-down
make run
make test
```

To run Go directly in the Codespace terminal, use `DATABASE_URL=postgres://bankuser:bankpass@localhost:5432/bankdb?sslmode=disable` because the Compose service hostname `db` is only resolvable inside Compose.
