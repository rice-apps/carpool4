"use client";

import { useRef, useState, type FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, ArrowRight, Info, MapPin } from "@phosphor-icons/react";
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
        <ArrowLeft size={17} /> {ride ? "Back to ride" : "Back to rides"}
      </Link>
      <div className="form-intro">
        <span className="section-kicker">
          {ride ? "Update your trip" : "Bring someone along"}
        </span>
        <h1>{ride ? "Edit your ride" : "Post a ride"}</h1>
        <p>
          {ride
            ? "Keep your fellow travelers up to date."
            : "Share your plans and make room for another Rice student."}
        </p>
      </div>
      {!ride && profile.data?.user && !profile.data.user.phone && (
        <div className="notice-box">
          <Info size={21} />
          <span>
            Add a phone number to your <Link href="/profile">profile</Link>{" "}
            before posting. Riders need a way to reach you.
          </span>
        </div>
      )}
      <form className="form-card" onSubmit={(event) => void save(event)}>
        <div className="form-section-label">
          <MapPin size={19} />
          <span>Your route</span>
        </div>
        <div className="form-grid two-columns">
          <label className="field">
            <span>
              Leaving from <span className="required-mark">*</span>
            </span>
            <select
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
            </select>
          </label>
          <label className="field">
            <span>
              Going to <span className="required-mark">*</span>
            </span>
            <select
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
            </select>
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
        <div className="form-divider" />
        <div className="form-section-label">
          <span className="section-number">02</span>
          <span>The details</span>
        </div>
        <div className="form-grid two-columns">
          <label className="field">
            <span>
              Departure date and time <span className="required-mark">*</span>
            </span>
            <input
              type="datetime-local"
              required
              value={when}
              min={localDateTime(new Date())}
              onChange={(event) => setWhen(event.target.value)}
            />
          </label>
          <label className="field">
            <span>
              Total seats, including you{" "}
              <span className="required-mark">*</span>
            </span>
            <input
              type="number"
              min={minCapacity}
              step={1}
              required
              value={capacity}
              onChange={(event) => setCapacity(Number(event.target.value))}
            />
            <small>
              {ride
                ? `${ride.riders.length} people are already on this ride.`
                : "For example, 4 means you and 3 riders."}
            </small>
          </label>
        </div>
        <label className="field">
          <span>
            Notes for riders <span className="optional-label">Optional</span>
          </span>
          <textarea
            value={notes}
            onChange={(event) => setNotes(event.target.value)}
            maxLength={500}
            rows={4}
            placeholder="Where to meet, luggage space, or anything else helpful"
          />
          <small>{notes.length}/500 characters</small>
        </label>
        {error && (
          <p className="form-error" role="alert">
            {error}
          </p>
        )}
        <div className="form-actions">
          <button
            className="button button-primary"
            type="submit"
            disabled={
              isPending ||
              locations.isLoading ||
              (!ride && !profile.data?.user?.phone)
            }
          >
            {isPending ? "Saving…" : ride ? "Save ride" : "Post ride"}{" "}
            <ArrowRight size={17} />
          </button>
          <Link
            className="button button-quiet"
            href={ride ? `/rides/${ride.id}` : "/"}
          >
            Cancel
          </Link>
        </div>
      </form>
    </div>
  );
}
