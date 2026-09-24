"use client";

import { RequireAuth } from "../../components/RouteGate";
import { ProfileForm } from "../../components/ProfileForm";

export default function OnboardingPage() {
  return (
    <RequireAuth>
      <div className="onboarding-page">
        <ProfileForm onboarding />
      </div>
    </RequireAuth>
  );
}
