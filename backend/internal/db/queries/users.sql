-- FindUser: Load one profile by user ID.
-- name: FindUser :one
SELECT id, first_name, last_name, email, phone, created_at, updated_at
FROM users
WHERE id = $1;

-- UpsertUser: Save a profile and return its stored values.
-- name: UpsertUser :one
INSERT INTO users (id, first_name, last_name, email, phone)
VALUES($1,$2, $3,$4,$5)
ON CONFLICT(id)
DO UPDATE SET
   id = EXCLUDED.id,
   first_name = EXCLUDED.first_name,
   last_name = EXCLUDED.last_name,
   email = EXCLUDED.email,
   phone = EXCLUDED.phone
RETURNING id, first_name, last_name, email, phone, created_at, updated_at;
