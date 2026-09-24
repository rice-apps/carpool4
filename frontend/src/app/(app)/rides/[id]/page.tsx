"use client";

import { useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import {
  ArrowLeft,
  ArrowRight,
  CalendarBlank,
  CheckCircle,
  Clock,
  EnvelopeSimple,
  NotePencil,
  PencilSimple,
  Phone,
  Users,
} from "@phosphor-icons/react";
import {
  cancelRide,
  getRide,
  joinRide,
  leaveRide,
} from "../../../../gen/carpool/v1/ride-RideService_connectquery";
import { RideStatus } from "../../../../gen/carpool/v1/ride_pb";
import { useAuth } from "../../../../lib/auth";
import {
  availableSeats,
  formatDeparture,
  rideState,
} from "../../../../lib/ride";
import { requestErrorMessage } from "../../../../lib/requestError";

export default function RideDetailsPage() {
  const id = useParams<{ id: string }>()?.id || "";
  const { session } = useAuth();
  const queryClient = useQueryClient();
  const { data, error, isLoading, refetch } = useQuery(getRide, { id });
  const join = useMutation(joinRide);
  const leave = useMutation(leaveRide);
  const cancel = useMutation(cancelRide);
  const [actionError, setActionError] = useState("");
  const [actionSuccess, setActionSuccess] = useState("");

  async function act(action: "join" | "leave" | "cancel") {
    if (
      action === "cancel" &&
      !window.confirm(
        "Cancel this ride? Everyone on it will see that it was cancelled.",
      )
    )
      return;
    setActionError("");
    setActionSuccess("");
    try {
      if (action === "join") await join.mutateAsync({ rideId: id });
      if (action === "leave") await leave.mutateAsync({ rideId: id });
      if (action === "cancel") await cancel.mutateAsync({ id });
      await queryClient.invalidateQueries();
      setActionSuccess(
        action === "join"
          ? "You joined this ride."
          : action === "leave"
            ? "You left this ride."
            : "Ride cancelled.",
      );
    } catch (cause) {
      setActionError(
        requestErrorMessage(
          cause,
          action === "join"
            ? "Joining rides"
            : action === "leave"
              ? "Leaving rides"
              : "Cancelling rides",
          cause instanceof Error
            ? cause.message
            : "The ride couldn’t be updated. Try again.",
        ),
      );
    }
  }

  if (isLoading)
    return (
      <div className="page-container list-message" role="status">
        Loading ride…
      </div>
    );
  if (error)
    return (
      <div className="page-container list-message" role="alert">
        {requestErrorMessage(
          error,
          "Getting ride details",
          "Couldn’t load this ride.",
        )}{" "}
        <button onClick={() => void refetch()}>Try again</button>
      </div>
    );
  if (!data?.ride)
    return <div className="page-container list-message">Ride not found.</div>;

  const ride = data.ride;
  const state = rideState(ride);
  const isOwner = ride.owner?.id === session?.user.id;
  const isRider = ride.riders.some((rider) => rider.id === session?.user.id);
  const pending = join.isPending || leave.isPending || cancel.isPending;

  return (
    <div className="page-container detail-page">
      <Link href="/" className="back-link">
        <ArrowLeft size={17} /> Back to rides
      </Link>
      <div className="detail-header">
        <div>
          <span className="section-kicker">Ride details</span>
          <h1>
            {ride.departureLocation?.title || "Departure"} <span>to</span>{" "}
            {ride.arrivalLocation?.title || "Arrival"}
          </h1>
        </div>
        <span className={`status-pill status-${state}`}>
          {state === "open"
            ? `${availableSeats(ride)} ${availableSeats(ride) === 1 ? "seat" : "seats"} left`
            : state}
        </span>
      </div>
      <div className="detail-grid">
        <div className="detail-main">
          <section className="detail-card trip-card">
            <h2>The trip</h2>
            <div className="trip-route">
              <div className="trip-stop">
                <span className="trip-dot" />
                <div>
                  <small>Leaving from</small>
                  <strong>{ride.departureLocation?.title}</strong>
                  <span>{ride.departureLocation?.address}</span>
                </div>
              </div>
              <div className="trip-connector" />
              <div className="trip-stop">
                <span className="trip-dot trip-dot-end" />
                <div>
                  <small>Arriving at</small>
                  <strong>{ride.arrivalLocation?.title}</strong>
                  <span>{ride.arrivalLocation?.address}</span>
                </div>
              </div>
            </div>
            <div className="trip-facts">
              <span>
                <CalendarBlank size={20} />
                {formatDeparture(ride, {
                  weekday: "long",
                  month: "long",
                  day: "numeric",
                  year: "numeric",
                })}
              </span>
              <span>
                <Clock size={20} />
                {formatDeparture(ride, { hour: "numeric", minute: "2-digit" })}
              </span>
              <span>
                <Users size={20} />
                {ride.riders.length} of {ride.capacity} seats taken
              </span>
            </div>
          </section>
          {ride.notes && (
            <section className="detail-card notes-card">
              <h2>
                <NotePencil size={21} /> A note from the driver
              </h2>
              <p>{ride.notes}</p>
            </section>
          )}
          <section className="detail-card people-card">
            <h2>
              People on this ride <span>{ride.riders.length}</span>
            </h2>
            <div className="people-list">
              {ride.riders.map((rider) => (
                <div className="person-row" key={rider.id}>
                  <span className="person-avatar">
                    {rider.firstName?.[0]}
                    {rider.lastName?.[0]}
                  </span>
                  <div>
                    <strong>
                      {rider.firstName} {rider.lastName}
                    </strong>
                    <span>
                      {rider.id === ride.owner?.id ? "Driver" : "Rider"}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </section>
        </div>
        <aside className="detail-sidebar">
          <section className="detail-card driver-card">
            <span className="section-kicker">Your driver</span>
            <div className="driver-summary">
              <span className="driver-avatar">
                {ride.owner?.firstName?.[0]}
                {ride.owner?.lastName?.[0]}
              </span>
              <div>
                <strong>
                  {ride.owner?.firstName} {ride.owner?.lastName}
                </strong>
                <span>Rice student</span>
              </div>
            </div>
            {ride.owner?.phone || ride.owner?.email ? (
              <div className="driver-contact">
                {ride.owner?.phone && (
                  <p>
                    <Phone size={18} />
                    {ride.owner.phone}
                  </p>
                )}
                {ride.owner?.email && (
                  <p>
                    <EnvelopeSimple size={18} />
                    {ride.owner.email}
                  </p>
                )}
              </div>
            ) : (
              <p className="privacy-note">
                Contact details are visible when you share a ride with this driver.
              </p>
            )}
          </section>
          {ride.status !== RideStatus.CANCELLED && (
            <section className="action-card">
              <h2>
                {isOwner
                  ? "You’re driving"
                  : isRider
                    ? "You’re on this ride"
                    : "Come along?"}
              </h2>
              <p>
                {isOwner
                  ? "Edit the details or cancel if plans change."
                  : isRider
                    ? "Your seat is saved. You can leave if your plans change."
                    : state === "full"
                      ? "This ride is full right now."
                      : "Join to save a seat and see your driver’s contact details."}
              </p>
              {isOwner ? (
                <div className="action-stack">
                  <Link
                    href={`/rides/${id}/edit`}
                    className="button button-primary"
                  >
                    <PencilSimple size={18} /> Edit ride
                  </Link>
                  <button
                    className="button button-outline-danger"
                    disabled={pending}
                    onClick={() => void act("cancel")}
                  >
                    Cancel ride
                  </button>
                </div>
              ) : isRider ? (
                <button
                  className="button button-outline-danger"
                  disabled={pending}
                  onClick={() => void act("leave")}
                >
                  {pending ? "Leaving…" : "Leave ride"}
                </button>
              ) : (
                <button
                  className="button button-primary"
                  disabled={pending || state === "full" || state === "past"}
                  onClick={() => void act("join")}
                >
                  {pending
                    ? "Joining…"
                    : state === "full"
                      ? "Ride is full"
                      : "Join this ride"}{" "}
                  <ArrowRight size={17} />
                </button>
              )}
            </section>
          )}
          {actionError && (
            <p className="form-error" role="alert">
              {actionError}
            </p>
          )}
          {actionSuccess && (
            <p className="form-success" role="status">
              <CheckCircle size={18} />
              {actionSuccess}
            </p>
          )}
        </aside>
      </div>
    </div>
  );
}
