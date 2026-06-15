# AI Usage

AI tool used: Codex in the Codex desktop app.

## Representative Prompts

- Read the assignment PDF and CSV in detail and design a React + Go + Neon implementation.
- Scaffold the Go importer and balance calculation with tests.
- Build the React review UI for import anomalies and settlement suggestions.
- Produce the required README, SCOPE, DECISIONS, and AI usage notes.
- Debug deployment setup for Azure App Service and Vercel.

## Incorrect AI Outputs Caught

1. Duplicate detection was initially too literal and missed `Dinner at Marina Bites` versus `dinner - marina bites`. I caught it with a backend test expecting `duplicate_expense`, then changed the duplicate key to normalize known semantic description buckets.

2. The first test helper used a made-up interface instead of the real `domain.ImportAnomaly` type. I caught it during review before the test run and replaced it with the concrete domain type.

3. The initial dependency setup used `concurrently` for convenience. `npm audit` showed a critical advisory through `shell-quote`, so I removed the dependency and kept separate `api:dev` and `web:dev` scripts.

4. The first package install pulled a vulnerable Vite/esbuild chain. I verified the advisory with `npm audit`, applied the recommended Vite major update, and rebuilt the frontend.

5. The settlement simplification initially used two independent `if` statements, which let a debtor become a creditor after its negative balance was flipped positive. I caught this when Dev disappeared from "Who Pays Whom" and added a regression test.

6. The importer originally treated Sam's deposit as a settlement. Comparing the balance output against the CSV notes showed it should be a non-shared transfer, so I changed deposits to skip pending review.

7. The membership timeline was initially assignment-specific. A later CSV test showed hardcoded names would appear for unrelated data, so I changed the importer to derive members from uploaded rows and added a regression test with different names.

8. The frontend initially called relative `/api` URLs. That works locally through Vite proxy but fails on Vercel. I changed the client to use `VITE_API_URL` for deployed builds.

## Engineer-of-Record Notes

I verified the implemented code with:

```bash
cd api && go test ./...
npm run web:build
npm audit --omit=dev
```

The current known product gap is persistence wiring: the Neon schema is ready, but the running local API still stores imported data in memory.
