#!/usr/bin/env bash
set -euo pipefail

# The Ubuntu runner provides native PostgreSQL. All data and credentials are disposable.
sudo systemctl start postgresql.service
sudo -u postgres psql -v ON_ERROR_STOP=1 <<'SQL'
ALTER ROLE postgres PASSWORD 'carpool-ci-only';
CREATE DATABASE carpool_ci;
SQL
export ADMIN_DATABASE_URL='postgresql://postgres:carpool-ci-only@127.0.0.1:5432/carpool_ci?sslmode=disable'
export PGPASSWORD=carpool-ci-only
psql -h 127.0.0.1 -U postgres -d carpool_ci -v ON_ERROR_STOP=1 <<'SQL'
CREATE ROLE anon NOLOGIN;
CREATE ROLE authenticated NOLOGIN;
CREATE ROLE supabase_auth_admin NOLOGIN;
CREATE SCHEMA auth;
CREATE TABLE auth.users(id uuid PRIMARY KEY);
SQL
export SUPABASE_URL=https://ci.example.invalid
export SUPABASE_ANON_KEY=ci-public-key
go -C backend run ./cmd/devdb provision
cp .local/pairs/profile.env .env
