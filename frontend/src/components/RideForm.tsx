"use client";

import { useRef, useState, type FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "./ui/button";
import { Card, CardContent } from "./ui/card";
import { Input } from "./ui/input";
import { NativeSelect } from "./ui/native-select";
import { Textarea } from "./ui/textarea";
import {
  createRide,
  listLocations,
  updateRide,
} from "../gen/carpool/v1/ride-RideService_connectquery";
import { getUser } from "../gen/carpool/v1/user-UserService_connectquery";
import type { Ride } from "../gen/carpool/v1/ride_pb";
import { departureDate, localDateTime } from "../lib/ride";
import { requestErrorMessage } from "../lib/requestError";

export function RideForm({ ride }: { ride?: Ride }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const locations = useQuery(listLocations, {});
  const profile = useQuery(getUser, {});
  const create = useMutation(createRide);
  const update = useMutation(updateRide);
  const submitting = useRef(false);
  const [from, setFrom] = useState(ride?.departureLocation?.id || "");
  const [to, setTo] = useState(ride?.arrivalLocation?.id || "");
  const [when, setWhen] = useState(
    ride && departureDate(ride) ? localDateTime(departureDate(ride)!) : "",
  );
  const [capacity, setCapacity] = useState(ride?.capacity || 4);
  const [notes, setNotes] = useState(ride?.notes || "");
  const [error, setError] = useState("");
  const minCapacity = ride?.riders.length || 1;
  const isPending = create.isPending || update.isPending;

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submitting.current) return;
    setError("");
    const departure = new Date(when);
    if (!from || !to) return setError("Choose both locations.");
    if (from === to)
      return setError("Choose different departure and arrival locations.");
    if (
      !when ||
      Number.isNaN(departure.getTime()) ||
      departure.getTime() <= Date.now()
    )
      return setError("Choose a future departure time.");
    if (localDateTime(departure) !== when)
      return setError("Choose a valid local departure time.");
    if (!Number.isInteger(capacity) || capacity < minCapacity)
      return setError(
        `Capacity must be at least ${minCapacity}, including everyone already on this ride.`,
      );
    if (notes.length > 500)
      return setError("Keep notes to 500 characters or fewer.");

    submitting.current = true;
    try {
      const input = {
        departureLocationId: from,
        arrivalLocationId: to,
        departureDate: timestampFromDate(departure),
        capacity,
        notes: notes.trim(),
      };
      const result = ride
        ? await update.mutateAsync({ id: ride.id, ...input })
        : await create.mutateAsync(input);
      await queryClient.invalidateQueries();
      const savedRideId = result.ride?.id || ride?.id;
      router.push(savedRideId ? `/rides/${savedRideId}` : "/my-rides");
    } catch (cause) {
      setError(
        requestErrorMessage(
          cause,
          ride ? "Updating rides" : "Posting rides",
          cause instanceof Error
            ? cause.message
            : "Couldn’t save the ride. Try again.",
        ),
      );
      submitting.current = false;
    }
  }

  return (
    <div className="page-container form-page">
      <Link href={ride ? `/rides/${ride.id}` : "/"} className="back-link">
        {ride ? "Back to ride" : "Back to rides"}
      </Link>
      <div className="form-intro">
        <h1>{ride ? "Edit your ride" : "Post a ride"}</h1>
        <p>Add your route, departure time, and available space.</p>
      </div>
      {!ride && profile.data?.user && !profile.data.user.phone && (
        <p className="mb-5 rounded-md border bg-muted p-4">
          Add a phone number to your{" "}
          <Link className="text-link" href="/profile">
            profile
          </Link>{" "}
          before posting. Riders need a way to reach you.
        </p>
      )}
      <Card className="shadow-none">
        <CardContent>
          <form className="space-y-6" onSubmit={(event) => void save(event)}>
            <div className="form-grid two-columns">
              <label className="field">
                <span>Leaving from</span>
                <NativeSelect
                  required
                  value={from}
                  onChange={(event) => setFrom(event.target.value)}
                >
                  <option value="">Select a location</option>
                  {locations.data?.locations.map((location) => (
                    <option value={location.id} key={location.id}>
                      {location.title}
                    </option>
                  ))}
                </NativeSelect>
              </label>
              <label className="field">
                <span>Going to</span>
                <NativeSelect
                  required
                  value={to}
                  onChange={(event) => setTo(event.target.value)}
                >
                  <option value="">Select a location</option>
                  {locations.data?.locations.map((location) => (
                    <option value={location.id} key={location.id}>
                      {location.title}
                    </option>
                  ))}
                </NativeSelect>
              </label>
            </div>
            {locations.error && (
              <p className="form-error" role="alert">
                {requestErrorMessage(
                  locations.error,
                  "Getting locations",
                  "Couldn’t load locations.",
                )}{" "}
                <button type="button" onClick={() => void locations.refetch()}>
                  Try again
                </button>
              </p>
            )}
            {!ride && profile.error && (
              <p className="form-error" role="alert">
                {requestErrorMessage(
                  profile.error,
                  "Getting your profile",
                  "Couldn’t load your profile.",
                )}{" "}
                <button type="button" onClick={() => void profile.refetch()}>
                  Try again
                </button>
              </p>
            )}
            <div className="form-grid two-columns">
              <label className="field">
                <span>Departure date and time</span>
                <Input
                  type="datetime-local"
                  required
                  value={when}
                  min={localDateTime(new Date())}
                  onChange={(event) => setWhen(event.target.value)}
                />
              </label>
              <label className="field">
                <span>Total seats, including you</span>
                <Input
                  type="number"
                  min={minCapacity}
                  step={1}
                  required
                  value={capacity}
                  onChange={(event) => setCapacity(Number(event.target.value))}
                  aria-describedby="capacity-help"
                />
                <small id="capacity-help">
                  {ride
                    ? `${ride.riders.length} people are already on this ride.`
                    : "For example, 4 means you and 3 riders."}
                </small>
              </label>
            </div>
            <label className="field">
              <span>Notes for riders (optional)</span>
              <Textarea
                value={notes}
                onChange={(event) => setNotes(event.target.value)}
                maxLength={500}
                rows={4}
                placeholder="Where to meet, luggage space, or anything else helpful"
                aria-describedby="notes-help"
              />
              <small id="notes-help">{notes.length}/500 characters</small>
            </label>
            {error && (
              <p className="form-error" role="alert">
                {error}
              </p>
            )}
            <div className="form-actions">
              <Button
                type="submit"
                disabled={
                  isPending ||
                  locations.isLoading ||
                  (!ride && !profile.data?.user?.phone)
                }
              >
                {isPending ? "Saving…" : ride ? "Save ride" : "Post ride"}
              </Button>
              <Button variant="outline" asChild>
                <Link href={ride ? `/rides/${ride.id}` : "/"}>Cancel</Link>
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
