"use client";

import type { ReactNode } from "react";
import { AppShell } from "../../components/AppShell";
import { RequireProfile } from "../../components/RouteGate";

export default function AppLayout({ children }: { children: ReactNode }) {
  return (
    <RequireProfile>
      <AppShell>{children}</AppShell>
    </RequireProfile>
  );
}
