import Link from "next/link";
import { ArrowUpRight, Clock, Users } from "@phosphor-icons/react";
import type { Ride } from "../gen/carpool/v1/ride_pb";
import {
  availableSeats,
  departureTime,
  formatDeparture,
  rideState,
} from "../lib/ride";

export function RideCard({ ride }: { ride: Ride }) {
  const state = rideState(ride);
  const departure = departureTime(ride);

  return (
    <Link href={`/rides/${ride.id}`} className="ride-card">
      <span className="ride-date" aria-hidden="true">
        <strong>
          {departure
            ? new Intl.DateTimeFormat("en-US", { day: "2-digit" }).format(
                departure,
              )
            : "--"}
        </strong>
        <span>
          {departure
            ? new Intl.DateTimeFormat("en-US", { month: "short" }).format(
                departure,
              )
            : "Date"}
        </span>
      </span>
      <span className="ride-card-main">
        <span className="ride-route">
          <strong>{ride.departureLocation?.title || "Departure"}</strong>
          <span className="route-line" />
          <strong>{ride.arrivalLocation?.title || "Arrival"}</strong>
        </span>
        <span className="ride-meta">
          <span>
            <Clock size={16} />
            {formatDeparture(ride, { hour: "numeric", minute: "2-digit" })}
          </span>
          <span>
            <Users size={16} />
            {ride.owner?.firstName || "Rice student"} is driving
          </span>
        </span>
      </span>
      <span className="ride-card-end">
        <span className={`status-pill status-${state}`}>
          {state === "open"
            ? `${availableSeats(ride)} ${availableSeats(ride) === 1 ? "seat" : "seats"} left`
            : state}
        </span>
        <ArrowUpRight size={20} />
      </span>
    </Link>
  );
}
