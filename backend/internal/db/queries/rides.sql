-- GetRide: Load one ride with its owner, locations, and occupants.
-- TODO(student): Write the GetRide SQL query.

-- CreateRide: Store a new ride and return its ID.
-- TODO(student): Write the CreateRide SQL query.
-- name: CreateRide :one
insert into rides (departure_time, departure_location_id, arrival_location_id, owner_id, notes, capacity, status)
values (sqlc.arg(departure_time), sqlc.arg(departure_location_id), sqlc.arg(arrival_location_id), sqlc.arg(owner_id), sqlc.arg(notes), sqlc.arg(capacity), sqlc.arg(status))
returning id;
-- UpdateRide: Replace a ride's editable fields.
-- TODO(student): Write the UpdateRide SQL query.
-- name: UpdateRide :execrows
    UPDATE rides
        SET departure_time = sqlc.arg(departure_time),
            arrival_location_id = sqlc.arg(arrival_location_id),
            departure_location_id = sqlc.arg(departure_location_id),
            capacity = sqlc.arg(capacity),
            notes = sqlc.arg(notes),
            updated_at = now()
        WHERE id = sqlc.arg(id);

-- CancelRide: Mark a ride cancelled while retaining its history.
-- TODO(student): Write the CancelRide SQL query.

-- AddRideOccupant: Record a user's ride membership.
-- TODO(student): Write the AddRideOccupant SQL query.

-- RemoveRideOccupant: Remove a user's ride membership and report whether a row changed.
-- TODO(student): Write the RemoveRideOccupant SQL query.

-- ListRides: Find active rides matching the supplied route and time filters.
-- TODO(student): Write the ListRides SQL query.

-- ListMyRides: Return the viewer's owned and joined rides, including history.
-- TODO(student): Write the ListMyRides SQL query.
