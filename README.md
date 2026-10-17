# Bank App

## Running in Codespaces

Copy the demo environment file, then start the database, migrations, and backend:

```sh
cp -n .env.example .env
docker compose up --build
```

The backend health check is available at `http://localhost:8080/health` and PostgreSQL is available on port `5432`.

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
make up
make down
make logs
make migrate-up
make migrate-down
make test
```

To run Go directly in the Codespace terminal, use `DATABASE_URL=postgres://bankuser:bankpass@localhost:5432/bankdb?sslmode=disable` because the Compose service hostname `db` is only resolvable inside Compose.
