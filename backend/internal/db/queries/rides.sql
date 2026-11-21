-- GetRide: Load one ride with its owner, locations, and occupants.
-- TODO(student): Write the GetRide SQL query.

-- CreateRide: Store a new ride and return its ID.
-- TODO(student): Write the CreateRide SQL query.

-- UpdateRide: Replace a ride's editable fields.
-- TODO(student): Write the UpdateRide SQL query.

-- CancelRide: Mark a ride cancelled while retaining its history.
-- TODO(student): Write the CancelRide SQL query.

-- AddRideOccupant: Record a user's ride membership.
-- TODO(student): Write the AddRideOccupant SQL query.

-- RemoveRideOccupant: Remove a user's ride membership and report whether a row changed.
-- TODO(student): Write the RemoveRideOccupant SQL query.

-- ListRides: Find active rides matching the supplied route and time filters.
-- TODO(student): Write the ListRides SQL query.
-- name: ListRides :many
SELECT sqlc.embed(r),
       sqlc.embed(dl),
       sqlc.embed(al),
       ride_riders(r.id) AS riders
    FROM rides r
        INNER JOIN locations AS dl ON rides.departure_location_id = dl.id
        INNER JOIN locations AS al ON rides.arrival_location_id = al.id WHERE rides.status = 'active'
            AND (sqlc.narg('departure_location_id')::uuid IS NULL) OR (rides.departure_location_id = sqlc.narg('departure_location_id'))
            AND (sqlc.narg('arrival_location_id')::uuid IS NULL) OR (rides.arrival_location_id = sqlc.narg('arrival_location_id'))
            AND (sqlc.narg('departure_before')::timestamptz IS NULL) OR (rides.departure_date <= sqlc.narg('departure_before'))
            AND (rides.departure_date >= sqlc.narg('departure_after'))
                 ORDER BY rides.departure_date ASC, rides.id ASC;
-- ListMyRides: Return the viewer's owned and joined rides, including history.
-- TODO(student): Write the ListMyRides SQL query.
