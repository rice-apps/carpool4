"use client";

import { useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
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
import { RequireProfile } from "../../../../components/RouteGate";
import { Badge } from "../../../../components/ui/badge";
import { Button } from "../../../../components/ui/button";
import { Card, CardContent, CardHeader } from "../../../../components/ui/card";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "../../../../components/ui/alert-dialog";

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
        <p>
          {requestErrorMessage(
            error,
            "Getting ride details",
            "Couldn’t load this ride.",
          )}
        </p>
        <Button variant="outline" onClick={() => void refetch()}>
          Try again
        </Button>
      </div>
    );
  if (!data?.ride)
    return <div className="page-container list-message">Ride not found.</div>;

  const ride = data.ride;
  const state = rideState(ride);
  const seats = availableSeats(ride);
  const isOwner = ride.owner?.id === session?.user.id;
  const isRider = ride.riders.some((rider) => rider.id === session?.user.id);
  const pending = join.isPending || leave.isPending || cancel.isPending;

  return (
    <div className="page-container detail-page">
      <Link href="/" className="back-link">
        Back to rides
      </Link>
      <div className="page-heading">
        <h1>
          {ride.departureLocation?.title || "Departure"} to{" "}
          {ride.arrivalLocation?.title || "Arrival"}
        </h1>
        <Badge variant={state === "cancelled" ? "outline" : "secondary"}>
          {state === "open"
            ? `${seats} ${seats === 1 ? "seat" : "seats"} left`
            : state}
        </Badge>
      </div>
      <div className="grid gap-5">
        <Card className="shadow-none">
          <CardHeader>
            <h2>Trip details</h2>
          </CardHeader>
          <CardContent>
            <dl className="grid gap-5 sm:grid-cols-2">
              <div>
                <dt className="text-sm text-muted-foreground">Leaving from</dt>
                <dd className="mt-1 font-medium">
                  {ride.departureLocation?.title}
                </dd>
                <dd className="text-sm text-muted-foreground">
                  {ride.departureLocation?.address}
                </dd>
              </div>
              <div>
                <dt className="text-sm text-muted-foreground">Going to</dt>
                <dd className="mt-1 font-medium">
                  {ride.arrivalLocation?.title}
                </dd>
                <dd className="text-sm text-muted-foreground">
                  {ride.arrivalLocation?.address}
                </dd>
              </div>
              <div>
                <dt className="text-sm text-muted-foreground">Departure</dt>
                <dd className="mt-1 font-medium">
                  {formatDeparture(ride, {
                    weekday: "long",
                    month: "long",
                    day: "numeric",
                    year: "numeric",
                    hour: "numeric",
                    minute: "2-digit",
                  })}
                </dd>
              </div>
              <div>
                <dt className="text-sm text-muted-foreground">Seats</dt>
                <dd className="mt-1 font-medium">
                  {ride.occupiedSeats} of {ride.capacity} seats taken, including
                  the driver
                </dd>
              </div>
            </dl>
          </CardContent>
        </Card>
        {session && ride.notes && (
          <Card className="shadow-none">
            <CardHeader>
              <h2>Notes from the driver</h2>
            </CardHeader>
            <CardContent>
              <p className="whitespace-pre-wrap break-words">{ride.notes}</p>
            </CardContent>
          </Card>
        )}
        {session && (
          <>
            <Card className="shadow-none">
              <CardHeader>
                <h2>People on this ride ({ride.riders.length})</h2>
              </CardHeader>
              <CardContent>
                <ul className="divide-y">
                  {ride.riders.map((rider) => (
                    <li
                      className="flex flex-wrap justify-between gap-2 py-3 first:pt-0 last:pb-0"
                      key={rider.id}
                    >
                      <span className="break-words">
                        {rider.firstName} {rider.lastName}
                      </span>
                      <span className="text-sm text-muted-foreground">
                        {rider.id === ride.owner?.id ? "Driver" : "Rider"}
                      </span>
                    </li>
                  ))}
                </ul>
              </CardContent>
            </Card>
            <Card className="shadow-none">
              <CardHeader>
                <h2>Driver contact</h2>
              </CardHeader>
              <CardContent className="space-y-2 break-words">
                <p className="font-medium">
                  {ride.owner?.firstName} {ride.owner?.lastName}
                </p>
                {ride.owner?.phone || ride.owner?.email ? (
                  <>
                    {ride.owner.phone && <p>{ride.owner.phone}</p>}
                    {ride.owner.email && <p>{ride.owner.email}</p>}
                  </>
                ) : (
                  <p className="text-muted-foreground">
                    Contact details are visible when you share a ride with this
                    driver.
                  </p>
                )}
              </CardContent>
            </Card>
          </>
        )}
        {!session && state === "open" && (
          <Card className="shadow-none">
            <CardHeader>
              <h2>Join this ride</h2>
              <p className="text-muted-foreground">
                Sign in with your Rice account to join this ride.
              </p>
            </CardHeader>
            <CardContent>
              <Button asChild>
                <Link
                  href={`/login?next=${encodeURIComponent(`/rides/${id}`)}`}
                >
                  Sign in to join
                </Link>
              </Button>
            </CardContent>
          </Card>
        )}
        {session && ride.status !== RideStatus.CANCELLED && (
          <RequireProfile>
            <Card className="shadow-none">
              <CardHeader>
                <h2>
                  {isOwner
                    ? "You’re driving"
                    : isRider
                      ? "You’re on this ride"
                      : "Join this ride"}
                </h2>
                <p className="text-muted-foreground">
                  {isOwner
                    ? "Edit the details or cancel if plans change."
                    : isRider
                      ? "Your seat is saved. You can leave if your plans change."
                      : state === "full"
                        ? "This ride is full."
                        : state === "past"
                          ? "This ride has departed."
                          : "Join to save a seat and see your driver’s contact details."}
                </p>
              </CardHeader>
              <CardContent>
                {isOwner ? (
                  <div className="form-actions">
                    <Button asChild>
                      <Link href={`/rides/${id}/edit`}>Edit ride</Link>
                    </Button>
                    <AlertDialog>
                      <AlertDialogTrigger asChild>
                        <Button variant="outline" disabled={pending}>
                          Cancel ride
                        </Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>Cancel this ride?</AlertDialogTitle>
                          <AlertDialogDescription>
                            Everyone on it will see that it was cancelled.
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>Keep ride</AlertDialogCancel>
                          <AlertDialogAction
                            variant="destructive"
                            disabled={pending}
                            onClick={() => void act("cancel")}
                          >
                            Cancel ride
                          </AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  </div>
                ) : isRider ? (
                  <Button
                    variant="outline"
                    disabled={pending}
                    onClick={() => void act("leave")}
                  >
                    {pending ? "Leaving…" : "Leave ride"}
                  </Button>
                ) : (
                  <Button
                    disabled={pending || state === "full" || state === "past"}
                    onClick={() => void act("join")}
                  >
                    {pending
                      ? "Joining…"
                      : state === "full"
                        ? "Ride is full"
                        : "Join this ride"}
                  </Button>
                )}
              </CardContent>
            </Card>
          </RequireProfile>
        )}
        {actionError && (
          <p className="form-error" role="alert">
            {actionError}
          </p>
        )}
        {actionSuccess && (
          <p className="form-success" role="status">
            {actionSuccess}
          </p>
        )}
      </div>
    </div>
  );
}
