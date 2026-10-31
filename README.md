# Rice Carpool

Rice Carpool is a ridesharing platform built for the Rice University community. It connects students to coordinate shared travel, making it simple to organize and join rides between Rice, Houston airports, and surrounding areas.

---

## Getting Started

Install **Go 1.27.1+** and **Node 20.19+**. Obtain your pair's complete configuration from the leads and save it as the root `.env`. Keep it out of version control.

```bash
npm run setup
npm run dev
```

The frontend runs on http://localhost:3000 and the Go API on http://localhost:8080. The database and Google sign-in are hosted. Development data is shared; SQL tests get private fixtures automatically. Press Ctrl+C to stop the local app processes.

Your supplied `.env` contains `DATABASE_URL`, `TEST_DATABASE_URL`, `SUPABASE_URL`, and `SUPABASE_ANON_KEY`. The provided helpers load these settings and select the right database objects; write ordinary SQL and application code as usual. Students need no administrator or OAuth client credentials.

---

## Finding Your Code

### Workspace Structure

| Path                            | Purpose                                                                   |
| :------------------------------ | :------------------------------------------------------------------------ |
| `proto/carpool/v1/`             | API Protocol Buffer messages and request validation schemas               |
| `backend/internal/rpc/`         | Parse requests, invoke application services, and construct RPC responses  |
| `backend/internal/app/`         | Business rules, authorization, database transactions, and contact privacy |
| `backend/internal/db/postgres/` | Convert application records to and from `sqlc` database calls             |
| `backend/internal/db/queries/`  | SQL query sources; assigned queries contain headers to fill               |
| `supabase/migrations/`          | Database schema migrations                                                |
| `frontend/src/`                 | Pre-built frontend client                                                 |

---

## Backend Rules

See [What the provided helpers do](HELPERS.md) for explanations and examples of sign-in identity,
contact privacy, database access, conversion, and error handling.

> [!IMPORTANT]
> Follow these guidelines strictly when writing backend logic:

1. **Caller Verification**: `ListRides`, `GetRide`, and `ListLocations` allow a missing bearer token. A supplied invalid token is rejected. Every other RPC requires sign-in and uses `rpc.getCurrentUser` for the caller's identity. Public handlers can check `auth.IsSignedIn` before retrieving identity. Protected app methods rely on verified identity and do not repeat the sign-in check; direct service callers must supply verified identity too. Request IDs select target entities; they do **not** establish caller identity.
2. **Transaction Integrity**: Begin every storage operation via `app.Database.BeginTx`. Repositories created from that transaction share a single `Serializable` snapshot. Defer rollback and build an authorized mutation result before committing.
3. **Contact Privacy**: Enforce privacy rules using `app.getVisibleProfile` and `app.buildRidesForViewer`. Guest ride responses include trip facts and `occupied_seats`, but no owner, riders, or notes. Signed-in users see contacts only when permitted. Do **not** expose raw stored contact information in RPC responses.
4. **Layer Separation**:
   - **RPC layer**: Parsing, service invocation, error mapping, and protobuf conversion.
   - **App layer (`app/`)**: Business logic and domain rules.
   - **DB layer (`db/postgres/`)**: `sqlc` data conversion.
5. **Code Generation**: Edit `.proto` and `.sql` source files, then regenerate code. **Do not** manually edit generated files in `backend/internal/gen/`, `backend/internal/db/sqlc/`, or `frontend/src/gen/`.

### Public browsing contract

The assigned backend operations start as stubs. Their implementations must follow these rules:

- Visitors can list locations and active rides without signing in. Discovery includes full rides and excludes departures more than one hour in the past.
- A visitor can open a ride only while it qualifies for discovery. Cancelled and older rides return `NotFound` to visitors; signed-in users can still open their ride history.
- Public ride results include the route, departure time, capacity, status, and occupied seats (including the driver). They omit the owner, riders, and notes. Signed-in results retain the existing contact privacy rules.
- Discovery and ride detail pages are public. Profile, ride history, creation, editing, and ride actions require sign-in and a completed profile. After sign-in and onboarding, the frontend returns users to their intended page.

---

## Testing & Code Generation

The student and reference code use `CurrentUser.ID` and `DepartureDate`. The protobuf and database
field is `departure_date`; generated TypeScript uses `departureDate`.

### Commands Quick Reference

#### Backend Development (run from `backend/`)

```bash
go build ./...    # Compile backend packages
go test ./...     # Run backend test suite
sqlc generate     # Generate Go code from SQL queries
sqlc diff         # Verify SQL query changes against database schema
```

#### Code Generation Tools

- **Protobuf**: Run `buf generate` in `proto/`.
- **Frontend**: Run `npm run generate` in `frontend/`.

#### Root Convenience Scripts

```bash
npm run setup         # Install packages and verify the supplied configuration
npm run dev           # Run local backend and frontend
npm run backend       # Run only the backend
npm run frontend      # Run only the frontend
npm run test:backend  # Run backend checks using the root .env
```

### Testing Guidelines

- **Database Integration Tests**: Use `backend/internal/testutil.SetupTestDB`. It loads your root `.env`, prepares fresh private fixtures, and cleans up automatically. Test connections cannot reset shared development data. `go test -short ./...` omits SQL fixtures.
- **CI Pipeline**: Operation test checks are visible in CI (initially non-blocking to allow incremental merges). Build and code generation checks **must** pass.
- **Frontend Checks**: From `frontend/`, run `npm test`, `npm run lint`, `npx tsc --noEmit`, and `npm run build`. Install the browser once with `npx playwright install chromium`, then run `npm run test:e2e` for mocked public-browsing flows.

---

## Contributors

Built by [RiceApps Studio](https://riceapps.org/students) :>

- Andrew Chu
- Angel  J Guerrero
- Audrey Zheng
- Calvin Wong
- Claire Li
- Dara Odukoya
- Eason Tang
- Ella Wang
- Emily Zheng
- Katherine Han
- Joya Roy
- Liya Ferede
- Marianna Argeiti
- Megan Cung
- Melody Cui
- Michael Wang
- Yiyi Sun
- Zimo Wang
