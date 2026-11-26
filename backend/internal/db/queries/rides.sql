-- GetRide: Load one ride with its owner, locations, and occupants.
-- name: GetRide :one
SELECT sqlc.embed(r),
       sqlc.embed(u),
       sqlc.embed(dl),
       sqlc.embed(al),
       ride_riders(r.id) riders
FROM rides r
         JOIN users u
              ON r.owner_id = u.id
         JOIN locations dl
              ON r.departure_location_id = dl.id
         JOIN locations al
              ON r.arrival_location_id = al.id
WHERE r.id = $1;

-- CreateRide: Store a new ride and return its ID.
-- name: CreateRide :one
insert into rides (departure_time, departure_location_id, arrival_location_id, owner_id, notes, capacity, status)
values (sqlc.arg(departure_time), sqlc.arg(departure_location_id), sqlc.arg(arrival_location_id), sqlc.arg(owner_id),
        sqlc.arg(notes), sqlc.arg(capacity), sqlc.arg(status)) returning id;

-- UpdateRide: Replace a ride's editable fields.
-- name: UpdateRide :execrows
UPDATE rides
SET departure_time        = sqlc.arg(departure_time),
    arrival_location_id   = sqlc.arg(arrival_location_id),
    departure_location_id = sqlc.arg(departure_location_id),
    capacity              = sqlc.arg(capacity),
    notes                 = sqlc.arg(notes),
    updated_at            = now()
WHERE id = sqlc.arg(id);

-- CancelRide: Mark a ride cancelled while retaining its history.
-- name: CancelRide :execrows
UPDATE rides
SET status     = 'cancelled',
    updated_at = now()
WHERE id = $1;

-- AddRideOccupant: Record a user's ride membership.
-- name: AddRideOccupant :exec
INSERT INTO ride_occupants (ride_id, user_id)
VALUES ($1, $2);

-- RemoveRideOccupant: Remove a user's ride membership and report whether a row changed.
-- name: RemoveRideOccupant :execrows
DELETE
FROM ride_occupants
WHERE ride_id = $1
  and user_id = $2;

-- ListRides: Find active rides matching the supplied route and time filters.
-- name: ListRides :many
SELECT sqlc.embed(r),
       sqlc.embed(dl),
       sqlc.embed(al),
       sqlc.embed(u),
       ride_riders(r.id) AS riders
FROM rides r
         INNER JOIN locations AS dl ON r.departure_location_id = dl.id
         INNER JOIN locations AS al ON r.arrival_location_id = al.id
         INNER JOIN users AS u ON u.id = r.owner_id
WHERE r.status = 'active'
  AND (sqlc.narg('departure_location_id')::uuid IS NULL OR r.departure_location_id = sqlc.narg('departure_location_id'))
  AND (sqlc.narg('arrival_location_id')::uuid IS NULL OR r.arrival_location_id = sqlc.narg('arrival_location_id'))
  AND (sqlc.narg('departure_before')::timestamptz IS NULL OR r.departure_time <= sqlc.narg('departure_before'))
  AND (r.departure_time >= sqlc.arg('departure_after'))
ORDER BY r.departure_time ASC, r.id ASC;

-- ListMyRides: Return the viewer's owned and joined rides, including history.
-- name: ListMyRides :many
SELECT sqlc.embed(r),
       sqlc.embed(u),
       sqlc.embed(dl),
       sqlc.embed(al),
       ride_riders(r.id) riders
FROM rides r
         INNER JOIN users u
                    ON u.id = r.owner_id
         INNER JOIN locations dl
                    ON r.departure_location_id = dl.id
         INNER JOIN locations al
                    ON r.arrival_location_id = al.id
WHERE r.owner_id = sqlc.arg(viewerID)
   OR EXISTS (SELECT 1
              FROM ride_occupants ro
              WHERE ro.ride_id = r.id
                AND ro.user_id = sqlc.arg(viewerID))
ORDER BY r.departure_time DESC, r.id DESC;
