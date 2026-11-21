"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useQuery } from "@connectrpc/connect-query";
import {
  listLocations,
  listRides,
} from "../../gen/carpool/v1/ride-RideService_connectquery";
import { RideCard } from "../../components/RideCard";
import { Button } from "../../components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../components/ui/card";
import { NativeSelect } from "../../components/ui/native-select";
import { requestErrorMessage } from "../../lib/requestError";
import { useAuth } from "../../lib/auth";
import { returnToKey, safeReturnTo } from "../../lib/returnTo";

export default function HomePage() {
  const router = useRouter();
  const { session, loading } = useAuth();
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const locations = useQuery(listLocations, {});
  const rides = useQuery(listRides, {
    departureLocationId: from || undefined,
    arrivalLocationId: to || undefined,
  });

  useEffect(() => {
    if (loading || !session) return;
    const destination = safeReturnTo(sessionStorage.getItem(returnToKey));
    sessionStorage.removeItem(returnToKey);
    if (destination !== "/") router.replace(destination);
  }, [loading, session, router]);

  return (
    <div className="page-container interior-page">
      <div className="page-heading">
        <div>
          <h1>Find a ride</h1>
          <p>Browse upcoming trips.</p>
        </div>
        <Button asChild>
          <Link href="/rides/new">Post a ride</Link>
        </Button>
      </div>
      <Card className="shadow-none">
        <CardHeader>
          <CardTitle>
            <h2>Filter rides</h2>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <form
            className="space-y-4"
            onSubmit={(event) => event.preventDefault()}
          >
            <div className="form-grid two-columns">
              <label className="field">
                <span>Leaving from</span>
                <NativeSelect
                  value={from}
                  onChange={(event) => setFrom(event.target.value)}
                  disabled={locations.isLoading}
                >
                  <option value="">Anywhere</option>
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
                  value={to}
                  onChange={(event) => setTo(event.target.value)}
                  disabled={locations.isLoading}
                >
                  <option value="">Anywhere</option>
                  {locations.data?.locations.map((location) => (
                    <option value={location.id} key={location.id}>
                      {location.title}
                    </option>
                  ))}
                </NativeSelect>
              </label>
            </div>
            <Button
              variant="ghost"
              size="sm"
              type="button"
              onClick={() => {
                setFrom(to);
                setTo(from);
              }}
            >
              Swap departure and arrival
            </Button>
          </form>
          {locations.error && (
            <p className="inline-error" role="alert">
              {requestErrorMessage(
                locations.error,
                "Getting locations",
                "Couldn’t load locations.",
              )}{" "}
              <button onClick={() => void locations.refetch()}>
                Try again
              </button>
            </p>
          )}
        </CardContent>
      </Card>
      <section className="mt-6" aria-labelledby="available-rides">
        <div className="mb-4 flex items-center justify-between gap-3">
          <h2 id="available-rides">Available rides</h2>
          {rides.data && (
            <span className="text-sm text-muted-foreground">
              {rides.data.rides.length}{" "}
              {rides.data.rides.length === 1 ? "ride" : "rides"}
            </span>
          )}
        </div>
        {rides.isLoading ? (
          <div className="list-message" role="status">
            Loading rides…
          </div>
        ) : rides.error ? (
          <div className="list-message" role="alert">
            <p>
              {requestErrorMessage(
                rides.error,
                "Getting rides",
                "Couldn’t load rides.",
              )}
            </p>
            <Button variant="outline" onClick={() => void rides.refetch()}>
              Try again
            </Button>
          </div>
        ) : rides.data?.rides.length ? (
          <div className="ride-list">
            {rides.data.rides.map((ride) => (
              <RideCard key={ride.id} ride={ride} />
            ))}
          </div>
        ) : (
          <div className="empty-state">
            <h2>No rides found</h2>
            <p>Try another route, or post your trip.</p>
            <Button variant="outline" asChild>
              <Link href="/rides/new">Post a ride</Link>
            </Button>
          </div>
        )}
      </section>
    </div>
  );
}
