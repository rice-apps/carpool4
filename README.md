# Rice Carpool

Rice Carpool is a ridesharing platform built for the Rice University community. It connects students to coordinate shared travel, making it simple to organize and join rides between Rice, Houston airports, and surrounding areas.

---

## Getting Started

You need **Go 1.27+** and **Node 20.19+**.

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
- Angel J Guerrero
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
