"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { ArrowRight, GoogleLogo } from "@phosphor-icons/react";
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
    <main className="login-layout">
      <section className="login-panel">
        <Brand />
        <div className="login-copy">
          <span className="section-kicker">For the Rice community</span>
          <h1>
            Going somewhere?
            <br />
            <em>Go together.</em>
          </h1>
          <p>
            Catch a ride with someone headed your way, or share the seats in
            yours.
          </p>
          <button
            className="button button-primary login-button"
            onClick={() => void signIn()}
            disabled={pending}
          >
            <GoogleLogo size={21} weight="bold" />
            {pending ? "Opening Google…" : "Continue with Rice Google"}
            <ArrowRight size={19} />
          </button>
          {error && (
            <p className="form-error" role="alert">
              {error}
            </p>
          )}
          <p className="login-footnote">
            A Rice-managed Google account is required.
          </p>
          <Link className="text-link" href="/">
            Browse rides without signing in
          </Link>
        </div>
        <span className="login-footer">Rice rides together.</span>
      </section>
      <section className="login-art" aria-hidden="true">
        <div className="art-orbit art-orbit-one" />
        <div className="art-orbit art-orbit-two" />
        <div className="art-sticker art-sticker-top">
          RICE UNIVERSITY <span>●</span> HOUSTON, TX
        </div>
        <div className="art-card">
          <span className="art-card-label">A little less driving alone</span>
          <div className="art-route">
            <span className="art-stop" />
            <span className="art-route-line" />
            <span className="art-stop art-stop-end" />
          </div>
          <div className="art-places">
            <span>Campus</span>
            <span>Everywhere</span>
          </div>
        </div>
        <div className="art-sticker art-sticker-bottom">
          SAME DIRECTION, BETTER COMPANY
        </div>
      </section>
    </main>
  );
}
