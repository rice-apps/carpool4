# Rice Carpool

Rice Carpool is a ridesharing platform built for the Rice University community. It connects students to coordinate shared travel, making it simple to organize and join rides between Rice, Houston airports, and surrounding areas.

---

## Prerequisites

Ensure you have the following installed before getting started:

- **Go**
- **Node.js**
- **Docker Desktop**

> [!NOTE]
> Request the shared **Google OAuth development credentials** from the leads before starting local configuration.

---

## Getting Started

### 1. Environment Configuration

1. Start **Docker Desktop**.
2. Create a local `.env` file from `.env.example`:

   ```bash
   # macOS or Linux
   cp .env.example .env
   ```

   ```powershell
   # Windows PowerShell
   Copy-Item .env.example .env
   ```

3. Add the supplied Google OAuth values to your `.env` file:
   - `SUPABASE_AUTH_EXTERNAL_GOOGLE_CLIENT_ID`
   - `SUPABASE_AUTH_EXTERNAL_GOOGLE_CLIENT_SECRET`

> [!WARNING]
> Do **not** commit `.env` to version control.

### 2. Installation & Running

Run the following commands from the repository root:

```bash
# First time setup: checks tools and installs dependencies
npm run setup

# Start local database, backend, and frontend
npm run dev
```

_Press `Ctrl+C` to stop local services._

### Port & Service Overview

| Service                 | Port(s)          | Description                                   |
| :---------------------- | :--------------- | :-------------------------------------------- |
| **Frontend**            | `3000`           | Next.js application (<http://localhost:3000>) |
| **Backend**             | `8080`           | Go Connect RPC server                         |
| **Supabase / Database** | `54321`, `54322` | Local PostgreSQL instance                     |

> [!NOTE]
>
> - `npm run setup` starts Supabase briefly to validate the environment, then stops it.
> - `npm run dev` keeps all data localized within your local Docker environment.

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

> [!IMPORTANT]
> Follow these guidelines strictly when writing backend logic:

1. **Caller Verification**: Read the verified caller context through `rpc.actorFromContext`. Request IDs select target entities; they do **not** establish caller identity.
2. **Transaction Integrity**: Begin every storage operation via `app.Database.BeginTx`. Repositories created from that transaction share a single `Serializable` snapshot. Defer rollback and build an authorized mutation result before committing.
3. **Contact Privacy**: Enforce privacy rules using `app.projectUser` and `app.projectRides`. Do **not** expose raw stored contact information in RPC responses.
4. **Layer Separation**:
   - **RPC layer**: Parsing, service invocation, error mapping, and protobuf conversion.
   - **App layer (`app/`)**: Business logic and domain rules.
   - **DB layer (`db/postgres/`)**: `sqlc` data conversion.
5. **Code Generation**: Edit `.proto` and `.sql` source files, then regenerate code. **Do not** manually edit generated files in `backend/internal/gen/`, `backend/internal/db/sqlc/`, or `frontend/src/gen/`.

---

## Testing & Code Generation

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
npm run db:start   # Start local Supabase database
npm run db:stop    # Stop Supabase database
npm run db:reset   # Reset local Supabase database
npm run backend    # Run Go backend individually
npm run frontend   # Run Next.js frontend individually
```

### Testing Guidelines

- **Database Integration Tests**: Use `backend/internal/testutil.SetupTestDB`. It boots a disposable PostgreSQL container when Docker is available or uses an explicitly set `TEST_DATABASE_URL`. It ignores standard `DATABASE_URL`.
- **CI Pipeline**: Operation test checks are visible in CI (initially non-blocking to allow incremental merges). Build and code generation checks **must** pass.

---

## Contributors

Built by [RiceApps Studio](https://riceapps.org/students) :>
