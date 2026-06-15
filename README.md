# Flat Ledger

Shared expenses app for the Spreetail assignment. The implementation is a React frontend, Go API, Neon-compatible Postgres schema, and GitHub Actions deployment workflow for the backend.

## Stack

- Frontend: React 19, TypeScript, Vite
- Backend: Go HTTP API
- Database target: Neon Postgres, schema in `db/migrations`
- Auth store: Neon/Postgres `users` table
- Import/expense store: in-memory for the running API, with schema prepared for persistence

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

- `VITE_API_TARGET`: API URL used by the Vite dev proxy in local development, for example `http://localhost:8080`.
- `VITE_API_URL`: deployed API URL used by the built frontend, for 

The Go API currently reads OS environment variables. If you keep values in `.env`, export them before starting the API:

```bash
set -a
source .env
set +a
npm run api:dev
```

Login and register use the `users` table in Postgres. They are disabled until `DATABASE_URL` is set and migrations have been applied.

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

Auth is backed by Postgres. Expense/import data still uses an in-memory store for local review while the schema is ready for persistence. The next implementation step is replacing `api/internal/store/memory.go` with a Postgres-backed store that writes imports, anomalies, expenses, shares, settlements, and review decisions transactionally.

## Import Flow

Upload the assignment CSV from the UI or POST it directly:

```bash
curl -F "file=@/path/to/expenses_export.csv" -F "usdRate=83.50" http://localhost:8080/api/imports
```

For local auto-seeding during API startup, place your own CSV at `sample-data/expenses_export.csv`. CSV files in `sample-data/` are ignored by Git.

The importer returns an import report containing:

- rows read
- imported expenses
- settlements detected from expense rows
- every anomaly found
- policy and action for every anomaly

The UI also lets a reviewer change the USD conversion rate before import. The default is `83.50`.

## Deployment

Backend deployment is configured in `.github/workflows/azure-backend.yml`.

Required GitHub secret:

- `AZURE_WEBAPP_PUBLISH_PROFILE`: full XML from Azure App Service's publish profile download.

Required Azure App Service settings:

- `DATABASE_URL`: Neon connection string.
- `ADDR`: `:8080`
- `WEBSITES_PORT`: `8080`

Azure startup command:

```bash
./startup.sh
```

Frontend deployment target is Vercel:

- Root directory: `web`
- Build command: `npm run build`
- Output directory: `dist`
- Environment variable: `VITE_API_URL=https://your-azure-app.azurewebsites.net`

Every push to `main` deploys the Go API through GitHub Actions once the secret is configured. Vercel redeploys the frontend from the same repository.

## AI Used

Codex was used as the coding collaborator. See `AI_USAGE.md` for details, including incorrect outputs caught during implementation.
