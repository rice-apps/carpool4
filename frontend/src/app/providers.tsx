"use client";

import { useState, type ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { TransportProvider } from "@connectrpc/connect-query";
import { AuthProvider, useAuth } from "../lib/auth";
import { isUnimplemented } from "../lib/requestError";
import { transport } from "../lib/transport";

export function Providers({ children }: { children: ReactNode }) {
  return (
    <AuthProvider>
      <SessionData>{children}</SessionData>
    </AuthProvider>
  );
}

function SessionData({ children }: { children: ReactNode }) {
  const { session } = useAuth();
  return (
    <DataProviders key={session?.user.id || "signed-out"}>
      {children}
    </DataProviders>
  );
}

function DataProviders({ children }: { children: ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 20_000,
            retry: (failures, error) => failures < 1 && !isUnimplemented(error),
          },
        },
      }),
  );

  return (
    <TransportProvider transport={transport}>
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    </TransportProvider>
  );
}
