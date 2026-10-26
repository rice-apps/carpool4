"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { Button } from "../../components/ui/button";
import { Card, CardContent, CardHeader } from "../../components/ui/card";
import { Brand } from "../../components/AppShell";
import { useAuth } from "../../lib/auth";
import { supabase } from "../../lib/supabase";
import { returnToKey, safeReturnTo } from "../../lib/returnTo";

export default function LoginPage() {
  const router = useRouter();
  const { session, loading } = useAuth();
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!loading && session) {
      const next = new URLSearchParams(window.location.search).get("next");
      router.replace(safeReturnTo(next));
    }
  }, [loading, session, router]);

  async function signIn() {
    setPending(true);
    setError("");
    sessionStorage.setItem(
      returnToKey,
      safeReturnTo(new URLSearchParams(window.location.search).get("next")),
    );
    const { error } = await supabase.auth.signInWithOAuth({
      provider: "google",
      options: {
        redirectTo: window.location.origin,
        queryParams: { hd: "rice.edu" },
      },
    });
    if (error) {
      setError(error.message);
      setPending(false);
    }
  }

  if (loading || session) return null;
  return (
    <main className="mx-auto max-w-md px-4 py-12">
      <div className="mb-8">
        <Brand />
      </div>
      <Card className="shadow-none">
        <CardHeader>
          <h1>Sign in</h1>
          <p className="text-muted-foreground">
            Use your Rice Google account to post or join a ride.
          </p>
        </CardHeader>
        <CardContent className="space-y-5">
          <Button
            className="w-full"
            onClick={() => void signIn()}
            disabled={pending}
          >
            {pending ? "Opening Google…" : "Continue with Rice Google"}
          </Button>
          {error && (
            <p className="form-error" role="alert">
              {error}
            </p>
          )}
          <p className="text-sm text-muted-foreground">
            A Rice-managed Google account is required.
          </p>
          <Link className="text-link" href="/">
            Browse rides without signing in
          </Link>
        </CardContent>
      </Card>
    </main>
  );
}
