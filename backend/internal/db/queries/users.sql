-- FindUser: Load one profile by user ID.
-- TODO(student): Write the FindUser SQL query.
-- name: FindUser :one
SELECT *
FROM users
WHERE id = sqlc.arg(id);

-- UpsertUser: Save a profile and return its stored values.
-- TODO(student): Write the UpsertUser SQL query.
-- name: UpsertUser :one
INSERT INTO users (id, first_name, last_name, email, phone)
VALUES(sqlc.arg(id),sqlc.arg(first_name), sqlc.arg(last_name),sqlc.arg(email),sqlc.arg(phone))
ON CONFLICT(id)
DO UPDATE SET
   id = EXCLUDED.id,
   first_name = EXCLUDED.first_name,
   last_name = EXCLUDED.last_name,
   email = EXCLUDED.email,
   phone = EXCLUDED.phone
RETURNING *;