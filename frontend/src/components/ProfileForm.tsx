"use client";

import { useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import { Code, ConnectError } from "@connectrpc/connect";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "./ui/button";
import { Card, CardContent } from "./ui/card";
import { Input } from "./ui/input";
import {
  getUser,
  updateUser,
} from "../gen/carpool/v1/user-UserService_connectquery";
import { type User } from "../gen/carpool/v1/user_pb";
import { useAuth } from "../lib/auth";
import { requestErrorMessage } from "../lib/requestError";
import { safeReturnTo } from "../lib/returnTo";
import { Brand } from "./AppShell";

export function ProfileForm({ onboarding = false }: { onboarding?: boolean }) {
  const profile = useQuery(getUser, {});
  if (profile.isLoading)
    return (
      <div className="loading-screen" role="status">
        Loading your profile…
      </div>
    );
  if (profile.error && ConnectError.from(profile.error).code !== Code.NotFound)
    return (
      <div className="gate-error" role="alert">
        <h1>
          {requestErrorMessage(
            profile.error,
            "Getting your profile",
            "We couldn’t load your profile",
          )}
        </h1>
        <Button onClick={() => void profile.refetch()}>Try again</Button>
      </div>
    );
  return <ProfileFields onboarding={onboarding} initial={profile.data?.user} />;
}

function ProfileFields({
  onboarding,
  initial,
}: {
  onboarding: boolean;
  initial?: User;
}) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { session, signOut } = useAuth();
  const update = useMutation(updateUser);
  const [firstName, setFirstName] = useState(initial?.firstName || "");
  const [lastName, setLastName] = useState(initial?.lastName || "");
  const [phone, setPhone] = useState(initial?.phone || "");
  const [error, setError] = useState("");
  const [saved, setSaved] = useState(false);

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setSaved(false);
    if (!firstName.trim() || !lastName.trim()) {
      setError("Enter your first and last name.");
      return;
    }
    try {
      await update.mutateAsync({
        firstName: firstName.trim(),
        lastName: lastName.trim(),
        phone: phone.trim(),
      });
      await queryClient.invalidateQueries();
      if (onboarding) {
        const next = new URLSearchParams(window.location.search).get("next");
        router.replace(safeReturnTo(next));
      } else {
        setSaved(true);
      }
    } catch (cause) {
      setError(
        requestErrorMessage(
          cause,
          "Saving your profile",
          cause instanceof Error
            ? cause.message
            : "Couldn’t save your profile. Try again.",
        ),
      );
    }
  }

  async function leave() {
    try {
      await signOut();
      queryClient.clear();
      router.replace("/login");
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "Couldn’t sign out. Try again.",
      );
    }
  }

  return (
    <div className={`profile-wrap ${onboarding ? "profile-onboarding" : ""}`}>
      {onboarding && (
        <div className="onboarding-brand">
          <Brand />
        </div>
      )}
      <div className="form-intro">
        <h1>{onboarding ? "Complete your profile" : "Your profile"}</h1>
        <p>
          {onboarding
            ? "Add your name to post or join a ride."
            : "Manage your name and contact details."}
        </p>
      </div>
      <Card className="shadow-none">
        <CardContent>
          <form className="space-y-6" onSubmit={(event) => void save(event)}>
            <div className="form-grid two-columns">
              <label className="field">
                <span>First name</span>
                <Input
                  value={firstName}
                  onChange={(event) => setFirstName(event.target.value)}
                  required
                  maxLength={100}
                  autoComplete="given-name"
                  placeholder="Your first name"
                />
              </label>
              <label className="field">
                <span>Last name</span>
                <Input
                  value={lastName}
                  onChange={(event) => setLastName(event.target.value)}
                  required
                  maxLength={100}
                  autoComplete="family-name"
                  placeholder="Your last name"
                />
              </label>
            </div>
            <label className="field">
              <span>Rice email</span>
              <Input
                value={session?.user.email || ""}
                readOnly
                className="bg-muted"
                aria-describedby="email-help"
              />
              <small id="email-help">
                Your email comes from your Rice Google account.
              </small>
            </label>
            <label className="field">
              <span>Phone number</span>
              <Input
                type="tel"
                value={phone}
                onChange={(event) => setPhone(event.target.value)}
                maxLength={32}
                autoComplete="tel"
                placeholder="(713) 555-0123"
                aria-describedby="phone-help"
              />
              <small id="phone-help">
                Required to post a ride. Only people who share a ride with you
                can see it.
              </small>
            </label>
            {error && (
              <p className="form-error" role="alert">
                {error}
              </p>
            )}
            {saved && (
              <p className="form-success" role="status">
                Profile saved.
              </p>
            )}
            <div className="form-actions">
              <Button type="submit" disabled={update.isPending}>
                {update.isPending
                  ? "Saving…"
                  : onboarding
                    ? "Finish profile"
                    : "Save changes"}
              </Button>
              {!onboarding && (
                <Button
                  variant="outline"
                  type="button"
                  onClick={() => void leave()}
                >
                  Sign out
                </Button>
              )}
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
