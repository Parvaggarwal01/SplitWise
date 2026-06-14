# Flat Ledger

Shared expenses app for the Spreetail assignment. The implementation is a React frontend, Go API, and Neon-compatible Postgres schema.

## Stack

- Frontend: React 19, TypeScript, Vite
- Backend: Go HTTP API
- Database target: Neon Postgres, schema in `db/migrations`
- Local development store: in-memory, seeded from `sample-data/expenses_export.csv`

## Setup

```bash
npm install
npm run api:test
npm run web:build
```

## Environment Variables

Copy examples before local setup:

```bash
cp .env.example .env
cp web/.env.example web/.env.local
```

Backend variables:

- `DATABASE_URL`: Neon Postgres connection string.
- `ADDR`: API bind address, defaults to `:8080`.

Frontend variables:

- `VITE_API_TARGET`: API URL used by the Vite dev proxy, for example `http://localhost:8080`.

The Go API currently reads OS environment variables. If you keep values in `.env`, export them before starting the API:

```bash
set -a
source .env
set +a
npm run api:dev
```

Run the API:

```bash
npm run api:dev
```

Run the frontend in another terminal:

```bash
npm run web:dev
```

Open `http://localhost:5173`.

## Neon Setup

Create a Neon project, copy the pooled connection string, and set:

```bash
export DATABASE_URL="postgresql://..."
```

Apply migrations:

```bash
psql "$DATABASE_URL" -f db/migrations/001_initial.sql
psql "$DATABASE_URL" -f db/migrations/002_seed_assignment.sql
```

The current API uses an in-memory store for local review while the schema is ready for Postgres persistence. The next implementation step is replacing `api/internal/store/memory.go` with a Postgres-backed store that writes imports, anomalies, expenses, shares, and settlements transactionally.

## Import Flow

Upload the assignment CSV from the UI or POST it directly:

```bash
curl -F "file=@sample-data/expenses_export.csv" http://localhost:8080/api/imports
```

The importer returns an import report containing:

- rows read
- imported expenses
- settlements detected from expense rows
- every anomaly found
- policy and action for every anomaly

## AI Used

Codex was used as the coding collaborator. See `AI_USAGE.md` for details, including incorrect outputs caught during implementation.
