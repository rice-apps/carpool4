"use client";

import { useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import { Code, ConnectError } from "@connectrpc/connect";
import { useQueryClient } from "@tanstack/react-query";
import { ArrowRight, CheckCircle, SignOut } from "@phosphor-icons/react";
import {
  getUser,
  updateUser,
} from "../gen/carpool/v1/user-UserService_connectquery";
import { type User } from "../gen/carpool/v1/user_pb";
import { useAuth } from "../lib/auth";
import { requestErrorMessage } from "../lib/requestError";
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
        <button
          className="button button-primary"
          onClick={() => void profile.refetch()}
        >
          Try again
        </button>
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
      if (onboarding) router.replace("/");
      else setSaved(true);
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
        <span className="section-kicker">
          {onboarding ? "One quick introduction" : "Your account"}
        </span>
        <h1>{onboarding ? "Let’s get to know you." : "Your profile"}</h1>
        <p>
          {onboarding
            ? "Add your name to meet fellow travelers. A phone number is needed when you post a ride."
            : "Keep your contact details current for the people you ride with."}
        </p>
      </div>
      <form className="form-card" onSubmit={(event) => void save(event)}>
        <div className="form-grid two-columns">
          <label className="field">
            <span>
              First name <span className="required-mark">*</span>
            </span>
            <input
              value={firstName}
              onChange={(event) => setFirstName(event.target.value)}
              required
              maxLength={100}
              autoComplete="given-name"
              placeholder="Your first name"
            />
          </label>
          <label className="field">
            <span>
              Last name <span className="required-mark">*</span>
            </span>
            <input
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
          <input
            value={session?.user.email || ""}
            readOnly
            className="read-only-input"
          />
          <small>Your email comes from your Rice Google account.</small>
        </label>
        <label className="field">
          <span>Phone number</span>
          <input
            type="tel"
            value={phone}
            onChange={(event) => setPhone(event.target.value)}
            maxLength={32}
            autoComplete="tel"
            placeholder="(713) 555-0123"
          />
          <small>
            Required to post a ride. Only people who share a ride with you can
            see it.
          </small>
        </label>
        {error && (
          <p className="form-error" role="alert">
            {error}
          </p>
        )}
        {saved && (
          <p className="form-success" role="status">
            <CheckCircle size={18} /> Profile saved.
          </p>
        )}
        <div className="form-actions">
          <button
            className="button button-primary"
            type="submit"
            disabled={update.isPending}
          >
            {update.isPending
              ? "Saving…"
              : onboarding
                ? "Finish profile"
                : "Save changes"}{" "}
            <ArrowRight size={17} />
          </button>
          {!onboarding && (
            <button
              className="button button-quiet"
              type="button"
              onClick={() => void leave()}
            >
              <SignOut size={18} /> Sign out
            </button>
          )}
        </div>
      </form>
    </div>
  );
}
