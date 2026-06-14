# Scope, Anomalies, and Schema

## Product Scope

Implemented:

- CSV import with anomaly reporting
- Equal, unequal, share, and percentage split parsing
- USD to INR conversion with documented fixed assignment rate
- Membership timeline for Aisha, Rohan, Priya, Meera, Dev, Sam, and Kabir
- Group balance summary and simplified settlement suggestions
- Settlement/payment detection
- React UI for upload, balances, anomaly review, members, and expense trace
- Neon-compatible relational schema

Not completed yet:

- Real login/session handling
- Postgres persistence wired into the API
- User approval workflow that mutates pending anomaly decisions
- Deployed public URL

## Import Policies

Default currency is INR unless CSV explicitly says USD. USD is converted using `1 USD = INR 83.50` so calculations are repeatable in review.

Dates are parsed as `DD-MM-YYYY`. `Mar-14` is parsed as `14 March 2026` because the whole sheet is in 2026. Ambiguous `04-05-2026` is parsed as `4 May 2026` but flagged for approval.

Duplicate-looking expenses are not silently merged. The first row is imported and the later row is skipped pending review.

Settlements are recorded separately from expenses and do not create participant shares.

Blocking anomalies skip the row. Warning anomalies import with documented normalization. Approval-required anomalies either import with visible review status or skip until approved, depending on risk.

## Anomaly Log

| CSV row | Problem | Policy/action |
| --- | --- | --- |
| 5 and 6 | Duplicate Marina Bites dinner with description variation | Import first, skip duplicate pending approval |
| 8 | Amount has thousands separator `"1,200"` | Normalize comma and import |
| 10 | Payer `priya` has wrong case | Normalize to `Priya` |
| 11 | Amount `899.995` has fractional paise | Round to nearest paise |
| 12 | Payer `Priya S` alias | Normalize to `Priya` |
| 13 | Unequal split excludes Aisha intentionally | Import because participants match split details |
| 14 | Missing payer | Blocking, skip expense |
| 15 | Settlement logged as expense | Record as settlement |
| 16 | Percentages total 110%, not 100% | Normalize weights and flag for approval |
| 20, 21, 25 | USD expenses | Convert to INR at fixed rate |
| 24 | Participant `Dev's friend Kabir` not a normal member | Normalize to visitor `Kabir` |
| 26 and 27 | Possible duplicate Thalassa dinner with different amount/payer | Import first, skip later pending approval |
| 28 | Negative USD amount | Treat as refund and import as negative expense |
| 29 | Date `Mar-14` | Parse as `14 March 2026` and flag |
| 30 | Missing currency | Default to INR and flag |
| 33 | Zero amount | Skip pending review |
| 35 | Ambiguous date `04-05-2026` | Parse as DD-MM-YYYY, flag for approval |
| 37 | Meera included after move-out | Flag membership violation for approval |
| 40 | Sam deposit logged in expense sheet | Treat as non-shared transfer and skip pending review |
| 43 | `equal` split has share details | Trust split type, ignore details, flag |

## Database Schema

The schema lives in `db/migrations`.

Core tables:

- `users`: login identities
- `groups`: expense groups
- `people`: canonical people independent of login
- `group_memberships`: join/leave periods
- `imports`: import batches
- `import_rows`: raw CSV row payloads
- `import_anomalies`: every detected data problem and review status
- `expenses`: normalized expenses in base currency
- `expense_shares`: participant-level owed shares
- `settlements`: repayments/payments
- `exchange_rates`: conversion policy and rates

The schema deliberately separates `people` from `users`, because Dev and Kabir can appear in expenses without having login accounts.
