# RiceApps Studio Carpool

Build a ridesharing app for Rice students. The frontend, API contract, database schema, authentication, and shared backend infrastructure are provided. Your pair implements its assigned operations through the RPC, application, Postgres, and SQL layers. Find your pair's tasks in the class tasking platform.

## Start locally

Install Go 1.27.1+, Node 20.19+, and Docker Desktop. Ask the leads for the shared Google OAuth development credentials. Start Docker Desktop, then make a local `.env` from `.env.example`:

```powershell
# Windows PowerShell
Copy-Item .env.example .env
```

```bash
# macOS or Linux
cp .env.example .env
```

Add the supplied values for `SUPABASE_AUTH_EXTERNAL_GOOGLE_CLIENT_ID` and `SUPABASE_AUTH_EXTERNAL_GOOGLE_CLIENT_SECRET`. Do not commit `.env`.

From the repository root:

```bash
npm run setup   # first time; checks tools and installs dependencies
npm run dev     # local database, backend, and frontend
```

Open <http://localhost:3000>. Press Ctrl+C to stop the local services. `npm run setup` starts Supabase briefly to validate the environment, then stops it. `npm run dev` starts Supabase on ports 54321 and 54322, the Go backend on 8080, and Next.js on 3000. All data stays in your local Docker environment.

The operation methods start as stubs. Search for `TODO(student)` to find the unfinished RPC, application, Postgres, and SQL work. Requests for unfinished operations return Connect `Unimplemented`; the starter behavior tests run and fail until your pair implements them.

## Find your code

| Path | Purpose |
| --- | --- |
| `proto/carpool/v1/` | API messages and request validation |
| `backend/internal/rpc/` | Parse requests, call the application service, and build responses |
| `backend/internal/app/` | Business rules, authorization, transactions, and contact privacy |
| `backend/internal/db/postgres/` | Convert application records to and from sqlc calls |
| `backend/internal/db/queries/` | SQL query source; assigned queries have headers to fill |
| `supabase/migrations/` | Database schema |
| `frontend/src/` | Supplied client |

Follow one request through files with matching names: `profile.go`, `locations.go`, `ride_details.go`, `ride_management.go`, `ride_membership.go`, or `ride_lists.go`. The application owns the repository interfaces in `backend/internal/app/repository.go`. Pairs can code against those interfaces while another pair completes a shared repository method. Coordinate early on `FindUser`, `GetRide`, and `AddRideOccupant`.

## Backend rules

- Read the verified caller through `rpc.actorFromContext`. Request IDs select targets; they do not establish caller identity.
- Begin every storage operation through `app.Database.BeginTx`. Repositories from that transaction share one Serializable snapshot. Defer rollback; build an authorized mutation result before committing.
- Use `app.projectUser` and `app.projectRides` for contact visibility. Do not add raw stored contacts to RPC responses.
- Keep RPC methods focused on parsing, the service call, error mapping, and protobuf conversion. Business rules belong in `app`; sqlc conversion belongs in `db/postgres`.
- Edit `.proto` and `.sql` sources, then regenerate. Do not edit `backend/internal/gen/`, `backend/internal/db/sqlc/`, or `frontend/src/gen/` by hand.

## Tests and generation

Run backend commands from `backend/` and frontend commands from `frontend/`:

```bash
cd backend
go build ./...
go test ./...
sqlc generate
sqlc diff
```

There are two active starter cases per operation. They are expected to fail at first. Make your assigned cases pass, then add tests for other rules and edge cases. The operation test check in CI is visible but initially does not block merges; build and generation checks must pass.

For database integration tests, use `backend/internal/testutil.SetupTestDB`. It starts a disposable PostgreSQL container when Docker is available, or uses an explicitly disposable `TEST_DATABASE_URL`. It ignores `DATABASE_URL`.

For protobuf generation, run `buf generate` in `proto/`. For frontend generation, run `npm run generate` in `frontend/`. The repository pins buf 1.73.0 in CI and sqlc 1.31.1. Install those tools when you begin changing the contract or queries.

Other root commands: `npm run db:start`, `npm run db:reset`, `npm run db:stop`, `npm run backend`, and `npm run frontend`.

## Contributors

- Andrew Chu
- Calvin Wong
