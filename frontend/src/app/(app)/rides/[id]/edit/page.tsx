"use client";

import { useParams } from "next/navigation";
import { useQuery } from "@connectrpc/connect-query";
import { getRide } from "../../../../../gen/carpool/v1/ride-RideService_connectquery";
import { RideForm } from "../../../../../components/RideForm";
import { useAuth } from "../../../../../lib/auth";
import { RideStatus } from "../../../../../gen/carpool/v1/ride_pb";
import { requestErrorMessage } from "../../../../../lib/requestError";

export default function EditRidePage() {
  const id = useParams<{ id: string }>()?.id || "";
  const { session } = useAuth();
  const { data, error, isLoading, refetch } = useQuery(getRide, { id });
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
  if (
    data.ride.owner?.id !== session?.user.id ||
    data.ride.status === RideStatus.CANCELLED
  )
    return (
      <div className="page-container list-message">
        This ride can’t be edited.
      </div>
    );
  return <RideForm ride={data.ride} />;
}
