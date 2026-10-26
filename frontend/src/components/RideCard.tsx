import Link from "next/link";
import type { Ride } from "../gen/carpool/v1/ride_pb";
import { availableSeats, formatDeparture, rideState } from "../lib/ride";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { Card, CardContent } from "./ui/card";

export function RideCard({ ride }: { ride: Ride }) {
  const state = rideState(ride);
  const seats = availableSeats(ride);
  const from = ride.departureLocation?.title || "Departure";
  const to = ride.arrivalLocation?.title || "Arrival";

  return (
    <Card className="py-5 shadow-none">
      <CardContent className="space-y-4">
        <div>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <h2>
              {from} to {to}
            </h2>
            <Badge variant={state === "cancelled" ? "outline" : "secondary"}>
              {state === "open"
                ? `${seats} ${seats === 1 ? "seat" : "seats"} left`
                : state}
            </Badge>
          </div>
          <p className="mt-2 text-sm text-muted-foreground">
            {formatDeparture(ride)}
          </p>
        </div>
        <div className="flex flex-wrap items-center justify-between gap-3 border-t pt-4">
          <span className="text-sm text-muted-foreground">
            {ride.owner?.firstName || "Rice student"} is driving
          </span>
          <Button variant="outline" size="sm" asChild>
            <Link
              href={`/rides/${ride.id}`}
              aria-label={`View ride from ${from} to ${to}`}
            >
              View ride
            </Link>
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
