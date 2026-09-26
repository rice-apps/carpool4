import type { ReactNode } from "react";
import { RequireProfile } from "../../../components/RouteGate";

export default function MyRidesLayout({ children }: { children: ReactNode }) {
  return <RequireProfile>{children}</RequireProfile>;
}
