"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useQuery } from "@connectrpc/connect-query";
import {
  ArrowRight,
  ArrowsDownUp,
  MagnifyingGlass,
  Plus,
} from "@phosphor-icons/react";
import {
  listLocations,
  listRides,
} from "../../gen/carpool/v1/ride-RideService_connectquery";
import { RideCard } from "../../components/RideCard";
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
    <>
      <section className="discovery-hero">
        <div className="page-container hero-inner">
          <div className="hero-copy">
            <span className="hero-tag">
              <span className="hero-tag-dot" />
              Your next trip, together
            </span>
            <h1>
              Find your way
              <br />
              <em>there, together.</em>
            </h1>
            <p>
              From campus to the airport and everywhere in between. Find a seat
              with another Rice student.
            </p>
          </div>
          <div className="hero-graphic" aria-hidden="true">
            <div className="hero-path">
              <span className="hero-path-dot" />
              <span className="hero-path-dot" />
            </div>
            <span className="hero-place hero-place-one">Rice</span>
            <span className="hero-place hero-place-two">Your next stop</span>
          </div>
        </div>
      </section>

      <div className="page-container discover-content">
        <form
          className="search-panel"
          onSubmit={(event) => event.preventDefault()}
        >
          <div className="search-label">
            <MagnifyingGlass size={19} />
            <span>Find a ride</span>
          </div>
          <label className="field">
            <span>Leaving from</span>
            <select
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
            </select>
          </label>
          <button
            className="swap-button"
            type="button"
            aria-label="Swap departure and arrival"
            onClick={() => {
              setFrom(to);
              setTo(from);
            }}
          >
            <ArrowsDownUp size={19} />
          </button>
          <label className="field">
            <span>Going to</span>
            <select
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
            </select>
          </label>
        </form>
        {locations.error && (
          <p className="inline-error" role="alert">
            {requestErrorMessage(
              locations.error,
              "Getting locations",
              "Couldn’t load locations.",
            )}{" "}
            <button onClick={() => void locations.refetch()}>Try again</button>
          </p>
        )}

        <div className="discovery-grid">
          <section className="ride-list-section">
            <div className="section-heading">
              <div>
                <span className="section-kicker">On the road</span>
                <h2>Available rides</h2>
              </div>
              <span className="result-count">
                {rides.data
                  ? `${rides.data.rides.length} ${rides.data.rides.length === 1 ? "ride" : "rides"}`
                  : ""}
              </span>
            </div>
            {rides.isLoading ? (
              <div className="list-message" role="status">
                Looking for rides…
              </div>
            ) : rides.error ? (
              <div className="list-message" role="alert">
                {requestErrorMessage(
                  rides.error,
                  "Getting rides",
                  "Couldn’t load rides.",
                )}{" "}
                <button onClick={() => void rides.refetch()}>Try again</button>
              </div>
            ) : rides.data?.rides.length ? (
              <div className="ride-list">
                {rides.data.rides.map((ride) => (
                  <RideCard key={ride.id} ride={ride} />
                ))}
              </div>
            ) : (
              <div className="empty-state">
                <span className="empty-icon">
                  <MagnifyingGlass size={28} />
                </span>
                <h3>No rides found yet</h3>
                <p>Try another route, or post the trip you have planned.</p>
                <Link className="text-link" href="/rides/new">
                  Post a ride <ArrowRight size={17} />
                </Link>
              </div>
            )}
          </section>
          <aside className="discover-aside">
            <div className="share-card">
              <span className="share-symbol">
                <Plus size={26} />
              </span>
              <h3>Have room in your ride?</h3>
              <p>
                A spare seat can make someone’s day. Share where you’re headed.
              </p>
              <Link className="button button-light" href="/rides/new">
                Post a ride <ArrowRight size={17} />
              </Link>
            </div>
            <p className="aside-note">
              Rides are shared by Rice students. Connect directly with your
              driver after you join.
            </p>
          </aside>
        </div>
      </div>
    </>
  );
}
