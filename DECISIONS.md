# Decision Log

## Use an audit-first importer

Options considered:

- Clean data silently during import
- Reject the whole file on the first bad row
- Import row-by-row with anomaly records

Chosen: import row-by-row with anomaly records. The assignment explicitly says crashes and silent guesses fail, and reviewers will trace individual rows.

## Use membership periods instead of current group membership only

Options considered:

- Store only current members
- Store join and leave dates

Chosen: membership periods. Sam joining mid-April and Meera leaving after March are core requirements, and balances need historical membership context.

Update: the UI no longer hardcodes the assignment members. The importer derives the displayed timeline from uploaded CSV names so a different CSV does not show Aisha/Rohan/Priya/Dev unless they appear in that file.

## Treat settlements separately from expenses

Options considered:

- Keep repayment rows as expenses
- Drop repayment rows
- Store repayment rows as settlements

Chosen: settlements. Repayments affect net balances but should not create shared participant shares.

## Exclude deposits from shared balances

Options considered:

- Treat deposits as settlements
- Treat deposits as shared expenses
- Flag deposits as non-shared transfers

Chosen: flag deposits as non-shared transfers and skip them pending review. Sam's deposit is a one-off transfer to Aisha, not a household expense that Meera or Kabir should settle through.

## User-selected exchange rate

Options considered:

- Live FX API
- User-entered import-time FX rate
- Fixed documented default with user override

Chosen: default to `1 USD = INR 83.50`, but allow the reviewer to override the rate during import. A live rate would make reviewer calculations change over time, while a configurable rate supports Priya's concern and live evaluation changes.

## Duplicate policy requires approval

Options considered:

- Last row wins
- First row wins silently
- Keep first and skip duplicate pending approval

Chosen: keep first and flag the later duplicate for approval. This preserves a working balance while surfacing the risky decision.

## React and Go boundaries

Options considered:

- Put parsing in the browser
- Put parsing in Go backend

Chosen: Go backend parsing. Import policy, audit trail, and balance math should be server-side and testable.

## Neon/Postgres schema

Options considered:

- Store imported report as JSON only
- Normalize expenses, shares, settlements, memberships, and anomalies

Chosen: normalized relational schema with raw row retention. The assignment requires relational DBs and traceability.

## Auth backed by Postgres first

Options considered:

- Keep demo-only local auth
- Implement auth against the Postgres `users` table
- Add a full external identity provider

Chosen: implement register/login against Postgres. This satisfies the assignment login requirement without adding an external provider. The current token is a simple app token, so session hardening remains future work.

## In-memory imports for current build

Options considered:

- Persist all imports immediately
- Keep imports in memory while finalizing importer policy

Chosen: keep import state in memory for the current iteration and document the limitation. The schema already supports persistence, but wiring every import row, anomaly, expense, share, settlement, and review decision transactionally is the next backend milestone.

## Azure and Vercel deployment split

Options considered:

- Host everything on Azure
- Host everything on Vercel
- Host backend on Azure App Service and frontend on Vercel

Chosen: backend on Azure App Service and frontend on Vercel. The Go API fits App Service, and the React/Vite frontend fits Vercel. The frontend uses `VITE_API_URL` so production calls the Azure API instead of Vercel's `/api` route.
