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

-- ListMyRides: Return the viewer's owned and joined rides, including history.
-- TODO(student): Write the ListMyRides SQL query.
-- name: ListMyRides :many
SELECT
    sqlc.embed(r),
    sqlc.embed(u),
    sqlc.embed(departure),
    sqlc.embed(arrival),
    ride_riders(r.id) riders
FROM rides r
         INNER JOIN ride_occupants ro
                    ON r.id = ro.ride_id
         INNER JOIN users u
                    ON ro.user_id = u.id
         INNER JOIN locations departure
                    ON r.departure_location_id = departure.id
         INNER JOIN locations arrival
                    ON r.arrival_location_id = arrival.id
WHERE ro.user_id = sqlc.arg(viewerID);