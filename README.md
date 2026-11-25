# Rice Carpool

Rice Carpool is a ridesharing platform built for the Rice University community. It connects students to coordinate shared travel, making it simple to organize and join rides between Rice, Houston airports, and surrounding areas.

---

## Getting Started

You need **Go 1.27.1+** and **Node 20.19+**. Get the `.env` from the leads and put it at the repo root. Do not commit it.

```bash
npm run setup
npm run dev
```

The frontend runs at http://localhost:3000. The API runs at http://localhost:8080. The database and Google sign-in are hosted, and development data is shared across the team. Ctrl+C stops everything.

The `.env` holds `DATABASE_URL`, `SUPABASE_URL`, and `SUPABASE_ANON_KEY`, which the helpers read to run the app. Write SQL and application code the normal way. You do not need admin credentials or an OAuth client.

Tests need none of this. `go test ./...` starts its own PostgreSQL container, applies the migrations and throws it away; the only prerequisite is a running Docker daemon.

If your Docker runtime does not provide `/var/run/docker.sock`, point Testcontainers at your socket once:

```bash
export DOCKER_HOST="$(docker context inspect --format '{{.Endpoints.docker.Host}}')"
export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock
```

---

## Testing & Code Generation


### Commands Quick Reference

#### Backend Development (run from `backend/`)

```bash
go build ./...        # Compile backend packages
go test ./...         # Run the backend test suite
sqlc generate         # Generate Go code from SQL queries
sqlc diff             # Verify SQL query changes against database schema
```

#### Code Generation

- **Protobuf**: `buf generate` in `proto/`
- **Frontend**: `npm run generate` in `frontend/`

#### Root Scripts

```bash
npm run setup         # Install packages and verify config
npm run dev           # Start backend + frontend
npm run backend       # Backend only
npm run frontend      # Frontend only
npm run test:backend  # Backend checks using root .env
```



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
