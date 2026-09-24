-- ListContactVisibleUserIDs returns target IDs whose contacts the viewer may see. A shared ride
-- counts even when it is in the past or cancelled; self is always visible.
-- Fetch all matches in one query so ride lists do not query once per rider.
-- name: ListContactVisibleUserIDs :many
WITH viewer_rides AS (
    SELECT id AS ride_id FROM rides WHERE owner_id = sqlc.arg('viewer_id')
    UNION
    SELECT ride_id FROM ride_occupants WHERE user_id = sqlc.arg('viewer_id')
)
SELECT u.id
FROM users u
WHERE u.id = ANY(sqlc.arg('target_ids')::uuid[])
  AND (
      u.id = sqlc.arg('viewer_id')
      OR EXISTS (
          SELECT 1
          FROM viewer_rides vr
          JOIN rides r ON r.id = vr.ride_id
          WHERE r.owner_id = u.id
             OR EXISTS (
                 SELECT 1 FROM ride_occupants target_ro
                 WHERE target_ro.ride_id = vr.ride_id AND target_ro.user_id = u.id
             )
      )
  );
