"use client";

import { useState } from "react";
import Link from "next/link";
import { useQuery } from "@connectrpc/connect-query";
import { listMyRides } from "../../../gen/carpool/v1/ride-RideService_connectquery";
import { RideCard } from "../../../components/RideCard";
import { Button } from "../../../components/ui/button";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "../../../components/ui/tabs";
import { rideState } from "../../../lib/ride";
import { requestErrorMessage } from "../../../lib/requestError";

export default function MyRidesPage() {
  const [view, setView] = useState("upcoming");
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
          <h1>My rides</h1>
          <p>Rides you have posted or joined.</p>
        </div>
        <Button asChild>
          <Link href="/rides/new">Post a ride</Link>
        </Button>
      </div>
      <Tabs value={view} onValueChange={setView}>
        <TabsList aria-label="Ride history">
          <TabsTrigger value="upcoming">Upcoming</TabsTrigger>
          <TabsTrigger value="history">Past &amp; cancelled</TabsTrigger>
        </TabsList>
        <TabsContent value={view} className="mt-4">
          {isLoading ? (
            <div className="list-message" role="status">
              Loading your rides…
            </div>
          ) : error ? (
            <div className="list-message" role="alert">
              <p>
                {requestErrorMessage(
                  error,
                  "Getting your rides",
                  "Couldn’t load your rides.",
                )}
              </p>
              <Button variant="outline" onClick={() => void refetch()}>
                Try again
              </Button>
            </div>
          ) : rides.length ? (
            <div className="ride-list">
              {rides.map((ride) => (
                <RideCard key={ride.id} ride={ride} />
              ))}
            </div>
          ) : (
            <div className="empty-state">
              <h2>
                {view === "upcoming"
                  ? "No upcoming rides"
                  : "No past rides yet"}
              </h2>
              <p>
                {view === "upcoming"
                  ? "Find a ride or post your own trip."
                  : "Your past and cancelled rides will appear here."}
              </p>
              {view === "upcoming" && (
                <Button variant="outline" asChild>
                  <Link href="/">Find a ride</Link>
                </Button>
              )}
            </div>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}
