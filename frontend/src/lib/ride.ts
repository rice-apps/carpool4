import { timestampDate } from "@bufbuild/protobuf/wkt";
import type { Ride } from "../gen/carpool/v1/ride_pb";
import { RideStatus } from "../gen/carpool/v1/ride_pb";

export function departureTime(ride: Ride): Date | null {
  return ride.departureTime ? timestampDate(ride.departureTime) : null;
}

export function formatDeparture(
  ride: Ride,
  options?: Intl.DateTimeFormatOptions,
): string {
  const date = departureTime(ride);
  if (!date) return "Time unavailable";
  return new Intl.DateTimeFormat(
    "en-US",
    options || {
      weekday: "short",
      month: "short",
      day: "numeric",
      hour: "numeric",
      minute: "2-digit",
    },
  ).format(date);
}

export function availableSeats(ride: Ride): number {
  return Math.max(0, ride.capacity - ride.occupiedSeats);
}

export function rideState(ride: Ride): "cancelled" | "past" | "full" | "open" {
  if (ride.status === RideStatus.CANCELLED) return "cancelled";
  const date = departureTime(ride);
  if (date && date.getTime() < Date.now()) return "past";
  return availableSeats(ride) === 0 ? "full" : "open";
}

export function localDateTime(date: Date): string {
  const parts = [
    date.getFullYear(),
    String(date.getMonth() + 1).padStart(2, "0"),
    String(date.getDate()).padStart(2, "0"),
  ];
  return `${parts.join("-")}T${String(date.getHours()).padStart(2, "0")}:${String(date.getMinutes()).padStart(2, "0")}`;
}
