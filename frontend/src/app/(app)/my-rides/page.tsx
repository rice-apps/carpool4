"use client";

import { useState } from "react";
import Link from "next/link";
import { useQuery } from "@connectrpc/connect-query";
import { ArrowRight, CarProfile } from "@phosphor-icons/react";
import { listMyRides } from "../../../gen/carpool/v1/ride-RideService_connectquery";
import { RideCard } from "../../../components/RideCard";
import { rideState } from "../../../lib/ride";
import { requestErrorMessage } from "../../../lib/requestError";

export default function MyRidesPage() {
  const [view, setView] = useState<"upcoming" | "history">("upcoming");
  const { data, error, isLoading, refetch } = useQuery(listMyRides, {});
  const rides =
    data?.rides.filter((ride) =>
      view === "upcoming"
        ? !["past", "cancelled"].includes(rideState(ride))
        : ["past", "cancelled"].includes(rideState(ride)),
    ) || [];

  return (
    <div className="page-container interior-page">
      <div className="page-heading">
        <div>
          <span className="section-kicker">Your journeys</span>
          <h1>My rides</h1>
          <p>Every ride you’ve posted or joined, all in one place.</p>
        </div>
        <Link className="button button-primary" href="/rides/new">
          Post a ride <ArrowRight size={18} />
        </Link>
      </div>
      <div className="tabs" role="tablist" aria-label="Ride history">
        <button
          role="tab"
          aria-selected={view === "upcoming"}
          className={view === "upcoming" ? "is-active" : ""}
          onClick={() => setView("upcoming")}
        >
          Upcoming
        </button>
        <button
          role="tab"
          aria-selected={view === "history"}
          className={view === "history" ? "is-active" : ""}
          onClick={() => setView("history")}
        >
          Past & cancelled
        </button>
      </div>
      {isLoading ? (
        <div className="list-message" role="status">
          Loading your rides…
        </div>
      ) : error ? (
        <div className="list-message" role="alert">
          {requestErrorMessage(
            error,
            "Getting your rides",
            "Couldn’t load your rides.",
          )}{" "}
          <button onClick={() => void refetch()}>Try again</button>
        </div>
      ) : rides.length ? (
        <div className="ride-list my-ride-list">
          {rides.map((ride) => (
            <RideCard key={ride.id} ride={ride} />
          ))}
        </div>
      ) : (
        <div className="empty-state">
          <span className="empty-icon">
            <CarProfile size={30} />
          </span>
          <h3>
            {view === "upcoming" ? "No upcoming rides" : "No past rides yet"}
          </h3>
          <p>
            {view === "upcoming"
              ? "Find a seat or share a trip to get moving."
              : "Your completed and cancelled rides will appear here."}
          </p>
          {view === "upcoming" && (
            <Link className="text-link" href="/">
              Find a ride <ArrowRight size={17} />
            </Link>
          )}
        </div>
      )}
    </div>
  );
}
