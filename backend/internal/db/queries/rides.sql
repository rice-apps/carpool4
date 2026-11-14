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
-- name: ListRides :exec
SELECT * FROM rides LEFT JOIN ride_occupants ON rides.id = ride_occupants.ride_id
                    LEFT JOIN locations AS L1 ON rides.departure_location_id = L1.id
                    LEFT JOIN locations AS L2 ON rides.arrival_location_id = L2.id WHERE rides.status = 'active'
                                                                                     AND (sqlc.narg('departure_location_id')::departure_location_id IS NULL OR rides.departure_location_id = sqlc.narg('departure_location_id')
        AND (sqlc.narg('arrival_location_id')::arrival_location_id IS NULL OR rides.arrival_location_id = sqlc.narg('arrival_location_id')
        AND (sqlc.narg('DepartureBefore')::DepartureBefore IS NULL OR rides.departure_date BETWEEN sqlc.narg('DepartureAfter') AND sqlc.narg('DepartureBefore'))
                 ORDER BY rides.departure_date DESC, rides.id ASC;
-- ListMyRides: Return the viewer's owned and joined rides, including history.
-- TODO(student): Write the ListMyRides SQL query.
