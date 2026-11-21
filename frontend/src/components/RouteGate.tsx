"use client";

import { useEffect, type ReactNode } from "react";
import { Code, ConnectError } from "@connectrpc/connect";
import { useQuery } from "@connectrpc/connect-query";
import { usePathname, useRouter } from "next/navigation";
import { getUser } from "../gen/carpool/v1/user-UserService_connectquery";
import { useAuth } from "../lib/auth";
import { isUnimplemented } from "../lib/requestError";
import { safeReturnTo } from "../lib/returnTo";
import { Button } from "./ui/button";

export function LoadingScreen() {
  return (
    <div className="loading-screen" role="status">
      Loading…
    </div>
  );
}

export function RequireAuth({ children }: { children: ReactNode }) {
  const { session, loading } = useAuth();
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    if (!loading && !session) {
      const next = encodeURIComponent(safeReturnTo(pathname));
      router.replace(`/login?next=${next}`);
    }
  }, [loading, session, router, pathname]);

  if (loading || !session) return <LoadingScreen />;
  return children;
}

function ProfileGate({ children }: { children: ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const { data, error, isLoading, refetch } = useQuery(getUser, {});
  const missing = error && ConnectError.from(error).code === Code.NotFound;
  const needsProfile = missing || (data && !data.user?.firstName);

  useEffect(() => {
    if (needsProfile) {
      const next = encodeURIComponent(safeReturnTo(pathname));
      router.replace(`/onboarding?next=${next}`);
    }
  }, [needsProfile, router, pathname]);

  if (isLoading || needsProfile) return <LoadingScreen />;
  // Other operations can still be used while profile lookup is being built.
  if (error && isUnimplemented(error)) return children;
  if (error)
    return (
      <div className="gate-error" role="alert">
        <h1>We couldn’t load your profile</h1>
        <p>{error.message}</p>
        <Button onClick={() => void refetch()}>Try again</Button>
      </div>
    );
  return children;
}

export function RequireProfile({ children }: { children: ReactNode }) {
  return (
    <RequireAuth>
      <ProfileGate>{children}</ProfileGate>
    </RequireAuth>
  );
}
